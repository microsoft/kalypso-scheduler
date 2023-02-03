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

// WorkloadRegistrationSpec defines the desired state of WorkloadRegistration
type WorkloadRegistrationSpec struct {
	Workload  ManifestsSpec `json:"workload"`
	Workspace string        `json:"workspace,omitempty"`
}

// WorkloadRegistrationStatus defines the observed state of WorkloadRegistration
type WorkloadRegistrationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WorkloadRegistration is the Schema for the workloadregistrations API
type WorkloadRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadRegistrationSpec   `json:"spec,omitempty"`
	Status WorkloadRegistrationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkloadRegistrationList contains a list of WorkloadRegistration
type WorkloadRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadRegistration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadRegistration{}, &WorkloadRegistrationList{})
}
