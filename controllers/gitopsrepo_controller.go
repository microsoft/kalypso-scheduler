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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/microsoft/kalypso-scheduler/scheduler"
	"github.com/mitchellh/hashstructure"
)

// GitOpsRepoReconciler reconciles a GitOpsRepo object
type GitOpsRepoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const prCreateTimeOut = 3 * time.Second

//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=gitopsrepoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=gitopsrepoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=gitopsrepoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=scheduler.kalypso.io,resources=clustertypes,verbs=get;list;watch

func (r *GitOpsRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling GitOps Repo ===")

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

	// check Ready status condition of all Schedule Policies in the namespace
	// if all are true, set ReadyToPR to true
	schedulePolicies := &schedulerv1alpha1.SchedulingPolicyList{}
	err = r.List(ctx, schedulePolicies, client.InNamespace(gitopsrepo.Namespace))
	if err != nil {
		return r.manageFailure(ctx, reqLogger, gitopsrepo, err, "Failed to list SchedulePolicies")
	}
	readyToPR := true
	for _, schedulePolicy := range schedulePolicies.Items {
		if !meta.IsStatusConditionTrue(schedulePolicy.Status.Conditions, schedulerv1alpha1.ReadyConditionType) {
			readyToPR = false
			//log not all schedule policies are ready
			reqLogger.Info("Not all Schedule Policies are ready")
			break
		}
	}

	if readyToPR {
		// check Ready status condition of all Assignments in the namespace
		// if all are true, set ReadyToPR to true
		assignments := &schedulerv1alpha1.AssignmentList{}
		err = r.List(ctx, assignments, client.InNamespace(gitopsrepo.Namespace))
		if err != nil {
			return r.manageFailure(ctx, reqLogger, gitopsrepo, err, "Failed to list Assignments")
		}
		for _, assignment := range assignments.Items {
			if !meta.IsStatusConditionTrue(assignment.Status.Conditions, schedulerv1alpha1.ReadyConditionType) {
				readyToPR = false
				//log not all assignments are ready
				reqLogger.Info("Not all Assignments are ready")
				break
			}
		}
	}

	if readyToPR {
		//log all assignments and schedule policies are ready
		reqLogger.Info("All Assignments and Schedule Policies are ready")

		repoContent, err := r.getRepoContent(ctx, reqLogger, gitopsrepo)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, gitopsrepo, err, "Failed to get repo content")
		}

		// get the hash of the repoContent
		repoContentHash, err := hashstructure.Hash(repoContent, nil)
		if err != nil {
			return r.manageFailure(ctx, reqLogger, gitopsrepo, err, "Failed to hash the repoContent")
		}

		// convert the hash to a string
		repoContentHashString := strconv.FormatUint(repoContentHash, 10)

		// log hashes
		reqLogger.Info("repoContentHash", "repoContentHash", repoContentHashString)
		reqLogger.Info("gitopsrepo.Status.RepoContentHash", "gitopsrepo.Status.RepoContentHash", gitopsrepo.Status.RepoContentHash)

		// if the hash is different from the one in the status, create a PR
		if gitopsrepo.Status.RepoContentHash != repoContentHashString {

			//get "ReadyForPR" status condition
			readyForPRCondition := meta.FindStatusCondition(gitopsrepo.Status.Conditions, schedulerv1alpha1.ReadyToPRConditionType)
			//chheck it was set before 3 seconds ago
			if readyForPRCondition != nil {
				readyForPRConditionTime := readyForPRCondition.LastTransitionTime.Time
				//wait until things calm down and then create a PR
				//so we dont' create a PR for every change
				if readyForPRConditionTime.Add(prCreateTimeOut).Before(time.Now()) {

					meta.SetStatusCondition(&gitopsrepo.Status.Conditions, metav1.Condition{
						Type:   schedulerv1alpha1.ReadyConditionType,
						Status: metav1.ConditionFalse,
						Reason: "CreatingPR",
					})

					updateErr := r.Status().Update(ctx, gitopsrepo)

					if updateErr != nil {
						reqLogger.Info("Error when updating status.")
						return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
					}

					// create a PR
					reqLogger.Info("!!!!!!!!!!!!!!!!!!!Creating a PR!!!!!!!!!!!!!!!!!!!!!!!!")
					githubRepo, err := scheduler.NewGithubRepo(ctx, &gitopsrepo.Spec)
					if err != nil {
						return r.manageFailure(ctx, reqLogger, gitopsrepo, err, "Failed to create a GithubRepo")
					}

					_, err = githubRepo.CreatePR(r.getDeploymentBranchName(), repoContent)

					if err != nil {
						if r.ignorePrAlreadyExists(err) == nil {
							reqLogger.Info("PR already exists")
						} else {
							return r.manageFailure(ctx, reqLogger, gitopsrepo, err, "Failed to create a PR")
						}
					}

					meta.SetStatusCondition(&gitopsrepo.Status.Conditions, metav1.Condition{
						Type:   schedulerv1alpha1.ReadyConditionType,
						Status: metav1.ConditionTrue,
						Reason: "PRCreated",
					})
					meta.RemoveStatusCondition(&gitopsrepo.Status.Conditions, schedulerv1alpha1.ReadyToPRConditionType)

					gitopsrepo.Status.RepoContentHash = repoContentHashString

					updateErr = r.Status().Update(ctx, gitopsrepo)

					if updateErr != nil {
						reqLogger.Info("Error when updating status.")
						return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
					}
				}

			} else {
				reqLogger.Info("!!!!!!!Ready for PR!!!!!!!!")
				// set the "ReadyForPR" status condition
				meta.SetStatusCondition(&gitopsrepo.Status.Conditions, metav1.Condition{
					Type:   schedulerv1alpha1.ReadyToPRConditionType,
					Status: metav1.ConditionTrue,
					Reason: "ReadyForPR",
				})
				updateErr := r.Status().Update(ctx, gitopsrepo)

				if updateErr != nil {
					reqLogger.Info("Error when updating status.")
					return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
				}

				//create a pr in a timeout
				return ctrl.Result{RequeueAfter: prCreateTimeOut}, nil

			}

		} else {
			if meta.IsStatusConditionTrue(gitopsrepo.Status.Conditions, schedulerv1alpha1.ReadyToPRConditionType) {
				meta.RemoveStatusCondition(&gitopsrepo.Status.Conditions, schedulerv1alpha1.ReadyToPRConditionType)
				updateErr := r.Status().Update(ctx, gitopsrepo)

				if updateErr != nil {
					reqLogger.Info("Error when updating status.")
					return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
				}

			}
		}

	}

	return ctrl.Result{}, nil
}

