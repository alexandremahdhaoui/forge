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
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// -----------------------------------------------------------------------------
// Tests for extractEngineURIs
// -----------------------------------------------------------------------------

func TestExtractEngineURIs_BuildOnly(t *testing.T) {
	spec := forge.Spec{
		Build: forge.Build{
			{
				Name:   "app1",
				Engine: "go://go-build",
				Spec:   map[string]interface{}{"args": []string{"-v"}},
			},
			{
				Name:   "app2",
				Engine: "go://container-build",
				Spec:   nil,
			},
		},
		Test: nil,
	}

	refs := extractEngineURIs(spec)

	if len(refs) != 2 {
		t.Errorf("extractEngineURIs() returned %d refs, want 2", len(refs))
	}

	// Verify first build engine
	found := false
	for _, ref := range refs {
		if ref.URI == "go://go-build" {
			found = true
			if ref.SpecType != "build" {
				t.Errorf("go-build ref.SpecType = %q, want %q", ref.SpecType, "build")
			}
			if ref.SpecName != "app1" {
				t.Errorf("go-build ref.SpecName = %q, want %q", ref.SpecName, "app1")
			}
			if ref.Spec == nil {
				t.Error("go-build ref.Spec = nil, want non-nil")
			}
			break
		}
	}
	if !found {
		t.Error("extractEngineURIs() did not return go://go-build")
	}

	// Verify second build engine
	found = false
	for _, ref := range refs {
		if ref.URI == "go://container-build" {
			found = true
			if ref.SpecType != "build" {
				t.Errorf("container-build ref.SpecType = %q, want %q", ref.SpecType, "build")
			}
			if ref.SpecName != "app2" {
				t.Errorf("container-build ref.SpecName = %q, want %q", ref.SpecName, "app2")
			}
			break
		}
	}
	if !found {
		t.Error("extractEngineURIs() did not return go://container-build")
	}
}

func TestExtractEngineURIs_TestOnly(t *testing.T) {
	spec := forge.Spec{
		Build: nil,
		Test: []forge.TestSpec{
			{
				Name:    "unit",
				Runner:  "go://go-test",
				Testenv: "", // Empty testenv should not be extracted
				Spec:    map[string]interface{}{"packages": []string{"./..."}},
			},
			{
				Name:    "lint",
				Runner:  "go://go-lint",
				Testenv: "noop", // "noop" testenv should not be extracted
				Spec:    nil,
			},
		},
	}

	refs := extractEngineURIs(spec)

	// Should only have 2 refs (the two runners, no testenv)
	if len(refs) != 2 {
		t.Errorf("extractEngineURIs() returned %d refs, want 2", len(refs))
	}

	// Verify go-test runner
	found := false
	for _, ref := range refs {
		if ref.URI == "go://go-test" {
			found = true
			if ref.SpecType != "test" {
				t.Errorf("go-test ref.SpecType = %q, want %q", ref.SpecType, "test")
			}
			if ref.SpecName != "unit" {
				t.Errorf("go-test ref.SpecName = %q, want %q", ref.SpecName, "unit")
			}
			break
		}
	}
	if !found {
		t.Error("extractEngineURIs() did not return go://go-test")
	}

	// Verify go-lint runner
	found = false
	for _, ref := range refs {
		if ref.URI == "go://go-lint" {
			found = true
			if ref.SpecType != "test" {
				t.Errorf("go-lint ref.SpecType = %q, want %q", ref.SpecType, "test")
			}
			break
		}
	}
	if !found {
		t.Error("extractEngineURIs() did not return go://go-lint")
	}
}

