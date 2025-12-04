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

// testenv-stub is a lightweight no-op testenv subengine for fast e2e testing.
// It does nothing except return mock metadata, allowing the testenv create/list/get/delete
// workflow to be tested without the overhead of creating real resources (like KIND clusters).
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

func main() {
	cli.Bootstrap(cli.Config{
		Name:           "testenv-stub",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
}

// runMCPServer starts the testenv-stub MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New("testenv-stub", Version)

	config := engineframework.TestEnvSubengineConfig{
		Name:       "testenv-stub",
		Version:    Version,
		CreateFunc: createStubEnv,
		DeleteFunc: deleteStubEnv,
	}

	if err := engineframework.RegisterTestEnvSubengineTools(server, config); err != nil {
		return err
	}

	return server.RunDefault()
}

// createStubEnv implements the CreateFunc for the stub test environment.
// It creates a minimal stub file and returns mock metadata without creating real resources.
func createStubEnv(ctx context.Context, input engineframework.CreateInput) (*engineframework.TestEnvArtifact, error) {
	log.Printf("Creating stub test environment: testID=%s, stage=%s", input.TestID, input.Stage)

	// Create a stub file in tmpDir to simulate artifact creation
	stubFilePath := filepath.Join(input.TmpDir, "stub-marker.txt")
	stubContent := []byte("stub test environment created at " + time.Now().Format(time.RFC3339))
	if err := os.WriteFile(stubFilePath, stubContent, 0o644); err != nil {
		return nil, err
	}

	log.Printf("Stub test environment created successfully: testID=%s", input.TestID)

	return &engineframework.TestEnvArtifact{
		TestID: input.TestID,
		Files: map[string]string{
			"testenv-stub.marker": "stub-marker.txt",
		},
		Metadata: map[string]string{
			"testenv-stub.createdAt": time.Now().Format(time.RFC3339),
			"testenv-stub.testID":    input.TestID,
			"testenv-stub.stage":     input.Stage,
		},
		ManagedResources: []string{stubFilePath},
		Env: map[string]string{
			"TESTENV_STUB_ACTIVE": "true",
		},
	}, nil
}

// deleteStubEnv implements the DeleteFunc for the stub test environment.
// It does nothing since the stub doesn't create real resources.
func deleteStubEnv(ctx context.Context, input engineframework.DeleteInput) error {
	log.Printf("Deleting stub test environment: testID=%s", input.TestID)

	// Nothing to clean up for stub - tmpDir cleanup is handled by the orchestrator
	log.Printf("Stub test environment deleted (no-op): testID=%s", input.TestID)
	return nil
}
