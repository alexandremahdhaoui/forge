//go:build unit

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"reflect"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// -----------------------------------------------------------------------------
// Tests for extractSubenginesFromSpec
// -----------------------------------------------------------------------------

func TestExtractSubenginesFromSpec_ValidSubengines(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		expected []forge.TestenvEngineSpec
	}{
		{
			name: "Single subengine with engine only",
			spec: map[string]interface{}{
				"subengines": []interface{}{
					map[string]interface{}{
						"engine": "go://testenv-kind",
					},
				},
			},
			expected: []forge.TestenvEngineSpec{
				{Engine: "go://testenv-kind"},
			},
		},
		{
			name: "Subengine with spec",
			spec: map[string]interface{}{
				"subengines": []interface{}{
					map[string]interface{}{
						"engine": "go://testenv-kind",
						"spec": map[string]interface{}{
							"kubeconfigPath": "kubeconfig.yaml",
						},
					},
				},
			},
			expected: []forge.TestenvEngineSpec{
				{
					Engine: "go://testenv-kind",
					Spec:   map[string]interface{}{"kubeconfigPath": "kubeconfig.yaml"},
				},
			},
		},
		{
			name: "Subengine with deferTemplates",
			spec: map[string]interface{}{
				"subengines": []interface{}{
					map[string]interface{}{
						"engine":         "go://testenv-helm-install",
						"deferTemplates": true,
					},
				},
			},
			expected: []forge.TestenvEngineSpec{
				{
					Engine:         "go://testenv-helm-install",
					DeferTemplates: true,
				},
			},
		},
		{
			name: "Multiple subengines",
			spec: map[string]interface{}{
				"subengines": []interface{}{
					map[string]interface{}{
						"engine": "go://testenv-kind",
					},
					map[string]interface{}{
						"engine": "go://testenv-lcr",
						"spec": map[string]interface{}{
							"enabled": true,
						},
					},
					map[string]interface{}{
						"engine":         "go://testenv-helm-install",
						"deferTemplates": true,
					},
				},
			},
			expected: []forge.TestenvEngineSpec{
				{Engine: "go://testenv-kind"},
				{Engine: "go://testenv-lcr", Spec: map[string]interface{}{"enabled": true}},
				{Engine: "go://testenv-helm-install", DeferTemplates: true},
			},
		},
		{
			name:     "No subengines key in spec",
			spec:     map[string]interface{}{"other": "value"},
			expected: nil,
		},
		{
			name:     "Empty spec",
			spec:     map[string]interface{}{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSubenginesFromSpec(tt.spec)
			if err != nil {
				t.Errorf("Expected no error, got: %+v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestExtractSubenginesFromSpec_InvalidSubengines(t *testing.T) {
	tests := []struct {
		name          string
		spec          map[string]interface{}
		expectedField string
	}{
		{
			name: "Subengines not an array",
			spec: map[string]interface{}{
				"subengines": "not-an-array",
			},
			expectedField: "spec.subengines",
		},
		{
			name: "Subengines element not an object",
			spec: map[string]interface{}{
				"subengines": []interface{}{
					"not-an-object",
				},
			},
			expectedField: "spec.subengines[0]",
		},
		{
			name: "Second subengine element not an object",
			spec: map[string]interface{}{
				"subengines": []interface{}{
					map[string]interface{}{"engine": "go://testenv-kind"},
					123, // invalid
				},
			},
			expectedField: "spec.subengines[1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSubenginesFromSpec(tt.spec)
			if err == nil {
				t.Error("Expected ValidationError, got nil")
			}
			if result != nil {
				t.Errorf("Expected nil result on error, got: %+v", result)
			}
			if err != nil && err.Field != tt.expectedField {
				t.Errorf("Expected field %q, got: %s", tt.expectedField, err.Field)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Tests for extractSubenginesFromForgeSpec
// -----------------------------------------------------------------------------

func TestExtractSubenginesFromForgeSpec_FromAlias(t *testing.T) {
	forgeSpec := &forge.Spec{
		Test: []forge.TestSpec{
			{
				Name:    "integration",
				Runner:  "go://go-test",
				Testenv: "alias://k8senv",
			},
		},
		Engines: []forge.EngineConfig{
			{
				Alias: "k8senv",
				Type:  forge.TestenvEngineConfigType,
				Testenv: []forge.TestenvEngineSpec{
					{Engine: "go://testenv-kind"},
					{Engine: "go://testenv-lcr", Spec: map[string]interface{}{"enabled": true}},
				},
			},
		},
	}

	result := extractSubenginesFromForgeSpec(forgeSpec, "integration")

	if len(result) != 2 {
		t.Fatalf("Expected 2 subengines, got %d", len(result))
	}
	if result[0].Engine != "go://testenv-kind" {
		t.Errorf("Expected first engine 'go://testenv-kind', got %s", result[0].Engine)
	}
	if result[1].Engine != "go://testenv-lcr" {
		t.Errorf("Expected second engine 'go://testenv-lcr', got %s", result[1].Engine)
	}
}

func TestExtractSubenginesFromForgeSpec_NoAlias(t *testing.T) {
	forgeSpec := &forge.Spec{
		Test: []forge.TestSpec{
			{
				Name:    "integration",
				Runner:  "go://go-test",
				Testenv: "go://testenv", // Direct reference, not alias
			},
		},
	}

	result := extractSubenginesFromForgeSpec(forgeSpec, "integration")

	if result != nil {
		t.Errorf("Expected nil for direct go:// reference, got %+v", result)
	}
}

func TestExtractSubenginesFromForgeSpec_StageNotFound(t *testing.T) {
	forgeSpec := &forge.Spec{
		Test: []forge.TestSpec{
			{
				Name:    "unit",
				Runner:  "go://go-test",
				Testenv: "alias://k8senv",
			},
		},
	}

	result := extractSubenginesFromForgeSpec(forgeSpec, "integration") // Wrong stage name

	if result != nil {
		t.Errorf("Expected nil for nonexistent stage, got %+v", result)
	}
}

func TestExtractSubenginesFromForgeSpec_AliasNotFound(t *testing.T) {
	forgeSpec := &forge.Spec{
		Test: []forge.TestSpec{
			{
				Name:    "integration",
				Runner:  "go://go-test",
				Testenv: "alias://nonexistent",
			},
		},
		Engines: []forge.EngineConfig{
			{
				Alias: "other-alias",
				Type:  forge.TestenvEngineConfigType,
				Testenv: []forge.TestenvEngineSpec{
					{Engine: "go://testenv-kind"},
				},
			},
		},
	}

	result := extractSubenginesFromForgeSpec(forgeSpec, "integration")

	if result != nil {
		t.Errorf("Expected nil for nonexistent alias, got %+v", result)
	}
}

func TestExtractSubenginesFromForgeSpec_NoTestenv(t *testing.T) {
	forgeSpec := &forge.Spec{
		Test: []forge.TestSpec{
			{
				Name:   "unit",
				Runner: "go://go-test",
				// No Testenv field
			},
		},
	}

	result := extractSubenginesFromForgeSpec(forgeSpec, "unit")

	if result != nil {
		t.Errorf("Expected nil for empty testenv, got %+v", result)
	}
}

func TestExtractSubenginesFromForgeSpec_NilSpec(t *testing.T) {
	result := extractSubenginesFromForgeSpec(nil, "integration")

	if result != nil {
		t.Errorf("Expected nil for nil forgeSpec, got %+v", result)
	}
}

// -----------------------------------------------------------------------------
// Tests for getSubengineConfig
// -----------------------------------------------------------------------------

func TestGetSubengineConfig_KindEngine(t *testing.T) {
	forgeSpec := &forge.Spec{
		Kindenv: forge.Kindenv{
			KubeconfigPath: "custom-kubeconfig.yaml",
		},
	}

	result := getSubengineConfig("go://testenv-kind", nil, forgeSpec)

	if result == nil {
		t.Fatal("Expected non-nil result for kind engine")
	}
	if result["kubeconfigPath"] != "custom-kubeconfig.yaml" {
		t.Errorf("Expected kubeconfigPath 'custom-kubeconfig.yaml', got %v", result["kubeconfigPath"])
	}
}

func TestGetSubengineConfig_LCREngine(t *testing.T) {
	forgeSpec := &forge.Spec{
		LocalContainerRegistry: forge.LocalContainerRegistry{
			Enabled:        true,
			CredentialPath: "creds.yaml",
			CaCrtPath:      "ca.crt",
			Namespace:      "registry-ns",
		},
	}

	result := getSubengineConfig("go://testenv-lcr", nil, forgeSpec)

	if result == nil {
		t.Fatal("Expected non-nil result for lcr engine")
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled=true, got %v", result["enabled"])
	}
	if result["namespace"] != "registry-ns" {
		t.Errorf("Expected namespace 'registry-ns', got %v", result["namespace"])
	}
	if result["caCrtPath"] != "ca.crt" {
		t.Errorf("Expected caCrtPath 'ca.crt', got %v", result["caCrtPath"])
	}
}

func TestGetSubengineConfig_HelmInstallEngine(t *testing.T) {
	subengineSpec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name": "my-chart",
				"repo": "https://charts.example.com",
			},
		},
	}

	forgeSpec := &forge.Spec{}

	result := getSubengineConfig("go://testenv-helm-install", subengineSpec, forgeSpec)

	if result == nil {
		t.Fatal("Expected non-nil result for helm-install engine")
	}
	if !reflect.DeepEqual(result, subengineSpec) {
		t.Errorf("Expected helm-install to return subengineSpec directly, got %+v", result)
	}
}

func TestGetSubengineConfig_AliasEngine(t *testing.T) {
	subengineSpec := map[string]interface{}{
		"custom": "value",
	}
	forgeSpec := &forge.Spec{
		Engines: []forge.EngineConfig{
			{Alias: "custom-env"},
		},
	}

	result := getSubengineConfig("alias://custom-env", subengineSpec, forgeSpec)

	if !reflect.DeepEqual(result, subengineSpec) {
		t.Errorf("Expected alias engine to return subengineSpec, got %+v", result)
	}
}

func TestGetSubengineConfig_UnknownEngine(t *testing.T) {
	subengineSpec := map[string]interface{}{
		"data": "value",
	}
	forgeSpec := &forge.Spec{}

	result := getSubengineConfig("go://unknown-engine", subengineSpec, forgeSpec)

	if !reflect.DeepEqual(result, subengineSpec) {
		t.Errorf("Expected unknown engine to return subengineSpec, got %+v", result)
	}
}

func TestGetSubengineConfig_NilForgeSpec(t *testing.T) {
	subengineSpec := map[string]interface{}{
		"data": "value",
	}

	result := getSubengineConfig("go://testenv-kind", subengineSpec, nil)

	if !reflect.DeepEqual(result, subengineSpec) {
		t.Errorf("Expected subengineSpec when forgeSpec is nil, got %+v", result)
	}
}

// -----------------------------------------------------------------------------
// Tests for aggregateResults
// -----------------------------------------------------------------------------

func TestAggregateResults_AllValid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv-kind",
				SpecType: "testenv-subengine",
				SpecName: "integration[0]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
		{
			Ref: engineReference{
				URI:      "go://testenv-lcr",
				SpecType: "testenv-subengine",
				SpecName: "integration[1]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
	}

	output := aggregateResults(results)

	if !output.Valid {
		t.Error("Expected Valid=true when all results are valid")
	}
	if len(output.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(output.Errors))
	}
}

func TestAggregateResults_OneInvalid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv-kind",
				SpecType: "testenv-subengine",
				SpecName: "integration[0]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
		{
			Ref: engineReference{
				URI:      "go://testenv-lcr",
				SpecType: "testenv-subengine",
				SpecName: "integration[1]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{Field: "spec.enabled", Message: "required field is missing"},
				},
			},
		},
	}

	output := aggregateResults(results)

	if output.Valid {
		t.Error("Expected Valid=false when one result is invalid")
	}
	if len(output.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(output.Errors))
	}
	if output.Errors[0].Field != "spec.enabled" {
		t.Errorf("Expected field 'spec.enabled', got %s", output.Errors[0].Field)
	}
	if output.Errors[0].Engine != "go://testenv-lcr" {
		t.Errorf("Expected engine 'go://testenv-lcr', got %s", output.Errors[0].Engine)
	}
}

