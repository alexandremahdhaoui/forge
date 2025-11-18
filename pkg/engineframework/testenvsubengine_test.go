//go:build unit

package engineframework

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockCreateFunc creates a mock CreateFunc for testing.
// Returns artifact if testID doesn't contain "fail".
// Returns error if returnError is true or testID contains "fail".
func mockCreateFunc(returnError bool) CreateFunc {
	return func(ctx context.Context, input CreateInput) (*TestEnvArtifact, error) {
		// Simulate failure if testID contains "fail"
		if strings.Contains(input.TestID, "fail") || returnError {
			return nil, errors.New("create failed: simulated error")
		}

		// Return success artifact
		return &TestEnvArtifact{
			TestID: input.TestID,
			Files: map[string]string{
				"testenv.kubeconfig": "kubeconfig",
				"testenv.config":     "config.yaml",
			},
			Metadata: map[string]string{
				"testenv.clusterName": "cluster-" + input.TestID,
				"testenv.registryURL": "localhost:5000",
			},
			ManagedResources: []string{
				"/tmp/" + input.TestID + "/kubeconfig",
				"/tmp/" + input.TestID + "/config.yaml",
			},
		}, nil
	}
}

// mockDeleteFunc creates a mock DeleteFunc for testing.
// Returns error if returnError is true or testID contains "fail".
func mockDeleteFunc(returnError bool) DeleteFunc {
	return func(ctx context.Context, input DeleteInput) error {
		// Simulate failure if testID contains "fail"
		if strings.Contains(input.TestID, "fail") || returnError {
			return errors.New("delete failed: simulated error")
		}

		// Success
		return nil
	}
}

func TestMakeCreateHandler_Success(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "integration",
		TmpDir: "/tmp/test-123",
		Metadata: map[string]string{
			"previous.key": "value",
		},
		Spec: map[string]any{
			"setting": "value",
		},
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should not be an error
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				t.Fatalf("handler returned error result: %s", textContent.Text)
			}
		}
		t.Fatalf("handler returned error result")
	}

	// Artifact should be returned
	if artifact == nil {
		t.Fatal("handler returned nil artifact")
	}

	// Artifact should be map[string]interface{}
	artifactMap, ok := artifact.(map[string]interface{})
	if !ok {
		t.Fatalf("artifact is not map[string]interface{}, got %T", artifact)
	}

	// Verify artifact fields
	if artifactMap["testID"] != "test-123" {
		t.Errorf("artifact.testID = %v, want %q", artifactMap["testID"], "test-123")
	}

	// Verify files field exists and is correct type
	files, ok := artifactMap["files"].(map[string]string)
	if !ok {
		t.Fatalf("artifact.files is not map[string]string, got %T", artifactMap["files"])
	}
	if len(files) != 2 {
		t.Errorf("artifact.files has %d entries, want 2", len(files))
	}

	// Verify metadata field exists and is correct type
	metadata, ok := artifactMap["metadata"].(map[string]string)
	if !ok {
		t.Fatalf("artifact.metadata is not map[string]string, got %T", artifactMap["metadata"])
	}
	if len(metadata) != 2 {
		t.Errorf("artifact.metadata has %d entries, want 2", len(metadata))
	}

	// Verify managedResources field exists and is correct type
	managedResources, ok := artifactMap["managedResources"].([]string)
	if !ok {
		t.Fatalf("artifact.managedResources is not []string, got %T", artifactMap["managedResources"])
	}
	if len(managedResources) != 2 {
		t.Errorf("artifact.managedResources has %d entries, want 2", len(managedResources))
	}
}

func TestMakeCreateHandler_CreateFuncError(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(true), // Always returns error
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "integration",
		TmpDir: "/tmp/test-123",
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error (errors converted to MCP results)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error
	if !result.IsError {
		t.Fatal("handler should return error result when CreateFunc fails")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite error: %v", artifact)
	}

	// Error message should contain "Create failed"
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if !strings.Contains(textContent.Text, "Create failed") {
				t.Errorf("error message does not contain 'Create failed': %s", textContent.Text)
			}
		}
	}
}

func TestMakeCreateHandler_MissingTestID(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "", // Missing testID
		Stage:  "integration",
		TmpDir: "/tmp/test-123",
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (validation failure)
	if !result.IsError {
		t.Fatal("handler should return error result when required field is missing")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite validation error: %v", artifact)
	}
}

