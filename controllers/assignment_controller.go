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

	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	corev1 "k8s.io/api/core/v1"

	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/microsoft/kalypso-scheduler/scheduler"
)

// AssignmentReconciler reconciles a Assignment object
type AssignmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	platformConfigLabel = "platform-config"
)

// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignments/finalizers,verbs=update
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=assignmentpackages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=templates,verbs=get;list;watch;
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=clustertypes,verbs=get;list;watch;
// +kubebuilder:rbac:groups=scheduler.kalypso.io,resources=deploymenttargets,verbs=get;list;watch;
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;

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

	condition := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionFalse,
		Reason: "RebuildingAssignmentPackage",
	}
	meta.SetStatusCondition(&assignment.Status.Conditions, condition)

	updateErr := r.Status().Update(ctx, assignment)
	if updateErr != nil {
		reqLogger.Info("Error when updating status.")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}

	// fetch the assignnment cluster type
	clusterType := &schedulerv1alpha1.ClusterType{}
	err = r.Get(ctx, client.ObjectKey{Name: assignment.Spec.ClusterType, Namespace: assignment.Namespace}, clusterType)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get ClusterType")
	}

	// fetch the deploymentTarget
	deploymentTarget := &schedulerv1alpha1.DeploymentTarget{}
	err = r.Get(ctx, client.ObjectKey{Name: assignment.Spec.DeploymentTarget, Namespace: assignment.Namespace}, deploymentTarget)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get DeploymentTarget")
	}

	templater, err := scheduler.NewTemplater(deploymentTarget)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get templater")
	}

	// get the reconciler manifests
	reconcilerManifests, err := r.getReconcilerManifests(ctx, clusterType, templater)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get reconciler manifests")
	}

	//log reconcilerManifests
	reqLogger.Info("Reconciler Manifests", "Manifests", reconcilerManifests)

	// get the namespace manifests
	namespaceManifests, err := r.getNamespaceManifests(ctx, clusterType, templater)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get namespace manifests")
	}

	// log namespaceManifests
	reqLogger.Info("Namespace Manifests", "Manifests", namespaceManifests)

	//get configManifests
	configManifests, err := r.getConfigManifests(ctx, clusterType, deploymentTarget)
	if err != nil {
		return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get config manifests")
	}

	// log configManifests
	reqLogger.Info("Config Manifests", "Manifests", configManifests)

	// get the assignment package by label selector if doesn't exist create it
	assignmentPackage := &schedulerv1alpha1.AssignmentPackage{}
	packageExists := true
	err = r.Get(ctx, client.ObjectKey{Name: assignment.Name, Namespace: assignment.Namespace}, assignmentPackage)
	if err != nil {
		if !errors.IsNotFound(err) {
			return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to get AssignmentPackage")
		}

		// create the assignment package
		assignmentPackage.SetName(assignment.Name)
		assignmentPackage.SetNamespace(assignment.Namespace)

		if err := ctrl.SetControllerReference(assignment, assignmentPackage, r.Scheme); err != nil {
			return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to set controller reference")
		}
		packageExists = false
	}

	assignmentPackage.Spec.ReconcilerManifests = reconcilerManifests
	assignmentPackage.Spec.NamespaceManifests = namespaceManifests
	assignmentPackage.Spec.ConfigManifests = configManifests

	assignmentPackage.SetLabels(map[string]string{
		schedulerv1alpha1.ClusterTypeLabel:      assignment.Spec.ClusterType,
		schedulerv1alpha1.WorkloadLabel:         assignment.Spec.Workload,
		schedulerv1alpha1.DeploymentTargetLabel: assignment.Spec.DeploymentTarget,
	})

	//log assignmentPackage
	reqLogger.Info("Assignment Package", "AssignmentPackage", assignmentPackage)

	if packageExists {
		err = r.Update(ctx, assignmentPackage)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to update assignment package")
		}
	} else {
		err = r.Create(ctx, assignmentPackage)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, assignment, err, "Failed to create assignment package")
		}
	}

	condition = metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "AssignmentPackageCreated",
	}
	meta.SetStatusCondition(&assignment.Status.Conditions, condition)

	updateErr = r.Status().Update(ctx, assignment)
	if updateErr != nil {
		reqLogger.Info("Error when updating status.")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}

	return ctrl.Result{}, nil
}