func (r *GitOpsRepoReconciler) getDeploymentBranchName() string {
	return fmt.Sprintf("deployment/%s", time.Now().Format("2006-01-02-15-04-05"))
}

// ignorePrAlreadyExists returns nil if the error is a PR already exists error
func (r *GitOpsRepoReconciler) ignorePrAlreadyExists(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "A pull request already exists") {
		return nil
	}
	return err
}

func (r *GitOpsRepoReconciler) getRepoContent(ctx context.Context, logger logr.Logger, gitopsrepo *schedulerv1alpha1.GitOpsRepo) (*schedulerv1alpha1.RepoContentType, error) {
	// create repoContent map
	repoContent := schedulerv1alpha1.NewRepoContentType()

	//fetch all cluster types in the namespace
	clusterTypes := &schedulerv1alpha1.ClusterTypeList{}
	err := r.List(ctx, clusterTypes, client.InNamespace(gitopsrepo.Namespace))
	if err != nil {
		return nil, err
	}

	//iterate over all cluster types
	for _, clusterType := range clusterTypes.Items {
		clusterTypeContent := schedulerv1alpha1.NewClusterContentType()
		repoContent.ClusterTypes[clusterType.Name] = *clusterTypeContent
	}

	//fetch all assignment packages in the namespace
	assignmentPackages := &schedulerv1alpha1.AssignmentPackageList{}
	err = r.List(ctx, assignmentPackages, client.InNamespace(gitopsrepo.Namespace))
	if err != nil {
		return nil, err
	}

	//iterate over all assignment packages
	for _, assignmentPackage := range assignmentPackages.Items {
		clusterTypeContent, ok := repoContent.ClusterTypes[assignmentPackage.Labels[schedulerv1alpha1.ClusterTypeLabel]]
		if !ok {
			clusterTypeContent = *schedulerv1alpha1.NewClusterContentType()
			repoContent.ClusterTypes[assignmentPackage.Labels[schedulerv1alpha1.ClusterTypeLabel]] = clusterTypeContent
		}
		clusterTypeContent.DeploymentTargets[assignmentPackage.Labels[schedulerv1alpha1.DeploymentTargetLabel]] = assignmentPackage.Spec
	}

	// list all BaseRepos in the namespace
	baserepos := &schedulerv1alpha1.BaseRepoList{}
	err = r.List(ctx, baserepos, client.InNamespace(gitopsrepo.Namespace))
	if err != nil {
		return nil, err
	}
	if len(baserepos.Items) > 1 {
		return nil, errors.New("There should be only one BaseRepo in the namespace")
	}

	if len(baserepos.Items) == 1 {
		repoContent.BaseRepo = baserepos.Items[0].Spec
	}

	return repoContent, nil
}