func TestAggregateResults_InfraError(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv-kind",
				SpecType: "testenv-subengine",
				SpecName: "integration[0]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:      false,
				InfraError: "failed to spawn MCP server",
			},
		},
	}

	output := aggregateResults(results)

	if output.Valid {
		t.Error("Expected Valid=false for infra error")
	}
	if len(output.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(output.Errors))
	}
	if output.Errors[0].Message != "failed to spawn MCP server" {
		t.Errorf("Expected infra error message, got %s", output.Errors[0].Message)
	}
}

func TestAggregateResults_MultipleErrors(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv-kind",
				SpecType: "testenv-subengine",
				SpecName: "integration[0]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{Field: "spec.clusterName", Message: "invalid cluster name"},
				},
			},
		},
		{
			Ref: engineReference{
				URI:      "go://testenv-lcr",
				SpecType: "testenv-subengine",
				SpecName: "integration[1]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{Field: "spec.enabled", Message: "required field"},
					{Field: "spec.namespace", Message: "invalid namespace"},
				},
			},
		},
	}

	output := aggregateResults(results)

	if output.Valid {
		t.Error("Expected Valid=false")
	}
	if len(output.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(output.Errors))
	}
}

func TestAggregateResults_Warnings(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv-kind",
				SpecType: "testenv-subengine",
				SpecName: "integration[0]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Message: "using default cluster name"},
				},
			},
		},
		{
			Ref: engineReference{
				URI:      "go://testenv-lcr",
				SpecType: "testenv-subengine",
				SpecName: "integration[1]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Message: "auto-push is disabled"},
				},
			},
		},
	}

	output := aggregateResults(results)

	if !output.Valid {
		t.Error("Expected Valid=true with only warnings")
	}
	if len(output.Warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(output.Warnings))
	}
}

