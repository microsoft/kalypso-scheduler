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
	"bytes"
	"context"
	"html/template"

	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Templater interface {
	ProcessTemplate(ctx context.Context, template *kalypsov1alpha1.Template) ([]unstructured.Unstructured, error)
}

// implements Templater interface
type templater struct {
	data dataType
}

// validate templater implements Templater interface
var _ Templater = (*templater)(nil)

type dataType struct {
	DeploymentTargetName string
	Repo                 string
	Branch               string
	Path                 string
	Namespace            string
	Environment          string
	Workspace            string
	Workload             string
}

// new templater function
func NewTemplater(deploymentTarget *kalypsov1alpha1.DeploymentTarget) (Templater, error) {
	return &templater{
		data: newData(deploymentTarget),
	}, nil
}

// implement ProcessTemplate function
func (t *templater) ProcessTemplate(ctx context.Context, template *kalypsov1alpha1.Template) ([]unstructured.Unstructured, error) {
	var processedTemplates []unstructured.Unstructured
	logger := log.FromContext(ctx)

	//itereate through the manifests
	for _, manifest := range template.Spec.Manifests {
		processedObject, err := t.replaceTemplateVariables(manifest.Object)
		if err != nil {
			logger.Error(err, "error replacing template variables")
			return nil, err
		}

		manifest.Object = processedObject
		// append manifest to processedTemplates
		processedTemplates = append(processedTemplates, manifest)

	}

	return processedTemplates, nil

}

// recursively replace template variables in a map with appropriate values
func (h *templater) replaceTemplateVariables(m map[string]interface{}) (map[string]interface{}, error) {
	for k, v := range m {
		switch v := v.(type) {
		case map[string]interface{}:
			// recurse
			mk, err := h.replaceTemplateVariables(v)
			if err != nil {
				return nil, err
			}
			m[k] = mk
		case string:
			//processs the string woth text/template
			t, err := template.New("template").Parse(v)
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			err = t.Execute(&buf, h.data)
			if err != nil {
				return nil, err
			}
			m[k] = buf.String()
		}
	}
	return m, nil
}

// create a new data struct
func newData(deploymentTarget *kalypsov1alpha1.DeploymentTarget) dataType {
	environment := deploymentTarget.Spec.Environment
	workspace := deploymentTarget.GetWorkspace()
	workload := deploymentTarget.GetWorkload()
	deploymentTargetName := deploymentTarget.Name
	namespace := deploymentTarget.GetTargetNamespace()

	return dataType{
		DeploymentTargetName: deploymentTargetName,
		Repo:                 deploymentTarget.Spec.Manifests.Repo,
		Branch:               deploymentTarget.Spec.Manifests.Branch,
		Path:                 deploymentTarget.Spec.Manifests.Path,
		Namespace:            namespace,
		Environment:          environment,
		Workspace:            workspace,
		Workload:             workload,
	}
}
