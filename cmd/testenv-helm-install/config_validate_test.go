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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate_ValidSpec(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name":       "my-chart",
				"chart":      "nginx",
				"repo":       "https://charts.bitnami.com/bitnami",
				"version":    "1.0.0",
				"namespace":  "default",
				"valuesFile": "values.yaml",
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_MinimalValidSpec(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name": "my-chart",
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_EmptySpec(t *testing.T) {
	spec := map[string]interface{}{}

	output := validateHelmInstallSpec(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_NilSpec(t *testing.T) {
	output := validateHelmInstallSpec(nil)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_EmptyChartsArray(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{},
	}

	output := validateHelmInstallSpec(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_MissingChartName(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"chart": "nginx",
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].name", output.Errors[0].Field)
	assert.Equal(t, "required field is missing", output.Errors[0].Message)
}

func TestConfigValidate_InvalidChartsType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": "not-an-array",
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected array")
}

func TestConfigValidate_InvalidChartType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			"not-an-object",
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0]", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected object")
}

func TestConfigValidate_InvalidNameType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name": 123,
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].name", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidChartFieldType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name":  "my-chart",
				"chart": 123,
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].chart", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidRepoType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name": "my-chart",
				"repo": 123,
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].repo", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidVersionType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name":    "my-chart",
				"version": 1.0,
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].version", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidNamespaceType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name":      "my-chart",
				"namespace": []string{"ns1", "ns2"},
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].namespace", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_InvalidValuesFileType(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name":       "my-chart",
				"valuesFile": true,
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 1)
	assert.Equal(t, "spec.charts[0].valuesFile", output.Errors[0].Field)
	assert.Contains(t, output.Errors[0].Message, "expected string")
}

func TestConfigValidate_MultipleCharts(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				"name":  "chart-1",
				"chart": "nginx",
			},
			map[string]interface{}{
				"name":      "chart-2",
				"namespace": "kube-system",
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestConfigValidate_MultipleChartsWithErrors(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				// Missing name
				"chart": "nginx",
			},
			map[string]interface{}{
				"name": "chart-2",
			},
			map[string]interface{}{
				// Missing name
				"repo": "https://example.com",
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	require.Len(t, output.Errors, 2)
	assert.Equal(t, "spec.charts[0].name", output.Errors[0].Field)
	assert.Equal(t, "spec.charts[2].name", output.Errors[1].Field)
}

func TestConfigValidate_MultipleErrorsInSingleChart(t *testing.T) {
	spec := map[string]interface{}{
		"charts": []interface{}{
			map[string]interface{}{
				// Missing name
				"chart":     123,  // wrong type
				"namespace": true, // wrong type
			},
		},
	}

	output := validateHelmInstallSpec(spec)

	assert.False(t, output.Valid)
	// 3 errors: missing name, invalid chart type, invalid namespace type
	assert.Len(t, output.Errors, 3)
}
