package scheduler

import (
	"context"
	"testing"
)

var (
	schema = `
	{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "Values",
			"description": "Values for the helm chart",
			"type": "object",
			"properties": {
				"stringRequired": {
					"type": "string"
				},
				"intRequired": {
					"type": "integer",
					"minimum": -90,
					"maximum": 90
		
					},
					"numberRequired": {
						"type": "number",
						"exclusiveMinimum": 0
					},
					"stringOptional": {
						"type": "string"
					},
					"phoneOptional": {
						"type": "string",
						"pattern": "^(\\([0-9]{3}\\))?[0-9]{3}-[0-9]{4}$"
					}
				},
				"required": [
					"stringRequired",
					"intRequired",
					"numberRequired"
				]
			}

`
	values = map[string]string{
		"stringRequired": "value1",
		"intRequired":    "0",
		"numberRequired": "3.14",
		"phoneOptional":  "(888)555-1212",
	}
)

// unit test for NewConfigValidator
func TestNewConfigValidator(t *testing.T) {
	validator := NewConfigValidator()
	if validator == nil {
		t.Errorf("validator is nil")
	}
}

// unit test for ValidateValues
func TestValidateValues(t *testing.T) {
	validator := NewConfigValidator()
	if validator == nil {
		t.Errorf("validator is nil")
	}

	err := validator.ValidateValues(context.Background(), values, schema)
	if err != nil {
		t.Errorf("error validating values: %v", err)
	}
}
