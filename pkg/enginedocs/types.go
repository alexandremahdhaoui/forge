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

// Package enginedocs provides shared types and utilities for distributed documentation
// management across forge engines. Each engine maintains its own documentation in a
// docs/ subdirectory with a list.yaml registry file.
package enginedocs

// DocStore represents the structure of a list.yaml documentation registry.
// Each engine maintains its own DocStore in cmd/{engine}/docs/list.yaml.
type DocStore struct {
	// Version is the schema version (e.g., "1.0")
	Version string `yaml:"version" json:"version"`
	// Engine is the engine name (must match the directory name, e.g., "go-build")
	Engine string `yaml:"engine" json:"engine"`
	// BaseURL is the URL prefix for constructing full document URLs
	// (e.g., "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main")
	BaseURL string `yaml:"baseURL" json:"baseURL"`
	// Docs is the list of documentation entries
	Docs []DocEntry `yaml:"docs" json:"docs"`
}

// DocEntry represents a single documentation item in the registry.
type DocEntry struct {
	// Name is a unique identifier for the document (e.g., "usage", "schema")
	Name string `yaml:"name" json:"name"`
	// Title is the human-readable title displayed in list output
	Title string `yaml:"title" json:"title"`
	// Description is a brief description of the document
	Description string `yaml:"description" json:"description"`
	// URL is the path relative to the repository root (e.g., "cmd/go-build/docs/usage.md")
	URL string `yaml:"url" json:"url"`
	// Tags is a list of searchable tags for filtering (optional)
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	// Required indicates whether this is a required document for the engine (optional)
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`
}

// Config provides per-engine configuration for documentation management.
// This is used by the enginedocs package to locate and validate documentation.
type Config struct {
	// EngineName is the engine identifier (e.g., "go-build")
	EngineName string `yaml:"engineName" json:"engineName"`
	// LocalDir is the path to the docs/ directory relative to the repository root
	// (e.g., "cmd/go-build/docs")
	LocalDir string `yaml:"localDir" json:"localDir"`
	// BaseURL is the GitHub raw URL prefix for remote fetching
	// (e.g., "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main")
	BaseURL string `yaml:"baseURL" json:"baseURL"`
	// RequiredDocs is the list of required document names that must exist
	// (e.g., ["usage", "schema"])
	RequiredDocs []string `yaml:"requiredDocs" json:"requiredDocs"`
}
