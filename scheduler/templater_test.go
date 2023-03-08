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
	"os"
	"testing"

	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Unit tests for the Templater interface.
const (
	gitopsDeploymentTargetFile = "testdata/deploymenttarget.yaml"
)

func TestNewTemplater(t *testing.T) {
	getNewTemplater(t)
}

func TestDeploymentTargetAttributes(t *testing.T) {
	templater := getNewTemplater(t)
	assert.NotEmpty(t, templater.data)

	assert.Equal(t, "test-deployment-target", templater.data.DeploymentTargetName)
	assert.Equal(t, "test-environment", templater.data.Environment)
	assert.Equal(t, "test-workload", templater.data.Workload)
	assert.Equal(t, "test-workspace", templater.data.Workspace)
	assert.Equal(t, "test-label-value", templater.data.Labels["test-label-key"])
	assert.Equal(t, "git", templater.data.Manifests["storage"])
	assert.Equal(t, "kustomize", templater.data.Manifests["type"])
}

func TestProcessTemplate(t *testing.T) {
	templater := getNewTemplater(t)
	template := readTemplateFromFile(t, "testdata/template.yaml")

	processedTemplates, err := templater.ProcessTemplate(context.TODO(), template)
	assert.NoError(t, err)
	assert.NotEmpty(t, processedTemplates)

	var unstructuredProcessedTemplates []unstructured.Unstructured

	//convert processedTemplates into a slice of unstructured objects
	for _, processedTemplate := range processedTemplates {
		var unstructuredObject map[string]interface{}
		err = yaml.Unmarshal([]byte(processedTemplate), &unstructuredObject)
		assert.NoError(t, err)
		unstructuredProcessedTemplates = append(unstructuredProcessedTemplates, unstructured.Unstructured{Object: unstructuredObject})
	}

	assert.Equal(t, "GitRepository", unstructuredProcessedTemplates[0].GetKind())
	assert.Equal(t, "test-deployment-target-kustomize", unstructuredProcessedTemplates[0].Object["metadata"].(map[string]interface{})["name"])
	assert.Equal(t, "Kustomization", unstructuredProcessedTemplates[1].GetKind())

}

func readDeploymentTargetFromFile(t *testing.T, filename string) *kalypsov1alpha1.DeploymentTarget {
	// read deployment target from a file
	deploymentTarget := &kalypsov1alpha1.DeploymentTarget{}
	deploymentTargetFile, err := os.Open(filename)
	assert.NoError(t, err)
	decoder := yaml.NewYAMLOrJSONDecoder(deploymentTargetFile, 4096)
	err = decoder.Decode(deploymentTarget)
	assert.NoError(t, err)
	return deploymentTarget
}

func getNewTemplater(t *testing.T) *templater {
	deploymentTarget := readDeploymentTargetFromFile(t, gitopsDeploymentTargetFile)

	templ, err := NewTemplater(deploymentTarget)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, templ)
	}
	return templ.(*templater)
}

func readTemplateFromFile(t *testing.T, filename string) *kalypsov1alpha1.Template {
	// read template from a file
	template := &kalypsov1alpha1.Template{}
	templateFile, err := os.Open(filename)
	assert.NoError(t, err)
	decoder := yaml.NewYAMLOrJSONDecoder(templateFile, 4096)
	err = decoder.Decode(template)
	assert.NoError(t, err)
	return template
}
