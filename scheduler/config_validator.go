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
	"strconv"

	chartutil "helm.sh/helm/v3/pkg/chartutil"
)

type ConfigValidator interface {
	ValidateValues(ctx context.Context, values map[string]string, schema string) error
}

// implements Validator interface
type validator struct {
}

var _ ConfigValidator = (*validator)(nil)

func NewConfigValidator() ConfigValidator {
	return &validator{}
}

func (v *validator) ValidateValues(ctx context.Context, values map[string]string, schema string) error {
	var chartutilValues chartutil.Values = make(map[string]interface{})
	// convert map[string]string to chartutil.Values
	for k, v := range values {
		if i, err := strconv.Atoi(v); err == nil {
			chartutilValues[k] = i
		} else if f, err := strconv.ParseFloat(v, 64); err == nil {
			chartutilValues[k] = f
		} else {
			chartutilValues[k] = v
		}
	}

	err := chartutil.ValidateAgainstSingleSchema(chartutilValues, []byte(schema))

	return err
}
