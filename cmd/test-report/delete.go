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

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	ID               string   `json:"id"`
	Success          bool     `json:"success"`
	DeletedFiles     []string `json:"deletedFiles,omitempty"`
	FailedFiles      []string `json:"failedFiles,omitempty"`
	ErrorMessage     string   `json:"errorMessage,omitempty"`
	PartiallyDeleted bool     `json:"partiallyDeleted"`
}

// cmdDelete deletes a test report and its associated artifact files.
func cmdDelete(reportID string) error {
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

	// Get test report to access artifact files list
	report, err := forge.GetTestReport(&store, reportID)
	if err != nil {
		return fmt.Errorf("failed to get test report: %w", err)
	}

	// Delete artifact files
	var deletedFiles []string
	var failedFiles []string

	for _, filePath := range report.ArtifactFiles {
		if err := os.Remove(filePath); err != nil {
			if !os.IsNotExist(err) {
				// File exists but couldn't be deleted
				failedFiles = append(failedFiles, filePath)
			}
			// If file doesn't exist, consider it already deleted
		} else {
			deletedFiles = append(deletedFiles, filePath)
		}
	}

	// Delete test report from store atomically
	// Use AtomicDeleteTestReport to avoid race conditions with concurrent writes
	if err := forge.AtomicDeleteTestReport(artifactStorePath, reportID); err != nil {
		// Report couldn't be deleted from store
		result := DeleteResult{
			ID:               reportID,
			Success:          false,
			DeletedFiles:     deletedFiles,
			FailedFiles:      failedFiles,
			ErrorMessage:     fmt.Sprintf("failed to delete report from artifact store: %v", err),
			PartiallyDeleted: len(deletedFiles) > 0,
		}
		outputResult(result)
		return fmt.Errorf("failed to delete test report: %w", err)
	}

	// Success
	result := DeleteResult{
		ID:               reportID,
		Success:          true,
		DeletedFiles:     deletedFiles,
		FailedFiles:      failedFiles,
		PartiallyDeleted: len(failedFiles) > 0,
	}

	if len(failedFiles) > 0 {
		result.ErrorMessage = "some files could not be deleted"
	}

	outputResult(result)
	return nil
}

// outputResult outputs the delete result as JSON.
func outputResult(result DeleteResult) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(result) // Ignore error since this is best-effort output
}
