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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WorkspaceLabel = "workspace"
	WorkloadLabel  = "workload"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentTargetSpec defines the desired state of DeploymentTarget
type DeploymentTargetSpec struct {
	//+kubebuilder:validation:MinLength=0
	Environment string `json:"environment"`

	Manifests ManifestsSpec `json:"manifests"`
}

type ManifestsSpec struct {
	//+kubebuilder:validation:MinLength=0
	Repo string `json:"repo"`

	//+kubebuilder:validation:MinLength=0
	Branch string `json:"branch"`

	//+kubebuilder:validation:MinLength=0
	Path string `json:"path"`
}

// DeploymentTargetStatus defines the observed state of DeploymentTarget
type DeploymentTargetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeploymentTarget is the Schema for the deploymenttargets API
type DeploymentTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentTargetSpec   `json:"spec,omitempty"`
	Status DeploymentTargetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentTargetList contains a list of DeploymentTarget
type DeploymentTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentTarget `json:"items"`
}

// get deployment target workspace
func (dt *DeploymentTarget) GetWorkspace() string {
	return dt.Labels[WorkspaceLabel]
}

// get deployment target workload
func (dt *DeploymentTarget) GetWorkload() string {
	return dt.Labels[WorkloadLabel]
}

// get deployment target namespace
func (dt *DeploymentTarget) GetTargetNamespace() string {
	return fmt.Sprintf("%s-%s-%s", dt.Spec.Environment, dt.GetWorkspace(), dt.GetName())
}

func init() {
	SchemeBuilder.Register(&DeploymentTarget{}, &DeploymentTargetList{})
}
