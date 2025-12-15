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

package mcputil

import (
	"encoding/json"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewValidateOutput_ReturnsValidOutput(t *testing.T) {
	out := NewValidateOutput()

	if out == nil {
		t.Fatal("Expected non-nil output")
	}

	if !out.Valid {
		t.Error("Expected Valid to be true")
	}

	if out.Errors == nil {
		t.Error("Expected Errors to be initialized (not nil)")
	}

	if len(out.Errors) != 0 {
		t.Errorf("Expected empty Errors, got %d errors", len(out.Errors))
	}

	if out.Warnings == nil {
		t.Error("Expected Warnings to be initialized (not nil)")
	}

	if len(out.Warnings) != 0 {
		t.Errorf("Expected empty Warnings, got %d warnings", len(out.Warnings))
	}
}

func TestAddValidationError_SetsValidFalseAndAppendsError(t *testing.T) {
	out := NewValidateOutput()

	AddValidationError(out, "spec.name", "required field is missing")

	if out.Valid {
		t.Error("Expected Valid to be false after adding error")
	}

	if len(out.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(out.Errors))
	}

	err := out.Errors[0]
	if err.Field != "spec.name" {
		t.Errorf("Expected field 'spec.name', got '%s'", err.Field)
	}

	if err.Message != "required field is missing" {
		t.Errorf("Expected message 'required field is missing', got '%s'", err.Message)
	}
}

func TestAddValidationError_AppendsMultipleErrors(t *testing.T) {
	out := NewValidateOutput()

	AddValidationError(out, "spec.name", "required field is missing")
	AddValidationError(out, "spec.timeout", "must be a positive integer")

	if out.Valid {
		t.Error("Expected Valid to be false")
	}

	if len(out.Errors) != 2 {
		t.Fatalf("Expected 2 errors, got %d", len(out.Errors))
	}

	if out.Errors[0].Field != "spec.name" {
		t.Errorf("Expected first error field 'spec.name', got '%s'", out.Errors[0].Field)
	}

	if out.Errors[1].Field != "spec.timeout" {
		t.Errorf("Expected second error field 'spec.timeout', got '%s'", out.Errors[1].Field)
	}
}

func TestAddValidationWarning_AppendsWarningWithoutChangingValid(t *testing.T) {
	out := NewValidateOutput()

	AddValidationWarning(out, "deprecated field will be removed in v2.0")

	if !out.Valid {
		t.Error("Expected Valid to remain true after adding warning")
	}

	if len(out.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(out.Warnings))
	}

	warning := out.Warnings[0]
	if warning.Message != "deprecated field will be removed in v2.0" {
		t.Errorf("Expected warning message, got '%s'", warning.Message)
	}

	if warning.Field != "" {
		t.Errorf("Expected empty field for warning without field, got '%s'", warning.Field)
	}
}

func TestAddValidationWarningWithField_AppendsWarningWithField(t *testing.T) {
	out := NewValidateOutput()

	AddValidationWarningWithField(out, "spec.oldField", "deprecated, use 'newField' instead")

	if !out.Valid {
		t.Error("Expected Valid to remain true after adding warning")
	}

	if len(out.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(out.Warnings))
	}

	warning := out.Warnings[0]
	if warning.Field != "spec.oldField" {
		t.Errorf("Expected field 'spec.oldField', got '%s'", warning.Field)
	}

	if warning.Message != "deprecated, use 'newField' instead" {
		t.Errorf("Expected warning message, got '%s'", warning.Message)
	}
}

func TestAddValidationWarning_MultipleWarnings(t *testing.T) {
	out := NewValidateOutput()

	AddValidationWarning(out, "first warning")
	AddValidationWarning(out, "second warning")

	if !out.Valid {
		t.Error("Expected Valid to remain true")
	}

	if len(out.Warnings) != 2 {
		t.Fatalf("Expected 2 warnings, got %d", len(out.Warnings))
	}
}

func TestAddValidationWarning_DoesNotChangeValidFalse(t *testing.T) {
	out := NewValidateOutput()

	// First add an error to set Valid=false
	AddValidationError(out, "spec.name", "required")

	// Then add a warning
	AddValidationWarning(out, "some warning")

	if out.Valid {
		t.Error("Expected Valid to remain false after adding warning to invalid output")
	}

	if len(out.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(out.Errors))
	}

	if len(out.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(out.Warnings))
	}
}

func TestValidateOutputResult_ValidOutput(t *testing.T) {
	out := NewValidateOutput()

	result := ValidateOutputResult(out)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.IsError {
		t.Error("Expected IsError to be false for valid output")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected Content to have at least one element")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected Content[0] to be *TextContent")
	}

	// Parse the JSON to verify it matches
	var parsed mcptypes.ConfigValidateOutput
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if !parsed.Valid {
		t.Error("Expected parsed Valid to be true")
	}
}

func TestValidateOutputResult_InvalidOutput(t *testing.T) {
	out := NewValidateOutput()
	AddValidationError(out, "spec.name", "required field is missing")

	result := ValidateOutputResult(out)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsError {
		t.Error("Expected IsError to be true for invalid output")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected Content to have at least one element")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected Content[0] to be *TextContent")
	}

	// Parse the JSON to verify it contains the error
	var parsed mcptypes.ConfigValidateOutput
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed.Valid {
		t.Error("Expected parsed Valid to be false")
	}

	if len(parsed.Errors) != 1 {
		t.Fatalf("Expected 1 error in parsed output, got %d", len(parsed.Errors))
	}

	if parsed.Errors[0].Field != "spec.name" {
		t.Errorf("Expected error field 'spec.name', got '%s'", parsed.Errors[0].Field)
	}

	if parsed.Errors[0].Message != "required field is missing" {
		t.Errorf("Expected error message 'required field is missing', got '%s'", parsed.Errors[0].Message)
	}
}

func TestValidateOutputResult_WithWarnings(t *testing.T) {
	out := NewValidateOutput()
	AddValidationWarning(out, "some warning")

	result := ValidateOutputResult(out)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Valid with warnings should NOT be an error
	if result.IsError {
		t.Error("Expected IsError to be false for valid output with warnings")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected Content[0] to be *TextContent")
	}

	var parsed mcptypes.ConfigValidateOutput
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(parsed.Warnings) != 1 {
		t.Fatalf("Expected 1 warning in parsed output, got %d", len(parsed.Warnings))
	}

	if parsed.Warnings[0].Message != "some warning" {
		t.Errorf("Expected warning message 'some warning', got '%s'", parsed.Warnings[0].Message)
	}
}

func TestValidateOutputResult_CompleteScenario(t *testing.T) {
	out := NewValidateOutput()
	AddValidationError(out, "spec.name", "required field is missing")
	AddValidationError(out, "spec.type", "invalid enum value")
	AddValidationWarning(out, "deprecated feature used")
	AddValidationWarningWithField(out, "spec.oldField", "use newField instead")

	result := ValidateOutputResult(out)

	if !result.IsError {
		t.Error("Expected IsError to be true")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected Content[0] to be *TextContent")
	}

	var parsed mcptypes.ConfigValidateOutput
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed.Valid {
		t.Error("Expected parsed Valid to be false")
	}

	if len(parsed.Errors) != 2 {
		t.Fatalf("Expected 2 errors, got %d", len(parsed.Errors))
	}

	if len(parsed.Warnings) != 2 {
		t.Fatalf("Expected 2 warnings, got %d", len(parsed.Warnings))
	}
}
