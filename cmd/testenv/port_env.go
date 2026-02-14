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
	"fmt"
	"strings"
)

// NormalizePortAllocEnvKey converts an allocateOpenPort identifier to an
// environment variable key. It replaces hyphens with underscores, uppercases
// the result, and prepends the PORTALLOC_ prefix.
//
// Example: "shaper-e2e-api" -> "PORTALLOC_SHAPER_E2E_API"
func NormalizePortAllocEnvKey(id string) string {
	return "PORTALLOC_" + strings.ToUpper(strings.ReplaceAll(id, "-", "_"))
}

// toInt converts a template argument to int. Supports int and float64
// (JSON numbers decode as float64 in Go templates).
func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case float64:
		return int(n), nil
	case int64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}
