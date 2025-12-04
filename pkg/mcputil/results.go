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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrorResult creates a standardized MCP error result.
//
// Parameters:
//   - message: error message to display
//
// Example usage:
//
//	return mcputil.ErrorResult(fmt.Sprintf("Build failed: %v", err)), nil, nil
func ErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		IsError: true,
	}
}

// SuccessResult creates a standardized MCP success result.
//
// Parameters:
//   - message: success message to display
//
// Example usage:
//
//	return mcputil.SuccessResult("Build completed successfully"), nil, nil
func SuccessResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		IsError: false,
	}
}

// SuccessResultWithArtifact creates a success result that returns an artifact.
// This is the most common pattern for MCP tool responses.
//
// Parameters:
//   - message: success message to display
//   - artifact: the artifact to return (typically forge.Artifact or similar)
//
// Returns:
//   - result: the MCP CallToolResult
//   - artifact: the artifact (passed through for MCP handler return)
//
// Example usage:
//
//	result, artifact := mcputil.SuccessResultWithArtifact("Built successfully", myArtifact)
//	return result, artifact, nil
func SuccessResultWithArtifact(message string, artifact any) (*mcp.CallToolResult, any) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		IsError: false,
	}
	return result, artifact
}

// ErrorResultWithArtifact creates an error result that also returns an artifact.
// This is useful when you want to return partial data even when an operation fails
// (e.g., returning a TestReport even when tests fail).
//
// Parameters:
//   - message: error message to display
//   - artifact: the artifact to return (typically contains partial or error information)
//
// Returns:
//   - result: the MCP CallToolResult with IsError set to true
//   - artifact: the artifact (passed through for MCP handler return)
//
// Example usage:
//
//	result, artifact := mcputil.ErrorResultWithArtifact("Tests failed", testReport)
//	return result, artifact, nil
func ErrorResultWithArtifact(message string, artifact any) (*mcp.CallToolResult, any) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		IsError: true,
	}
	return result, artifact
}
