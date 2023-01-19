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
	"testing"

	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
)

func TestNewScheduler(t *testing.T) {
	schedulingPolicy := &kalypsov1alpha1.SchedulingPolicy{}
	scheduler, err := NewScheduler(schedulingPolicy)
	if err != nil {
		t.Errorf("error creating scheduler: %v", err)
	}
	if scheduler == nil {
		t.Errorf("scheduler is nil")
	}
}

func TestSelectClusterTypes(t *testing.T) {
	// TODO
	// 1. create a scheduler
	// 2. create a list of cluster types
	// 3. call SelectClusterTypes
	// 4. check the returned list of cluster types

	// // 1. create a scheduler
	// schedulingPolicy := &kalypsov1alpha1.SchedulingPolicy{}
	// scheduler, err := NewScheduler(schedulingPolicy)
	// if err != nil {
	// 	t.Errorf("error creating scheduler: %v", err)
	// }
	// if scheduler == nil {
	// 	t.Errorf("scheduler is nil")
	// }

	// // 2. create a list of cluster types
	// clusterTypes := []kalypsov1alpha1.ClusterType{
	// 	{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name: "cluster-type-1",
	// 			Labels: map[string]string{
	// 				"cluster-type": "cluster-type-1",
	// 			},
	// 		},
	// 	},
	// 	{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name: "cluster-type-2",
	// 			Labels: map[string]string{
	// 				"cluster-type": "cluster-type-2",
	// 			},
	// 		},
	// 	},
	// 	{
	// 		ObjectMeta: metav1.ObjectMeta{
	// 			Name: "cluster-type-3",
	// 			Labels: map[string]string{
	// 				"cluster-type": "cluster-type-3",
	// 			},
	// 		},
	// 	},
	// }

	// // 3. call SelectClusterTypes
	// selectedClusterTypes := scheduler.SelectClusterTypes(clusterTypes)

	// // 4. check the returned list of cluster types
	// if len(selectedClusterTypes) != 1 {
	// 	t.Errorf("expected 1 cluster type, got %d", len(selectedClusterTypes))
	// }

}

func TestSelectDeploymentTargets(t *testing.T) {
	// TODO
}

func TestIsClusterTypeCompliant(t *testing.T) {
	// TODO
}

func TestIsDeploymentTargetCompliant(t *testing.T) {
	// TODO
}

func TestSchedule(t *testing.T) {
	// TODO
}
