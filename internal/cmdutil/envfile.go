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

// parseExportedVarNames extracts the list of environment variable names
// that are expected to be set after sourcing the file. It handles both
// export and unset statements, with the last statement for each variable winning.
func parseExportedVarNames(content []byte) ([]string, error) {
	exportRegex := regexp.MustCompile(`^\s*export\s+([A-Z_][A-Z0-9_]*)=`)
	unsetRegex := regexp.MustCompile(`^\s*unset\s+([A-Z_][A-Z0-9_]*)`)

	// Track last line number for each exported and unset variable
	lastExportLine := make(map[string]int)
	lastUnsetLine := make(map[string]int)

	lines := bytes.Split(content, []byte{'\n'})
	for lineNum, line := range lines {
		lineStr := string(line)

		// Check for export statement
		if matches := exportRegex.FindStringSubmatch(lineStr); matches != nil {
			varName := matches[1]
			lastExportLine[varName] = lineNum
		}

		// Check for unset statement
		if matches := unsetRegex.FindStringSubmatch(lineStr); matches != nil {
			varName := matches[1]
			lastUnsetLine[varName] = lineNum
		}
	}

	// Collect variables where last export > last unset (or no unset exists)
	var expectedVars []string
	for varName, exportLine := range lastExportLine {
		unsetLine, wasUnset := lastUnsetLine[varName]
		if !wasUnset || exportLine > unsetLine {
			expectedVars = append(expectedVars, varName)
		}
	}

	return expectedVars, nil
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

// parseEnvOutput parses the null-terminated environment output
// and filters it to only include expected variables.
func parseEnvOutput(output []byte, expectedVars []string) (map[string]string, error) {
	// Create a set of expected variable names for quick lookup
	expectedSet := make(map[string]bool)
	for _, varName := range expectedVars {
		expectedSet[varName] = true
	}

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

		varName := string(parts[0])
		varValue := string(parts[1])

		// Only include if in expected list
		if expectedSet[varName] {
			result[varName] = varValue
		}
	}

	return result, nil
}

// SourceEnvFile sources an environment file by executing it in a bash shell
// and setting the exported variables in the current process.
// This function performs security validation, parses the file to identify
// expected variables, executes the file in a shell, and sets the resulting
// environment variables.
func SourceEnvFile(envFilePath string) error {
	// Step 1: Validate the path
	if err := validateEnvFilePath(envFilePath); err != nil {
		return fmt.Errorf("invalid env file path: %w", err)
	}

	// Step 2: Read file content
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	// Step 3: Parse exported variable names
	expectedVars, err := parseExportedVarNames(content)
	if err != nil {
		return fmt.Errorf("failed to parse exported variables: %w", err)
	}

	// Step 4: Execute the file in a shell
	output, err := executeEnvFileInShell(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to execute env file: %w", err)
	}

	// Step 5: Parse the environment output
	envMap, err := parseEnvOutput(output, expectedVars)
	if err != nil {
		return fmt.Errorf("failed to parse env output: %w", err)
	}

	// Step 6: Set environment variables
	for varName, varValue := range envMap {
		if err := os.Setenv(varName, varValue); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", varName, err)
		}
	}

	// Step 7: Log success (count only, no variable names or values)
	fmt.Fprintf(os.Stderr, "Sourced %d environment variables from %s\n", len(envMap), envFilePath)

	return nil
}
