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

package controllers

import (
	"context"
	"time"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
)

// GitOpsRepoReconciler reconciles a GitOpsRepo object
type GitOpsRepoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=gitopsrepoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=gitopsrepoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=gitopsrepoes/finalizers,verbs=update

func (r *GitOpsRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Assignment ===")

	// Fetch the GitOpsRepo instance
	gitopsrepo := &schedulerv1alpha1.GitOpsRepo{}
	err := r.Get(ctx, req.NamespacedName, gitopsrepo)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get GitOpsRepo")

		}
		return ctrl.Result{}, ignroredNotFound
	}

	// Check if the resource is being deleted
	if !gitopsrepo.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Check if the status condioion ReadyToPR is true
	if !meta.IsStatusConditionTrue(gitopsrepo.Status.Conditions, schedulerv1alpha1.ReadyToPRConditionType) {
		// check Ready status condition of all Schedule Policies in the namespace
		// if all are true, set ReadyToPR to true
		schedulePolicies := &schedulerv1alpha1.SchedulingPolicyList{}
		err = r.List(ctx, schedulePolicies, client.InNamespace(gitopsrepo.Namespace))
		if err != nil {
			reqLogger.Error(err, "Failed to list SchedulePolicies")
			return ctrl.Result{}, err
		}
		readyToPR := true
		for _, schedulePolicy := range schedulePolicies.Items {
			if !meta.IsStatusConditionTrue(schedulePolicy.Status.Conditions, schedulerv1alpha1.ReadyConditionType) {
				readyToPR = false
				break
			}
		}

		if readyToPR {
			// check Ready status condition of all Assignments in the namespace
			// if all are true, set ReadyToPR to true
			assignments := &schedulerv1alpha1.AssignmentList{}
			err = r.List(ctx, assignments, client.InNamespace(gitopsrepo.Namespace))
			if err != nil {
				reqLogger.Error(err, "Failed to list Assignments")
				return ctrl.Result{}, err
			}
			for _, assignment := range assignments.Items {
				if !meta.IsStatusConditionTrue(assignment.Status.Conditions, schedulerv1alpha1.ReadyConditionType) {
					readyToPR = false
					break
				}
			}
		}

		if readyToPR {
			// set ReadyToPR to true
			meta.SetStatusCondition(&gitopsrepo.Status.Conditions, metav1.Condition{
				Type:   schedulerv1alpha1.ReadyToPRConditionType,
				Status: metav1.ConditionTrue,
				Reason: "All SchedulePolicies and Assignments are ready",
			})

			updateErr := r.Status().Update(ctx, gitopsrepo)
			if updateErr != nil {
				reqLogger.Error(updateErr, "Error when updating status.")
				return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
			}

			//Create a PR in the next reconcile in 3 seconds
			//so all assignments can bombard this GitOpsRepo
			return ctrl.Result{RequeueAfter: time.Second * 3}, nil

		}

	}
	//check last transaction time of the ReadyToPR condition
	//if it is more than 3 seconds, create a PR
	readyToPRCondition := meta.FindStatusCondition(gitopsrepo.Status.Conditions, schedulerv1alpha1.ReadyToPRConditionType)
	if readyToPRCondition != nil {
		if readyToPRCondition.LastTransitionTime.Time.Add(time.Second * 3).Before(time.Now()) {
			//log a mesage and create a PR
			reqLogger.Info("!!!!!!!!!!!!!!!!!!!Creating a PR!!!!!!!!!!!!!!!!!!!!!!!!")
			//TODO: create a PR			
			// figure out how to not create a PR on every controller restart
			// perhaps hash the content of the PR and store it in the status - that's correct!
			// https://pkg.go.dev/github.com/mitchellh/hashstructure
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitOpsRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.GitOpsRepo{}).
		Complete(r)
}
