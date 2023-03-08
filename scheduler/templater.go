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
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Templater interface {
	ProcessTemplate(ctx context.Context, template *kalypsov1alpha1.Template) ([]string, error)
}

// implements Templater interface
type templater struct {
	data dataType
}

// validate templater implements Templater interface
var _ Templater = (*templater)(nil)

type dataType struct {
	DeploymentTargetName string
	Namespace            string
	Environment          string
	Workspace            string
	Workload             string
	Labels               map[string]string
	Manifests            map[string]string
}

// new templater function
func NewTemplater(deploymentTarget *kalypsov1alpha1.DeploymentTarget) (Templater, error) {
	return &templater{
		data: newData(deploymentTarget),
	}, nil
}

// implement ProcessTemplate function
func (t *templater) ProcessTemplate(ctx context.Context, template *kalypsov1alpha1.Template) ([]string, error) {
	var processedTemplates []string
	logger := log.FromContext(ctx)
	logger.Info("Hi there")

	//itereate through the manifests
	for _, manifest := range template.Spec.Manifests {
		processedObject, err := t.replaceTemplateVariables(manifest)
		if err != nil {
			logger.Error(err, "error replacing template variables")
			return nil, err
		}

		if processedObject != nil && *processedObject != "" {
			processedTemplates = append(processedTemplates, *processedObject)
		}
	}

	return processedTemplates, nil

}

// recursively replace template variables in a map with appropriate values
func (h *templater) replaceTemplateVariables(s string) (*string, error) {
	//processs the string with text/template
	t, err := template.New("template").Parse(s)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, h.data)
	if err != nil {
		return nil, err
	}
	rs := buf.String()
	return &rs, nil
}

// create a new data struct
func newData(deploymentTarget *kalypsov1alpha1.DeploymentTarget) dataType {
	environment := deploymentTarget.Spec.Environment
	workspace := deploymentTarget.GetWorkspace()
	workload := deploymentTarget.GetWorkload()
	deploymentTargetName := deploymentTarget.Name
	namespace := deploymentTarget.GetTargetNamespace()
	manifests := deploymentTarget.Spec.Manifests

	return dataType{
		DeploymentTargetName: deploymentTargetName,
		Namespace:            namespace,
		Environment:          environment,
		Workspace:            workspace,
		Workload:             workload,
		Labels:               deploymentTarget.GetLabels(),
		Manifests:            manifests,
	}
}
