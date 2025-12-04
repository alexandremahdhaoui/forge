//go:build unit

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

package engineframework

import (
	"strings"
	"testing"
	"time"
)

func TestGetGitVersion(t *testing.T) {
	// This test runs in the forge repository, so it should succeed
	version, err := GetGitVersion()
	if err != nil {
		t.Logf("Warning: GetGitVersion() failed (this may be expected in some environments): %v", err)
		// Don't fail the test - git may not be available in all test environments
		if version != "unknown" {
			t.Errorf("GetGitVersion() error case returned version = %q, want %q", version, "unknown")
		}
		return
	}

	// Version should be a non-empty string (commit SHA)
	if version == "" {
		t.Error("GetGitVersion() returned empty version")
	}

	// Version should be hex string (git commit SHA)
	if len(version) < 7 || len(version) > 40 {
		t.Errorf("GetGitVersion() returned version with unexpected length: %d (expected 7-40 characters)", len(version))
	}

	// Should not be "unknown"
	if version == "unknown" {
		t.Error("GetGitVersion() returned 'unknown' despite no error")
	}

	t.Logf("GetGitVersion() returned: %s", version)
}

func TestCreateVersionedArtifact(t *testing.T) {
	// Record time before call
	beforeTime := time.Now().UTC()

	artifact, err := CreateVersionedArtifact("my-app", "binary", "./build/bin/my-app")

	// Record time after call
	afterTime := time.Now().UTC()

	// If git is not available, this may fail - that's okay for this test
	if err != nil {
		t.Logf("CreateVersionedArtifact() failed (git may not be available): %v", err)
		return
	}

	// Verify artifact fields
	if artifact.Name != "my-app" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "my-app")
	}

	if artifact.Type != "binary" {
		t.Errorf("artifact.Type = %q, want %q", artifact.Type, "binary")
	}

	if artifact.Location != "./build/bin/my-app" {
		t.Errorf("artifact.Location = %q, want %q", artifact.Location, "./build/bin/my-app")
	}

	// Version should be set and non-empty
	if artifact.Version == "" {
		t.Error("artifact.Version is empty, expected git commit SHA")
	}

	if artifact.Version == "unknown" {
		t.Error("artifact.Version = 'unknown', expected git commit SHA")
	}

	// Timestamp should be set and valid RFC3339
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	} else {
		parsedTime, err := time.Parse(time.RFC3339, artifact.Timestamp)
		if err != nil {
			t.Errorf("artifact.Timestamp %q is not valid RFC3339: %v", artifact.Timestamp, err)
		}

		// Timestamp should be reasonably close to now (within 5 seconds)
		if parsedTime.Before(beforeTime.Add(-5*time.Second)) || parsedTime.After(afterTime.Add(5*time.Second)) {
			t.Errorf("artifact.Timestamp %v is not reasonably close to call time (before: %v, after: %v)", parsedTime, beforeTime, afterTime)
		}
	}

	t.Logf("CreateVersionedArtifact() created artifact with version: %s, timestamp: %s", artifact.Version, artifact.Timestamp)
}

func TestCreateArtifact(t *testing.T) {
	// Record time before call
	beforeTime := time.Now().UTC()

	artifact := CreateArtifact("openapi-client", "generated", "./pkg/generated")

	// Record time after call
	afterTime := time.Now().UTC()

	// Verify artifact fields
	if artifact.Name != "openapi-client" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "openapi-client")
	}

	if artifact.Type != "generated" {
		t.Errorf("artifact.Type = %q, want %q", artifact.Type, "generated")
	}

	if artifact.Location != "./pkg/generated" {
		t.Errorf("artifact.Location = %q, want %q", artifact.Location, "./pkg/generated")
	}

	// Version should be EMPTY (generated artifacts don't have versions)
	if artifact.Version != "" {
		t.Errorf("artifact.Version = %q, want empty string (generated code has no version)", artifact.Version)
	}

	// Timestamp should be set and valid RFC3339
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	} else {
		parsedTime, err := time.Parse(time.RFC3339, artifact.Timestamp)
		if err != nil {
			t.Errorf("artifact.Timestamp %q is not valid RFC3339: %v", artifact.Timestamp, err)
		}

		// Timestamp should be reasonably close to now (within 5 seconds)
		if parsedTime.Before(beforeTime.Add(-5*time.Second)) || parsedTime.After(afterTime.Add(5*time.Second)) {
			t.Errorf("artifact.Timestamp %v is not reasonably close to call time (before: %v, after: %v)", parsedTime, beforeTime, afterTime)
		}
	}

	t.Logf("CreateArtifact() created artifact with timestamp: %s", artifact.Timestamp)
}

func TestCreateCustomArtifact(t *testing.T) {
	// Record time before call
	beforeTime := time.Now().UTC()

	artifact := CreateCustomArtifact("my-app", "container", "localhost:5000/my-app:v1.2.3", "v1.2.3")

	// Record time after call
	afterTime := time.Now().UTC()

	// Verify artifact fields
	if artifact.Name != "my-app" {
		t.Errorf("artifact.Name = %q, want %q", artifact.Name, "my-app")
	}

	if artifact.Type != "container" {
		t.Errorf("artifact.Type = %q, want %q", artifact.Type, "container")
	}

	if artifact.Location != "localhost:5000/my-app:v1.2.3" {
		t.Errorf("artifact.Location = %q, want %q", artifact.Location, "localhost:5000/my-app:v1.2.3")
	}

	// Version should be the custom version we provided
	if artifact.Version != "v1.2.3" {
		t.Errorf("artifact.Version = %q, want %q", artifact.Version, "v1.2.3")
	}

	// Timestamp should be set and valid RFC3339
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	} else {
		parsedTime, err := time.Parse(time.RFC3339, artifact.Timestamp)
		if err != nil {
			t.Errorf("artifact.Timestamp %q is not valid RFC3339: %v", artifact.Timestamp, err)
		}

		// Timestamp should be reasonably close to now (within 5 seconds)
		if parsedTime.Before(beforeTime.Add(-5*time.Second)) || parsedTime.After(afterTime.Add(5*time.Second)) {
			t.Errorf("artifact.Timestamp %v is not reasonably close to call time (before: %v, after: %v)", parsedTime, beforeTime, afterTime)
		}
	}

	t.Logf("CreateCustomArtifact() created artifact with version: %s, timestamp: %s", artifact.Version, artifact.Timestamp)
}

