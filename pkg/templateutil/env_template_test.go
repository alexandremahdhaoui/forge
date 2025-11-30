//go:build unit

package templateutil

import (
	"strings"
	"testing"
)

func TestExpandTemplates_BasicExpansion(t *testing.T) {
	spec := map[string]interface{}{
		"path": "{{.Env.KUBECONFIG}}",
	}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	if result["path"] != "/tmp/kubeconfig" {
		t.Errorf("Expected path='/tmp/kubeconfig', got '%v'", result["path"])
	}
}

func TestExpandTemplates_MultipleVariables(t *testing.T) {
	spec := map[string]interface{}{
		"kubeconfig": "{{.Env.KUBECONFIG}}",
		"registry":   "{{.Env.TESTENV_LCR_FQDN}}",
	}

	env := map[string]string{
		"KUBECONFIG":       "/tmp/kubeconfig",
		"TESTENV_LCR_FQDN": "localhost:5000",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	if result["kubeconfig"] != "/tmp/kubeconfig" {
		t.Errorf("Expected kubeconfig='/tmp/kubeconfig', got '%v'", result["kubeconfig"])
	}

	if result["registry"] != "localhost:5000" {
		t.Errorf("Expected registry='localhost:5000', got '%v'", result["registry"])
	}
}

func TestExpandTemplates_NestedMap(t *testing.T) {
	spec := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"path": "{{.Env.KUBECONFIG}}",
			},
		},
	}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	level1, ok := result["level1"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected level1 to be map[string]interface{}, got %T", result["level1"])
	}

	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected level2 to be map[string]interface{}, got %T", level1["level2"])
	}

	if level2["path"] != "/tmp/kubeconfig" {
		t.Errorf("Expected path='/tmp/kubeconfig', got '%v'", level2["path"])
	}
}

func TestExpandTemplates_Array(t *testing.T) {
	spec := map[string]interface{}{
		"paths": []interface{}{
			"{{.Env.KUBECONFIG}}",
			"{{.Env.TESTENV_LCR_FQDN}}",
			"/static/path",
		},
	}

	env := map[string]string{
		"KUBECONFIG":       "/tmp/kubeconfig",
		"TESTENV_LCR_FQDN": "localhost:5000",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	paths, ok := result["paths"].([]interface{})
	if !ok {
		t.Fatalf("Expected paths to be []interface{}, got %T", result["paths"])
	}

	if len(paths) != 3 {
		t.Fatalf("Expected 3 paths, got %d", len(paths))
	}

	if paths[0] != "/tmp/kubeconfig" {
		t.Errorf("Expected paths[0]='/tmp/kubeconfig', got '%v'", paths[0])
	}

	if paths[1] != "localhost:5000" {
		t.Errorf("Expected paths[1]='localhost:5000', got '%v'", paths[1])
	}

	if paths[2] != "/static/path" {
		t.Errorf("Expected paths[2]='/static/path', got '%v'", paths[2])
	}
}

func TestExpandTemplates_NestedArrayInMap(t *testing.T) {
	spec := map[string]interface{}{
		"config": map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{
					"path": "{{.Env.KUBECONFIG}}",
				},
			},
		},
	}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	config, ok := result["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected config to be map[string]interface{}, got %T", result["config"])
	}

	files, ok := config["files"].([]interface{})
	if !ok {
		t.Fatalf("Expected files to be []interface{}, got %T", config["files"])
	}

	fileMap, ok := files[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected files[0] to be map[string]interface{}, got %T", files[0])
	}

	if fileMap["path"] != "/tmp/kubeconfig" {
		t.Errorf("Expected path='/tmp/kubeconfig', got '%v'", fileMap["path"])
	}
}

