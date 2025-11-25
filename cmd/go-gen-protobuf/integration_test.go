//go:build integration

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// TestBuildIntegration tests the full protobuf build flow with real protoc execution.
func TestBuildIntegration(t *testing.T) {
	// Check for required tools
	requiredTools := []string{
		"protoc",
		"protoc-gen-go",
		"protoc-gen-go-grpc",
	}

	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("required tool not found: %s", tool)
		}
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "proto")
	destDir := filepath.Join(tmpDir, "generated")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	// Create minimal valid .proto file
	protoContent := `syntax = "proto3";
package test;
option go_package = "./testpb";
message TestMessage {
  string name = 1;
}
`
	protoFile := filepath.Join(srcDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to write proto file: %v", err)
	}

	// Call build() with valid BuildInput
	input := mcptypes.BuildInput{
		Name: "test-protobuf",
		Src:  srcDir,
		Dest: destDir,
	}

	artifact, err := build(context.Background(), input)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	// Verify artifact has correct name
	if artifact.Name != "test-protobuf" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "test-protobuf")
	}

	// Verify artifact type is "protobuf"
	if artifact.Type != "protobuf" {
		t.Errorf("artifact.Type = %q, want %q", artifact.Type, "protobuf")
	}

	// Verify artifact has DependencyDetectorEngine set
	if artifact.DependencyDetectorEngine != "go://go-gen-protobuf" {
		t.Errorf("artifact.DependencyDetectorEngine = %q, want %q", artifact.DependencyDetectorEngine, "go://go-gen-protobuf")
	}

	// Verify dependencies populated with .proto file
	if len(artifact.Dependencies) == 0 {
		t.Error("artifact.Dependencies is empty, expected at least one dependency")
	} else {
		// Check that the proto file is in dependencies
		foundProtoFile := false
		for _, dep := range artifact.Dependencies {
			if dep.Type != "file" {
				t.Errorf("dependency.Type = %q, want %q", dep.Type, "file")
			}
			if strings.HasSuffix(dep.FilePath, "test.proto") {
				foundProtoFile = true
			}
			if dep.Timestamp == "" {
				t.Error("dependency.Timestamp is empty")
			}
		}
		if !foundProtoFile {
			t.Error("test.proto not found in artifact.Dependencies")
		}
	}

	// Verify generated .pb.go file exists
	// With paths=source_relative, the output path matches the input proto file's relative path
	generatedFile := filepath.Join(destDir, "test.pb.go")
	if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
		t.Errorf("Generated file not found at %s", generatedFile)
	}

	// Verify artifact location matches dest
	if artifact.Location != destDir {
		t.Errorf("artifact.Location = %q, want %q", artifact.Location, destDir)
	}

	// Verify artifact timestamp is set
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	}
}

// TestBuildIntegrationWithGRPC tests that gRPC code is also generated.
func TestBuildIntegrationWithGRPC(t *testing.T) {
	// Check for required tools
	requiredTools := []string{
		"protoc",
		"protoc-gen-go",
		"protoc-gen-go-grpc",
	}

	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("required tool not found: %s", tool)
		}
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "proto")
	destDir := filepath.Join(tmpDir, "generated")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	// Create .proto file with a service definition for gRPC
	protoContent := `syntax = "proto3";
package test;
option go_package = "./testpb";

message PingRequest {
  string message = 1;
}

message PingResponse {
  string message = 1;
}

service TestService {
  rpc Ping(PingRequest) returns (PingResponse);
}
`
	protoFile := filepath.Join(srcDir, "service.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("Failed to write proto file: %v", err)
	}

	// Call build() with valid BuildInput
	input := mcptypes.BuildInput{
		Name: "test-grpc",
		Src:  srcDir,
		Dest: destDir,
	}

	artifact, err := build(context.Background(), input)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	// Verify artifact
	if artifact.Name != "test-grpc" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "test-grpc")
	}

	// Verify generated .pb.go file exists
	// With paths=source_relative, output path matches the input proto file's relative path
	pbFile := filepath.Join(destDir, "service.pb.go")
	if _, err := os.Stat(pbFile); os.IsNotExist(err) {
		t.Errorf("Generated pb file not found at %s", pbFile)
	}

	// Verify generated _grpc.pb.go file exists
	grpcFile := filepath.Join(destDir, "service_grpc.pb.go")
	if _, err := os.Stat(grpcFile); os.IsNotExist(err) {
		t.Errorf("Generated gRPC file not found at %s", grpcFile)
	}
}

// TestBuildIntegrationMultipleFiles tests compilation of multiple proto files.
func TestBuildIntegrationMultipleFiles(t *testing.T) {
	// Check for required tools
	requiredTools := []string{
		"protoc",
		"protoc-gen-go",
		"protoc-gen-go-grpc",
	}

	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("required tool not found: %s", tool)
		}
	}

	// Create temp directory structure
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "proto")
	destDir := filepath.Join(tmpDir, "generated")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	// Create first proto file
	proto1 := `syntax = "proto3";
package test;
option go_package = "./testpb";
message Message1 {
  string field1 = 1;
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "message1.proto"), []byte(proto1), 0o644); err != nil {
		t.Fatalf("Failed to write proto file 1: %v", err)
	}

	// Create second proto file
	proto2 := `syntax = "proto3";
package test;
option go_package = "./testpb";
message Message2 {
  int32 field2 = 1;
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "message2.proto"), []byte(proto2), 0o644); err != nil {
		t.Fatalf("Failed to write proto file 2: %v", err)
	}

	// Call build()
	input := mcptypes.BuildInput{
		Name: "test-multi",
		Src:  srcDir,
		Dest: destDir,
	}

	artifact, err := build(context.Background(), input)
	if err != nil {
		t.Fatalf("build() failed: %v", err)
	}

	// Verify both proto files are in dependencies
	if len(artifact.Dependencies) != 2 {
		t.Errorf("artifact.Dependencies has %d entries, want 2", len(artifact.Dependencies))
	}

	// Verify both generated files exist
	// With paths=source_relative, output path matches the input proto file's relative path
	files := []string{
		filepath.Join(destDir, "message1.pb.go"),
		filepath.Join(destDir, "message2.pb.go"),
	}
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Generated file not found at %s", f)
		}
	}
}
