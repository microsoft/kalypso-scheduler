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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaulFluxNamespace = "flux-system"
	FluxInterval        = 10 * time.Second
)

type Flux interface {
	CreateFluxReferenceResources(name, namespace, targetnamespace, url, branch, path, commit string) error
}

// implements Flux interface
type flux struct {
	ctx    context.Context
	client client.Client
	owner  metav1.Object
	schema *runtime.Scheme
}

// validate flux implements Flux interface
var _ Flux = (*flux)(nil)

// new flux function
func NewFlux(ctx context.Context, client client.Client, owner metav1.Object, schema *runtime.Scheme) Flux {
	return &flux{
		ctx:    ctx,
		client: client,
		owner:  owner,
		schema: schema,
	}
}

func (f *flux) CreateFluxGitRepository(name, namespace, url, branch, commit string) (*sourcev1.GitRepository, error) {
	gitRepo := &sourcev1.GitRepository{}
	repoExists := true
	err := f.client.Get(f.ctx, client.ObjectKey{Name: name, Namespace: namespace}, gitRepo)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		gitRepo.SetName(name)
		gitRepo.SetNamespace(namespace)

		// if err := ctrl.SetControllerReference(f.owner, gitRepo, f.schema); err != nil {
		// 	return nil, err
		// }
		repoExists = false
	}
	gitRepo.Spec.URL = url
	gitRepo.Spec.Reference = &sourcev1.GitRepositoryRef{
		Branch: branch,
		Commit: commit,
	}

	//TODO: remove hardcoding
	gitRepo.Spec.SecretRef = &meta.LocalObjectReference{
		Name: "cluster-config-dev-auth",
	}

	gitRepo.Spec.Interval = metav1.Duration{Duration: FluxInterval}

	if repoExists {
		err = f.client.Update(f.ctx, gitRepo)
		if err != nil {
			return nil, err
		}
	} else {
		err = f.client.Create(f.ctx, gitRepo)
		if err != nil {
			return nil, err
		}
	}

	return gitRepo, nil
}

// Create Flux Kustomization
func (f *flux) CreateFluxKsutomization(name, namespace, targetnamespace, path string) (*kustomizev1.Kustomization, error) {
	kustomization := &kustomizev1.Kustomization{}
	kustomizationExists := true
	err := f.client.Get(f.ctx, client.ObjectKey{Name: name, Namespace: namespace}, kustomization)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		kustomization.SetName(name)
		kustomization.SetNamespace(namespace)
		// if err := ctrl.SetControllerReference(f.owner, kustomization, f.schema); err != nil {
		// 	return nil, err
		// }
		kustomizationExists = false
	}
	kustomization.Spec.Path = path
	kustomization.Spec.Prune = true
	kustomization.Spec.Interval = metav1.Duration{Duration: FluxInterval}
	kustomization.Spec.SourceRef = kustomizev1.CrossNamespaceSourceReference{
		Kind: sourcev1.GitRepositoryKind,
		Name: name,
	}
	kustomization.Spec.TargetNamespace = targetnamespace

	if kustomizationExists {
		err = f.client.Update(f.ctx, kustomization)
		if err != nil {
			return nil, err
		}
	} else {
		err = f.client.Create(f.ctx, kustomization)
		if err != nil {
			return nil, err
		}
	}

	return kustomization, nil
}

func (f *flux) CreateFluxReferenceResources(name, namespace, targetnamespace, url, branch, path, commit string) error {

	//create Flux GitRepository
	_, err := f.CreateFluxGitRepository(name, namespace, url, branch, commit)
	if err != nil {
		return err
	}

	//create Flux Kustomization
	_, err = f.CreateFluxKsutomization(name, namespace, targetnamespace, path)
	if err != nil {
		return err
	}

	return nil
}
