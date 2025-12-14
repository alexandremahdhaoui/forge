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

package mcptypes

import "fmt"

// ValidateString validates that a field is a string or absent.
// Returns the string value and nil error if field is absent or valid.
// Returns empty string and ValidationError if field has wrong type.
func ValidateString(spec map[string]interface{}, field string) (string, *ValidationError) {
	val, ok := spec[field]
	if !ok {
		return "", nil // Field is absent, which is OK for optional fields
	}

	str, ok := val.(string)
	if !ok {
		return "", &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: fmt.Sprintf("expected string, got %T", val),
		}
	}

	return str, nil
}

// ValidateStringRequired validates that a field is a string and present.
// Returns the string value and nil error if field is present and valid.
// Returns empty string and ValidationError if field is absent or has wrong type.
func ValidateStringRequired(spec map[string]interface{}, field string) (string, *ValidationError) {
	val, ok := spec[field]
	if !ok {
		return "", &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: "required field is missing",
		}
	}

	str, ok := val.(string)
	if !ok {
		return "", &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: fmt.Sprintf("expected string, got %T", val),
		}
	}

	if str == "" {
		return "", &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: "required field cannot be empty",
		}
	}

	return str, nil
}

// ValidateStringSlice validates that a field is a []string or absent.
// Returns the slice and nil error if field is absent or valid.
// Returns nil and ValidationError if field has wrong type or contains non-string elements.
func ValidateStringSlice(spec map[string]interface{}, field string) ([]string, *ValidationError) {
	val, ok := spec[field]
	if !ok {
		return nil, nil // Field is absent, which is OK for optional fields
	}

	// JSON unmarshaling produces []interface{} for arrays
	slice, ok := val.([]interface{})
	if !ok {
		// Also accept []string directly (e.g., from Go code)
		if strSlice, ok := val.([]string); ok {
			return strSlice, nil
		}
		return nil, &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: fmt.Sprintf("expected []string, got %T", val),
		}
	}

	result := make([]string, 0, len(slice))
	for i, elem := range slice {
		str, ok := elem.(string)
		if !ok {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("spec.%s[%d]", field, i),
				Message: fmt.Sprintf("expected string, got %T", elem),
			}
		}
		result = append(result, str)
	}

	return result, nil
}

// ValidateStringMap validates that a field is a map[string]string or absent.
// Returns the map and nil error if field is absent or valid.
// Returns nil and ValidationError if field has wrong type or contains non-string values.
func ValidateStringMap(spec map[string]interface{}, field string) (map[string]string, *ValidationError) {
	val, ok := spec[field]
	if !ok {
		return nil, nil // Field is absent, which is OK for optional fields
	}

	// JSON unmarshaling produces map[string]interface{} for objects
	m, ok := val.(map[string]interface{})
	if !ok {
		// Also accept map[string]string directly (e.g., from Go code)
		if strMap, ok := val.(map[string]string); ok {
			return strMap, nil
		}
		return nil, &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: fmt.Sprintf("expected map[string]string, got %T", val),
		}
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		str, ok := v.(string)
		if !ok {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("spec.%s.%s", field, k),
				Message: fmt.Sprintf("expected string value, got %T", v),
			}
		}
		result[k] = str
	}

	return result, nil
}

// ValidateBool validates that a field is a bool or absent.
// Returns the bool value and nil error if field is absent or valid.
// Returns false and ValidationError if field has wrong type.
func ValidateBool(spec map[string]interface{}, field string) (bool, *ValidationError) {
	val, ok := spec[field]
	if !ok {
		return false, nil // Field is absent, which is OK for optional fields
	}

	b, ok := val.(bool)
	if !ok {
		return false, &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: fmt.Sprintf("expected bool, got %T", val),
		}
	}

	return b, nil
}

// ValidateInt validates that a field is an int or absent.
// JSON numbers are unmarshaled as float64, so this function accepts both int and float64.
// Returns the int value and nil error if field is absent or valid.
// Returns 0 and ValidationError if field has wrong type.
func ValidateInt(spec map[string]interface{}, field string) (int, *ValidationError) {
	val, ok := spec[field]
	if !ok {
		return 0, nil // Field is absent, which is OK for optional fields
	}

	// Accept int directly
	if i, ok := val.(int); ok {
		return i, nil
	}

	// JSON unmarshaling produces float64 for numbers
	f, ok := val.(float64)
	if !ok {
		return 0, &ValidationError{
			Field:   fmt.Sprintf("spec.%s", field),
			Message: fmt.Sprintf("expected int, got %T", val),
		}
	}

	return int(f), nil
}
