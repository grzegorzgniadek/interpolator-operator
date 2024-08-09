/*
Copyright 2024.

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
	"context"
	"fmt"
	"reflect"
	"strings"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	interpolatorv1 "interpolator.io/interpolator/api/v1"
)

// InterpolatorReconciler reconciles a Interpolator object
type InterpolatorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=interpolator.io,resources=interpolators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=interpolator.io,resources=interpolators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=interpolator.io,resources=interpolators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Interpolator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile

const (
	typeAvailableInterpolator = "Synced"
	typeDegradedInterpolator  = "Degraded"
	interpolatorFinalizer     = "interpolator.interpolator.io/finalizer"
)

func (r *InterpolatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconciliation for Interpolator", "NamespacedName", req.NamespacedName)

	// Fetch the Interpolator instance
	interpolator := &interpolatorv1.Interpolator{}
	err := r.Get(ctx, req.NamespacedName, interpolator)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("Interpolator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Interpolator")
		return ctrl.Result{}, err
	}
	if interpolator.Status.Conditions == nil || len(interpolator.Status.Conditions) == 0 {
		meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeDegradedInterpolator, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconcilation"})
		if err := r.Status().Update(ctx, interpolator); err != nil {
			log.Error(err, "Failed to update Interpolator status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, interpolator); err != nil {
			log.Error(err, "Failed to re-fetch Interpolator")
			return ctrl.Result{}, err
		}
	}

	if !controllerutil.ContainsFinalizer(interpolator, interpolatorFinalizer) {
		log.Info("Adding finalizer to Interpolator")
		if ok := controllerutil.AddFinalizer(interpolator, interpolatorFinalizer); !ok {
			log.Error(err, "Failed to add finalizer to interpolator resource")
			return ctrl.Result{Requeue: true}, err
		}
		if err := r.Update(ctx, interpolator); err != nil {
			log.Error(err, "Failed to update interpolator resource to add finalizer")
			return ctrl.Result{}, err
		}
	}
	if interpolator.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(interpolator, interpolatorFinalizer) {
			log.Info("Performing finalizer operations for Interpolator before delete CR")
			meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeDegradedInterpolator, Status: metav1.ConditionUnknown, Reason: "Finalizing", Message: fmt.Sprintf("performing finalizer operations for custom resource %s", interpolator.Name)})
			if err := r.Status().Update(ctx, interpolator); err != nil {
				log.Error(err, "Failed to update Interpolator status")
				return ctrl.Result{}, err
			}
			r.doFinalizerOperatotionsForInterpolator(interpolator)
			if err := r.Get(ctx, req.NamespacedName, interpolator); err != nil {
				log.Error(err, "Failed to re-fetch interpolator")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeDegradedInterpolator, Status: metav1.ConditionTrue, Reason: "Finalizing", Message: fmt.Sprintf("finalizer operations for custom resource %s were successfully accomplished", interpolator.Name)})
			if err := r.Status().Update(ctx, interpolator); err != nil {
				log.Error(err, "Failed to update interpolator status")
				return ctrl.Result{}, err
			}
			log.Info("Removing finalizer for Interpolator after successful performed operations")
			if ok := controllerutil.RemoveFinalizer(interpolator, interpolatorFinalizer); !ok {
				log.Error(err, "Failed to remove finalizer for Interpolator")
				return ctrl.Result{Requeue: true}, fmt.Errorf("failed to remove finalizer")
			}
			if err := r.Update(ctx, interpolator); err != nil {
				log.Error(err, "Failed to remove finalizer for Interpolator")
			}

		}
		return ctrl.Result{}, nil
	}

	/////////////////////////////     Interpolation logic ///////////////////////////////////////////////////////
	FinalSecrets := make(map[string][]byte)
	secretValues := make(map[string]string)

	log.Info("Fetching and processing input secrets")
	for i, secret := range interpolator.Spec.InputSecret {
		if secret.Kind == "ConfigMap" {
			configmaps := &v1.ConfigMap{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}, configmaps); err != nil {
				log.Error(err, "Failed to get data from secret", "name", secret.Name, "namespace", secret.Namespace)
				return ctrl.Result{}, err
			}
			secretValues[secret.Key] = configmaps.Data[secret.Key]
			interpolator.Spec.InputSecret[i].Value = configmaps.Data[secret.Key]
		} else if secret.Kind == "Secret" {
			secrets := &v1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}, secrets); err != nil {
				log.Error(err, "Failed to get data from secret", "name", secret.Name, "namespace", secret.Namespace)
				return ctrl.Result{}, err
			}
			secretValues[secret.Key] = string(secrets.Data[secret.Key])
			interpolator.Spec.InputSecret[i].Value = string(secrets.Data[secret.Key])
		}
	}

	log.Info("Interpolating secrets")
	for _, secret := range interpolator.Spec.InputSecret {
		// Find the corresponding NewSecret template
		var templateValue string
		var outputKey string
		templateFound := false
		for _, newSecret := range interpolator.Spec.OutputSecret {
			if newSecret.SourceKey == secret.Key {
				templateValue = newSecret.Value
				outputKey = newSecret.OutputKey
				templateFound = true
				break
			}
		}
		// Replace placeholders in templateValue
		if !templateFound || templateValue == "" {
			templateValue = secret.Value
		} else {
			for key, value := range secretValues {
				placeholder := fmt.Sprintf("{{ %s }}", key)
				templateValue = strings.ReplaceAll(templateValue, placeholder, value)
			}
		}
		if outputKey == "" {
			outputKey = secret.Key
		}
		// Convert the final value to []byte and store it in the map
		FinalSecrets[outputKey] = []byte(templateValue)
	}

	secret := &v1.Secret{}
	secret, _ = r.secretForInterpolator(interpolator, FinalSecrets)
	err = r.Create(ctx, secret)
	if apierrors.IsAlreadyExists(err) {
		log.Info("Secret already exists", "name", secret.Name, "namespace", secret.Namespace)
		existingSecret := &v1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}, existingSecret); err != nil {
			log.Error(err, "Failed to fetch existing secret", "name", secret.Name, "namespace", secret.Namespace)
			return ctrl.Result{}, err
		}
		if !reflect.DeepEqual(existingSecret.Data, FinalSecrets) {
			existingSecret.Data = FinalSecrets
			if err := r.Update(ctx, secret); err != nil {
				log.Error(err, "Failed to update existing secret", "name", secret.Name, "namespace", secret.Namespace)
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeAvailableInterpolator, Status: metav1.ConditionTrue, Reason: "Syncing", Message: "Secret not synced with newer data"})
			if err := r.Status().Update(ctx, interpolator); err != nil {
				log.Error(err, "Failed to update Interpolator status")
				return ctrl.Result{}, err
			}
			log.Info("final secret", "status", "updated")
		} else {
			log.Info("No update needed for existing secret")
			meta.SetStatusCondition(&interpolator.Status.Conditions, metav1.Condition{Type: typeAvailableInterpolator, Status: metav1.ConditionTrue, Reason: "Synced", Message: "Secret synced with newer data"})
			if err := r.Status().Update(ctx, interpolator); err != nil {
				log.Error(err, "Failed to update Interpolator status")
				return ctrl.Result{}, err
			}
			log.Info("Final secret status synced")
		}
	} else {
		log.Error(err, "Failed to create new secret")
		return ctrl.Result{}, err
	}

	log.Info("Finished reconciliation for Interpolator", "NamespacedName", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *InterpolatorReconciler) secretForInterpolator(interpolator *interpolatorv1.Interpolator, rmap map[string][]byte) (*v1.Secret, error) {

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      interpolator.Spec.OutputSecretName,
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

func (r *InterpolatorReconciler) doFinalizerOperatotionsForInterpolator(cr *interpolatorv1.Interpolator) {
	r.Recorder.Event(cr, "Warning", "Deleting", fmt.Sprintf("Custom resource %s is being deleted from namespace %s", cr.Name, cr.Namespace))

}

// SetupWithManager sets up the controller with the Manager.
func (r *InterpolatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&interpolatorv1.Interpolator{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
