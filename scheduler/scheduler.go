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
	"fmt"

	"github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type Scheduler interface {
	SelectClusterTypes(ctx context.Context, allClusterTypes []kalypsov1alpha1.ClusterType) ([]kalypsov1alpha1.ClusterType, error)
	SelectDeploymentTargets(ctx context.Context, allDeploymentTargets []kalypsov1alpha1.DeploymentTarget) ([]kalypsov1alpha1.DeploymentTarget, error)
	IsClusterTypeCompliant(ctx context.Context, clusterType kalypsov1alpha1.ClusterType) bool
	IsDeploymentTargetCompliant(ctx context.Context, deploymentTarget kalypsov1alpha1.DeploymentTarget) bool
	Schedule(ctx context.Context, clusterTypes []kalypsov1alpha1.ClusterType, deploymentTargets []kalypsov1alpha1.DeploymentTarget) ([]kalypsov1alpha1.Assignment, error)
}

// implements Scheduler interface
type scheduler struct {
	schedulingPolicy                *kalypsov1alpha1.SchedulingPolicy
	clusterTypesLabelsSelector      labels.Selector
	deploymentTargetsLabelsSelector labels.Selector
}

// validate scheduler implements Scheduler interface
var _ Scheduler = (*scheduler)(nil)

// new scheduler function
func NewScheduler(schedulingPolicy *kalypsov1alpha1.SchedulingPolicy) (Scheduler, error) {
	clusterTypesLabelsSelector, err := metav1.LabelSelectorAsSelector(&schedulingPolicy.Spec.ClusterTypeSelector.LabelSelector)
	if err != nil {
		return nil, err
	}

	deploymentTargetsLabelsSelector, err := metav1.LabelSelectorAsSelector(&schedulingPolicy.Spec.DeploymentTargetSelector.LabelSelector)
	if err != nil {
		return nil, err
	}

	return &scheduler{
		schedulingPolicy:                schedulingPolicy,
		clusterTypesLabelsSelector:      clusterTypesLabelsSelector,
		deploymentTargetsLabelsSelector: deploymentTargetsLabelsSelector,
	}, nil
}

// SelectClusterTypes selects cluster types that match scheduling policy labels
func (s *scheduler) SelectClusterTypes(ctx context.Context, allClusterTypes []kalypsov1alpha1.ClusterType) ([]kalypsov1alpha1.ClusterType, error) {
	// iterate over all cluster types and return the ones that match the labels selector
	var selectedClusterTypes []kalypsov1alpha1.ClusterType
	for _, clusterType := range allClusterTypes {
		if s.IsClusterTypeCompliant(ctx, clusterType) {
			selectedClusterTypes = append(selectedClusterTypes, clusterType)
		}
	}

	return selectedClusterTypes, nil
}

// SelectDeploymentTargets selects deployment targets based on the scheduler implementation
func (s *scheduler) SelectDeploymentTargets(ctx context.Context, allDeploymentTargets []kalypsov1alpha1.DeploymentTarget) ([]kalypsov1alpha1.DeploymentTarget, error) {
	// iterate over all deployment targets and return the ones that match the labels selector
	var selectedDeploymentTargets []kalypsov1alpha1.DeploymentTarget
	for _, deploymentTarget := range allDeploymentTargets {
		if s.IsDeploymentTargetCompliant(ctx, deploymentTarget) {
			selectedDeploymentTargets = append(selectedDeploymentTargets, deploymentTarget)
		}
	}

	return selectedDeploymentTargets, nil
}

// IsClusterTypeCompliant checks if the cluster type is compliant with the scheduler implementation
func (s *scheduler) IsClusterTypeCompliant(ctx context.Context, clusterType kalypsov1alpha1.ClusterType) bool {
	return s.clusterTypesLabelsSelector.Matches(labels.Set(clusterType.GetLabels()))
}

// IsDeploymentTargetCompliant checks if the deployment target is compliant with the scheduler implementation
func (s *scheduler) IsDeploymentTargetCompliant(ctx context.Context, deploymentTarget kalypsov1alpha1.DeploymentTarget) bool {
	var policyWorkspace = s.schedulingPolicy.Spec.DeploymentTargetSelector.Workspace
	if policyWorkspace != "" {
		if policyWorkspace != deploymentTarget.GetWorkspace() {
			return false
		}
	}

	return s.deploymentTargetsLabelsSelector.Matches(labels.Set(deploymentTarget.GetLabels()))
}

// Schedule schedules the deployment targets on cluster types
func (s *scheduler) Schedule(ctx context.Context, clusterTypes []kalypsov1alpha1.ClusterType, deploymentTargets []kalypsov1alpha1.DeploymentTarget) ([]kalypsov1alpha1.Assignment, error) {

	var assignments []kalypsov1alpha1.Assignment

	//var selectedDeploymentTargets []kalypsov1alpha1.DeploymentTarget
	selectedDeploymentTargets, err := s.SelectDeploymentTargets(ctx, deploymentTargets)
	if err != nil {
		return nil, err
	}

	selectedClusterTypes, err := s.SelectClusterTypes(ctx, clusterTypes)
	if err != nil {
		return nil, err
	}

	// build a matrix of assignments
	for _, clusterType := range selectedClusterTypes {
		for _, deploymentTarget := range selectedDeploymentTargets {
			assignments = append(assignments,
				s.assign(deploymentTarget.GetName(), deploymentTarget.GetWorkload(), clusterType.GetName(), s.schedulingPolicy.GetName()))
		}
	}

	return assignments, nil
}

// assign creates a new Assignment object
func (s *scheduler) assign(deploymentTarget string, workload string, clusterType string, schedulingPolicy string) v1alpha1.Assignment {
	name := fmt.Sprintf("%s-%s-%s", workload, deploymentTarget, clusterType)
	assignment := v1alpha1.Assignment{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.AssignmentKind,
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.AssignmentSpec{
			Workload:         workload,
			DeploymentTarget: deploymentTarget,
			ClusterType:      clusterType,
		},
	}
	assignment.ObjectMeta.SetLabels(map[string]string{
		v1alpha1.AssignmentSchedulingPolicyLabel: schedulingPolicy,
	},
	)

	return assignment
}
