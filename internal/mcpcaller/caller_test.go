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

package mcpcaller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCaller(t *testing.T) {
	caller := NewCaller("v1.0.0")
	require.NotNil(t, caller)
	assert.Equal(t, "v1.0.0", caller.forgeVersion)
}

func TestCaller_GetMCPCaller(t *testing.T) {
	caller := NewCaller("v1.0.0")
	mcpCaller := caller.GetMCPCaller()

	// Verify the returned function is not nil and has the correct signature
	require.NotNil(t, mcpCaller)

	// The MCPCaller type should be: func(command string, args []string, toolName string, params interface{}) (interface{}, error)
	// We cannot easily test the actual MCP call without spawning a real process,
	// but we can verify the function is correctly returned
	var _ MCPCaller = mcpCaller
}

func TestCaller_GetEngineResolver(t *testing.T) {
	caller := NewCaller("v1.0.0")
	resolver := caller.GetEngineResolver()

	// Verify the returned function is not nil and has the correct signature
	require.NotNil(t, resolver)

	// The EngineResolver type should be: func(engineURI string) (command string, args []string, err error)
	var _ EngineResolver = resolver
}

func TestCaller_ResolveEngine_GoURI(t *testing.T) {
	caller := NewCaller("v1.0.0")

	// Test resolving a go:// URI
	command, args, err := caller.ResolveEngine("go://go-build")
	require.NoError(t, err)
	assert.Equal(t, "go", command)
	assert.Contains(t, args, "run")
	// The args should contain the package path with version
	assert.True(t, len(args) >= 2, "expected at least 2 args (run, package@version)")
}

func TestCaller_ResolveEngine_AliasURI_NotSupported(t *testing.T) {
	caller := NewCaller("v1.0.0")

	// Test that alias:// URIs are not supported
	_, _, err := caller.ResolveEngine("alias://my-alias")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alias:// URIs not supported")
}

func TestCaller_ResolveEngine_InvalidURI(t *testing.T) {
	caller := NewCaller("v1.0.0")

	// Test that invalid URIs return an error
	_, _, err := caller.ResolveEngine("invalid://something")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported engine protocol")
}

func TestCaller_ResolveEngine_EmptyGoPath(t *testing.T) {
	caller := NewCaller("v1.0.0")

	// Test that empty go:// path returns an error
	_, _, err := caller.ResolveEngine("go://")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty engine path")
}

func TestMCPCaller_Signature(t *testing.T) {
	// This test verifies that MCPCaller has the correct signature
	// matching internal/orchestrate/orchestrate.go:24
	var caller MCPCaller = func(command string, args []string, toolName string, params interface{}) (interface{}, error) {
		return nil, nil
	}
	require.NotNil(t, caller)
}

func TestEngineResolver_Signature(t *testing.T) {
	// This test verifies that EngineResolver has the correct signature
	// matching internal/orchestrate/orchestrate.go:29
	var resolver EngineResolver = func(engineURI string) (command string, args []string, err error) {
		return "", nil, nil
	}
	require.NotNil(t, resolver)
}

// MockMCPCaller is a helper for testing code that uses MCPCaller
func MockMCPCaller(returnValue interface{}, returnErr error) MCPCaller {
	return func(command string, args []string, toolName string, params interface{}) (interface{}, error) {
		return returnValue, returnErr
	}
}

// MockEngineResolver is a helper for testing code that uses EngineResolver
func MockEngineResolver(command string, args []string, err error) EngineResolver {
	return func(engineURI string) (string, []string, error) {
		return command, args, err
	}
}

func TestMockMCPCaller(t *testing.T) {
	// Test that MockMCPCaller works as expected
	expectedResult := map[string]any{"key": "value"}
	mockCaller := MockMCPCaller(expectedResult, nil)

	result, err := mockCaller("cmd", []string{"arg1"}, "tool", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestMockEngineResolver(t *testing.T) {
	// Test that MockEngineResolver works as expected
	mockResolver := MockEngineResolver("go", []string{"run", "pkg@v1.0.0"}, nil)

	command, args, err := mockResolver("go://some-engine")
	require.NoError(t, err)
	assert.Equal(t, "go", command)
	assert.Equal(t, []string{"run", "pkg@v1.0.0"}, args)
}
