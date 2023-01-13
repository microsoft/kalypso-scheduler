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

package scheduler

import (
	"context"

	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type Scheduler interface {
	SelectClusterTypes(ctx context.Context, allClusterTypes []kalypsov1alpha1.ClusterType) ([]kalypsov1alpha1.ClusterType, error)
	SelectDeploymentTargets(ctx context.Context, allDeploymentTargets []kalypsov1alpha1.DeploymentTarget) ([]kalypsov1alpha1.DeploymentTarget, error)
	IsClusterTypeCompliant(ctx context.Context, clusterType kalypsov1alpha1.ClusterType) (bool, error)
	IsDeploymentTargetCompliant(ctx context.Context, deploymentTarget kalypsov1alpha1.DeploymentTarget) (bool, error)
}

// implements Scheduler interface
type scheduler struct {
	schedulingPolicy kalypsov1alpha1.SchedulingPolicy
}

// validate scheduler implements Scheduler interface
var _ Scheduler = (*scheduler)(nil)

// new scheduler function
func NewScheduler(schedulingPolicy kalypsov1alpha1.SchedulingPolicy) Scheduler {
	return &scheduler{
		schedulingPolicy: schedulingPolicy,
	}
}

// SelectClusterTypes selects cluster types that match scheduling policy labels
func (s *scheduler) SelectClusterTypes(ctx context.Context, allClusterTypes []kalypsov1alpha1.ClusterType) ([]kalypsov1alpha1.ClusterType, error) {
	// get cluster types labels selector from scheduling policy
	clusterTypesLabelsSelector, err := metav1.LabelSelectorAsSelector(&s.schedulingPolicy.Spec.ClusterTypeSelector.LabelSelector)
	if err != nil {
		return nil, err
	}

	// iterate over all cluster types and return the ones that match the labels selector

	var selectedClusterTypes []kalypsov1alpha1.ClusterType
	for _, clusterType := range allClusterTypes {
		if clusterTypesLabelsSelector.Matches(labels.Set(clusterType.GetLabels())) {
			selectedClusterTypes = append(selectedClusterTypes, clusterType)
		}
	}

	return selectedClusterTypes, nil
}

// SelectDeploymentTargets selects deployment targets based on the scheduler implementation
func (s *scheduler) SelectDeploymentTargets(ctx context.Context, allDeploymentTargets []kalypsov1alpha1.DeploymentTarget) ([]kalypsov1alpha1.DeploymentTarget, error) {
	return allDeploymentTargets, nil
}

// IsClusterTypeCompliant checks if the cluster type is compliant with the scheduler implementation
func (s *scheduler) IsClusterTypeCompliant(ctx context.Context, clusterType kalypsov1alpha1.ClusterType) (bool, error) {
	return true, nil
}

// IsDeploymentTargetCompliant checks if the deployment target is compliant with the scheduler implementation
func (s *scheduler) IsDeploymentTargetCompliant(ctx context.Context, deploymentTarget kalypsov1alpha1.DeploymentTarget) (bool, error) {
	return true, nil
}

// get cluster types labels selector from scheduling policy
func (s *scheduler) getClusterTypesLabelsSelector() (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(&s.schedulingPolicy.Spec.ClusterTypeSelector.LavelSelector)
}
