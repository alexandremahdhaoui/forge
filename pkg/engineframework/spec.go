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

package engineframework

import "fmt"

// ExtractString safely extracts a string value from a spec map.
// Returns the string value and true if the key exists and is a string.
// Returns empty string and false if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"name": "my-app", "count": 42}
//	name, ok := ExtractString(spec, "name")  // "my-app", true
//	missing, ok := ExtractString(spec, "missing")  // "", false
//	wrong, ok := ExtractString(spec, "count")  // "", false (wrong type)
func ExtractString(spec map[string]any, key string) (string, bool) {
	if spec == nil {
		return "", false
	}

	value, exists := spec[key]
	if !exists {
		return "", false
	}

	str, ok := value.(string)
	if !ok {
		return "", false
	}

	return str, true
}

// ExtractStringWithDefault safely extracts a string value from a spec map with a default value.
// Returns the string value if the key exists and is a string.
// Returns the default value if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"name": "my-app"}
//	name := ExtractStringWithDefault(spec, "name", "default")  // "my-app"
//	missing := ExtractStringWithDefault(spec, "missing", "default")  // "default"
func ExtractStringWithDefault(spec map[string]any, key, defaultValue string) string {
	value, ok := ExtractString(spec, key)
	if !ok {
		return defaultValue
	}
	return value
}

// ExtractStringSlice safely extracts a []string value from a spec map.
// Returns the slice and true if the key exists and is a []string or []any containing only strings.
// Returns nil and false if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"tags": []string{"a", "b"}, "numbers": []int{1, 2}}
//	tags, ok := ExtractStringSlice(spec, "tags")  // ["a", "b"], true
//	missing, ok := ExtractStringSlice(spec, "missing")  // nil, false
//	wrong, ok := ExtractStringSlice(spec, "numbers")  // nil, false
func ExtractStringSlice(spec map[string]any, key string) ([]string, bool) {
	if spec == nil {
		return nil, false
	}

	value, exists := spec[key]
	if !exists {
		return nil, false
	}

	// Try []string first (common case)
	if slice, ok := value.([]string); ok {
		return slice, true
	}

	// Try []any with string elements (JSON unmarshal case)
	anySlice, ok := value.([]any)
	if !ok {
		return nil, false
	}

	result := make([]string, len(anySlice))
	for i, item := range anySlice {
		str, ok := item.(string)
		if !ok {
			return nil, false
		}
		result[i] = str
	}

	return result, true
}

// ExtractStringSliceWithDefault safely extracts a []string value from a spec map with a default value.
// Returns the slice if the key exists and is a valid []string.
// Returns the default value if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"tags": []string{"a", "b"}}
//	tags := ExtractStringSliceWithDefault(spec, "tags", []string{"default"})  // ["a", "b"]
//	missing := ExtractStringSliceWithDefault(spec, "missing", []string{"default"})  // ["default"]
func ExtractStringSliceWithDefault(spec map[string]any, key string, defaultValue []string) []string {
	value, ok := ExtractStringSlice(spec, key)
	if !ok {
		return defaultValue
	}
	return value
}

// ExtractStringMap safely extracts a map[string]string value from a spec map.
// Returns the map and true if the key exists and is a map[string]string or map[string]any with string values.
// Returns nil and false if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"labels": map[string]string{"app": "foo"}}
//	labels, ok := ExtractStringMap(spec, "labels")  // {"app": "foo"}, true
//	missing, ok := ExtractStringMap(spec, "missing")  // nil, false
func ExtractStringMap(spec map[string]any, key string) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}

	value, exists := spec[key]
	if !exists {
		return nil, false
	}

	// Try map[string]string first (common case)
	if m, ok := value.(map[string]string); ok {
		return m, true
	}

	// Try map[string]any with string values (JSON unmarshal case)
	anyMap, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}

	result := make(map[string]string, len(anyMap))
	for k, v := range anyMap {
		str, ok := v.(string)
		if !ok {
			return nil, false
		}
		result[k] = str
	}

	return result, true
}

