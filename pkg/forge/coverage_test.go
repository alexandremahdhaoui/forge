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

package forge

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCoverage_EnabledDefaultsFalse(t *testing.T) {
	// Go zero value should be false
	var c Coverage
	if c.Enabled != false {
		t.Errorf("expected Enabled to default to false, got %v", c.Enabled)
	}
}

func TestCoverage_JSONSerialization(t *testing.T) {
	// Test with Enabled=true
	c1 := Coverage{Enabled: true, Percentage: 75.5, FilePath: "coverage.out"}
	data, err := json.Marshal(c1)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var c2 Coverage
	if err := json.Unmarshal(data, &c2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if c2.Enabled != true {
		t.Errorf("expected Enabled=true after round-trip, got %v", c2.Enabled)
	}
	if c2.Percentage != 75.5 {
		t.Errorf("expected Percentage=75.5, got %v", c2.Percentage)
	}
}

func TestCoverage_YAMLSerialization(t *testing.T) {
	// Test with Enabled=false (non-coverage runner)
	c1 := Coverage{Enabled: false, Percentage: 0}
	data, err := yaml.Marshal(c1)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var c2 Coverage
	if err := yaml.Unmarshal(data, &c2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if c2.Enabled != false {
		t.Errorf("expected Enabled=false after round-trip, got %v", c2.Enabled)
	}
}
