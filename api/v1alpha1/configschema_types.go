/*
Copyright 2023 microsoft.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigSchemaSpec defines the desired state of ConfigSchema
type ConfigSchemaSpec struct {
	//+kubebuilder:pruning:PreserveUnknownFields
	//+kubebuilder:validation:MinLength=0
	Schema string `json:"schema"`
}

// ConfigSchemaStatus defines the observed state of ConfigSchema
type ConfigSchemaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ConfigSchema is the Schema for the configschemas API
type ConfigSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSchemaSpec   `json:"spec,omitempty"`
	Status ConfigSchemaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigSchemaList contains a list of ConfigSchema
type ConfigSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigSchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigSchema{}, &ConfigSchemaList{})
}
