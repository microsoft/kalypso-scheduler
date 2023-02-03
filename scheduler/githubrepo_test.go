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

package scheduler

import (
	"context"
	"fmt"
	"os"
	"testing"

	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	ctx        = context.Background()
	gitOpsRepo = &kalypsov1alpha1.GitOpsRepoSpec{
		ManifestsSpec: kalypsov1alpha1.ManifestsSpec{
			Repo:   "https://github.com/microsoft/kalypso-gitops",
			Path:   ".",
			Branch: "dev",
		},
	}
	reconcilerManifestsFile = "./testdata/reconciler-manifests.yaml"
	namespaceManifestsFile  = "./testdata/namespace-manifests.yaml"
)

// Test NewGithubRepo
func TestNewGithubRepo(t *testing.T) {

	githubRepo, err := NewGithubRepo(ctx,
		gitOpsRepo)
	if err != nil {
		t.Errorf("error creating github repo: %v", err)
	}
	if githubRepo == nil {
		t.Errorf("github repo is nil")
	}

}

// Test CreatePR
func TestCreatePR(t *testing.T) {
	githubRepo, err := NewGithubRepo(ctx,
		gitOpsRepo)
	if err != nil {
		t.Errorf("error creating github repo: %v", err)
	}
	if githubRepo == nil {
		t.Errorf("github repo is nil")
	}

	fmt.Println("reconcilerManifests")
	reconcilerManifests := getManifestsYaml(t, reconcilerManifestsFile)
	fmt.Println(reconcilerManifests)

	fmt.Println("namespaceManifests")
	namespaceManifests := getManifestsYaml(t, namespaceManifestsFile)
	fmt.Println(namespaceManifests)

	//Initialize the package
	assignmentPackageSpec := &kalypsov1alpha1.AssignmentPackageSpec{
		ReconcilerManifests: reconcilerManifests,
		NamespaceManifests:  namespaceManifests,
	}

	repoContentType := kalypsov1alpha1.NewRepoContentType()
	repoContentType.ClusterTypes["drone"] = *kalypsov1alpha1.NewClusterContentType()
	repoContentType.ClusterTypes["drone"].DeploymentTargets["hello-world-app-functional-test"] = *assignmentPackageSpec

	_, err = githubRepo.CreatePR("unit-test", repoContentType)
	if err != nil {
		t.Errorf("can't create PR: %v", err)
	}
	//t.Errorf("Something went wrong")

}
func getManifestsYaml(t *testing.T, filename string) []unstructured.Unstructured {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("could not read the file: %v", err)
	}

	var manifestsObject map[string]interface{}

	// Unmarshal the YAML data into the struct
	err = yaml.Unmarshal(data, &manifestsObject)
	if err != nil {
		t.Errorf("could not unmarshal the data: %v", err)
	}
	fmt.Println(manifestsObject)
	//iterate over manifests
	for _, manifest := range manifestsObject {
		fmt.Println(manifest)
	}
	manifests := unstructured.Unstructured{Object: manifestsObject}

	return []unstructured.Unstructured{manifests}

}
