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
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestErrorResult_CreatesErrorResult(t *testing.T) {
	message := "Test error message"
	result := ErrorResult(message)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsError {
		t.Error("Expected IsError to be true")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected Content to have at least one element")
	}

	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text != message {
			t.Errorf("Expected message '%s', got '%s'", message, textContent.Text)
		}
	} else {
		t.Error("Expected Content[0] to be *TextContent")
	}
}

func TestSuccessResult_CreatesSuccessResult(t *testing.T) {
	message := "Operation successful"
	result := SuccessResult(message)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.IsError {
		t.Error("Expected IsError to be false")
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected Content to have at least one element")
	}

	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
		if textContent.Text != message {
			t.Errorf("Expected message '%s', got '%s'", message, textContent.Text)
		}
	} else {
		t.Error("Expected Content[0] to be *TextContent")
	}
}

func TestSuccessResultWithArtifact_ReturnsArtifact(t *testing.T) {
	message := "Built successfully"
	artifactData := "test-artifact"

	result, artifact := SuccessResultWithArtifact(message, artifactData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.IsError {
		t.Error("Expected IsError to be false")
	}

	if artifact == nil {
		t.Fatal("Expected non-nil artifact")
	}

	if artifact.(string) != artifactData {
		t.Errorf("Expected artifact '%s', got '%s'", artifactData, artifact.(string))
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected Content to have at least one element")
	}
}

func TestSuccessResultWithArtifact_WithComplexArtifact(t *testing.T) {
	type ComplexArtifact struct {
		Name    string
		Version string
	}

	message := "Complex build successful"
	artifactData := ComplexArtifact{
		Name:    "my-app",
		Version: "v1.0.0",
	}

	result, artifact := SuccessResultWithArtifact(message, artifactData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if artifact == nil {
		t.Fatal("Expected non-nil artifact")
	}

	complexArtifact, ok := artifact.(ComplexArtifact)
	if !ok {
		t.Fatal("Expected artifact to be ComplexArtifact")
	}

	if complexArtifact.Name != "my-app" {
		t.Errorf("Expected artifact name 'my-app', got '%s'", complexArtifact.Name)
	}

	if complexArtifact.Version != "v1.0.0" {
		t.Errorf("Expected artifact version 'v1.0.0', got '%s'", complexArtifact.Version)
	}
}
