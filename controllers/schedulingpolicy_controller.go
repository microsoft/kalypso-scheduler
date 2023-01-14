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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
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
		if client.IgnoreNotFound(err) != nil {
			reqLogger.Error(err, "Failed to get SchedulingPolicy")
			return ctrl.Result{}, err
		}
	}

	// TODO(eedorenko): if the scheduling policy is deleted, we should delete the assignmnents

	// fetch the list of clusterctypes in the namespace
	clusterTypes := &schedulerv1alpha1.ClusterTypeList{}
	err = r.List(ctx, clusterTypes, client.InNamespace(req.Namespace))
	if err != nil {
		reqLogger.Error(err, "Failed to list ClusterTypes")
		return ctrl.Result{}, err
	}

	// iterate over the cluster types
	for _, clusterType := range clusterTypes.Items {
		// log the cluster type
		reqLogger.Info("ClusterType", "Name", clusterType.Name)
	}

	// TODO(user): upadte the status of the SchedulingPolicy

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulingPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.SchedulingPolicy{}).
		Complete(r)
}