func TestMakeCreateHandler_MissingStage(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "", // Missing stage
		TmpDir: "/tmp/test-123",
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (validation failure)
	if !result.IsError {
		t.Fatal("handler should return error result when required field is missing")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite validation error: %v", artifact)
	}
}

func TestMakeCreateHandler_MissingTmpDir(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "integration",
		TmpDir: "", // Missing tmpDir
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (validation failure)
	if !result.IsError {
		t.Fatal("handler should return error result when required field is missing")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite validation error: %v", artifact)
	}
}

func TestMakeCreateHandler_NilArtifact(t *testing.T) {
	// Test defensive handling of nil artifact from CreateFunc
	nilArtifactFunc := func(ctx context.Context, input CreateInput) (*TestEnvArtifact, error) {
		return nil, nil // Returns nil artifact (shouldn't happen, but defensive)
	}

	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: nilArtifactFunc,
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "integration",
		TmpDir: "/tmp/test-123",
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (nil artifact)
	if !result.IsError {
		t.Fatal("handler should return error result when CreateFunc returns nil artifact")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite nil artifact from CreateFunc: %v", artifact)
	}

	// Error message should mention nil artifact
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if !strings.Contains(textContent.Text, "nil artifact") {
				t.Errorf("error message should mention nil artifact: %s", textContent.Text)
			}
		}
	}
}

func TestMakeDeleteHandler_Success(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeDeleteHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := DeleteInput{
		TestID: "test-123",
		Metadata: map[string]string{
			"testenv.clusterName": "cluster-test-123",
		},
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should not be an error
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				t.Fatalf("handler returned error result: %s", textContent.Text)
			}
		}
		t.Fatalf("handler returned error result")
	}

	// Artifact should be nil (delete doesn't return artifacts)
	if artifact != nil {
		t.Errorf("handler returned artifact for delete: %v", artifact)
	}
}

func TestMakeDeleteHandler_DeleteFuncError(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(true), // Always returns error
	}

	handler := makeDeleteHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := DeleteInput{
		TestID: "test-123",
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error (errors converted to MCP results)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error
	if !result.IsError {
		t.Fatal("handler should return error result when DeleteFunc fails")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite error: %v", artifact)
	}

	// Error message should contain "Delete failed"
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			if !strings.Contains(textContent.Text, "Delete failed") {
				t.Errorf("error message does not contain 'Delete failed': %s", textContent.Text)
			}
		}
	}
}

func TestMakeDeleteHandler_MissingTestID(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeDeleteHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := DeleteInput{
		TestID: "", // Missing testID
	}

	result, artifact, err := handler(ctx, req, input)
	// Should not return Go error
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Result should be an error (validation failure)
	if !result.IsError {
		t.Fatal("handler should return error result when required field is missing")
	}

	// Artifact should be nil
	if artifact != nil {
		t.Errorf("handler returned artifact despite validation error: %v", artifact)
	}
}

func TestRegisterTestEnvSubengineTools(t *testing.T) {
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	server := mcpserver.New("testenv-test", "1.0.0")

	err := RegisterTestEnvSubengineTools(server, config)
	if err != nil {
		t.Fatalf("RegisterTestEnvSubengineTools returned error: %v", err)
	}

	// NOTE: We can't easily test that tools are registered without
	// examining internal server state or running a full MCP server.
	// This test just verifies the function doesn't error.
	// Real validation happens in integration tests.
}

