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

package testrunner

import (
	"strings"
	"testing"
)

func TestAssertResult_ExactMatch(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "string match",
			actual:   map[string]interface{}{"key": "value"},
			expected: map[string]interface{}{"key": "value"},
			wantErr:  false,
		},
		{
			name:     "string mismatch",
			actual:   map[string]interface{}{"key": "actual"},
			expected: map[string]interface{}{"key": "expected"},
			wantErr:  true,
		},
		{
			name:     "numeric int match",
			actual:   map[string]interface{}{"exitCode": 0},
			expected: map[string]interface{}{"exitCode": 0},
			wantErr:  false,
		},
		{
			name:     "numeric float64 match",
			actual:   map[string]interface{}{"exitCode": float64(0)},
			expected: map[string]interface{}{"exitCode": float64(0)},
			wantErr:  false,
		},
		{
			name:     "nil actual mismatch",
			actual:   map[string]interface{}{"key": nil},
			expected: map[string]interface{}{"key": "value"},
			wantErr:  true,
		},
		{
			name:     "empty map passes with empty expected",
			actual:   map[string]interface{}{},
			expected: map[string]interface{}{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_Length(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "correct length",
			actual:   map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"length": 3}},
			wantErr:  false,
		},
		{
			name:     "incorrect length",
			actual:   map[string]interface{}{"items": []interface{}{"a"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"length": 3}},
			wantErr:  true,
		},
		{
			name:     "length on non-array",
			actual:   map[string]interface{}{"items": "not-an-array"},
			expected: map[string]interface{}{"items": map[string]interface{}{"length": 1}},
			wantErr:  true,
		},
		{
			name:     "length with float64 expected",
			actual:   map[string]interface{}{"items": []interface{}{"a", "b"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"length": float64(2)}},
			wantErr:  false,
		},
		{
			name:     "length with non-numeric expected",
			actual:   map[string]interface{}{"items": []interface{}{"a"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"length": "two"}},
			wantErr:  true,
		},
		{
			name:     "empty array length zero",
			actual:   map[string]interface{}{"items": []interface{}{}},
			expected: map[string]interface{}{"items": map[string]interface{}{"length": 0}},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_Contains(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "string contains substring",
			actual:   map[string]interface{}{"stdout": "hello world version 1.0"},
			expected: map[string]interface{}{"stdout": map[string]interface{}{"contains": []interface{}{"hello", "version"}}},
			wantErr:  false,
		},
		{
			name:     "string missing substring",
			actual:   map[string]interface{}{"stdout": "hello world"},
			expected: map[string]interface{}{"stdout": map[string]interface{}{"contains": []interface{}{"goodbye"}}},
			wantErr:  true,
		},
		{
			name:     "array contains element",
			actual:   map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"contains": []interface{}{"b"}}},
			wantErr:  false,
		},
		{
			name:     "array missing element",
			actual:   map[string]interface{}{"items": []interface{}{"a", "b"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"contains": []interface{}{"z"}}},
			wantErr:  true,
		},
		{
			name:     "contains on non-string-non-array",
			actual:   map[string]interface{}{"val": 42},
			expected: map[string]interface{}{"val": map[string]interface{}{"contains": []interface{}{"42"}}},
			wantErr:  true,
		},
		{
			name:     "contains with non-list expected",
			actual:   map[string]interface{}{"stdout": "hello"},
			expected: map[string]interface{}{"stdout": map[string]interface{}{"contains": "hello"}},
			wantErr:  true,
		},
		{
			name:     "string contains empty list",
			actual:   map[string]interface{}{"stdout": "hello"},
			expected: map[string]interface{}{"stdout": map[string]interface{}{"contains": []interface{}{}}},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_NotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "non-empty string",
			actual:   map[string]interface{}{"val": "hello"},
			expected: map[string]interface{}{"val": map[string]interface{}{"notEmpty": true}},
			wantErr:  false,
		},
		{
			name:     "empty string",
			actual:   map[string]interface{}{"val": ""},
			expected: map[string]interface{}{"val": map[string]interface{}{"notEmpty": true}},
			wantErr:  true,
		},
		{
			name:     "nil value",
			actual:   map[string]interface{}{"val": nil},
			expected: map[string]interface{}{"val": map[string]interface{}{"notEmpty": true}},
			wantErr:  true,
		},
		{
			name:     "non-empty array",
			actual:   map[string]interface{}{"items": []interface{}{"a"}},
			expected: map[string]interface{}{"items": map[string]interface{}{"notEmpty": true}},
			wantErr:  false,
		},
		{
			name:     "empty array",
			actual:   map[string]interface{}{"items": []interface{}{}},
			expected: map[string]interface{}{"items": map[string]interface{}{"notEmpty": true}},
			wantErr:  true,
		},
		{
			name:     "non-empty map",
			actual:   map[string]interface{}{"obj": map[string]interface{}{"k": "v"}},
			expected: map[string]interface{}{"obj": map[string]interface{}{"notEmpty": true}},
			wantErr:  false,
		},
		{
			name:     "empty map",
			actual:   map[string]interface{}{"obj": map[string]interface{}{}},
			expected: map[string]interface{}{"obj": map[string]interface{}{"notEmpty": true}},
			wantErr:  true,
		},
		{
			name:     "non-nil non-collection passes",
			actual:   map[string]interface{}{"val": 42},
			expected: map[string]interface{}{"val": map[string]interface{}{"notEmpty": true}},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_Matches(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "regex matches",
			actual:   map[string]interface{}{"version": "v1.2.3"},
			expected: map[string]interface{}{"version": map[string]interface{}{"matches": `^v\d+\.\d+\.\d+$`}},
			wantErr:  false,
		},
		{
			name:     "regex does not match",
			actual:   map[string]interface{}{"version": "abc"},
			expected: map[string]interface{}{"version": map[string]interface{}{"matches": `^\d+$`}},
			wantErr:  true,
		},
		{
			name:     "invalid regex pattern",
			actual:   map[string]interface{}{"val": "hello"},
			expected: map[string]interface{}{"val": map[string]interface{}{"matches": `[invalid`}},
			wantErr:  true,
		},
		{
			name:     "matches on non-string actual (converted via Sprintf)",
			actual:   map[string]interface{}{"code": 42},
			expected: map[string]interface{}{"code": map[string]interface{}{"matches": `^42$`}},
			wantErr:  false,
		},
		{
			name:     "matches with non-string expected pattern",
			actual:   map[string]interface{}{"val": "hello"},
			expected: map[string]interface{}{"val": map[string]interface{}{"matches": 123}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_NestedObject(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "nested field match",
			actual: map[string]interface{}{
				"result": map[string]interface{}{
					"status": "ok",
					"count":  3,
				},
			},
			expected: map[string]interface{}{
				"result": map[string]interface{}{
					"status": "ok",
				},
			},
			wantErr: false,
		},
		{
			name: "nested field mismatch",
			actual: map[string]interface{}{
				"result": map[string]interface{}{
					"status": "error",
				},
			},
			expected: map[string]interface{}{
				"result": map[string]interface{}{
					"status": "ok",
				},
			},
			wantErr: true,
		},
		{
			name:     "nested expected but actual is not a map",
			actual:   map[string]interface{}{"result": "string-value"},
			expected: map[string]interface{}{"result": map[string]interface{}{"status": "ok"}},
			wantErr:  true,
		},
		{
			name: "deeply nested match",
			actual: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": map[string]interface{}{
						"value": "deep",
					},
				},
			},
			expected: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": map[string]interface{}{
						"value": "deep",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "nested expected but actual is nil",
			actual:   map[string]interface{}{"result": nil},
			expected: map[string]interface{}{"result": map[string]interface{}{"status": "ok"}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_NumericComparison(t *testing.T) {
	tests := []struct {
		name     string
		actual   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "int actual matches float64 expected",
			actual:   map[string]interface{}{"exitCode": float64(0)},
			expected: map[string]interface{}{"exitCode": 0},
			wantErr:  false,
		},
		{
			name:     "float64 actual matches int expected",
			actual:   map[string]interface{}{"exitCode": 0},
			expected: map[string]interface{}{"exitCode": float64(0)},
			wantErr:  false,
		},
		{
			name:     "int actual matches int expected",
			actual:   map[string]interface{}{"exitCode": 1},
			expected: map[string]interface{}{"exitCode": 1},
			wantErr:  false,
		},
		{
			name:     "int mismatch",
			actual:   map[string]interface{}{"exitCode": 1},
			expected: map[string]interface{}{"exitCode": 0},
			wantErr:  true,
		},
		{
			name:     "float mismatch",
			actual:   map[string]interface{}{"val": float64(1.5)},
			expected: map[string]interface{}{"val": float64(2.5)},
			wantErr:  true,
		},
		{
			name:     "int64 matches float64",
			actual:   map[string]interface{}{"val": int64(42)},
			expected: map[string]interface{}{"val": float64(42)},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssertResult(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssertResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAssertResult_MultipleFailures(t *testing.T) {
	actual := map[string]interface{}{
		"exitCode": float64(1),
		"stdout":   "error occurred",
	}
	expected := map[string]interface{}{
		"exitCode": 0,
		"stdout":   "success",
	}

	err := AssertResult(actual, expected)
	if err == nil {
		t.Fatal("expected error for multiple failures, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "assertion failures") {
		t.Errorf("expected error to contain 'assertion failures', got: %s", errStr)
	}
	if !strings.Contains(errStr, "exitCode") {
		t.Errorf("expected error to mention 'exitCode', got: %s", errStr)
	}
	if !strings.Contains(errStr, "stdout") {
		t.Errorf("expected error to mention 'stdout', got: %s", errStr)
	}
}
