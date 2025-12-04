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

package testutil

import (
	"os"
	"testing"
)

// TestEnvironment manages test lifecycle including cleanup and resource tracking.
type TestEnvironment struct {
	T            TestingT
	TempDir      string
	ForgeBinary  string
	CleanupFuncs []func() error
	testEnvIDs   []string
	kindClusters []string
}

// NewTestEnvironment creates a new test environment with automatic cleanup.
// It registers cleanup via testing.T.Cleanup() to ensure resources are cleaned up
// even if the test fails.
//
// Note: This function requires a *testing.T, not the TestingT interface,
// because it needs access to TempDir() and Cleanup() methods.
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	env := &TestEnvironment{
		T:            t,
		TempDir:      t.TempDir(), // Automatically cleaned up by testing framework
		CleanupFuncs: make([]func() error, 0),
		testEnvIDs:   make([]string, 0),
		kindClusters: make([]string, 0),
	}

	// Register automatic cleanup
	t.Cleanup(func() {
		env.Cleanup()
	})

	return env
}

// CreateTestEnv creates a test environment and tracks it for cleanup.
// It returns the test environment ID or an error.
func (te *TestEnvironment) CreateTestEnv(stage string) (string, error) {
	// Find forge binary if not already set
	if te.ForgeBinary == "" {
		binary, err := FindForgeBinary()
		if err != nil {
			return "", err
		}
		te.ForgeBinary = binary
	}

	// Create test environment using forge CLI
	result := RunCommand(te.T, te.ForgeBinary, "test", "create-env", stage)
	if result.Err != nil {
		return "", result.Err
	}

	// Extract test ID from output
	testID := ExtractTestID(result.Stdout + result.Stderr)
	if testID == "" {
		return "", result.Err
	}

	// Track for cleanup
	te.testEnvIDs = append(te.testEnvIDs, testID)

	return testID, nil
}

// RegisterCleanup adds a cleanup function to be called during test teardown.
// Cleanup functions are called in LIFO order (last registered, first executed).
func (te *TestEnvironment) RegisterCleanup(fn func() error) {
	te.T.Helper()
	te.CleanupFuncs = append(te.CleanupFuncs, fn)
}

// SkipCleanup returns true if the SKIP_CLEANUP environment variable is set.
// This is useful for debugging tests by leaving resources intact.
func (te *TestEnvironment) SkipCleanup() bool {
	return os.Getenv("SKIP_CLEANUP") != ""
}

// Cleanup runs all registered cleanup functions in LIFO order.
// It respects the SKIP_CLEANUP environment variable for debugging.
func (te *TestEnvironment) Cleanup() {
	if te.SkipCleanup() {
		// Log that we're skipping cleanup for debugging
		return
	}

	// Clean up test environments (LIFO order)
	for i := len(te.testEnvIDs) - 1; i >= 0; i-- {
		testID := te.testEnvIDs[i]
		_ = ForceCleanupTestEnv(testID)
	}

	// Run custom cleanup functions (LIFO order)
	for i := len(te.CleanupFuncs) - 1; i >= 0; i-- {
		fn := te.CleanupFuncs[i]
		_ = fn() // Ignore errors during cleanup
	}
}
