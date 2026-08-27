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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// InterpolatorSpec defines the desired state of Interpolator
type InterpolatorSpec struct {
	// Name of output resource
	OutputName string `json:"outputName"`
	// Type of Output resource, Can be ConfigMap or Secret
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	OutputKind string `json:"outputKind"`
	// Output secret array of structs
	OutputSecrets []InterpolatorOutputSecrets `json:"outputSecrets,omitempty"`
	// Input secret array of structs
	InputSecrets []InterpolatorInputSecrets `json:"inputSecrets"`
}

type InterpolatorInputSecrets struct {
	// Name of input resource
	Name string `json:"name,omitempty"`
	// Type of input resource, Can be ConfigMap or Secret
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	Kind string `json:"kind,omitempty"`
	// Namespace of input resource
	Namespace string `json:"namespace,omitempty"`
	// Key of input resource
	Key string `json:"key,omitempty"`
	// Value of input resource
	Value string `json:"value,omitempty"`
}

type InterpolatorOutputSecrets struct {
	// Source key for value
	SourceKey string `json:"sourcekey,omitempty"`
	// Output  key for value, if empty the SourceKey is master
	OutputKey string `json:"outputkey,omitempty"`
	// Templated value of output key
	Value string `json:"value,omitempty"`
}

// InterpolatorStatus defines the observed state of Interpolator.
type InterpolatorStatus struct {
	Conditions     []metav1.Condition `json:"conditions,omitempty"`
	LastSyncedTime *metav1.Time       `json:"lastSyncedTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Last Update",type="date",JSONPath=".status.lastSyncedTime"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].status"
// +kubebuilder:resource:shortName={"inter"}
// Interpolator is the Schema for the interpolators API
type Interpolator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// Spec defines the desired state of Interpolator
	// +required
	Spec InterpolatorSpec `json:"spec"`

	// Status defines the observed state of Interpolator
	// +optional
	Status InterpolatorStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InterpolatorList contains a list of Interpolator
type InterpolatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Interpolator `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Interpolator{}, &InterpolatorList{})
		return nil
	})
}