func TestExpandTemplates_UndefinedVariable(t *testing.T) {
	spec := map[string]interface{}{
		"path": "{{.Env.UNDEFINED_VAR}}",
	}

	env := map[string]string{
		"KUBECONFIG":       "/tmp/kubeconfig",
		"TESTENV_LCR_FQDN": "localhost:5000",
	}

	_, err := ExpandTemplates(spec, env)
	if err == nil {
		t.Fatal("Expected error for undefined variable, got nil")
	}

	errMsg := err.Error()

	// Check error message contains required information
	if !strings.Contains(errMsg, "UNDEFINED_VAR") {
		t.Errorf("Error message should contain variable name 'UNDEFINED_VAR', got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "{{.Env.UNDEFINED_VAR}}") {
		t.Errorf("Error message should contain template string '{{.Env.UNDEFINED_VAR}}', got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "Available") {
		t.Errorf("Error message should list available variables, got: %s", errMsg)
	}

	// Check that available variables are mentioned
	if !strings.Contains(errMsg, "KUBECONFIG") || !strings.Contains(errMsg, "TESTENV_LCR_FQDN") {
		t.Errorf("Error message should list available variables (KUBECONFIG, TESTENV_LCR_FQDN), got: %s", errMsg)
	}
}

func TestExpandTemplates_NoTemplates(t *testing.T) {
	spec := map[string]interface{}{
		"path":   "/static/path",
		"number": 42,
		"bool":   true,
		"nested": map[string]interface{}{
			"value": "no template here",
		},
	}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	// Verify spec passes through unchanged
	if result["path"] != "/static/path" {
		t.Errorf("Expected path='/static/path', got '%v'", result["path"])
	}

	if result["number"] != 42 {
		t.Errorf("Expected number=42, got '%v'", result["number"])
	}

	if result["bool"] != true {
		t.Errorf("Expected bool=true, got '%v'", result["bool"])
	}

	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected nested to be map[string]interface{}, got %T", result["nested"])
	}

	if nested["value"] != "no template here" {
		t.Errorf("Expected nested.value='no template here', got '%v'", nested["value"])
	}
}

func TestExpandTemplates_EmptyEnv(t *testing.T) {
	spec := map[string]interface{}{
		"path": "/static/path",
	}

	env := map[string]string{}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	if result["path"] != "/static/path" {
		t.Errorf("Expected path='/static/path', got '%v'", result["path"])
	}
}

func TestExpandTemplates_EmptySpec(t *testing.T) {
	spec := map[string]interface{}{}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestExpandTemplates_PartialTemplate(t *testing.T) {
	// Test template embedded in larger string
	spec := map[string]interface{}{
		"command": "kubectl --kubeconfig {{.Env.KUBECONFIG}} get pods",
	}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	expected := "kubectl --kubeconfig /tmp/kubeconfig get pods"
	if result["command"] != expected {
		t.Errorf("Expected command='%s', got '%v'", expected, result["command"])
	}
}

func TestExpandTemplates_MultipleTemplatesInString(t *testing.T) {
	// Test multiple templates in a single string
	spec := map[string]interface{}{
		"url": "https://{{.Env.TESTENV_LCR_HOST}}:{{.Env.TESTENV_LCR_PORT}}/repo",
	}

	env := map[string]string{
		"TESTENV_LCR_HOST": "localhost",
		"TESTENV_LCR_PORT": "5000",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	expected := "https://localhost:5000/repo"
	if result["url"] != expected {
		t.Errorf("Expected url='%s', got '%v'", expected, result["url"])
	}
}

func TestExpandTemplates_NonStringValues(t *testing.T) {
	// Test that non-string values are preserved
	spec := map[string]interface{}{
		"number":  123,
		"float":   45.67,
		"bool":    false,
		"null":    nil,
		"string":  "{{.Env.KUBECONFIG}}",
		"numbers": []interface{}{1, 2, 3},
	}

	env := map[string]string{
		"KUBECONFIG": "/tmp/kubeconfig",
	}

	result, err := ExpandTemplates(spec, env)
	if err != nil {
		t.Fatalf("ExpandTemplates failed: %v", err)
	}

	// Verify non-string values pass through
	if result["number"] != 123 {
		t.Errorf("Expected number=123, got '%v'", result["number"])
	}

	if result["float"] != 45.67 {
		t.Errorf("Expected float=45.67, got '%v'", result["float"])
	}

	if result["bool"] != false {
		t.Errorf("Expected bool=false, got '%v'", result["bool"])
	}

	if result["null"] != nil {
		t.Errorf("Expected null=nil, got '%v'", result["null"])
	}

	// Verify string template was expanded
	if result["string"] != "/tmp/kubeconfig" {
		t.Errorf("Expected string='/tmp/kubeconfig', got '%v'", result["string"])
	}

	// Verify numeric array preserved
	numbers, ok := result["numbers"].([]interface{})
	if !ok {
		t.Fatalf("Expected numbers to be []interface{}, got %T", result["numbers"])
	}
	if len(numbers) != 3 || numbers[0] != 1 || numbers[1] != 2 || numbers[2] != 3 {
		t.Errorf("Expected numbers=[1,2,3], got %v", numbers)
	}
}

func TestExpandTemplates_UndefinedVariableInNestedStructure(t *testing.T) {
	// Test error handling in nested structures
	spec := map[string]interface{}{
		"config": map[string]interface{}{
			"paths": []interface{}{
				"{{.Env.VALID_VAR}}",
				"{{.Env.INVALID_VAR}}",
			},
		},
	}

	env := map[string]string{
		"VALID_VAR": "/tmp/valid",
	}

	_, err := ExpandTemplates(spec, env)
	if err == nil {
		t.Fatal("Expected error for undefined variable in nested structure, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "INVALID_VAR") {
		t.Errorf("Error message should contain 'INVALID_VAR', got: %s", errMsg)
	}
}