// ExtractStringMapWithDefault safely extracts a map[string]string value from a spec map with a default value.
// Returns the map if the key exists and is a valid map[string]string.
// Returns the default value if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"labels": map[string]string{"app": "foo"}}
//	labels := ExtractStringMapWithDefault(spec, "labels", map[string]string{"default": "value"})  // {"app": "foo"}
//	missing := ExtractStringMapWithDefault(spec, "missing", map[string]string{"default": "value"})  // {"default": "value"}
func ExtractStringMapWithDefault(spec map[string]any, key string, defaultValue map[string]string) map[string]string {
	value, ok := ExtractStringMap(spec, key)
	if !ok {
		return defaultValue
	}
	return value
}

// ExtractBool safely extracts a bool value from a spec map.
// Returns the bool value and true if the key exists and is a bool.
// Returns false and false if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"enabled": true, "name": "foo"}
//	enabled, ok := ExtractBool(spec, "enabled")  // true, true
//	missing, ok := ExtractBool(spec, "missing")  // false, false
//	wrong, ok := ExtractBool(spec, "name")  // false, false (wrong type)
func ExtractBool(spec map[string]any, key string) (bool, bool) {
	if spec == nil {
		return false, false
	}

	value, exists := spec[key]
	if !exists {
		return false, false
	}

	b, ok := value.(bool)
	if !ok {
		return false, false
	}

	return b, true
}

// ExtractBoolWithDefault safely extracts a bool value from a spec map with a default value.
// Returns the bool value if the key exists and is a bool.
// Returns the default value if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"enabled": true}
//	enabled := ExtractBoolWithDefault(spec, "enabled", false)  // true
//	missing := ExtractBoolWithDefault(spec, "missing", false)  // false
func ExtractBoolWithDefault(spec map[string]any, key string, defaultValue bool) bool {
	value, ok := ExtractBool(spec, key)
	if !ok {
		return defaultValue
	}
	return value
}

// ExtractInt safely extracts an int value from a spec map.
// Returns the int value and true if the key exists and is an int, int64, or float64 that represents an integer.
// Returns 0 and false if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"count": 42, "rate": 3.14, "name": "foo"}
//	count, ok := ExtractInt(spec, "count")  // 42, true
//	missing, ok := ExtractInt(spec, "missing")  // 0, false
//	wrong, ok := ExtractInt(spec, "name")  // 0, false (wrong type)
func ExtractInt(spec map[string]any, key string) (int, bool) {
	if spec == nil {
		return 0, false
	}

	value, exists := spec[key]
	if !exists {
		return 0, false
	}

	// Try int first
	if i, ok := value.(int); ok {
		return i, true
	}

	// Try int64
	if i64, ok := value.(int64); ok {
		return int(i64), true
	}

	// Try float64 (JSON numbers are always float64)
	if f, ok := value.(float64); ok {
		// Check if it's actually an integer value
		if f == float64(int(f)) {
			return int(f), true
		}
	}

	return 0, false
}

// ExtractIntWithDefault safely extracts an int value from a spec map with a default value.
// Returns the int value if the key exists and is a valid integer.
// Returns the default value if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"count": 42}
//	count := ExtractIntWithDefault(spec, "count", 10)  // 42
//	missing := ExtractIntWithDefault(spec, "missing", 10)  // 10
func ExtractIntWithDefault(spec map[string]any, key string, defaultValue int) int {
	value, ok := ExtractInt(spec, key)
	if !ok {
		return defaultValue
	}
	return value
}

// ExtractMap safely extracts a map[string]any value from a spec map.
// Returns the map and true if the key exists and is a map[string]any.
// Returns nil and false if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"config": map[string]any{"timeout": 30}}
//	config, ok := ExtractMap(spec, "config")  // {"timeout": 30}, true
//	missing, ok := ExtractMap(spec, "missing")  // nil, false
func ExtractMap(spec map[string]any, key string) (map[string]any, bool) {
	if spec == nil {
		return nil, false
	}

	value, exists := spec[key]
	if !exists {
		return nil, false
	}

	m, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}

	return m, true
}

