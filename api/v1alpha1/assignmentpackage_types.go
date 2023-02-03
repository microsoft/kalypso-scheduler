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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ClusterTypeLabel      = "cluster-type"
	DeploymentTargetLabel = "deployment-target"
)

// AssignmentPackageSpec defines the desired state of AssignmentPackage
type AssignmentPackageSpec struct {
	//+kubebuilder:pruning:PreserveUnknownFields
	ReconcilerManifests []unstructured.Unstructured `json:"reconcilerManifests,omitempty"`

	//+kubebuilder:pruning:PreserveUnknownFields
	NamespaceManifests []unstructured.Unstructured `json:"namespaceManifests,omitempty"`

	//+kubebuilder:pruning:PreserveUnknownFields
	ConfigManifests []unstructured.Unstructured `json:"configManifests,omitempty"`
}

// AssignmentPackageStatus defines the observed state of AssignmentPackage
type AssignmentPackageStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AssignmentPackage is the Schema for the assignmentpackages API
type AssignmentPackage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AssignmentPackageSpec   `json:"spec,omitempty"`
	Status AssignmentPackageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AssignmentPackageList contains a list of AssignmentPackage
type AssignmentPackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AssignmentPackage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AssignmentPackage{}, &AssignmentPackageList{})
}
