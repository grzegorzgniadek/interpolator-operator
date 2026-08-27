/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"text/template"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sprig "github.com/Masterminds/sprig/v3"
	interpolatorv1 "github.com/grzegorzgniadek/interpolator-operator/api/v1"
)

// InterpolatorReconciler reconciles a Interpolator object
type InterpolatorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=interpolator.io,resources=interpolators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=interpolator.io,resources=interpolators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=interpolator.io,resources=interpolators/finalizers,verbs=update
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Interpolator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile

const (
	typeAvailableInterpolator = "Synced"
	typeDegradedInterpolator  = "Degraded"
	interpolatorFinalizer     = "interpolator.interpolator.io/finalizer"
	resourceTypeSecret        = "Secret"
	resourceTypeConfigMap     = "ConfigMap"
)

func (r *InterpolatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("interpolator", req.NamespacedName)
	logger.Info("starting reconciliation for Interpolator", "NamespacedName", req.NamespacedName)

	// Fetch the Interpolator instance
	interpolator := &interpolatorv1.Interpolator{}
	interpolatorList := &interpolatorv1.InterpolatorList{}
	now := metav1.Now()

	if err := r.List(ctx, interpolatorList); err != nil {
		logger.Error(err, "failed to list custom resources")
	}
	count := len(interpolatorList.Items)
	interCount.Set(float64(count))

	if err := r.Get(ctx, req.NamespacedName, interpolator); err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then, it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			logger.Info("Interpolator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		logger.Error(err, "unable to fetch Interpolator")
		return ctrl.Result{}, err
	}

	if len(interpolator.Status.Conditions) == 0 {
		meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeDegradedInterpolator, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		logger.Info("adding status condition reconciling")
		if err := r.Status().Update(ctx, interpolator); err != nil {
			logger.Error(err, "failed to update Interpolator status-Reconcilling")
			return ctrl.Result{}, nil
		}
	}

	if !interpolator.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(interpolator, interpolatorFinalizer) {
			logger.Info("performing finalizer operations for Interpolator before delete CR")
			r.doFinalizerOperationsForInterpolator(interpolator)

			logger.Info("removing finalizer for Interpolator after successful performed operations")
			controllerutil.RemoveFinalizer(interpolator, interpolatorFinalizer)
			if err := r.Update(ctx, interpolator); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(interpolator, interpolatorFinalizer) {
		logger.Info("adding finalizer to Interpolator")
		controllerutil.AddFinalizer(interpolator, interpolatorFinalizer)
		if err := r.Update(ctx, interpolator); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Interpolation logic
	FinalSecrets := make(map[string][]byte)
	FinalSecretsString := make(map[string]string)

	secretValues := make(map[string]string)

	logger.Info("fetching and processing input secrets")
	for i, inputsecret := range interpolator.Spec.InputSecrets {
		switch inputsecret.Kind {
		case resourceTypeSecret:
			secrets := &v1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: inputsecret.Namespace, Name: inputsecret.Name}, secrets); err != nil {
				logger.Error(err, "failed to get data from secret", "name", inputsecret.Name, "namespace", inputsecret.Namespace)
				return ctrl.Result{}, err
			}
			secretValues[inputsecret.Key] = string(secrets.Data[inputsecret.Key])
			interpolator.Spec.InputSecrets[i].Value = string(secrets.Data[inputsecret.Key])
		case resourceTypeConfigMap:
			configmaps := &v1.ConfigMap{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: inputsecret.Namespace, Name: inputsecret.Name}, configmaps); err != nil {
				logger.Error(err, "failed to get data from configmap", "name", inputsecret.Name, "namespace", inputsecret.Namespace)
				return ctrl.Result{}, err
			}
			secretValues[inputsecret.Key] = configmaps.Data[inputsecret.Key]
			interpolator.Spec.InputSecrets[i].Value = configmaps.Data[inputsecret.Key]
		}
	}

	logger.Info("interpolating secrets")
	for _, secret := range interpolator.Spec.InputSecrets {
		// Find the corresponding NewSecret template
		var templateValue string
		var outputKey string
		templateFound := false
		for _, newSecret := range interpolator.Spec.OutputSecrets {
			if newSecret.SourceKey == secret.Key {
				templateValue = newSecret.Value
				outputKey = newSecret.OutputKey
				templateFound = true
				break
			}
		}
		// Replace placeholders in templateValue using text/template + sprig.
		// We expose each key as a zero-arg template function so templates
		// can reference keys as `{{ key }}` and also use sprig functions.
		if !templateFound || templateValue == "" {
			templateValue = secret.Value
		} else {
			funcMap := sprig.TxtFuncMap()
			for k, v := range secretValues {
				vv := v
				// expose key as function returning its value so `{{ key }}` works
				funcMap[k] = func() string { return vv }
			}

			tmpl, err := template.New("interpolator").Funcs(funcMap).Parse(templateValue)
			if err != nil {
				logger.Error(err, "failed to parse template", "template", templateValue)
				templateValue = secret.Value
			} else {
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, nil); err != nil {
					logger.Error(err, "failed to execute template", "template", templateValue)
					templateValue = secret.Value
				} else {
					templateValue = buf.String()
				}
			}
		}
		if outputKey == "" {
			outputKey = secret.Key
		}
		// Convert the final value to []byte and store it in the map
		switch interpolator.Spec.OutputKind {
		case resourceTypeSecret:
			FinalSecrets[outputKey] = []byte(templateValue)
		case resourceTypeConfigMap:
			FinalSecretsString[outputKey] = templateValue
		}
	}

	secret := &v1.Secret{}
	configmap := &v1.ConfigMap{}

	switch interpolator.Spec.OutputKind {
	case resourceTypeSecret:
		secret, _ = r.secretForInterpolator(interpolator, FinalSecrets)
		err := r.Create(ctx, secret)
		if apierrors.IsAlreadyExists(err) {
			logger.Info("secret already exists", "name", secret.Name, "namespace", secret.Namespace)
			existingSecret := &v1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}, existingSecret); err != nil {
				logger.Error(err, "failed to fetch existing secret", "name", secret.Name, "namespace", secret.Namespace)
				return ctrl.Result{}, err
			}
			if !reflect.DeepEqual(existingSecret.Data, FinalSecrets) {
				existingSecret.Data = FinalSecrets
				if err := r.Update(ctx, existingSecret); err != nil {
					logger.Error(err, "failed to update existing secret", "name", secret.Name, "namespace", secret.Namespace)
					return ctrl.Result{}, err
				}

				meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeAvailableInterpolator, Status: metav1.ConditionTrue, Reason: "Synced", Message: "Secret synced with newer data", LastTransitionTime: metav1.Now()})
				interpolator.Status.LastSyncedTime = &now
				if err := r.Status().Update(ctx, interpolator); err != nil {
					logger.Error(err, "failed to update Interpolator status-Synced")
					return ctrl.Result{}, err
				}
				logger.Info("updated output secret")
			} else {
				logger.Info("no update needed for output secret")
				meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeAvailableInterpolator, Status: metav1.ConditionTrue, Reason: "Synced", Message: "Secret synced with newer data", LastTransitionTime: metav1.Now()})
				interpolator.Status.LastSyncedTime = &now
				if err := r.Status().Update(ctx, interpolator); err != nil {
					logger.Error(err, "failed to update Interpolator status-Synced")
					return ctrl.Result{}, err
				}
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}
	case resourceTypeConfigMap:
		configmap, _ = r.configmapForInterpolator(interpolator, FinalSecretsString)
		err := r.Create(ctx, configmap)
		if apierrors.IsAlreadyExists(err) {
			logger.Info("configMap already exists", "name", configmap.Name, "namespace", configmap.Namespace)
			existingConfigMap := &v1.ConfigMap{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: configmap.Namespace, Name: configmap.Name}, existingConfigMap); err != nil {
				logger.Error(err, "failed to fetch existing configmap", "name", configmap.Name, "namespace", configmap.Namespace)
				return ctrl.Result{}, err
			}
			if !reflect.DeepEqual(existingConfigMap.Data, FinalSecretsString) {
				existingConfigMap.Data = FinalSecretsString
				if err := r.Update(ctx, existingConfigMap); err != nil {
					logger.Error(err, "failed to update existing configmap", "name", configmap.Name, "namespace", configmap.Namespace)
					return ctrl.Result{}, err
				}

				meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeAvailableInterpolator, Status: metav1.ConditionTrue, Reason: "Synced", Message: "ConfigMap synced with newer data", LastTransitionTime: metav1.Now()})
				interpolator.Status.LastSyncedTime = &now
				if err := r.Status().Update(ctx, interpolator); err != nil {
					logger.Error(err, "failed to update Interpolator status-Synced")
					return ctrl.Result{}, err
				}
				logger.Info("updated output configmap")
			} else {
				logger.Info("no update needed for output configmap")
				meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeAvailableInterpolator, Status: metav1.ConditionTrue, Reason: "Synced", Message: "ConfigMap synced with newer data", LastTransitionTime: metav1.Now()})
				interpolator.Status.LastSyncedTime = &now
				if err := r.Status().Update(ctx, interpolator); err != nil {
					logger.Error(err, "failed to update Interpolator status-Synced")
					return ctrl.Result{}, err
				}
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}
	}

	r.Recorder.Eventf(interpolator, nil, "Normal", "Synced", "Synced", "custom resource %s is synced", req.Name)
	logger.Info("finished reconciliation for Interpolator", "NamespacedName", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *InterpolatorReconciler) secretForInterpolator(interpolator *interpolatorv1.Interpolator, rmap map[string][]byte) (*v1.Secret, error) {

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      interpolator.Spec.OutputName,
			Namespace: interpolator.Namespace,
		},
		Data: rmap,
		Type: "Opaque",
	}
	if err := ctrl.SetControllerReference(interpolator, secret, r.Scheme); err != nil {
		return nil, err
	}
	return secret, nil
}

