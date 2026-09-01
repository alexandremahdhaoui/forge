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

package cmdutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// LoadEnvFile loads environment variables from a file.
//
// Supported formats:
//   - KEY=VALUE
//   - export KEY=VALUE
//   - KEY="VALUE with spaces"
//   - # comments
//
// Empty lines and comments (starting with #) are skipped.
// If the file doesn't exist, returns an empty map (not an error).
func LoadEnvFile(path string) (map[string]string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return empty map if file doesn't exist (not an error)
		return make(map[string]string), nil
	}

	// Read file contents
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	envVars := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove "export " prefix if present
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		// Split on first '=' sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format in env file at line %d: %s", lineNum+1, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		envVars[key] = value
	}

	return envVars, nil
}

// validateEnvFilePath validates the path to an environment file for security.
// It rejects absolute paths, parent directory traversal, and shell metacharacters.
func validateEnvFilePath(path string) error {
	// Check if path is absolute
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("absolute paths are not allowed: %s", path)
	}

	// Check for parent directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("parent directory traversal is not allowed: %s", path)
	}

	// Validate path with regex (alphanumeric, dot, underscore, slash, hyphen only)
	validPathRegex := regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	if !validPathRegex.MatchString(path) {
		return fmt.Errorf("path contains invalid characters: %s", path)
	}

	// Check for shell metacharacters explicitly (additional layer of defense)
	shellMetachars := []string{";", "|", "&", "$", "(", ")", "`", ">", "<"}
	for _, char := range shellMetachars {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains shell metacharacter '%s': %s", char, path)
		}
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env file does not exist: %s", path)
		}
		return fmt.Errorf("failed to stat env file: %w", err)
	}

	return nil
}

// shellBookkeepingVars are set or changed by the shell that executes the
// env file, not by the file itself, so the diff must not import them.
var shellBookkeepingVars = map[string]bool{
	"SHLVL": true, "_": true, "PWD": true, "OLDPWD": true, "SHELL": true,
}

// shellQuote provides POSIX-compliant shell quoting for paths.
// It wraps the path in single quotes and escapes any single quotes using the
// standard '\” pattern (end quote, escaped quote, start quote).
//
// Design decision: We use a custom implementation instead of an external library
// (like github.com/alessio/shellescape) to avoid adding external dependencies.
// This implementation follows the POSIX standard for single-quote escaping and
// is sufficient for our use case of quoting validated file paths.
func shellQuote(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

// executeEnvFileInShell executes the environment file in a bash shell
// and captures the resulting environment variables.
func executeEnvFileInShell(path string) ([]byte, error) {
	// Quote the path for shell safety
	quotedPath := shellQuote(path)

	// Build the shell command: source the file and print env with null-termination
	shellCmd := fmt.Sprintf("source %s && env -0", quotedPath)

	// Execute in bash
	cmd := exec.Command("/bin/bash", "-c", shellCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute env file: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// parseEnvOutput parses the null-terminated environment output into a map.
func parseEnvOutput(output []byte) map[string]string {
	result := make(map[string]string)

	// Split on null bytes
	entries := bytes.Split(output, []byte{0})
	for _, entry := range entries {
		if len(entry) == 0 {
			continue
		}

		// Split on first '=' only
		parts := bytes.SplitN(entry, []byte{'='}, 2)
		if len(parts) != 2 {
			continue
		}

		result[string(parts[0])] = string(parts[1])
	}

	return result
}

// SourceEnvFile sources an environment file by executing it in a bash shell
// and applying to this process what the execution changed: every variable
// the file added or modified is set, and every variable it unset is unset.
//
// The delta is measured against this process's environment rather than
// parsed out of the file's text. A textual parse only sees `export NAME=`
// lines in the file itself and silently drops whatever a sourced sub-file
// exports - which is exactly how a generated env file sourced from a
// managed block lost every variable except the one the block exported
// directly.
func SourceEnvFile(envFilePath string) error {
	// Step 1: Validate the path
	if err := validateEnvFilePath(envFilePath); err != nil {
		return fmt.Errorf("invalid env file path: %w", err)
	}

	// Step 2: Execute the file in a shell and capture the resulting env
	output, err := executeEnvFileInShell(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to execute env file: %w", err)
	}

	after := parseEnvOutput(output)

	// Step 3: The environment the shell started from
	before := make(map[string]string)
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		before[parts[0]] = parts[1]
	}

	// Step 4: Apply what changed
	applied := 0

	for name, value := range after {
		if shellBookkeepingVars[name] {
			continue
		}
		if prev, ok := before[name]; ok && prev == value {
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", name, err)
		}
		applied++
	}

	for name := range before {
		if shellBookkeepingVars[name] {
			continue
		}
		if _, ok := after[name]; ok {
			continue
		}
		if err := os.Unsetenv(name); err != nil {
			return fmt.Errorf("failed to unset environment variable %s: %w", name, err)
		}
		applied++
	}

	// Step 5: Log success (count only, no variable names or values)
	fmt.Fprintf(os.Stderr, "Sourced %d environment variables from %s\n", applied, envFilePath)

	return nil
}