func TestRegisterTestEnvSubengineTools_IntegrationTest(t *testing.T) {
	// This test verifies that the registered tools can actually be called
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	server := mcpserver.New("testenv-test", "1.0.0")

	err := RegisterTestEnvSubengineTools(server, config)
	if err != nil {
		t.Fatalf("RegisterTestEnvSubengineTools returned error: %v", err)
	}

	// Create handlers
	createHandler := makeCreateHandler(config)
	deleteHandler := makeDeleteHandler(config)

	// Test create handler
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	createInput := CreateInput{
		TestID: "integration-test",
		Stage:  "integration",
		TmpDir: "/tmp/integration-test",
	}

	createResult, createArtifact, createErr := createHandler(ctx, req, createInput)
	if createErr != nil {
		t.Errorf("create handler returned error: %v", createErr)
	}
	if createResult.IsError {
		t.Error("create handler returned error result")
	}
	if createArtifact == nil {
		t.Error("create handler returned nil artifact")
	}

	// Test delete handler
	deleteInput := DeleteInput{
		TestID: "integration-test",
	}

	deleteResult, deleteArtifact, deleteErr := deleteHandler(ctx, req, deleteInput)
	if deleteErr != nil {
		t.Errorf("delete handler returned error: %v", deleteErr)
	}
	if deleteResult.IsError {
		t.Error("delete handler returned error result")
	}
	if deleteArtifact != nil {
		t.Error("delete handler returned artifact (should be nil)")
	}
}

func TestCreateFunc_ContextPropagation(t *testing.T) {
	// Test that context is properly propagated to CreateFunc
	var receivedCtx context.Context

	testCreate := func(ctx context.Context, input CreateInput) (*TestEnvArtifact, error) {
		receivedCtx = ctx
		return &TestEnvArtifact{
			TestID:   input.TestID,
			Files:    map[string]string{},
			Metadata: map[string]string{},
		}, nil
	}

	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: testCreate,
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	// Create context with value
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("test"), "value")
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "integration",
		TmpDir: "/tmp/test-123",
	}

	_, _, err := handler(ctx, req, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Verify context was propagated
	if receivedCtx == nil {
		t.Fatal("context was not propagated to CreateFunc")
	}

	// Verify context value
	if val := receivedCtx.Value(ctxKey("test")); val != "value" {
		t.Errorf("context value = %v, want %q", val, "value")
	}
}

func TestDeleteFunc_ContextPropagation(t *testing.T) {
	// Test that context is properly propagated to DeleteFunc
	var receivedCtx context.Context

	testDelete := func(ctx context.Context, input DeleteInput) error {
		receivedCtx = ctx
		return nil
	}

	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: testDelete,
	}

	handler := makeDeleteHandler(config)

	// Create context with value
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("test"), "value")
	req := &mcp.CallToolRequest{}
	input := DeleteInput{
		TestID: "test-123",
	}

	_, _, err := handler(ctx, req, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// Verify context was propagated
	if receivedCtx == nil {
		t.Fatal("context was not propagated to DeleteFunc")
	}

	// Verify context value
	if val := receivedCtx.Value(ctxKey("test")); val != "value" {
		t.Errorf("context value = %v, want %q", val, "value")
	}
}

func TestArtifactFields(t *testing.T) {
	// Test that all artifact fields are properly set
	config := TestEnvSubengineConfig{
		Name:       "testenv-test",
		Version:    "1.0.0",
		CreateFunc: mockCreateFunc(false),
		DeleteFunc: mockDeleteFunc(false),
	}

	handler := makeCreateHandler(config)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := CreateInput{
		TestID: "test-123",
		Stage:  "integration",
		TmpDir: "/tmp/test-123",
	}

	_, artifact, err := handler(ctx, req, input)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	artifactMap, ok := artifact.(map[string]interface{})
	if !ok {
		t.Fatalf("artifact is not map[string]interface{}, got %T", artifact)
	}

	// Verify all expected fields are present
	expectedFields := []string{"testID", "files", "metadata", "managedResources"}
	for _, field := range expectedFields {
		if _, exists := artifactMap[field]; !exists {
			t.Errorf("artifact is missing field %q", field)
		}
	}

	// Verify testID
	if artifactMap["testID"] != "test-123" {
		t.Errorf("artifact.testID = %v, want %q", artifactMap["testID"], "test-123")
	}

	// Verify files is map[string]string
	if _, ok := artifactMap["files"].(map[string]string); !ok {
		t.Errorf("artifact.files is not map[string]string, got %T", artifactMap["files"])
	}

	// Verify metadata is map[string]string
	if _, ok := artifactMap["metadata"].(map[string]string); !ok {
		t.Errorf("artifact.metadata is not map[string]string, got %T", artifactMap["metadata"])
	}

	// Verify managedResources is []string
	if _, ok := artifactMap["managedResources"].([]string); !ok {
		t.Errorf("artifact.managedResources is not []string, got %T", artifactMap["managedResources"])
	}
}
