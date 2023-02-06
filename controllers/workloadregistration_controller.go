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
	"fmt"
	"time"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
)

// WorkloadRegistrationReconciler reconciles a WorkloadRegistration object
type WorkloadRegistrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=workloadregistrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=workloadregistrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=workloadregistrations/finalizers,verbs=update

func (r *WorkloadRegistrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Workload Registration ===")

	// Fetch the WorkloadRegistration instance
	workloadRegistration := &schedulerv1alpha1.WorkloadRegistration{}
	deleted := false
	err := r.Get(ctx, req.NamespacedName, workloadRegistration)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get Workload Registration")
			return ctrl.Result{}, ignroredNotFound
		} else {
			deleted = true
		}

	}

	// Check if the resource is being deleted
	if !workloadRegistration.ObjectMeta.DeletionTimestamp.IsZero() {
		deleted = true
	}

	flux := NewFlux(ctx, r.Client)
	name := fmt.Sprintf("%s-%s", req.Namespace, req.Name)
	if deleted {
		err := flux.DeleteFluxReferenceResources(name, DefaulFluxNamespace)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, workloadRegistration, err, "Failed to delete flux resources")
		}
		reqLogger.Info(fmt.Sprintf("Flux resources %s in %s namespace deleted successfully", name, DefaulFluxNamespace))
	} else {

		// Create flux resources

		err = flux.CreateFluxReferenceResources(name, DefaulFluxNamespace,
			workloadRegistration.Namespace,
			workloadRegistration.Spec.Workload.Repo,
			workloadRegistration.Spec.Workload.Branch,
			workloadRegistration.Spec.Workload.Path,
			"")

		if err != nil {
			return r.manageFailure(ctx, reqLogger, workloadRegistration, err, "Failed to create flux resources")
		}

		condition := metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
			Reason: "FluxResourcesCreated",
		}
		meta.SetStatusCondition(&workloadRegistration.Status.Conditions, condition)

		updateErr := r.Status().Update(ctx, workloadRegistration)
		if updateErr != nil {
			reqLogger.Info("Error when updating status.")
			return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
		}
	}

	return ctrl.Result{}, nil
}

// Gracefully handle errors
func (h *WorkloadRegistrationReconciler) manageFailure(ctx context.Context, logger logr.Logger, workloadRegistration *schedulerv1alpha1.WorkloadRegistration, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&workloadRegistration.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, workloadRegistration)
	if updateErr != nil {
		logger.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.WorkloadRegistration{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