func (r *InterpolatorReconciler) configmapForInterpolator(interpolator *interpolatorv1.Interpolator, rmap map[string]string) (*v1.ConfigMap, error) {

	configmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      interpolator.Spec.OutputName,
			Namespace: interpolator.Namespace,
		},
		Data: rmap,
	}
	if err := ctrl.SetControllerReference(interpolator, configmap, r.Scheme); err != nil {
		return nil, err
	}
	return configmap, nil
}

func (r *InterpolatorReconciler) doFinalizerOperationsForInterpolator(cr *interpolatorv1.Interpolator) {
	r.Recorder.Eventf(cr, nil, "Warning", "Deleting", "Deleting", "custom resource %s is being deleted from namespace %s", cr.Name, cr.Namespace)
}

func (r *InterpolatorReconciler) isInputResource(ctx context.Context, obj client.Object) bool {
	interpolatorList := &interpolatorv1.InterpolatorList{}
	listOpts := &client.ListOptions{
		Namespace: obj.GetNamespace(), // Only check Interpolators in the same namespace
	}
	if err := r.List(ctx, interpolatorList, listOpts); err != nil {
		return true // If we can't list, assume it's an input resource to be safe
	}

	for _, interpolator := range interpolatorList.Items {
		for _, inputRes := range interpolator.Spec.InputSecrets {
			if inputRes.Name == obj.GetName() && inputRes.Namespace == obj.GetNamespace() {
				return true
			}
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *InterpolatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	inputResourcePredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return r.isInputResource(context.Background(), obj)
	})

	// Only reconcile when the spec changes, not on status updates
	specChangedPredicate := predicate.GenerationChangedPredicate{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&interpolatorv1.Interpolator{}).
		Owns(&v1.Secret{}, builder.WithPredicates(specChangedPredicate)).
		Owns(&v1.ConfigMap{}, builder.WithPredicates(specChangedPredicate)).
		Watches(
			&v1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
			builder.WithPredicates(inputResourcePredicate),
		).
		Watches(
			&v1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(inputResourcePredicate),
		).
		Complete(r)
}

