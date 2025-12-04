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
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// cmdGet retrieves and displays details about a specific test report.
func cmdGet(reportID string) error {
	// Get artifact store path from environment variable
	artifactStorePath := os.Getenv("FORGE_ARTIFACT_STORE_PATH")
	if artifactStorePath == "" {
		config, err := forge.ReadSpec()
		if err != nil {
			return fmt.Errorf("failed to read forge.yaml: %w", err)
		}
		artifactStorePath, err = forge.GetArtifactStorePath(config.ArtifactStorePath)
		if err != nil {
			return fmt.Errorf("failed to get artifact store path: %w", err)
		}
	}

	// Read artifact store
	store, err := forge.ReadArtifactStore(artifactStorePath)
	if err != nil {
		return fmt.Errorf("failed to read artifact store: %w", err)
	}

	// Get test report
	report, err := forge.GetTestReport(&store, reportID)
	if err != nil {
		return fmt.Errorf("failed to get test report: %w", err)
	}

	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode test report: %w", err)
	}

	return nil
}