// Gracefully handle errors
func (h *GitOpsRepoReconciler) manageFailure(ctx context.Context, logger logr.Logger, gitopsrepo *schedulerv1alpha1.GitOpsRepo, err error, message string) (ctrl.Result, error) {
	logger.Error(err, message)

	//crerate a condition
	condition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  "UpdateFailed",
		Message: err.Error(),
	}

	meta.SetStatusCondition(&gitopsrepo.Status.Conditions, condition)

	updateErr := h.Status().Update(ctx, gitopsrepo)
	if updateErr != nil {
		logger.Info("Error when updating status. Requeued")
		return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
	}
	return ctrl.Result{}, err
}

func (r *GitOpsRepoReconciler) findGitOpsRepo(object client.Object) []reconcile.Request {
	// Find all GitOps repos in the namespace
	gitopsrepos := &schedulerv1alpha1.GitOpsRepoList{}
	err := r.List(context.TODO(), gitopsrepos, client.InNamespace(object.GetNamespace()))
	if err != nil {
		return []reconcile.Request{}
	}

	//Create requests for the gitops repos
	requests := make([]reconcile.Request, len(gitopsrepos.Items))
	for i, gitopsrepo := range gitopsrepos.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      gitopsrepo.Name,
				Namespace: gitopsrepo.Namespace,
			},
		}
	}

	return requests
}

// Reocnile only if the spec has changed or it was triggered by the watched objects (e.g. SchedulingPolicy or Assignment)
func (r *GitOpsRepoReconciler) normalPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if _, ok := e.ObjectOld.(*schedulerv1alpha1.GitOpsRepo); ok {
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			}

			if schedulingPolicy, ok := e.ObjectNew.(*schedulerv1alpha1.SchedulingPolicy); ok {
				return meta.IsStatusConditionTrue(schedulingPolicy.Status.Conditions, schedulerv1alpha1.ReadyConditionType)
			}

			if assignment, ok := e.ObjectNew.(*schedulerv1alpha1.Assignment); ok {
				return meta.IsStatusConditionTrue(assignment.Status.Conditions, schedulerv1alpha1.ReadyConditionType)
			}

			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return !e.DeleteStateUnknown
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitOpsRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulerv1alpha1.GitOpsRepo{}).
		Watches(
			&source.Kind{Type: &schedulerv1alpha1.SchedulingPolicy{}},
			handler.EnqueueRequestsFromMapFunc(r.findGitOpsRepo)).
		Watches(
			&source.Kind{Type: &schedulerv1alpha1.Assignment{}},
			handler.EnqueueRequestsFromMapFunc(r.findGitOpsRepo)).
		Watches(
			&source.Kind{Type: &schedulerv1alpha1.ClusterType{}},
			handler.EnqueueRequestsFromMapFunc(r.findGitOpsRepo)).
		WithEventFilter(r.normalPredicate()).
		Complete(r)
}

// update diagram
// visibility - commit status, health checks
//TODO: remove hardcoding !!!!
// gitRepo.Spec.SecretRef = &meta.LocalObjectReference{
// 	Name: "cluster-config-dev-auth",
// }
// debug info and events
// nice crd output
//
