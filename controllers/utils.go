package controllers

import (
	"context"
	"fmt"
	"strconv"

	schedulerv1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/mitchellh/hashstructure"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// get hash string
func getHashString(object interface{}) (string, error) {
	hash, err := hashstructure.Hash(object, nil)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(hash, 10), nil
}

// find GitOps repo
func findGitOpsRepo(ctx context.Context, r client.Client, object client.Object) (*schedulerv1alpha1.GitOpsRepo, error) {
	//list gitops repos in the object namespace
	gitopsrepos := &schedulerv1alpha1.GitOpsRepoList{}
	listOpts := []client.ListOption{
		client.InNamespace(object.GetNamespace()),
	}

	err := r.List(ctx, gitopsrepos, listOpts...)
	if err != nil {
		return nil, err
	}

	if len(gitopsrepos.Items) > 0 {
		return &gitopsrepos.Items[0], nil
	} else {
		return nil, fmt.Errorf("no GitOps repo found in namespace %s", object.GetNamespace())
	}

}
