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
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	kalypsov1alpha1 "github.com/microsoft/kalypso-scheduler/api/v1alpha1"
	"github.com/mitchellh/hashstructure"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Templater interface {
	ProcessTemplate(ctx context.Context, template *kalypsov1alpha1.Template) ([]string, error)
	GetTargetNamespace() string
}

// implements Templater interface
type templater struct {
	data dataType
}

// validate templater implements Templater interface
var _ Templater = (*templater)(nil)

var funcMap = template.FuncMap{
	"toYaml":    toYAML,
	"stringify": stringify,
	"hash":      hash,
	"unquote":   unquote,
}

type dataType struct {
	DeploymentTargetName string
	Namespace            string
	Environment          string
	Workspace            string
	Workload             string
	Labels               map[string]string
	Manifests            map[string]string
	ClusterType          string
	ConfigData           map[string]interface{}
}

// new templater function
func NewTemplater(deploymentTarget *kalypsov1alpha1.DeploymentTarget, clusterType *kalypsov1alpha1.ClusterType, configData map[string]interface{}) (Templater, error) {
	return &templater{
		data: newData(deploymentTarget, clusterType, configData),
	}, nil
}

// implement ProcessTemplate function
func (t *templater) ProcessTemplate(ctx context.Context, template *kalypsov1alpha1.Template) ([]string, error) {
	var processedTemplates []string
	logger := log.FromContext(ctx)

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
	t, err := template.New("template").Funcs(sprig.TxtFuncMap()).Funcs(funcMap).Parse(s)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, h.data)
	if err != nil {
		return nil, err
	}
	rs := buf.String()

	// handle nested templates
	if strings.Contains(rs, "{{") {
		return h.replaceTemplateVariables(rs)
	}

	return &rs, nil
}

// get deployment target namespace
func (h *templater) GetTargetNamespace() string {
	return h.data.Namespace
}

func buildTargetNamespace(deploymentTarget *kalypsov1alpha1.DeploymentTarget, clusterType *kalypsov1alpha1.ClusterType) string {
	return fmt.Sprintf("%s-%s-%s", deploymentTarget.Spec.Environment, clusterType.Name, deploymentTarget.Name)
}

// create a new data struct
func newData(deploymentTarget *kalypsov1alpha1.DeploymentTarget, clusterType *kalypsov1alpha1.ClusterType, configData map[string]interface{}) dataType {
	environment := deploymentTarget.Spec.Environment
	workspace := deploymentTarget.GetWorkspace()
	workload := deploymentTarget.GetWorkload()
	deploymentTargetName := deploymentTarget.Name
	namespace := buildTargetNamespace(deploymentTarget, clusterType)
	manifests := deploymentTarget.Spec.Manifests
	labels := deploymentTarget.GetLabels()
	clusterTypeName := clusterType.Name

	return dataType{
		DeploymentTargetName: deploymentTargetName,
		Namespace:            namespace,
		Environment:          environment,
		Workspace:            workspace,
		Workload:             workload,
		Labels:               labels,
		Manifests:            manifests,
		ClusterType:          clusterTypeName,
		ConfigData:           configData,
	}
}

func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

func hash(v interface{}) string {
	hashValue, err := hashstructure.Hash(v, nil)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(hashValue, 10)
}

func stringify(v interface{}) string {
	// if v is a a map, iterate over keys, if a key is an array or a map, marshal it into string
	if m, ok := v.(map[string]interface{}); ok {
		newMap := make(map[string]interface{}, len(v.(map[string]interface{})))
		for key, value := range m {
			newMap[key] = value
			switch value.(type) {
			case map[string]interface{}, []interface{}:
				data, err := yaml.Marshal(value)
				if err != nil {
					return ""
				}
				newMap[key] = string(data)
			}
		}
		return toYAML(newMap)
	}
	return toYAML(v)
}

func unquote(v interface{}) string {
	if s, ok := v.(string); ok {
		return strings.Trim(strings.Trim(strings.TrimSpace(s), "\""), "'")
	}
	return ""
}