func TestExtractEngineURIs_TestWithTestenv(t *testing.T) {
	spec := forge.Spec{
		Build: nil,
		Test: []forge.TestSpec{
			{
				Name:    "integration",
				Runner:  "go://go-test",
				Testenv: "go://testenv",
				Spec:    map[string]interface{}{"tags": []string{"integration"}},
			},
		},
	}

	refs := extractEngineURIs(spec)

	// Should have 2 refs: runner + testenv
	if len(refs) != 2 {
		t.Errorf("extractEngineURIs() returned %d refs, want 2", len(refs))
	}

	// Verify runner
	foundRunner := false
	for _, ref := range refs {
		if ref.URI == "go://go-test" {
			foundRunner = true
			if ref.SpecType != "test" {
				t.Errorf("go-test ref.SpecType = %q, want %q", ref.SpecType, "test")
			}
			if ref.Spec == nil {
				t.Error("go-test ref.Spec = nil, want non-nil")
			}
			break
		}
	}
	if !foundRunner {
		t.Error("extractEngineURIs() did not return go://go-test")
	}

	// Verify testenv
	foundTestenv := false
	for _, ref := range refs {
		if ref.URI == "go://testenv" {
			foundTestenv = true
			if ref.SpecType != "testenv" {
				t.Errorf("testenv ref.SpecType = %q, want %q", ref.SpecType, "testenv")
			}
			if ref.SpecName != "integration" {
				t.Errorf("testenv ref.SpecName = %q, want %q", ref.SpecName, "integration")
			}
			// testenv should have nil Spec (it gets forgeSpec instead)
			if ref.Spec != nil {
				t.Errorf("testenv ref.Spec = %v, want nil", ref.Spec)
			}
			break
		}
	}
	if !foundTestenv {
		t.Error("extractEngineURIs() did not return go://testenv")
	}
}

func TestExtractEngineURIs_Deduplication(t *testing.T) {
	spec := forge.Spec{
		Build: forge.Build{
			{
				Name:   "app1",
				Engine: "go://go-build",
			},
			{
				Name:   "app2",
				Engine: "go://go-build", // Same engine
			},
			{
				Name:   "app3",
				Engine: "go://go-build", // Same engine again
			},
		},
		Test: []forge.TestSpec{
			{
				Name:    "unit",
				Runner:  "go://go-test",
				Testenv: "go://testenv",
			},
			{
				Name:    "integration",
				Runner:  "go://go-test", // Same runner
				Testenv: "go://testenv", // Same testenv
			},
		},
	}

	refs := extractEngineURIs(spec)

	// Should deduplicate: go-build (1) + go-test (1) + testenv (1) = 3
	if len(refs) != 3 {
		t.Errorf("extractEngineURIs() returned %d refs, want 3 (deduplicated)", len(refs))
	}

	// Verify each unique engine appears exactly once
	uriCounts := make(map[string]int)
	for _, ref := range refs {
		uriCounts[ref.URI]++
	}

	for uri, count := range uriCounts {
		if count != 1 {
			t.Errorf("URI %q appears %d times, want 1", uri, count)
		}
	}

	// Verify expected URIs are present
	expectedURIs := []string{"go://go-build", "go://go-test", "go://testenv"}
	for _, expectedURI := range expectedURIs {
		if _, ok := uriCounts[expectedURI]; !ok {
			t.Errorf("Expected URI %q not found in refs", expectedURI)
		}
	}
}

func TestExtractEngineURIs_EmptySpec(t *testing.T) {
	spec := forge.Spec{
		Build: nil,
		Test:  nil,
	}

	refs := extractEngineURIs(spec)

	if len(refs) != 0 {
		t.Errorf("extractEngineURIs() returned %d refs, want 0", len(refs))
	}
}

func TestExtractEngineURIs_EmptyEngineURI(t *testing.T) {
	spec := forge.Spec{
		Build: forge.Build{
			{
				Name:   "app1",
				Engine: "", // Empty engine should be skipped
			},
		},
		Test: []forge.TestSpec{
			{
				Name:    "unit",
				Runner:  "", // Empty runner should be skipped
				Testenv: "",
			},
		},
	}

	refs := extractEngineURIs(spec)

	if len(refs) != 0 {
		t.Errorf("extractEngineURIs() returned %d refs, want 0 (empty URIs should be skipped)", len(refs))
	}
}

// -----------------------------------------------------------------------------
// Tests for aggregateResults
// -----------------------------------------------------------------------------

func TestAggregateResults_AllValid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "build",
				SpecName: "app1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:    true,
				Errors:   nil,
				Warnings: nil,
			},
		},
		{
			Ref: engineReference{
				URI:      "go://go-test",
				SpecType: "test",
				SpecName: "unit",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:    true,
				Errors:   nil,
				Warnings: nil,
			},
		},
	}

	combined := aggregateResults(results)

	if !combined.Valid {
		t.Error("aggregateResults() combined.Valid = false, want true")
	}
	if len(combined.Errors) != 0 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 0", len(combined.Errors))
	}
	if len(combined.Warnings) != 0 {
		t.Errorf("aggregateResults() len(combined.Warnings) = %d, want 0", len(combined.Warnings))
	}
}

