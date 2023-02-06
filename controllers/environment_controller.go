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

	v1 "k8s.io/api/core/v1"
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

// EnvironmentReconciler reconciles a Environment object
type EnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=environments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=environments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=environments/finalizers,verbs=update
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kustomize.toolkit.fluxcd.io,resources=kustomizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Environemnt ===")

	// Fetch the Environment instance
	environment := &schedulerv1alpha1.Environment{}
	deleted := false
	err := r.Get(ctx, req.NamespacedName, environment)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get Base Rep")
			return ctrl.Result{}, ignroredNotFound
		} else {
			deleted = true
		}

	}

	// Check if the resource is being deleted
	if !environment.ObjectMeta.DeletionTimestamp.IsZero() {
		deleted = true
	}

	nameSpaceName := req.Name
	fluxName := req.Name
	flux := NewFlux(ctx, r.Client)

	namespace := &v1.Namespace{}
	if deleted {
		// delete flux resources
		err := flux.DeleteFluxReferenceResources(fluxName, DefaulFluxNamespace)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, environment, err, "Failed to delete flux resources")
		}
		reqLogger.Info(fmt.Sprintf("Flux resources %s in %s namespace deleted successfully", fluxName, DefaulFluxNamespace))

		// delete the namespace
		err = r.Get(ctx, client.ObjectKey{Name: nameSpaceName}, namespace)
		if err != nil {
			ignroredNotFound := client.IgnoreNotFound(err)
			if ignroredNotFound != nil {
				return r.manageFailure(ctx, reqLogger, environment, err, "Failed to get Namespace")
			}
		}
		reqLogger.Info(fmt.Sprintf("Namespace %s deleted successfully", nameSpaceName))

	} else {
		// get the namespace and if it does not exist create it
		err = r.Get(ctx, client.ObjectKey{Name: nameSpaceName}, namespace)
		if err != nil {
			ignroredNotFound := client.IgnoreNotFound(err)
			if ignroredNotFound != nil {
				return r.manageFailure(ctx, reqLogger, environment, err, "Failed to get Namespace")
			}
			if namespace.Name == "" {
				namespace.Name = nameSpaceName
				if err := r.Create(ctx, namespace); err != nil {
					return r.manageFailure(ctx, reqLogger, environment, err, "Failed to create Namespace")
				}
			}

		}

		reqLogger.Info(fmt.Sprintf("Namespace %s created successfully", nameSpaceName))

		if err := flux.CreateFluxReferenceResources(fluxName, DefaulFluxNamespace, nameSpaceName,
			environment.Spec.ControlPlane.Repo,
			environment.Spec.ControlPlane.Branch,
			environment.Spec.ControlPlane.Path,
			""); err != nil {
			return r.manageFailure(ctx, reqLogger, environment, err, "Failed to create flux resources for the environment")
		}

		reqLogger.Info(fmt.Sprintf("Flux resources %s in %s namespace created successfully", fluxName, DefaulFluxNamespace))

		condition := metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
			Reason: "FluxResourcesCreated",
		}
		meta.SetStatusCondition(&environment.Status.Conditions, condition)

		updateErr := r.Status().Update(ctx, environment)
		if updateErr != nil {
			reqLogger.Info("Error when updating status.")
			return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
		}

	}

	return ctrl.Result{}, nil
}

// Gracefully handle errors
func (h *EnvironmentReconciler) manageFailure(ctx context.Context, logger logr.Logger, environment *schedulerv1alpha1.Environment, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&environment.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, environment)
	if updateErr != nil {
		logger.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.Environment{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
