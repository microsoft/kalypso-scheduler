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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/microsoft/kalypso-scheduler/scheduler"
)

// SchedulingPolicyReconciler reconciles a SchedulingPolicy object
type SchedulingPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	ClusterTypeField      = "spec.clusterType"
	ReconcilerField       = "spec.reconciler"
	NamespaceServiceField = "spec.namespaceService"
)

// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=schedulingpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=schedulingpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=schedulingpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=clustertypes,verbs=get;list;watch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=deploymenttargets,verbs=get;list;watch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kustomize.toolkit.fluxcd.io,resources=kustomizations,verbs=get;list;watch;create;update;patch;delete

func (r *SchedulingPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Scheduling Policy ===")

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

	condition := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionFalse,
		Reason: "Rescheduling",
	}
	meta.SetStatusCondition(&schedulingPolicy.Status.Conditions, condition)

	updateErr := r.Status().Update(ctx, schedulingPolicy)
	if updateErr != nil {
		reqLogger.Info("Error when updating status.")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}

	// fetch the list of clusterctypes in the namespace
	clusterTypes := &schedulerv1alpha1.ClusterTypeList{}
	err = r.List(ctx, clusterTypes, client.InNamespace(req.Namespace))
	if err != nil {
		return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to list ClusterTypes")
	}

	// fetch the list if deployment targets in the namespace
	deploymentTargets := &schedulerv1alpha1.DeploymentTargetList{}
	err = r.List(ctx, deploymentTargets, client.InNamespace(req.Namespace))
	if err != nil {
		return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to list DeploymentTargets")
	}

	// schedule the deployment targets
	scheduler, err := scheduler.NewScheduler(schedulingPolicy)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to create scheduler")
	}

	assignments, err := scheduler.Schedule(ctx, clusterTypes.Items, deploymentTargets.Items)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to schedule")
	}
	reqLogger.Info("Number of assignments", "count", len(assignments))

	//log the assignments
	for _, assignment := range assignments {
		reqLogger.Info("Assignment", "name", assignment.Name, "clusterType", assignment.Spec.ClusterType, "deploymentTarget", assignment.Spec.DeploymentTarget)
	}

	// fetch the list of assignments in the namespace by label that are owned by the scheduling policy
	assignmentsList := &schedulerv1alpha1.AssignmentList{}
	err = r.List(ctx, assignmentsList, client.InNamespace(req.Namespace), client.MatchingLabels{schedulerv1alpha1.AssignmentSchedulingPolicyLabel: schedulingPolicy.Name})
	if err != nil {
		return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to list Assignments")
	}

	// iterate over the existing assignments and delete the ones that are not in the new assignments
	for _, assignment := range assignmentsList.Items {
		found := false
		for _, newAssignment := range assignments {
			if assignment.Spec == newAssignment.Spec {
				found = true
				break
			}
		}
		if !found {
			err = r.Delete(ctx, &assignment)
			if err != nil {
				return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to delete Assignment")
			}
			reqLogger.Info("Deleted Assignment", "name", assignment.Name, "clusterType", assignment.Spec.ClusterType, "deploymentTarget", assignment.Spec.DeploymentTarget)
		}
	}

	// iterate over the new assignments and create the ones that are not in the existing assignments
	for _, newAssignment := range assignments {
		found := false
		for _, assignment := range assignmentsList.Items {
			if assignment.Spec == newAssignment.Spec {
				found = true
				break
			}
		}
		if !found {
			// set the namespace of the assignment
			newAssignment.Namespace = req.Namespace

			// set the owner of the assignment
			if err := ctrl.SetControllerReference(schedulingPolicy, &newAssignment, r.Scheme); err != nil {
				return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to set owner of Assignment")
			}

			err = r.Create(ctx, &newAssignment)
			if err != nil {
				return r.manageFailure(ctx, reqLogger, schedulingPolicy, err, "Failed to create Assignment")
			}
			reqLogger.Info("Created Assignment", "name", newAssignment.Name, "clusterType", newAssignment.Spec.ClusterType, "deploymentTarget", newAssignment.Spec.DeploymentTarget)
		}
	}

	condition = metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "AssignmentsCreated",
	}
	meta.SetStatusCondition(&schedulingPolicy.Status.Conditions, condition)

	updateErr = r.Status().Update(ctx, schedulingPolicy)
	if updateErr != nil {
		reqLogger.Info("Error when updating status.")
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
		logger.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

func (r *SchedulingPolicyReconciler) findPolicies(object client.Object) []reconcile.Request {
	// Find the scheduling policies
	schedulingPolicies := &schedulerv1alpha1.SchedulingPolicyList{}
	err := r.List(context.TODO(), schedulingPolicies, client.InNamespace(object.GetNamespace()))
	if err != nil {
		return []reconcile.Request{}
	}

	//Create requests for the scheduling policies
	//TODO: compose an object name on the request to include the cluster type or deployment target name or
	//just implment controllers for each of those types
	//keep it as is now for the simplicity
	requests := make([]reconcile.Request, len(schedulingPolicies.Items))
	for i, item := range schedulingPolicies.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulingPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the field index for the reconciler field in the cluster type
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &schedulerv1alpha1.ClusterType{}, ReconcilerField, func(rawObj client.Object) []string {
		return []string{rawObj.(*schedulerv1alpha1.ClusterType).Spec.Reconciler}
	}); err != nil {
		return err
	}

	// Add the field index for the namespaceService field in the cluster type
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &schedulerv1alpha1.ClusterType{}, NamespaceServiceField, func(rawObj client.Object) []string {
		return []string{rawObj.(*schedulerv1alpha1.ClusterType).Spec.NamespaceService}
	}); err != nil {
		return err
	}

	// Add the field index for the cluster type in the assignment
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &schedulerv1alpha1.Assignment{}, ClusterTypeField, func(rawObj client.Object) []string {
		return []string{rawObj.(*schedulerv1alpha1.Assignment).Spec.ClusterType}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.SchedulingPolicy{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&schedulerv1alpha1.Assignment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &schedulerv1alpha1.ClusterType{}},
			handler.EnqueueRequestsFromMapFunc(r.findPolicies)).
		Watches(
			&source.Kind{Type: &schedulerv1alpha1.DeploymentTarget{}},
			handler.EnqueueRequestsFromMapFunc(r.findPolicies)).
		Complete(r)
}