func TestCreateVersionedArtifact_AllFields(t *testing.T) {
	// Test that all artifact fields are correctly populated
	artifact, err := CreateVersionedArtifact("test-binary", "binary", "/path/to/binary")
	if err != nil {
		t.Skipf("Skipping test (git not available): %v", err)
	}

	// Ensure no unexpected fields are set (should only have Name, Type, Location, Version, Timestamp)
	// Artifact struct has exactly these fields - verify they're all non-empty (except custom fields)
	if artifact.Name == "" {
		t.Error("artifact.Name is empty")
	}
	if artifact.Type == "" {
		t.Error("artifact.Type is empty")
	}
	if artifact.Location == "" {
		t.Error("artifact.Location is empty")
	}
	if artifact.Version == "" {
		t.Error("artifact.Version is empty")
	}
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	}
}

func TestCreateArtifact_AllFields(t *testing.T) {
	// Test that all artifact fields are correctly populated
	artifact := CreateArtifact("test-generated", "generated", "/path/to/generated")

	// Ensure correct fields are set
	if artifact.Name == "" {
		t.Error("artifact.Name is empty")
	}
	if artifact.Type == "" {
		t.Error("artifact.Type is empty")
	}
	if artifact.Location == "" {
		t.Error("artifact.Location is empty")
	}
	// Version should be EMPTY for generated artifacts
	if artifact.Version != "" {
		t.Errorf("artifact.Version = %q, want empty", artifact.Version)
	}
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	}
}

func TestCreateCustomArtifact_AllFields(t *testing.T) {
	// Test that all artifact fields are correctly populated
	artifact := CreateCustomArtifact("test-custom", "custom-type", "/path/to/custom", "custom-version")

	// Ensure all fields are set
	if artifact.Name == "" {
		t.Error("artifact.Name is empty")
	}
	if artifact.Type == "" {
		t.Error("artifact.Type is empty")
	}
	if artifact.Location == "" {
		t.Error("artifact.Location is empty")
	}
	if artifact.Version == "" {
		t.Error("artifact.Version is empty")
	}
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty")
	}
}

func TestCreateArtifact_EmptyInputs(t *testing.T) {
	// Test with empty strings (should not panic, just create artifact with empty fields)
	artifact := CreateArtifact("", "", "")

	if artifact.Name != "" {
		t.Errorf("artifact.Name = %q, want empty", artifact.Name)
	}
	if artifact.Type != "" {
		t.Errorf("artifact.Type = %q, want empty", artifact.Type)
	}
	if artifact.Location != "" {
		t.Errorf("artifact.Location = %q, want empty", artifact.Location)
	}
	// Version should still be empty
	if artifact.Version != "" {
		t.Errorf("artifact.Version = %q, want empty", artifact.Version)
	}
	// Timestamp should still be set (even with empty inputs)
	if artifact.Timestamp == "" {
		t.Error("artifact.Timestamp is empty (should always be set)")
	}
}

func TestTimestampFormat(t *testing.T) {
	// Verify all functions use RFC3339 format consistently
	tests := []struct {
		name        string
		getArtifact func() string
	}{
		{
			name: "CreateArtifact",
			getArtifact: func() string {
				return CreateArtifact("test", "type", "loc").Timestamp
			},
		},
		{
			name: "CreateCustomArtifact",
			getArtifact: func() string {
				return CreateCustomArtifact("test", "type", "loc", "v1").Timestamp
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp := tt.getArtifact()

			// Parse as RFC3339
			parsedTime, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				t.Errorf("Timestamp %q is not valid RFC3339: %v", timestamp, err)
			}

			// Verify it's in UTC (should end with 'Z')
			if !strings.HasSuffix(timestamp, "Z") {
				t.Errorf("Timestamp %q is not in UTC (should end with 'Z')", timestamp)
			}

			// Verify it's recent (within last minute)
			now := time.Now().UTC()
			if parsedTime.Before(now.Add(-1*time.Minute)) || parsedTime.After(now.Add(1*time.Minute)) {
				t.Errorf("Timestamp %v is not recent (now: %v)", parsedTime, now)
			}
		})
	}
}

func TestCreateVersionedArtifact_TimestampFormat(t *testing.T) {
	artifact, err := CreateVersionedArtifact("test", "type", "loc")
	if err != nil {
		t.Skipf("Skipping (git not available): %v", err)
	}

	timestamp := artifact.Timestamp

	// Parse as RFC3339
	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Errorf("Timestamp %q is not valid RFC3339: %v", timestamp, err)
	}

	// Verify it's in UTC (should end with 'Z')
	if !strings.HasSuffix(timestamp, "Z") {
		t.Errorf("Timestamp %q is not in UTC (should end with 'Z')", timestamp)
	}

	// Verify it's recent (within last minute)
	now := time.Now().UTC()
	if parsedTime.Before(now.Add(-1*time.Minute)) || parsedTime.After(now.Add(1*time.Minute)) {
		t.Errorf("Timestamp %v is not recent (now: %v)", parsedTime, now)
	}
}