func TestAggregateResults_NilOutput(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI: "go://testenv-kind",
			},
			Output: nil, // Nil output should be skipped
		},
		{
			Ref: engineReference{
				URI: "go://testenv-lcr",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
	}

	output := aggregateResults(results)

	if !output.Valid {
		t.Error("Expected Valid=true when nil outputs are skipped")
	}
}

func TestAggregateResults_Empty(t *testing.T) {
	results := []validationResult{}

	output := aggregateResults(results)

	if !output.Valid {
		t.Error("Expected Valid=true for empty results")
	}
	if len(output.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(output.Errors))
	}
}

func TestAggregateResults_EngineContextPreserved(t *testing.T) {
	// Test that errors already have engine set preserve it
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv-lcr",
				SpecType: "testenv-subengine",
				SpecName: "integration[1]",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:   "spec.enabled",
						Message: "required",
						Engine:  "go://nested-engine", // Pre-set engine
					},
				},
			},
		},
	}

	output := aggregateResults(results)

	if output.Errors[0].Engine != "go://nested-engine" {
		t.Errorf("Expected engine 'go://nested-engine' to be preserved, got %s", output.Errors[0].Engine)
	}
}

// -----------------------------------------------------------------------------
// Tests for structToMap
// -----------------------------------------------------------------------------

func TestStructToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{
			name: "Kindenv struct",
			input: forge.Kindenv{
				KubeconfigPath: "kubeconfig.yaml",
			},
			expected: map[string]interface{}{
				"kubeconfigPath": "kubeconfig.yaml",
			},
		},
		{
			name: "LocalContainerRegistry struct",
			input: forge.LocalContainerRegistry{
				Enabled:        true,
				CredentialPath: "creds.yaml",
				CaCrtPath:      "ca.crt",
				Namespace:      "registry",
			},
			expected: map[string]interface{}{
				"enabled":        true,
				"credentialPath": "creds.yaml",
				"caCrtPath":      "ca.crt",
				"namespace":      "registry",
			},
		},
		{
			name:     "Empty struct",
			input:    forge.Kindenv{},
			expected: map[string]interface{}{},
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := structToMap(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Key %q: expected %v, got %v", k, v, result[k])
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Tests for configValidateInputToParams
// -----------------------------------------------------------------------------

func TestConfigValidateInputToParams(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: map[string]interface{}{
			"command": "go test",
		},
		ConfigPath: "/path/to/forge.yaml",
		SpecType:   "testenv-subengine",
		SpecName:   "integration[0]",
	}

	params := configValidateInputToParams(input)

	if params == nil {
		t.Fatal("Expected non-nil params")
	}
	if params["configPath"] != "/path/to/forge.yaml" {
		t.Errorf("Expected configPath '/path/to/forge.yaml', got %v", params["configPath"])
	}
	if params["specType"] != "testenv-subengine" {
		t.Errorf("Expected specType 'testenv-subengine', got %v", params["specType"])
	}
	if params["specName"] != "integration[0]" {
		t.Errorf("Expected specName 'integration[0]', got %v", params["specName"])
	}
}

// -----------------------------------------------------------------------------
// Tests for parseConfigValidateOutput
// -----------------------------------------------------------------------------

func TestParseConfigValidateOutput_ValidOutput(t *testing.T) {
	result := map[string]interface{}{
		"valid": true,
		"errors": []interface{}{
			map[string]interface{}{
				"field":   "spec.command",
				"message": "required field",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if output == nil {
		t.Fatal("Expected non-nil output")
	}
	if !output.Valid {
		t.Error("Expected Valid=true")
	}
}

func TestParseConfigValidateOutput_NilResult(t *testing.T) {
	output, err := parseConfigValidateOutput(nil)
	if err != nil {
		t.Fatalf("Expected no error for nil result, got: %v", err)
	}
	if output == nil {
		t.Fatal("Expected non-nil output for nil result")
	}
	if !output.Valid {
		t.Error("Expected Valid=true for nil result (assumes engine didn't implement config-validate)")
	}
}

func TestParseConfigValidateOutput_WithErrors(t *testing.T) {
	result := map[string]interface{}{
		"valid": false,
		"errors": []interface{}{
			map[string]interface{}{
				"field":   "spec.args[0]",
				"message": "expected string",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if output.Valid {
		t.Error("Expected Valid=false")
	}
	if len(output.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(output.Errors))
	}
}

func TestParseConfigValidateOutput_WithWarnings(t *testing.T) {
	result := map[string]interface{}{
		"valid": true,
		"warnings": []interface{}{
			map[string]interface{}{
				"field":   "spec.timeout",
				"message": "using default timeout",
			},
		},
	}

	output, err := parseConfigValidateOutput(result)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(output.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(output.Warnings))
	}
}

func TestParseConfigValidateOutput_InfraError(t *testing.T) {
	result := map[string]interface{}{
		"valid":      false,
		"infraError": "engine process failed",
	}

	output, err := parseConfigValidateOutput(result)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if output.InfraError != "engine process failed" {
		t.Errorf("Expected infraError 'engine process failed', got %s", output.InfraError)
	}
}

// -----------------------------------------------------------------------------
// Tests for validateTestenvSpec (higher-level validation)
// -----------------------------------------------------------------------------

func TestValidateTestenvSpec_NoSubengines(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec:     map[string]interface{}{}, // No subengines
		SpecName: "unit",
	}

	output := validateTestenvSpec(nil, input)

	if !output.Valid {
		t.Error("Expected Valid=true when no subengines to validate")
	}
}

func TestValidateTestenvSpec_SubengineWithoutEngine(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: map[string]interface{}{
			"subengines": []interface{}{
				map[string]interface{}{
					// Missing "engine" field
					"spec": map[string]interface{}{},
				},
			},
		},
		SpecName: "integration",
	}

	output := validateTestenvSpec(nil, input)

	if output.Valid {
		t.Error("Expected Valid=false when subengine has no engine field")
	}
	if len(output.Errors) == 0 {
		t.Error("Expected at least one error for missing engine field")
	}

	found := false
	for _, e := range output.Errors {
		if e.Field == "testenv[0].engine" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error for field 'testenv[0].engine'")
	}
}

func TestValidateTestenvSpec_InvalidSubenginesStructure(t *testing.T) {
	input := mcptypes.ConfigValidateInput{
		Spec: map[string]interface{}{
			"subengines": "not-an-array",
		},
		SpecName: "integration",
	}

	output := validateTestenvSpec(nil, input)

	if output.Valid {
		t.Error("Expected Valid=false for invalid subengines structure")
	}
	if len(output.Errors) == 0 {
		t.Error("Expected at least one error for invalid structure")
	}
}

func TestValidateTestenvSpec_FromForgeSpec(t *testing.T) {
	// Note: This test cannot fully verify MCP calls without integration tests.
	// It verifies the structure extraction from forgeSpec works correctly.

	forgeSpec := &forge.Spec{
		Test: []forge.TestSpec{
			{
				Name:    "integration",
				Runner:  "go://go-test",
				Testenv: "alias://k8senv",
			},
		},
		Engines: []forge.EngineConfig{
			{
				Alias: "k8senv",
				Type:  forge.TestenvEngineConfigType,
				Testenv: []forge.TestenvEngineSpec{
					{Engine: "go://testenv-kind"},
					// Note: The actual MCP call would fail in unit tests
					// but we can verify the extraction logic
				},
			},
		},
	}

	input := mcptypes.ConfigValidateInput{
		Spec:      map[string]interface{}{},
		ForgeSpec: forgeSpec,
		SpecName:  "integration",
	}

	// Note: This will attempt MCP calls which will fail in unit tests.
	// The test verifies the function doesn't panic and handles the error gracefully.
	output := validateTestenvSpec(nil, input)

	// We expect some errors because MCP calls will fail in unit tests
	// but the function should not panic
	_ = output // Just verify it doesn't panic
}

// -----------------------------------------------------------------------------
// Tests for engine reference types
// -----------------------------------------------------------------------------

func TestEngineReference_Fields(t *testing.T) {
	ref := engineReference{
		URI:      "go://testenv-kind",
		SpecType: "testenv-subengine",
		SpecName: "integration[0]",
	}

	if ref.URI != "go://testenv-kind" {
		t.Errorf("Expected URI 'go://testenv-kind', got %s", ref.URI)
	}
	if ref.SpecType != "testenv-subengine" {
		t.Errorf("Expected SpecType 'testenv-subengine', got %s", ref.SpecType)
	}
	if ref.SpecName != "integration[0]" {
		t.Errorf("Expected SpecName 'integration[0]', got %s", ref.SpecName)
	}
}

func TestValidationResult_Fields(t *testing.T) {
	result := validationResult{
		Ref: engineReference{
			URI: "go://testenv-kind",
		},
		Output: &mcptypes.ConfigValidateOutput{
			Valid: true,
		},
	}

	if result.Ref.URI != "go://testenv-kind" {
		t.Errorf("Expected Ref.URI 'go://testenv-kind', got %s", result.Ref.URI)
	}
	if result.Output == nil {
		t.Error("Expected non-nil Output")
	}
	if !result.Output.Valid {
		t.Error("Expected Output.Valid=true")
	}
}
