//go:build unit && generate

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
	"context"
	"os"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// TestGenerateForDir is a helper test to generate code for a specific directory.
// Usage: go test -v -tags generate -run TestGenerateForDir ./cmd/forge-dev -args <dir>
func TestGenerateForDir(t *testing.T) {
	args := os.Args
	// Find -args position and get the directory after it
	srcDir := ""
	for i, arg := range args {
		if arg == "-args" || arg == "--args" {
			if i+1 < len(args) {
				srcDir = args[i+1]
			}
			break
		}
	}

	// Also check for TEST_GENERATE_DIR env var
	if srcDir == "" {
		srcDir = os.Getenv("TEST_GENERATE_DIR")
	}

	if srcDir == "" {
		t.Skip("No source directory specified. Set TEST_GENERATE_DIR env var or use -args <dir>")
	}

	input := mcptypes.BuildInput{
		Name:   "generate",
		Src:    srcDir,
		Engine: "go://forge-dev",
	}

	artifact, err := generate(context.Background(), input)
	if err != nil {
		t.Fatalf("generate() error: %v", err)
	}

	t.Logf("Generated files in: %s", artifact.Location)
	t.Logf("Checksum: %s", artifact.Version)
}
