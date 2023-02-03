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
	ReadyToPRConditionType = "ReadyToPR"
	PRConditionType        = "PR"
	ReadyConditionType     = "Ready"
)

// GitOpsRepoSpec defines the desired state of GitOpsRepo
type GitOpsRepoSpec struct {
	ManifestsSpec `json:",inline"`
	//TODO
	//AutoMerge
}

type RepoContentType struct {
	ClusterTypes map[string]ClusterContentType
	BaseRepo     BaseRepoSpec
}

type ClusterContentType struct {
	DeploymentTargets map[string]AssignmentPackageSpec
}

// NewClusterContentType creates a new ClusterContentType
func NewClusterContentType() *ClusterContentType {
	return &ClusterContentType{
		DeploymentTargets: make(map[string]AssignmentPackageSpec),
	}
}

// newRepoContentType creates a new RepoContentType
func NewRepoContentType() *RepoContentType {
	return &RepoContentType{
		ClusterTypes: make(map[string]ClusterContentType),
	}
}

// GitOpsRepoStatus defines the observed state of GitOpsRepo
type GitOpsRepoStatus struct {
	RepoContentHash string             `json:"repoContentHash,omitempty"`
	Conditions      []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GitOpsRepo is the Schema for the gitopsrepoes API
type GitOpsRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitOpsRepoSpec   `json:"spec,omitempty"`
	Status GitOpsRepoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GitOpsRepoList contains a list of GitOpsRepo
type GitOpsRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitOpsRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitOpsRepo{}, &GitOpsRepoList{})
}
