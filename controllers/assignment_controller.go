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
)

// AssignmentReconciler reconciles a Assignment object
type AssignmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments/finalizers,verbs=update

func (r *AssignmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Assignment ===")

	// Fetch the Assignment instance
	assignment := &schedulerv1alpha1.Assignment{}

	err := r.Get(ctx, req.NamespacedName, assignment)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get Assignment")

		}
		return ctrl.Result{}, ignroredNotFound
	}

	// Check if the resource is being deleted
	if !assignment.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// fetch the templates in the namespace
	templates := &schedulerv1alpha1.TemplateList{}
	err = r.List(ctx, templates, client.InNamespace(req.Namespace))
	if err != nil {
		r.manageFailure(ctx, reqLogger, assignment, err, "Failed to list ClusterTypes")
	}

	// log the templates
	for _, template := range templates.Items {
		reqLogger.Info("Template", "Name", template.Name)
		reqLogger.Info("Template", "Manifests", template.Spec.Manifests)
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// Gracefully handle errors
func (h *AssignmentReconciler) manageFailure(ctx context.Context, logger logr.Logger, assignment *schedulerv1alpha1.Assignment, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Configured",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&assignment.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, assignment)
	if updateErr != nil {
		logger.Error(updateErr, "Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AssignmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.Assignment{}).
		Complete(r)
}
