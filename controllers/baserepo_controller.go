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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
)

// BaseRepoReconciler reconciles a BaseRepo object
type BaseRepoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=baserepoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=baserepoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=baserepoes/finalizers,verbs=update
func (r *BaseRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Base Repo ===")

	// Fetch the BaseRepo instance
	baserepo := &schedulerv1alpha1.BaseRepo{}
	err := r.Get(ctx, req.NamespacedName, baserepo)
	if err != nil {
		ignroredNotFound := client.IgnoreNotFound(err)
		if ignroredNotFound != nil {
			reqLogger.Error(err, "Failed to get GitOpsRepo")
		}
		return ctrl.Result{}, ignroredNotFound
	}

	// Check if the resource is being deleted
	if !baserepo.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	//TODO: delete flux resources if the baserepo is deleted

	flux := NewFlux(ctx, r.Client, baserepo, r.Scheme)
	name := fmt.Sprintf("%s-%s", baserepo.Name, baserepo.Namespace)
	if err := flux.CreateFluxReferenceResources(name, DefaulFluxNamespace, baserepo.Namespace,
		baserepo.Spec.Repo,
		baserepo.Spec.Branch,
		baserepo.Spec.Path,
		baserepo.Spec.Commit); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BaseRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.BaseRepo{}).
		// Owns(&sourcev1.GitRepository{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Owns(&kustomizev1.Kustomization{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
