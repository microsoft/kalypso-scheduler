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

const (
	AssignmentKind                  = "Assignment"
	AssignmentSchedulingPolicyLabel = "scheduling-policy"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AssignmentSpec defines the desired state of Assignment
type AssignmentSpec struct {
	//+kubebuilder:validation:MinLength=0
	Workload string `json:"workload"`

	//+kubebuilder:validation:MinLength=0
	DeploymentTarget string `json:"deploymentTarget"`

	//+kubebuilder:validation:MinLength=0
	ClusterType string `json:"clusterType"`
}

// AssignmentStatus defines the observed state of Assignment
type AssignmentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	//optional
	GitIssueStatus GitIssueStatus `json:"gitIssueStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Assignment is the Schema for the assignments API
type Assignment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AssignmentSpec   `json:"spec,omitempty"`
	Status AssignmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AssignmentList contains a list of Assignment
type AssignmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Assignment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Assignment{}, &AssignmentList{})
}
