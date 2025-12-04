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
	"fmt"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// extractOpenAPIConfigFromInput extracts OpenAPI config from BuildInput.Spec.
//
// Expected Spec structure (see context.md Decision 7 for full details):
//
//	{
//	  // Source (EITHER sourceFile OR sourceDir+name+version)
//	  "sourceFile": string,              // Explicit path (recommended)
//	  // OR
//	  "sourceDir": string,                // Directory for templating
//	  "name": string,                     // API name for templating
//	  "version": string,                  // API version for templating
//
//	  // Destination
//	  "destinationDir": string,           // Defaults to "./pkg/generated"
//
//	  // Client/Server
//	  "client": {"enabled": bool, "packageName": string},
//	  "server": {"enabled": bool, "packageName": string},
//	}
//
// Validation rules:
//  1. MUST provide EITHER sourceFile OR (sourceDir AND name AND version)
//  2. IF client.enabled=true THEN client.packageName is required
//  3. IF server.enabled=true THEN server.packageName is required
//  4. At least one of client.enabled or server.enabled must be true
//
// Note: Paths in Spec are kept as-is (relative or absolute). Relative paths will be resolved
// when executing commands, based on the working directory where the command is run.
func extractOpenAPIConfigFromInput(input mcptypes.BuildInput) (*forge.GenerateOpenAPIConfig, error) {
	spec := input.Spec
	if spec == nil {
		return nil, fmt.Errorf("BuildInput.Spec is required")
	}

	// Extract source file fields
	sourceFile, _ := spec["sourceFile"].(string)
	sourceDir, _ := spec["sourceDir"].(string)
	name, _ := spec["name"].(string)
	version, _ := spec["version"].(string)

	// Extract destination directory
	destinationDir, _ := spec["destinationDir"].(string)
	if destinationDir == "" {
		destinationDir = "./pkg/generated" // Apply default
	}
	// Keep paths as-is (relative or absolute) - they will be resolved when executing commands

	// Extract client configuration
	var clientEnabled bool
	var clientPackageName string
	if clientMap, ok := spec["client"].(map[string]interface{}); ok {
		if enabled, ok := clientMap["enabled"]; ok {
			enabledBool, ok := enabled.(bool)
			if !ok {
				return nil, fmt.Errorf("client.enabled must be a boolean")
			}
			clientEnabled = enabledBool
		}
		clientPackageName, _ = clientMap["packageName"].(string)
	}

	// Extract server configuration
	var serverEnabled bool
	var serverPackageName string
	if serverMap, ok := spec["server"].(map[string]interface{}); ok {
		if enabled, ok := serverMap["enabled"]; ok {
			enabledBool, ok := enabled.(bool)
			if !ok {
				return nil, fmt.Errorf("server.enabled must be a boolean")
			}
			serverEnabled = enabledBool
		}
		serverPackageName, _ = serverMap["packageName"].(string)
	}

	// Validation Rule 1: MUST provide EITHER sourceFile OR (sourceDir AND name AND version)
	hasSourceFile := sourceFile != ""
	hasTemplatedSource := sourceDir != "" && name != "" && version != ""

	if !hasSourceFile && !hasTemplatedSource {
		return nil, fmt.Errorf("must provide either 'sourceFile' or all of 'sourceDir', 'name', and 'version'")
	}

	// Validation Rule 2: IF client.enabled=true THEN client.packageName is required
	if clientEnabled && clientPackageName == "" {
		return nil, fmt.Errorf("client.packageName is required when client.enabled=true")
	}

	// Validation Rule 3: IF server.enabled=true THEN server.packageName is required
	if serverEnabled && serverPackageName == "" {
		return nil, fmt.Errorf("server.packageName is required when server.enabled=true")
	}

	// Validation Rule 4: At least one of client.enabled or server.enabled must be true
	if !clientEnabled && !serverEnabled {
		return nil, fmt.Errorf("at least one of client.enabled or server.enabled must be true")
	}

	// Build the config structure for the new design
	// Decision 1: One BuildSpec per version - NO versions array
	// We create a clean structure that doGenerate() will understand

	// Determine the source path
	var actualSourcePath string
	if sourceFile != "" {
		// Pattern 1: Explicit sourceFile (recommended)
		// Keep as-is (relative or absolute)
		actualSourcePath = sourceFile
	} else {
		// Pattern 2: Templated from sourceDir+name+version
		// Template it now instead of relying on doGenerate to do it
		// Use sourceFileTemplate const from main.go: "%s.%s.yaml"
		filename := fmt.Sprintf(sourceFileTemplate, name, version)
		// Preserve the sourceDir path exactly as provided (including ./ prefix if present)
		if sourceDir == "" || sourceDir == "." {
			actualSourcePath = filename
		} else {
			actualSourcePath = filepath.Join(sourceDir, filename)
		}
	}

	// Create config with EMPTY Versions array
	// doGenerate() has been updated to handle this case
	config := &forge.GenerateOpenAPIConfig{
		Defaults: forge.GenerateOpenAPIDefaults{
			DestinationDir: destinationDir,
		},
		Specs: []forge.GenerateOpenAPISpec{
			{
				Source:         actualSourcePath, // Fully resolved source path
				DestinationDir: destinationDir,
				Versions:       []string{}, // Empty - no versions array in new design
				Client: forge.GenOpts{
					Enabled:     clientEnabled,
					PackageName: clientPackageName,
				},
				Server: forge.GenOpts{
					Enabled:     serverEnabled,
					PackageName: serverPackageName,
				},
			},
		},
	}

	return config, nil
}
