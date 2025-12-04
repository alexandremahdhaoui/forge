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

import (
	"context"
	"fmt"
	"log"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateInput represents the input for testenv subengine create operations.
//
// This is the standard input format for all testenv subengines (e.g., testenv-kind, testenv-lcr).
// The testenv orchestrator calls each subengine's "create" tool with this input.
//
// Fields:
//   - TestID: Unique identifier for this test environment instance (required)
//   - Stage: Test stage name from forge.yaml (required)
//   - TmpDir: Temporary directory allocated for this test environment (required)
//   - RootDir: Project root directory for path resolution (optional)
//   - Metadata: Metadata from previous subengines in the chain (optional)
//   - Spec: Optional spec for configuration override from forge.yaml
//   - Env: Accumulated environment variables from previous sub-engines (optional)
//   - EnvPropagation: Optional EnvPropagation configuration from spec (optional)
//
// Example:
//
//	input := CreateInput{
//	    TestID:   "test-abc123",
//	    Stage:    "integration",
//	    TmpDir:   "/tmp/forge-test-abc123",
//	    RootDir:  "/home/user/project",
//	    Metadata: map[string]string{"testenv-lcr.registryFQDN": "testenv-lcr.testenv-lcr.svc.cluster.local:31906"},
//	    Spec:     map[string]any{"kindVersion": "v1.27.0"},
//	    Env:      map[string]string{"TESTENV_LCR_FQDN": "testenv-lcr.testenv-lcr.svc.cluster.local:31906"},
//	}
type CreateInput struct {
	TestID         string                `json:"testID"`                   // Test environment ID (required)
	Stage          string                `json:"stage"`                    // Test stage name (required)
	TmpDir         string                `json:"tmpDir"`                   // Temporary directory for this test environment (required)
	RootDir        string                `json:"rootDir,omitempty"`        // Project root directory (optional, for path resolution)
	Metadata       map[string]string     `json:"metadata"`                 // Metadata from previous testenv-subengines (optional)
	Spec           map[string]any        `json:"spec,omitempty"`           // Optional spec for configuration override
	Env            map[string]string     `json:"env,omitempty"`            // Accumulated environment variables from previous sub-engines (optional)
	EnvPropagation *forge.EnvPropagation `json:"envPropagation,omitempty"` // Optional EnvPropagation configuration from spec
}

// DeleteInput represents the input for testenv subengine delete operations.
//
// This is the standard input format for all testenv subengines.
// The testenv orchestrator calls each subengine's "delete" tool with this input.
//
// Fields:
//   - TestID: Unique identifier for the test environment instance to delete (required)
//   - Metadata: Metadata from the test environment (optional, useful for cleanup)
//
// Example:
//
//	input := DeleteInput{
//	    TestID:   "test-abc123",
//	    Metadata: map[string]string{"testenv-kind.clusterName": "myapp-test-abc123"},
//	}
type DeleteInput struct {
	TestID   string            `json:"testID"`   // Test environment ID (required)
	Metadata map[string]string `json:"metadata"` // Metadata from test environment (optional)
}

// TestEnvArtifact represents the artifact returned by testenv subengine create operations.
//
// This is the standard artifact format for all testenv subengines.
// The artifact is passed to the test runner and returned to the caller.
//
// Fields:
//   - TestID: Test environment ID
//   - Files: Map of logical names to relative file paths (relative to TmpDir)
//   - Metadata: Key-value metadata for downstream consumers
//   - ManagedResources: List of resources to clean up (file paths, cluster names, etc.)
//   - Env: Environment variables exported by this sub-engine (optional)
//
// Example:
//
//	artifact := TestEnvArtifact{
//	    TestID: "test-abc123",
//	    Files: map[string]string{
//	        "testenv-kind.kubeconfig": "kubeconfig",
//	    },
//	    Metadata: map[string]string{
//	        "testenv-kind.clusterName":    "myapp-test-abc123",
//	        "testenv-kind.kubeconfigPath": "/tmp/forge-test-abc123/kubeconfig",
//	    },
//	    ManagedResources: []string{"/tmp/forge-test-abc123/kubeconfig"},
//	    Env: map[string]string{
//	        "KUBECONFIG": "/tmp/forge-test-abc123/kubeconfig",
//	    },
//	}
type TestEnvArtifact struct {
	TestID           string            `json:"testID"`           // Test environment ID
	Files            map[string]string `json:"files"`            // Map of logical names to relative file paths
	Metadata         map[string]string `json:"metadata"`         // Metadata for downstream consumers
	ManagedResources []string          `json:"managedResources"` // Resources to clean up
	Env              map[string]string `json:"env,omitempty"`    // Environment variables exported by this sub-engine
}

// CreateFunc is the signature for testenv subengine create operations.
//
// Implementations must:
//   - Validate input fields (testID, stage, tmpDir are required)
//   - Create the test environment resource (cluster, registry, etc.)
//   - Return TestEnvArtifact on success with files, metadata, and managedResources
//   - Return error on failure
//
// The framework handles:
//   - MCP tool registration
//   - Result formatting
//   - Error conversion to MCP responses
//   - Artifact serialization
//
// Example:
//
//	func myCreateFunc(ctx context.Context, input CreateInput) (*TestEnvArtifact, error) {
//	    // Create resource (e.g., kind cluster)
//	    clusterName := fmt.Sprintf("myapp-%s", input.TestID)
//	    if err := createCluster(clusterName); err != nil {
//	        return nil, fmt.Errorf("failed to create cluster: %w", err)
//	    }
//
//	    // Return artifact
//	    return &TestEnvArtifact{
//	        TestID: input.TestID,
//	        Files: map[string]string{
//	            "kubeconfig": "kubeconfig",
//	        },
//	        Metadata: map[string]string{
//	            "clusterName": clusterName,
//	        },
//	        ManagedResources: []string{"/path/to/kubeconfig"},
//	    }, nil
//	}
type CreateFunc func(ctx context.Context, input CreateInput) (*TestEnvArtifact, error)

// DeleteFunc is the signature for testenv subengine delete operations.
//
// Implementations must:
//   - Validate input fields (testID is required)
//   - Delete the test environment resource (cluster, registry, etc.)
//   - Return error on failure (or nil for best-effort cleanup)
//
// The framework handles:
//   - MCP tool registration
//   - Result formatting
//   - Error conversion to MCP responses
//
// IMPORTANT: Delete operations should be best-effort. If resources are already gone,
// don't return an error. Only return errors for actual failures that need attention.
//
// Example:
//
//	func myDeleteFunc(ctx context.Context, input DeleteInput) error {
//	    // Reconstruct cluster name from testID
//	    clusterName := input.Metadata["clusterName"]
//	    if clusterName == "" {
//	        clusterName = fmt.Sprintf("myapp-%s", input.TestID)
//	    }
//
//	    // Delete cluster (best-effort)
//	    if err := deleteCluster(clusterName); err != nil {
//	        log.Printf("Warning: failed to delete cluster: %v", err)
//	        return nil // Don't fail on cleanup errors
//	    }
//
//	    return nil
//	}
type DeleteFunc func(ctx context.Context, input DeleteInput) error

// TestEnvSubengineConfig configures testenv subengine tool registration.
//
// Fields:
//   - Name: Engine name (e.g., "testenv-kind", "testenv-lcr")
//   - Version: Engine version string (e.g., "1.0.0" or git commit hash)
//   - CreateFunc: The create operation implementation function
//   - DeleteFunc: The delete operation implementation function
//
// Example:
//
//	config := TestEnvSubengineConfig{
//	    Name:       "testenv-kind",
//	    Version:    "1.0.0",
//	    CreateFunc: myCreateFunc,
//	    DeleteFunc: myDeleteFunc,
//	}
type TestEnvSubengineConfig struct {
	Name       string     // Engine name (e.g., "testenv-kind")
	Version    string     // Engine version
	CreateFunc CreateFunc // Create operation implementation
	DeleteFunc DeleteFunc // Delete operation implementation
}

// RegisterTestEnvSubengineTools registers create and delete tools with the MCP server.
//
// This function automatically:
//   - Registers "create" tool that calls the CreateFunc
//   - Registers "delete" tool that calls the DeleteFunc
//   - Validates required input fields (TestID, Stage, TmpDir for create; TestID for delete)
//   - Converts function errors to MCP error responses
//   - Returns TestEnvArtifact on successful create
//   - Uses SuccessResultWithArtifact for create operations
//   - Uses SuccessResult for delete operations
//
// Parameters:
//   - server: The MCP server instance
//   - config: TestEnvSubengine configuration with Name, Version, CreateFunc, and DeleteFunc
//
// Returns:
//   - nil on success
//   - error if tool registration fails (e.g., duplicate tool names)
//
// Example:
//
//	func runMCPServer() error {
//	    server := mcpserver.New("testenv-kind", "1.0.0")
//
//	    config := TestEnvSubengineConfig{
//	        Name:       "testenv-kind",
//	        Version:    "1.0.0",
//	        CreateFunc: myCreateFunc,
//	        DeleteFunc: myDeleteFunc,
//	    }
//
//	    if err := RegisterTestEnvSubengineTools(server, config); err != nil {
//	        return err
//	    }
//
//	    return server.RunDefault()
//	}
func RegisterTestEnvSubengineTools(server *mcpserver.Server, config TestEnvSubengineConfig) error {
	// Register create tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "create",
		Description: fmt.Sprintf("Create a test environment resource using %s", config.Name),
	}, makeCreateHandler(config))

	// Register delete tool
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "delete",
		Description: fmt.Sprintf("Delete a test environment resource using %s", config.Name),
	}, makeDeleteHandler(config))

	return nil
}