func TestAggregateResults_SomeInvalid(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "build",
				SpecName: "app1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
		{
			Ref: engineReference{
				URI:      "go://go-test",
				SpecType: "test",
				SpecName: "unit",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:   "spec.packages",
						Message: "must be a string array",
					},
				},
			},
		},
	}

	combined := aggregateResults(results)

	if combined.Valid {
		t.Error("aggregateResults() combined.Valid = true, want false")
	}
	if len(combined.Errors) != 1 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 1", len(combined.Errors))
	}
	if combined.Errors[0].Field != "spec.packages" {
		t.Errorf("combined.Errors[0].Field = %q, want %q", combined.Errors[0].Field, "spec.packages")
	}
	if combined.Errors[0].Message != "must be a string array" {
		t.Errorf("combined.Errors[0].Message = %q, want %q", combined.Errors[0].Message, "must be a string array")
	}
}

func TestAggregateResults_InfraError(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://unknown-engine",
				SpecType: "build",
				SpecName: "app1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid:      false,
				InfraError: "failed to spawn engine process: executable not found",
			},
		},
	}

	combined := aggregateResults(results)

	if combined.Valid {
		t.Error("aggregateResults() combined.Valid = true, want false")
	}
	if len(combined.Errors) != 1 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 1", len(combined.Errors))
	}

	// InfraError should be converted to a ValidationError
	err := combined.Errors[0]
	if err.Field != "" {
		t.Errorf("infraError converted to ValidationError.Field = %q, want empty string", err.Field)
	}
	if err.Message != "failed to spawn engine process: executable not found" {
		t.Errorf("infraError converted to ValidationError.Message = %q, want original infra error", err.Message)
	}
	if err.Engine != "go://unknown-engine" {
		t.Errorf("infraError converted to ValidationError.Engine = %q, want %q", err.Engine, "go://unknown-engine")
	}
	if err.SpecType != "build" {
		t.Errorf("infraError converted to ValidationError.SpecType = %q, want %q", err.SpecType, "build")
	}
	if err.SpecName != "app1" {
		t.Errorf("infraError converted to ValidationError.SpecName = %q, want %q", err.SpecName, "app1")
	}
}

func TestAggregateResults_MergesWarnings(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "build",
				SpecName: "app1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Field: "spec.args", Message: "deprecated argument '-ldflags' used"},
				},
			},
		},
		{
			Ref: engineReference{
				URI:      "go://go-test",
				SpecType: "test",
				SpecName: "unit",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
				Warnings: []mcptypes.ValidationWarning{
					{Field: "spec.timeout", Message: "timeout not set, using default 10m"},
					{Field: "", Message: "consider enabling race detector"},
				},
			},
		},
	}

	combined := aggregateResults(results)

	if !combined.Valid {
		t.Error("aggregateResults() combined.Valid = false, want true (warnings don't invalidate)")
	}
	if len(combined.Errors) != 0 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 0", len(combined.Errors))
	}
	// Should have 3 warnings total (1 from go-build + 2 from go-test)
	if len(combined.Warnings) != 3 {
		t.Errorf("aggregateResults() len(combined.Warnings) = %d, want 3", len(combined.Warnings))
	}
}

func TestAggregateResults_SetsEngineContext(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "build",
				SpecName: "my-app",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:   "spec.args",
						Message: "invalid argument",
						// Engine not set by the engine itself
						Engine:   "",
						SpecType: "",
						SpecName: "",
					},
				},
			},
		},
	}

	combined := aggregateResults(results)

	if combined.Valid {
		t.Error("aggregateResults() combined.Valid = true, want false")
	}
	if len(combined.Errors) != 1 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 1", len(combined.Errors))
	}

	// Verify context was set on the error
	err := combined.Errors[0]
	if err.Engine != "go://go-build" {
		t.Errorf("aggregateResults() set Engine = %q, want %q", err.Engine, "go://go-build")
	}
	if err.SpecType != "build" {
		t.Errorf("aggregateResults() set SpecType = %q, want %q", err.SpecType, "build")
	}
	if err.SpecName != "my-app" {
		t.Errorf("aggregateResults() set SpecName = %q, want %q", err.SpecName, "my-app")
	}
	// Original fields should be preserved
	if err.Field != "spec.args" {
		t.Errorf("aggregateResults() preserved Field = %q, want %q", err.Field, "spec.args")
	}
	if err.Message != "invalid argument" {
		t.Errorf("aggregateResults() preserved Message = %q, want %q", err.Message, "invalid argument")
	}
}

