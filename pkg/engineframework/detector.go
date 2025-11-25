package engineframework

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FindDetector locates a dependency detector binary by name.
// It searches in the following order:
//  1. PATH environment variable
//  2. ./build/bin directory (common for forge self-build)
//
// Returns the absolute path to the binary or an error if not found.
func FindDetector(name string) (string, error) {
	// Try to find in PATH
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}

	// Try in build directory (common for forge self-build)
	buildPath := filepath.Join(".", "build", "bin", name)
	if _, err := os.Stat(buildPath); err == nil {
		absPath, err := filepath.Abs(buildPath)
		if err != nil {
			return "", fmt.Errorf("found detector at %s but failed to resolve absolute path: %w", buildPath, err)
		}
		return absPath, nil
	}

	return "", fmt.Errorf("%s not found in PATH or ./build/bin", name)
}

// CallDetector calls a detector MCP server and returns dependencies.
// It spawns the detector as a subprocess, connects via MCP, calls the specified tool,
// and converts the response to []forge.ArtifactDependency.
//
// Parameters:
//   - ctx: context for the operation
//   - detectorPath: absolute path to the detector binary
//   - toolName: name of the MCP tool to call (e.g., "detectDependencies")
//   - input: input parameters for the tool (will be serialized to JSON)
//
// Returns:
//   - []forge.ArtifactDependency: list of detected dependencies
//   - error: if connection fails, tool call fails, or response parsing fails
func CallDetector(ctx context.Context, detectorPath, toolName string, input any) ([]forge.ArtifactDependency, error) {
	// Create command to spawn MCP server
	cmd := exec.Command(detectorPath, "--mcp")
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr // Forward logs

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "detector-client",
		Version: "v1.0.0",
	}, nil)

	// Create command transport
	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	// Connect to the MCP server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to detector: %w", err)
	}
	defer func() { _ = session.Close() }()

	// Call the tool
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: input,
	})
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Check if result indicates an error
	if result.IsError {
		errMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				errMsg = textContent.Text
			}
		}
		return nil, fmt.Errorf("detection failed: %s", errMsg)
	}

	// Parse structured content
	if result.StructuredContent == nil {
		return nil, fmt.Errorf("no structured content returned from detector")
	}

	// Convert structured content to DetectDependenciesOutput
	var output mcptypes.DetectDependenciesOutput
	jsonBytes, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal detector output: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal detector output: %w", err)
	}

	// Convert mcptypes.Dependency to forge.ArtifactDependency
	artifactDeps := make([]forge.ArtifactDependency, len(output.Dependencies))
	for i, dep := range output.Dependencies {
		artifactDeps[i] = forge.ArtifactDependency{
			Type:            dep.Type,
			FilePath:        dep.FilePath,
			ExternalPackage: dep.ExternalPackage,
			Timestamp:       dep.Timestamp,
			Semver:          dep.Semver,
		}
	}

	return artifactDeps, nil
}