// Gracefully handle errors
func (h *AssignmentReconciler) manageFailure(ctx context.Context, logger logr.Logger, assignment *schedulerv1alpha1.Assignment, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&assignment.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, assignment)
	if updateErr != nil {
		logger.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

// get the reconciler manifests
func (r *AssignmentReconciler) getReconcilerManifests(ctx context.Context, clusterType *schedulerv1alpha1.ClusterType, templater scheduler.Templater) ([]unstructured.Unstructured, error) {

	// fetch the cluster type reconciler template
	template := &schedulerv1alpha1.Template{}
	err := r.Get(ctx, client.ObjectKey{Name: clusterType.Spec.Reconciler, Namespace: clusterType.Namespace}, template)
	if err != nil {
		return nil, err
	}

	reconcilerManifests, err := templater.ProcessTemplate(ctx, template)
	if err != nil {
		return nil, err
	}

	return reconcilerManifests, nil
}

// get the namespace manifests
func (r *AssignmentReconciler) getNamespaceManifests(ctx context.Context, clusterType *schedulerv1alpha1.ClusterType, templater scheduler.Templater) ([]unstructured.Unstructured, error) {
	// fetch the cluster type namespace template
	template := &schedulerv1alpha1.Template{}
	err := r.Get(ctx, client.ObjectKey{Name: clusterType.Spec.NamespaceService, Namespace: clusterType.Namespace}, template)
	if err != nil {
		return nil, err
	}

	namespaceManifests, err := templater.ProcessTemplate(ctx, template)
	if err != nil {
		return nil, err
	}

	return namespaceManifests, nil
}

// get the config manifests
func (r *AssignmentReconciler) getConfigManifests(ctx context.Context, clusterType *schedulerv1alpha1.ClusterType, deploymentTarget *schedulerv1alpha1.DeploymentTarget) ([]unstructured.Unstructured, error) {
	// fetch all config maps in the cluster type namespace that have the label "platform-config: true"
	configMaps := &corev1.ConfigMapList{}
	err := r.List(ctx, configMaps, client.InNamespace(clusterType.Namespace), client.MatchingLabels{platformConfigLabel: "true"})
	if err != nil {
		return nil, err
	}

	//iterate ovrer the config maps and select those that satisfy the cluster type labels
	var clusterConfigData map[string]string = make(map[string]string)
	for _, configMap := range configMaps.Items {
		if r.isConfigForClusterType(&configMap, clusterType) {
			//add config map data to the cluster config data
			for key, value := range configMap.Data {
				clusterConfigData[key] = value
			}
		}
	}

	var manifests []unstructured.Unstructured
	manifest := unstructured.Unstructured{}
	manifest.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	})
	manifest.SetName(platformConfigLabel)
	manifest.SetNamespace(deploymentTarget.GetTargetNamespace())
	manifest.Object["data"] = clusterConfigData
	manifests = append(manifests, manifest)

	return manifests, nil
}

func (r *AssignmentReconciler) isConfigForClusterType(config *corev1.ConfigMap, clusterType *schedulerv1alpha1.ClusterType) bool {
	matches := true
	for key, value := range config.Labels {
		//TODO: have own labels namespace
		if key != FluxOwnerLabel && key != FluxNamespaceLabel && key != platformConfigLabel {
			if key == schedulerv1alpha1.ClusterTypeLabel {
				if value != clusterType.Name {
					matches = false
					break
				}
			} else {
				clusterTypeLabeValue := clusterType.Labels[key]
				if clusterTypeLabeValue != value {
					matches = false
					break
				}
			}
		}

	}
	return matches
}

func (r *AssignmentReconciler) findAssignmentsForTemplate(object client.Object) []reconcile.Request {
	//get template
	template := &schedulerv1alpha1.Template{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}, template)
	if err != nil {
		return []reconcile.Request{}
	}

	//get cluster types that use this template as a reconciler
	clusterTypes := &schedulerv1alpha1.ClusterTypeList{}
	err = r.List(context.TODO(), clusterTypes, client.InNamespace(object.GetNamespace()), client.MatchingFields{ReconcilerField: template.Name})
	if err != nil {
		return []reconcile.Request{}
	}

	//get cluster types that use this template as a namespace service
	clusterTypesNameSpace := &schedulerv1alpha1.ClusterTypeList{}
	err = r.List(context.TODO(), clusterTypesNameSpace, client.InNamespace(object.GetNamespace()), client.MatchingFields{NamespaceServiceField: template.Name})
	if err != nil {
		return []reconcile.Request{}
	}

	//append the two lists
	clusterTypes.Items = append(clusterTypes.Items, clusterTypesNameSpace.Items...)

	var requests []reconcile.Request
	// iterate over the cluster types and find the assignments
	for _, clusterType := range clusterTypes.Items {
		assignments := &schedulerv1alpha1.AssignmentList{}
		err = r.List(context.TODO(), assignments, client.InNamespace(object.GetNamespace()), client.MatchingFields{ClusterTypeField: clusterType.Name})
		if err != nil {
			return []reconcile.Request{}
		}

		for _, item := range assignments.Items {
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
			requests = append(requests, request)
		}
	}

	return requests
}

// findAssignmentsForConfigMap finds all assignments
func (r *AssignmentReconciler) findAssignmentsForConfigMap(object client.Object) []reconcile.Request {

	var requests []reconcile.Request
	assignments := &schedulerv1alpha1.AssignmentList{}
	err := r.List(context.TODO(), assignments, client.InNamespace(object.GetNamespace()))
	if err != nil {
		return []reconcile.Request{}
	}

	for _, item := range assignments.Items {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
		requests = append(requests, request)
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *AssignmentReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.Assignment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&schedulerv1alpha1.AssignmentPackage{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &schedulerv1alpha1.Template{}},
			handler.EnqueueRequestsFromMapFunc(r.findAssignmentsForTemplate)).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findAssignmentsForConfigMap)).
		Complete(r)
}