func TestAggregateResults_PreservesExistingEngineContext(t *testing.T) {
	// When an engine already sets the Engine field (e.g., recursive orchestrators),
	// aggregateResults should NOT overwrite it
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://testenv",
				SpecType: "testenv",
				SpecName: "integration",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{
						Field:    "kindenv.image",
						Message:  "invalid image reference",
						Engine:   "go://testenv-kind", // Already set by recursive orchestrator
						SpecType: "",
						SpecName: "",
					},
				},
			},
		},
	}

	combined := aggregateResults(results)

	if len(combined.Errors) != 1 {
		t.Fatalf("aggregateResults() len(combined.Errors) = %d, want 1", len(combined.Errors))
	}

	err := combined.Errors[0]
	// Engine should NOT be overwritten since it was already set
	if err.Engine != "go://testenv-kind" {
		t.Errorf("aggregateResults() overwrote Engine = %q, want %q (should preserve existing)", err.Engine, "go://testenv-kind")
	}
	// But SpecType and SpecName should still be set
	if err.SpecType != "testenv" {
		t.Errorf("aggregateResults() set SpecType = %q, want %q", err.SpecType, "testenv")
	}
	if err.SpecName != "integration" {
		t.Errorf("aggregateResults() set SpecName = %q, want %q", err.SpecName, "integration")
	}
}

func TestAggregateResults_NilOutput(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "build",
				SpecName: "app1",
			},
			Output: nil, // Nil output should be skipped
		},
		{
			Ref: engineReference{
				URI:      "go://go-test",
				SpecType: "test",
				SpecName: "unit",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: true,
			},
		},
	}

	combined := aggregateResults(results)

	// Should handle nil output gracefully
	if !combined.Valid {
		t.Error("aggregateResults() combined.Valid = false, want true")
	}
	if len(combined.Errors) != 0 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 0", len(combined.Errors))
	}
}

func TestAggregateResults_EmptyResults(t *testing.T) {
	results := []validationResult{}

	combined := aggregateResults(results)

	if !combined.Valid {
		t.Error("aggregateResults() combined.Valid = false, want true (empty results should be valid)")
	}
	if len(combined.Errors) != 0 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 0", len(combined.Errors))
	}
	if len(combined.Warnings) != 0 {
		t.Errorf("aggregateResults() len(combined.Warnings) = %d, want 0", len(combined.Warnings))
	}
}

func TestAggregateResults_MultipleErrorsFromSingleEngine(t *testing.T) {
	results := []validationResult{
		{
			Ref: engineReference{
				URI:      "go://go-build",
				SpecType: "build",
				SpecName: "app1",
			},
			Output: &mcptypes.ConfigValidateOutput{
				Valid: false,
				Errors: []mcptypes.ValidationError{
					{Field: "spec.args", Message: "must be an array"},
					{Field: "spec.env", Message: "must be a map"},
					{Field: "spec.workDir", Message: "must be a string"},
				},
			},
		},
	}

	combined := aggregateResults(results)

	if combined.Valid {
		t.Error("aggregateResults() combined.Valid = true, want false")
	}
	if len(combined.Errors) != 3 {
		t.Errorf("aggregateResults() len(combined.Errors) = %d, want 3", len(combined.Errors))
	}

	// All errors should have context set
	for i, err := range combined.Errors {
		if err.Engine != "go://go-build" {
			t.Errorf("combined.Errors[%d].Engine = %q, want %q", i, err.Engine, "go://go-build")
		}
		if err.SpecType != "build" {
			t.Errorf("combined.Errors[%d].SpecType = %q, want %q", i, err.SpecType, "build")
		}
		if err.SpecName != "app1" {
			t.Errorf("combined.Errors[%d].SpecName = %q, want %q", i, err.SpecName, "app1")
		}
	}
}
