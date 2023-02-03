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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SchedulingPolicySpec defines the desired state of SchedulingPolicy
type SchedulingPolicySpec struct {
	DeploymentTargetSelector DeploymentTargetSelectorSpec `json:"deploymentTargetSelector"`
	ClusterTypeSelector      ClusterTypeSelectorSpec      `json:"clusterTypeSelector"`
}

type DeploymentTargetSelectorSpec struct {
	//+optional
	Workspace string `json:"workspace,omitempty"`

	LabelSelector metav1.LabelSelector `json:"labelSelector"`
}

type ClusterTypeSelectorSpec struct {
	LabelSelector metav1.LabelSelector `json:"labelSelector"`
}

// SchedulingPolicyStatus defines the observed state of SchedulingPolicy
type SchedulingPolicyStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SchedulingPolicy is the Schema for the schedulingpolicies API
type SchedulingPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SchedulingPolicySpec   `json:"spec,omitempty"`
	Status SchedulingPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SchedulingPolicyList contains a list of SchedulingPolicy
type SchedulingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SchedulingPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SchedulingPolicy{}, &SchedulingPolicyList{})
}
