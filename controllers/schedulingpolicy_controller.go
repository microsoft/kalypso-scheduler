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

	"github.com/go-logr/logr"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/microsoft/kalypso-scheduler/scheduler"
)

// SchedulingPolicyReconciler reconciles a SchedulingPolicy object
type SchedulingPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=schedulingpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=schedulingpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=schedulingpolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SchedulingPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *SchedulingPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Scheduling Policy")

	// Fetch the SchedulingPolicy instance
	schedulingPolicy := &schedulerv1alpha1.SchedulingPolicy{}
	err := r.Get(ctx, req.NamespacedName, schedulingPolicy)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get SchedulingPolicy")

		}
		return ctrl.Result{}, ignroredNotFound
	}

	// Check if the resource is being deleted
	if !schedulingPolicy.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// fetch the list of clusterctypes in the namespace
	clusterTypes := &schedulerv1alpha1.ClusterTypeList{}
	err = r.List(ctx, clusterTypes, client.InNamespace(req.Namespace))
	if err != nil {
		r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to list ClusterTypes")
	}

	// fetch the list if deployment targets in the namespace
	deploymentTargets := &schedulerv1alpha1.DeploymentTargetList{}
	err = r.List(ctx, deploymentTargets, client.InNamespace(req.Namespace))
	if err != nil {
		r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to list DeploymentTargets")
	}

	// schedule the deployment targets
	scheduler, err := scheduler.NewScheduler(schedulingPolicy)
	if err != nil {
		r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to create scheduler")
	}

	assignments, err := scheduler.Schedule(ctx, clusterTypes.Items, deploymentTargets.Items)
	reqLogger.Info("Number of assignments", "count", len(assignments))

	//log the assignments
	for _, assignment := range assignments {
		reqLogger.Info("Assignment", "name", assignment.Name, "clusterType", assignment.Spec.ClusterType, "deploymentTarget", assignment.Spec.DeploymentTarget)
	}

	// fetch the list of assignments in the namespace
	assignmentsList := &schedulerv1alpha1.AssignmentList{}
	err = r.List(ctx, assignmentsList, client.InNamespace(req.Namespace))
	if err != nil {
		r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to list Assignments")
	}

	// iterate over the existing assignments and delete the ones that are not in the new assignments
	for _, assignment := range assignmentsList.Items {
		found := false
		for _, newAssignment := range assignments {
			if assignment.Name == newAssignment.Name {
				found = true
				break
			}
		}
		if !found {
			err = r.Delete(ctx, &assignment)
			if err != nil {
				r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to delete Assignment")
			}
			reqLogger.Info("Deleted Assignment", "name", assignment.Name, "clusterType", assignment.Spec.ClusterType, "deploymentTarget", assignment.Spec.DeploymentTarget)
		}
	}

	// iterate over the new assignments and create the ones that are not in the existing assignments
	for _, newAssignment := range assignments {
		found := false
		for _, assignment := range assignmentsList.Items {
			if assignment.Name == newAssignment.Name {
				found = true
				break
			}
		}
		if !found {
			// set the namespace of the assignment
			newAssignment.Namespace = req.Namespace

			// set the owner of the assignment
			if err := ctrl.SetControllerReference(schedulingPolicy, &newAssignment, r.Scheme); err != nil {
				r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to set owner of Assignment")
			}

			err = r.Create(ctx, &newAssignment)
			if err != nil {
				r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to create Assignment")
			}
			reqLogger.Info("Created Assignment", "name", newAssignment.Name, "clusterType", newAssignment.Spec.ClusterType, "deploymentTarget", newAssignment.Spec.DeploymentTarget)
		}
	}

	condition := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Scheduled",
	}
	meta.SetStatusCondition(&schedulingPolicy.Status.Conditions, condition)

	updateErr := r.Status().Update(ctx, schedulingPolicy)
	if updateErr != nil {
		reqLogger.Error(updateErr, "Error when updating status.")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}

	return ctrl.Result{}, nil
}

// Gracefully handle errors
func (h *SchedulingPolicyReconciler) manageFailure(ctx context.Context, logger logr.Logger, schedulingPolicy *schedulerv1alpha1.SchedulingPolicy, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&schedulingPolicy.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, schedulingPolicy)
	if updateErr != nil {
		logger.Error(updateErr, "Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulingPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.SchedulingPolicy{}).
		Complete(r)
}