// makeCreateHandler creates an MCP handler function from a CreateFunc.
//
// The returned handler:
//   - Validates required input fields (TestID, Stage, TmpDir)
//   - Calls the CreateFunc with the input
//   - Converts CreateFunc errors to MCP error responses
//   - Returns TestEnvArtifact as artifact on success
//   - Uses SuccessResultWithArtifact for successful creates
//
// This is an internal helper function used by RegisterTestEnvSubengineTools.
func makeCreateHandler(config TestEnvSubengineConfig) func(context.Context, *mcp.CallToolRequest, CreateInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CreateInput) (*mcp.CallToolResult, any, error) {
		log.Printf("Creating test environment resource: testID=%s, stage=%s using %s", input.TestID, input.Stage, config.Name)

		// Validate required input fields
		if result := mcputil.ValidateRequiredWithPrefix("Create failed", map[string]string{
			"testID": input.TestID,
			"stage":  input.Stage,
			"tmpDir": input.TmpDir,
		}); result != nil {
			return result, nil, nil
		}

		// Call the CreateFunc
		artifact, err := config.CreateFunc(ctx, input)
		if err != nil {
			// Creation error
			return mcputil.ErrorResult(fmt.Sprintf("Create failed: %v", err)), nil, nil
		}

		// Check if artifact is nil (shouldn't happen, but defensive)
		if artifact == nil {
			return mcputil.ErrorResult("Create function returned nil artifact"), nil, nil
		}

		// Convert artifact to map[string]interface{} for MCP serialization
		artifactMap := map[string]interface{}{
			"testID":           artifact.TestID,
			"files":            artifact.Files,
			"metadata":         artifact.Metadata,
			"managedResources": artifact.ManagedResources,
			"env":              artifact.Env,
		}

		// Return success with artifact
		result, returnedArtifact := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Created test environment resource using %s", config.Name),
			artifactMap,
		)
		return result, returnedArtifact, nil
	}
}

// makeDeleteHandler creates an MCP handler function from a DeleteFunc.
//
// The returned handler:
//   - Validates required input fields (TestID)
//   - Calls the DeleteFunc with the input
//   - Converts DeleteFunc errors to MCP error responses
//   - Uses SuccessResult for successful deletes
//
// This is an internal helper function used by RegisterTestEnvSubengineTools.
func makeDeleteHandler(config TestEnvSubengineConfig) func(context.Context, *mcp.CallToolRequest, DeleteInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input DeleteInput) (*mcp.CallToolResult, any, error) {
		log.Printf("Deleting test environment resource: testID=%s using %s", input.TestID, config.Name)

		// Validate required input fields
		if result := mcputil.ValidateRequiredWithPrefix("Delete failed", map[string]string{
			"testID": input.TestID,
		}); result != nil {
			return result, nil, nil
		}

		// Call the DeleteFunc
		err := config.DeleteFunc(ctx, input)
		if err != nil {
			// Deletion error
			return mcputil.ErrorResult(fmt.Sprintf("Delete failed: %v", err)), nil, nil
		}

		// Return success
		return mcputil.SuccessResult(fmt.Sprintf("Deleted test environment resource using %s", config.Name)), nil, nil
	}
}
