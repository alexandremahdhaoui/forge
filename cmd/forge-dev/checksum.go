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
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const (
	// ChecksumPrefix is the prefix for checksum values.
	ChecksumPrefix = "sha256:"
	// ChecksumHeaderPrefix is the prefix for checksum header comments in generated files.
	ChecksumHeaderPrefix = "// SourceChecksum: "
)

// ComputeSourceChecksum computes a SHA256 checksum of the concatenated contents of
// the configuration file and the OpenAPI spec file.
func ComputeSourceChecksum(configPath, specPath string) (string, error) {
	// Read config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	// Read spec file
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return "", fmt.Errorf("reading spec file %s: %w", specPath, err)
	}

	// Concatenate contents and compute hash
	h := sha256.New()
	h.Write(configData)
	h.Write(specData)

	checksum := hex.EncodeToString(h.Sum(nil))
	return ChecksumPrefix + checksum, nil
}

// ReadChecksumFromFile reads the checksum from a generated file's header comment.
// It looks for a line starting with "// SourceChecksum: " and extracts the checksum value.
// Returns an empty string and no error if the file doesn't exist or doesn't have a checksum.
func ReadChecksumFromFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("opening file %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	// Only scan the first few lines (header comments)
	lineCount := 0
	maxLines := 10

	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		lineCount++

		if strings.HasPrefix(line, ChecksumHeaderPrefix) {
			checksum := strings.TrimPrefix(line, ChecksumHeaderPrefix)
			return strings.TrimSpace(checksum), nil
		}

		// Stop if we've passed the header comments
		if !strings.HasPrefix(line, "//") && line != "" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}

	return "", nil
}

// ChecksumHeader formats a checksum for inclusion in a generated file header.
// It returns a line like: "// SourceChecksum: sha256:abc123..."
func ChecksumHeader(checksum string) string {
	return ChecksumHeaderPrefix + checksum
}

// ChecksumMatches compares a computed checksum with an existing file's checksum.
// Returns true if they match, indicating regeneration is not needed.
func ChecksumMatches(computed, existing string) bool {
	if computed == "" || existing == "" {
		return false
	}
	return computed == existing
}
