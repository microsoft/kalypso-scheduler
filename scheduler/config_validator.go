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
	"errors"
	"fmt"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/xeipuuv/gojsonschema"
)

type ConfigValidator interface {
	ValidateValues(ctx context.Context, values map[string]interface{}, schema string) error
}

// implements Validator interface
type validator struct {
}

var _ ConfigValidator = (*validator)(nil)

func NewConfigValidator() ConfigValidator {
	return &validator{}
}

func (v *validator) ValidateValues(ctx context.Context, values map[string]interface{}, schema string) error {
	var chartutilValues map[string]interface{} = make(map[string]interface{})
	for k, v := range values {
		if s, ok := v.(string); ok {
			if i, err := strconv.Atoi(s); err == nil {
				chartutilValues[k] = i
			} else if f, err := strconv.ParseFloat(s, 64); err == nil {
				chartutilValues[k] = f
			} else {
				chartutilValues[k] = v
			}
		} else {
			chartutilValues[k] = v
		}

	}

	err := v.validateAgainstSingleSchema(chartutilValues, []byte(schema))

	return err
}

func (v *validator) validateAgainstSingleSchema(values map[string]interface{}, schemaJSON []byte) (reterr error) {
	defer func() {
		if r := recover(); r != nil {
			reterr = fmt.Errorf("unable to validate schema: %s", r)
		}
	}()

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return err
	}
	if bytes.Equal(valuesJSON, []byte("null")) {
		valuesJSON = []byte("{}")
	}
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
	valuesLoader := gojsonschema.NewBytesLoader(valuesJSON)

	result, err := gojsonschema.Validate(schemaLoader, valuesLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var sb strings.Builder
		for _, desc := range result.Errors() {
			sb.WriteString(fmt.Sprintf("- %s\n", desc))
		}
		return errors.New(sb.String())
	}

	return nil
}
