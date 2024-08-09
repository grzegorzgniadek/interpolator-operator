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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InterpolatorSpec defines the desired state of Interpolator
type InterpolatorSpec struct {
	// Output secret Name
	OutputSecretName string `json:"outputSecretName"`
	//Output secret array of structs
	OutputSecret []InterpolatorOutputSecret `json:"outputSecret,omitempty"`
	//Input secret array of structs
	InputSecret []InterpolatorInputSecret `json:"inputSecret"`
}

type InterpolatorInputSecret struct {
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

type InterpolatorOutputSecret struct {
	// Source key for value
	SourceKey string `json:"sourcekey,omitempty"`
	// Output  key for value, if empty the SourceKey is master
	OutputKey string `json:"outputkey,omitempty"`
	// Templated value of output key
	Value string `json:"value,omitempty"`
}

// InterpolatorStatus defines the observed state of Interpolator
type InterpolatorStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Last Update",type="date",JSONPath=".status.conditions[?(@.status==\"True\")].lastTransitionTime"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.status==\"True\")].status"
// +kubebuilder:resource:shortName={"inter"}
// Interpolator is the Schema for the interpolators API
type Interpolator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InterpolatorSpec   `json:"spec,omitempty"`
	Status InterpolatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InterpolatorList contains a list of Interpolator
type InterpolatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Interpolator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Interpolator{}, &InterpolatorList{})
}