func (r *InterpolatorReconciler) findObjectsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	interpolatorList := &interpolatorv1.InterpolatorList{}
	listOpts := &client.ListOptions{
		Namespace: secret.GetNamespace(), // Only check Interpolators in the same namespace
	}
	if err := r.List(ctx, interpolatorList, listOpts); err != nil {
		return []reconcile.Request{}
	}

	requestMap := make(map[string]reconcile.Request) // Use map to deduplicate
	for _, interpolator := range interpolatorList.Items {
		for _, inputSecret := range interpolator.Spec.InputSecrets {
			if inputSecret.Kind == resourceTypeSecret &&
				inputSecret.Name == secret.GetName() &&
				inputSecret.Namespace == secret.GetNamespace() {
				key := fmt.Sprintf("%s/%s", interpolator.GetNamespace(), interpolator.GetName())
				requestMap[key] = reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      interpolator.GetName(),
						Namespace: interpolator.GetNamespace(),
					},
				}
				break // No need to check other inputs if we found a match
			}
		}
	}

	requests := make([]reconcile.Request, 0, len(requestMap))
	for _, req := range requestMap {
		requests = append(requests, req)
	}
	return requests
}

func (r *InterpolatorReconciler) findObjectsForConfigMap(ctx context.Context, configMap client.Object) []reconcile.Request {
	interpolatorList := &interpolatorv1.InterpolatorList{}
	listOpts := &client.ListOptions{
		Namespace: configMap.GetNamespace(), // Only check Interpolators in the same namespace
	}
	if err := r.List(ctx, interpolatorList, listOpts); err != nil {
		return []reconcile.Request{}
	}

	requestMap := make(map[string]reconcile.Request) // Use map to deduplicate
	for _, interpolator := range interpolatorList.Items {
		for _, inputSecret := range interpolator.Spec.InputSecrets {
			if inputSecret.Kind == resourceTypeConfigMap &&
				inputSecret.Name == configMap.GetName() &&
				inputSecret.Namespace == configMap.GetNamespace() {
				key := fmt.Sprintf("%s/%s", interpolator.GetNamespace(), interpolator.GetName())
				requestMap[key] = reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      interpolator.GetName(),
						Namespace: interpolator.GetNamespace(),
					},
				}
				break
			}
		}
	}

	requests := make([]reconcile.Request, 0, len(requestMap))
	for _, req := range requestMap {
		requests = append(requests, req)
	}
	return requests
}
