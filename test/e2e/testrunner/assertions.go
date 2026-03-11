//go:build e2e || unit

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
	"fmt"
	"regexp"
	"strings"
)

// AssertResult checks that actual matches expected using the assertion types:
//   - Exact match: field: "value"
//   - Nested object: field: {subfield: "value"}
//   - Array length: field: {length: N}
//   - Array contains: field: {contains: ["val1"]}
//   - Not empty: field: {notEmpty: true}
//   - Regex: field: {matches: "^pattern$"}
//   - Exit code: exitCode: 0 (top-level exact match)
func AssertResult(actual map[string]interface{}, expected map[string]interface{}) error {
	var errs []string
	for key, exp := range expected {
		if err := assertField(key, actual[key], exp); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("assertion failures:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// assertField asserts a single field value against an expected value.
func assertField(path string, actual interface{}, expected interface{}) error {
	switch exp := expected.(type) {
	case map[string]interface{}:
		return assertMapExpectation(path, actual, exp)
	default:
		return assertExactMatch(path, actual, expected)
	}
}

// assertMapExpectation handles map-typed expectations. If the map contains
// special assertion keys (length, contains, notEmpty, matches), it runs
// those assertions. Otherwise, it recurses into nested object comparison.
func assertMapExpectation(path string, actual interface{}, exp map[string]interface{}) error {
	// Check for special assertion keys.
	if _, ok := exp["length"]; ok {
		return assertLength(path, actual, exp["length"])
	}
	if _, ok := exp["contains"]; ok {
		return assertContains(path, actual, exp["contains"])
	}
	if _, ok := exp["notEmpty"]; ok {
		return assertNotEmpty(path, actual)
	}
	if _, ok := exp["matches"]; ok {
		return assertMatches(path, actual, exp["matches"])
	}

	// Nested object: recurse into subfields.
	actualMap, ok := actual.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s: expected nested object, got %T", path, actual)
	}

	var errs []string
	for key, val := range exp {
		subPath := path + "." + key
		if err := assertField(subPath, actualMap[key], val); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// assertExactMatch compares actual against expected using string representation.
func assertExactMatch(path string, actual interface{}, expected interface{}) error {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	// Handle numeric comparison: YAML may parse integers, but JSON
	// unmarshals numbers as float64.
	if actualStr != expectedStr {
		// Try comparing as numbers if both look numeric.
		if toFloat(actual) != nil && toFloat(expected) != nil {
			if *toFloat(actual) == *toFloat(expected) {
				return nil
			}
		}
		return fmt.Errorf("%s: expected %v (%T), got %v (%T)",
			path, expected, expected, actual, actual)
	}
	return nil
}

// assertLength checks that a slice has the expected length.
func assertLength(path string, actual interface{}, expected interface{}) error {
	slice, ok := toSlice(actual)
	if !ok {
		return fmt.Errorf("%s: expected array for length check, got %T", path, actual)
	}

	expectedLen, ok := toInt(expected)
	if !ok {
		return fmt.Errorf("%s: 'length' must be a number, got %T", path, expected)
	}

	if len(slice) != expectedLen {
		return fmt.Errorf("%s: expected length %d, got %d", path, expectedLen, len(slice))
	}
	return nil
}

// assertContains checks that a string or slice contains all expected values.
// For strings: checks that the string contains each expected substring.
// For slices: checks that all expected items are present in the slice.
func assertContains(path string, actual interface{}, expected interface{}) error {
	expectedList, ok := expected.([]interface{})
	if !ok {
		return fmt.Errorf("%s: 'contains' must be a list, got %T", path, expected)
	}

	// If actual is a string, check substring containment.
	if actualStr, ok := actual.(string); ok {
		for _, want := range expectedList {
			wantStr := fmt.Sprintf("%v", want)
			if !strings.Contains(actualStr, wantStr) {
				return fmt.Errorf("%s: string does not contain %q\n  actual: %q", path, wantStr, actualStr)
			}
		}
		return nil
	}

	// If actual is a slice, check element containment.
	slice, ok := toSlice(actual)
	if !ok {
		return fmt.Errorf("%s: expected string or array for contains check, got %T", path, actual)
	}

	for _, want := range expectedList {
		found := false
		wantStr := fmt.Sprintf("%v", want)
		for _, item := range slice {
			if fmt.Sprintf("%v", item) == wantStr {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s: array does not contain %v", path, want)
		}
	}
	return nil
}

// assertNotEmpty checks that a value is non-empty.
func assertNotEmpty(path string, actual interface{}) error {
	if actual == nil {
		return fmt.Errorf("%s: expected non-empty value, got nil", path)
	}

	switch v := actual.(type) {
	case string:
		if v == "" {
			return fmt.Errorf("%s: expected non-empty string, got empty", path)
		}
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("%s: expected non-empty array, got empty", path)
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return fmt.Errorf("%s: expected non-empty map, got empty", path)
		}
	}
	return nil
}

// assertMatches checks that a string value matches a regex pattern.
func assertMatches(path string, actual interface{}, expected interface{}) error {
	actualStr, ok := actual.(string)
	if !ok {
		actualStr = fmt.Sprintf("%v", actual)
	}

	pattern, ok := expected.(string)
	if !ok {
		return fmt.Errorf("%s: 'matches' must be a string, got %T", path, expected)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("%s: invalid regex %q: %w", path, pattern, err)
	}

	if !re.MatchString(actualStr) {
		return fmt.Errorf("%s: value %q does not match pattern %q", path, actualStr, pattern)
	}
	return nil
}

// toSlice attempts to convert a value to []interface{}.
func toSlice(v interface{}) ([]interface{}, bool) {
	slice, ok := v.([]interface{})
	return slice, ok
}

// toInt attempts to convert a numeric value to int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// toFloat attempts to convert a value to *float64. Returns nil if not numeric.
func toFloat(v interface{}) *float64 {
	var f float64
	switch n := v.(type) {
	case int:
		f = float64(n)
	case int64:
		f = float64(n)
	case float64:
		f = n
	default:
		return nil
	}
	return &f
}
