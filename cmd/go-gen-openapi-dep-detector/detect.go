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
	"log"
	"os"
	"time"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// DetectOpenAPIDependencies detects dependencies for OpenAPI code generation.
// It iterates over the input spec sources, stats each file to get its timestamp,
// and returns them as dependencies. If a spec file is not found, it returns an error.
//
// Note: $ref resolution (ResolveRefs) is not implemented in v1 and will log a warning.
func DetectOpenAPIDependencies(input mcptypes.DetectOpenAPIDependenciesInput) (mcptypes.DetectDependenciesOutput, error) {
	var deps []mcptypes.Dependency

	for _, specPath := range input.SpecSources {
		// Verify file exists and get timestamp
		info, err := os.Stat(specPath)
		if err != nil {
			return mcptypes.DetectDependenciesOutput{}, fmt.Errorf("spec file not found: %s: %w", specPath, err)
		}

		deps = append(deps, mcptypes.Dependency{
			Type:      "file",
			FilePath:  specPath,
			Timestamp: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// v1: ResolveRefs is ignored (no $ref resolution)
	if input.ResolveRefs {
		log.Printf("Warning: $ref resolution requested but not implemented in v1")
	}

	return mcptypes.DetectDependenciesOutput{Dependencies: deps}, nil
}
