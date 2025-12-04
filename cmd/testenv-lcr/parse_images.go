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
)

// parseImagesFromSpec extracts and validates images from CreateInput.Spec.
// Returns empty slice if "images" key is not present.
func parseImagesFromSpec(spec map[string]any) ([]ImageSource, error) {
	raw, ok := spec["images"]
	if !ok {
		return nil, nil // No images configured, valid case
	}

	// Marshal and unmarshal to convert map[string]any to []ImageSource
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal images: %w", err)
	}

	var images []ImageSource
	if err := json.Unmarshal(jsonBytes, &images); err != nil {
		return nil, fmt.Errorf("failed to unmarshal images: %w", err)
	}

	// Validate all images
	if err := ValidateImages(images); err != nil {
		return nil, fmt.Errorf("invalid images configuration: %w", err)
	}

	return images, nil
}
