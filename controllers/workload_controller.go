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
	"strings"
	"time"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
)

// WorkloadReconciler reconciles a Workload object
type WorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=workloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=workloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=workloads/finalizers,verbs=update
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=deploymenttargets,verbs=get;list;watch;create;update;patch;delete

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Workload  ===")

	// Fetch the Workload instance
	workload := &schedulerv1alpha1.Workload{}
	err := r.Get(ctx, req.NamespacedName, workload)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get Workload")
		}
		return ctrl.Result{}, ignroredNotFound
	}

	// Check if the resource is being deleted
	if !workload.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// fetch the list of deploymenttargets in the namespace by label that are owned by the workload
	deploymentTargets := &schedulerv1alpha1.DeploymentTargetList{}
	listOpts := []client.ListOption{
		client.InNamespace(workload.Namespace),
		client.MatchingLabels(map[string]string{"workload": workload.Name}),
	}
	err = r.List(ctx, deploymentTargets, listOpts...)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, workload, err, "Failed to list DeploymentTargets")
	}

	// iterate over the existing deploymenttargets and delete the ones that are not in the workload.deploymenttargets
	for _, deploymentTarget := range deploymentTargets.Items {
		found := false
		for _, workloadDeploymentTarget := range workload.Spec.DeploymentTargets {
			deploymentTargetName := r.buildDeploymentTargetName(workload, workloadDeploymentTarget.Name)
			if deploymentTarget.Name == deploymentTargetName {
				found = true
				break
			}
		}
		if !found {
			err = r.Delete(ctx, &deploymentTarget)
			if err != nil {
				return r.manageFailure(ctx, reqLogger, workload, err, "Failed to delete DeploymentTarget")
			}
			reqLogger.Info("Deleted DeploymentTarget", "DeploymentTarget.Name", deploymentTarget.Name)
		}

	}

	// iterate over the workload.deploymenttargets and create the ones that are not in the existing deploymenttargets
	// and update the ones that are
	for _, workloadDeploymentTarget := range workload.Spec.DeploymentTargets {
		var deploymentTarget *schedulerv1alpha1.DeploymentTarget = nil
		exists := false
		deploymentTargetName := r.buildDeploymentTargetName(workload, workloadDeploymentTarget.Name)

		for _, existingDeploymentTarget := range deploymentTargets.Items {
			if existingDeploymentTarget.Name == deploymentTargetName {
				deploymentTarget = &existingDeploymentTarget
				exists = true
				break
			}
		}

		if deploymentTarget == nil {
			// create the deploymenttarget
			deploymentTarget = &schedulerv1alpha1.DeploymentTarget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentTargetName,
					Namespace: workload.Namespace,
					Labels: map[string]string{
						"workload": workload.Name,
					},
				},
			}
			err = ctrl.SetControllerReference(workload, deploymentTarget, r.Scheme)
			if err != nil {
				return r.manageFailure(ctx, reqLogger, workload, err, "Failed to set controller reference")
			}
		}

		deploymentTarget.Spec = workloadDeploymentTarget.DeploymentTargetSpec

		// compose the deploymenttarget labels of workloadDeploymentTarget labels and workload labels
		deploymentTargetLabels := make(map[string]string)
		for key, value := range workloadDeploymentTarget.Labels {
			deploymentTargetLabels[key] = value
		}
		for key, value := range workload.Labels {
			deploymentTargetLabels[key] = value
		}
		deploymentTargetLabels[schedulerv1alpha1.WorkloadLabel] = workload.Name
		workspaceLabel, err := r.getWorkspaceLabel(ctx, workload)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, workload, err, "Failed to get workspace label")
		}
		deploymentTargetLabels[schedulerv1alpha1.WorkspaceLabel] = *workspaceLabel
		deploymentTarget.Labels = deploymentTargetLabels

		if exists {
			err = r.Update(ctx, deploymentTarget)
			if err != nil {
				return r.manageFailure(ctx, reqLogger, workload, err, "Failed to update DeploymentTarget")
			}
			reqLogger.Info("Updated DeploymentTarget", "DeploymentTarget.Name", deploymentTarget.Name)
		} else {
			err = r.Create(ctx, deploymentTarget)
			if err != nil {
				return r.manageFailure(ctx, reqLogger, workload, err, "Failed to create DeploymentTarget")
			}
			reqLogger.Info("Created DeploymentTarget", "DeploymentTarget.Name", deploymentTarget.Name)
		}

	}

	// update the status
	condition := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "DeploymentTargetsCreated",
	}
	meta.SetStatusCondition(&workload.Status.Conditions, condition)

	updateErr := r.Status().Update(ctx, workload)
	if updateErr != nil {
		reqLogger.Info("Error when updating status.")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}

	return ctrl.Result{}, nil
}

func (r *WorkloadReconciler) buildDeploymentTargetName(workload *schedulerv1alpha1.Workload, deploymentTargetName string) string {
	return workload.Name + "-" + deploymentTargetName
}

func (r *WorkloadReconciler) getWorkspaceLabel(ctx context.Context, workload *schedulerv1alpha1.Workload) (*string, error) {
	workspaceLabel := ""
	if workload.Labels != nil {
		fluxKustomizationName := workload.Labels[FluxOwnerLabel]
		// extract the workspace label from the flux kustomization name by removing the namespace- prefix
		if fluxKustomizationName != "" {
			workloadRegistrationName := strings.SplitN(fluxKustomizationName, "-", 2)[1]
			// get the workload registration
			workloadRegistration := &schedulerv1alpha1.WorkloadRegistration{}
			err := r.Get(ctx, types.NamespacedName{Name: workloadRegistrationName, Namespace: workload.Namespace}, workloadRegistration)
			if err != nil {
				return nil, err
			}
			workspaceLabel = workloadRegistration.Spec.Workspace
		}
	}
	return &workspaceLabel, nil
}

// Gracefully handle errors
func (h *WorkloadReconciler) manageFailure(ctx context.Context, logger logr.Logger, workload *schedulerv1alpha1.Workload, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&workload.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, workload)
	if updateErr != nil {
		logger.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.Workload{}).
		Owns(&schedulerv1alpha1.DeploymentTarget{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Complete(r)
}