// ExtractMapWithDefault safely extracts a map[string]any value from a spec map with a default value.
// Returns the map if the key exists and is a valid map[string]any.
// Returns the default value if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"config": map[string]any{"timeout": 30}}
//	config := ExtractMapWithDefault(spec, "config", map[string]any{"default": true})  // {"timeout": 30}
//	missing := ExtractMapWithDefault(spec, "missing", map[string]any{"default": true})  // {"default": true}
func ExtractMapWithDefault(spec map[string]any, key string, defaultValue map[string]any) map[string]any {
	value, ok := ExtractMap(spec, key)
	if !ok {
		return defaultValue
	}
	return value
}

// RequireString extracts a required string value from a spec map.
// Returns the string value and nil error if the key exists and is a string.
// Returns an error if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"name": "my-app"}
//	name, err := RequireString(spec, "name")  // "my-app", nil
//	missing, err := RequireString(spec, "missing")  // "", error
func RequireString(spec map[string]any, key string) (string, error) {
	value, ok := ExtractString(spec, key)
	if !ok {
		return "", fmt.Errorf("required field %q is missing or has wrong type (expected string)", key)
	}
	return value, nil
}

// RequireStringSlice extracts a required []string value from a spec map.
// Returns the slice and nil error if the key exists and is a valid []string.
// Returns an error if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"tags": []string{"a", "b"}}
//	tags, err := RequireStringSlice(spec, "tags")  // ["a", "b"], nil
//	missing, err := RequireStringSlice(spec, "missing")  // nil, error
func RequireStringSlice(spec map[string]any, key string) ([]string, error) {
	value, ok := ExtractStringSlice(spec, key)
	if !ok {
		return nil, fmt.Errorf("required field %q is missing or has wrong type (expected []string)", key)
	}
	return value, nil
}

// RequireStringMap extracts a required map[string]string value from a spec map.
// Returns the map and nil error if the key exists and is a valid map[string]string.
// Returns an error if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"labels": map[string]string{"app": "foo"}}
//	labels, err := RequireStringMap(spec, "labels")  // {"app": "foo"}, nil
//	missing, err := RequireStringMap(spec, "missing")  // nil, error
func RequireStringMap(spec map[string]any, key string) (map[string]string, error) {
	value, ok := ExtractStringMap(spec, key)
	if !ok {
		return nil, fmt.Errorf("required field %q is missing or has wrong type (expected map[string]string)", key)
	}
	return value, nil
}

// RequireBool extracts a required bool value from a spec map.
// Returns the bool value and nil error if the key exists and is a bool.
// Returns an error if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"enabled": true}
//	enabled, err := RequireBool(spec, "enabled")  // true, nil
//	missing, err := RequireBool(spec, "missing")  // false, error
func RequireBool(spec map[string]any, key string) (bool, error) {
	value, ok := ExtractBool(spec, key)
	if !ok {
		return false, fmt.Errorf("required field %q is missing or has wrong type (expected bool)", key)
	}
	return value, nil
}

// RequireInt extracts a required int value from a spec map.
// Returns the int value and nil error if the key exists and is a valid integer.
// Returns an error if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"count": 42}
//	count, err := RequireInt(spec, "count")  // 42, nil
//	missing, err := RequireInt(spec, "missing")  // 0, error
func RequireInt(spec map[string]any, key string) (int, error) {
	value, ok := ExtractInt(spec, key)
	if !ok {
		return 0, fmt.Errorf("required field %q is missing or has wrong type (expected int)", key)
	}
	return value, nil
}

// RequireMap extracts a required map[string]any value from a spec map.
// Returns the map and nil error if the key exists and is a valid map[string]any.
// Returns an error if the key doesn't exist or has the wrong type.
//
// Example:
//
//	spec := map[string]any{"config": map[string]any{"timeout": 30}}
//	config, err := RequireMap(spec, "config")  // {"timeout": 30}, nil
//	missing, err := RequireMap(spec, "missing")  // nil, error
func RequireMap(spec map[string]any, key string) (map[string]any, error) {
	value, ok := ExtractMap(spec, key)
	if !ok {
		return nil, fmt.Errorf("required field %q is missing or has wrong type (expected map[string]any)", key)
	}
	return value, nil
}
