//go:build unit

package testenvutil

import (
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// Helper to create int pointer
func intPtr(v int) *int {
	return &v
}

func TestCalculateEffectivePriority(t *testing.T) {
	tests := []struct {
		name             string
		envKey           string
		envPropagation   *forge.EnvPropagation
		expectedPriority int
	}{
		{
			name:             "nil envPropagation returns default",
			envKey:           "VAR1",
			envPropagation:   nil,
			expectedPriority: 65536,
		},
		{
			name:   "nil priority returns default via GetEffectivePriority",
			envKey: "VAR1",
			envPropagation: &forge.EnvPropagation{
				Priority: nil,
			},
			expectedPriority: 65536,
		},
		{
			name:   "explicit priority 0 returns 0",
			envKey: "VAR1",
			envPropagation: &forge.EnvPropagation{
				Priority: intPtr(0),
			},
			expectedPriority: 0,
		},
		{
			name:   "explicit priority 100 returns 100",
			envKey: "VAR1",
			envPropagation: &forge.EnvPropagation{
				Priority: intPtr(100),
			},
			expectedPriority: 100,
		},
		{
			name:   "per-env override takes precedence over default",
			envKey: "VAR1",
			envPropagation: &forge.EnvPropagation{
				Priority: intPtr(100),
				Envs: map[string]forge.EnvPropagationOverride{
					"VAR1": {Priority: intPtr(50)},
				},
			},
			expectedPriority: 50,
		},
		{
			name:   "per-env override priority 0 takes precedence",
			envKey: "VAR1",
			envPropagation: &forge.EnvPropagation{
				Priority: intPtr(100),
				Envs: map[string]forge.EnvPropagationOverride{
					"VAR1": {Priority: intPtr(0)},
				},
			},
			expectedPriority: 0,
		},
		{
			name:   "per-env with nil priority uses default",
			envKey: "VAR1",
			envPropagation: &forge.EnvPropagation{
				Priority: intPtr(100),
				Envs: map[string]forge.EnvPropagationOverride{
					"VAR1": {Priority: nil},
				},
			},
			expectedPriority: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateEffectivePriority(tt.envKey, tt.envPropagation)
			if result != tt.expectedPriority {
				t.Errorf("CalculateEffectivePriority() = %d, want %d", result, tt.expectedPriority)
			}
		})
	}
}

func TestMergeEnv_Disabled(t *testing.T) {
	accumulated := map[string]string{"VAR1": "value1"}
	newEnv := map[string]string{"VAR2": "value2"}
	envProp := &forge.EnvPropagation{Disabled: true}

	result := MergeEnv(accumulated, newEnv, envProp, 0)

	if len(result) != 1 {
		t.Errorf("Expected 1 var, got %d", len(result))
	}
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", result["VAR1"])
	}
	if _, exists := result["VAR2"]; exists {
		t.Errorf("VAR2 should not be merged when propagation is disabled")
	}
}

func TestMergeEnv_Whitelist(t *testing.T) {
	accumulated := map[string]string{}
	newEnv := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
	}
	envProp := &forge.EnvPropagation{
		Whitelist: []string{"VAR1", "VAR3"},
	}

	result := MergeEnv(accumulated, newEnv, envProp, 0)

	if len(result) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(result))
	}
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", result["VAR1"])
	}
	if result["VAR3"] != "value3" {
		t.Errorf("Expected VAR3=value3, got %s", result["VAR3"])
	}
	if _, exists := result["VAR2"]; exists {
		t.Errorf("VAR2 should be filtered by whitelist")
	}
}

func TestMergeEnv_Blacklist(t *testing.T) {
	accumulated := map[string]string{}
	newEnv := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
	}
	envProp := &forge.EnvPropagation{
		Blacklist: []string{"VAR2"},
	}

	result := MergeEnv(accumulated, newEnv, envProp, 0)

	if len(result) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(result))
	}
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", result["VAR1"])
	}
	if result["VAR3"] != "value3" {
		t.Errorf("Expected VAR3=value3, got %s", result["VAR3"])
	}
	if _, exists := result["VAR2"]; exists {
		t.Errorf("VAR2 should be filtered by blacklist")
	}
}

func TestMergeEnv_SimpleOverride(t *testing.T) {
	accumulated := map[string]string{"VAR1": "old"}
	newEnv := map[string]string{"VAR1": "new"}
	envProp := (*forge.EnvPropagation)(nil)

	result := MergeEnv(accumulated, newEnv, envProp, 1)

	if result["VAR1"] != "new" {
		t.Errorf("Expected VAR1=new (override), got %s", result["VAR1"])
	}
}

func TestEnvSourceTracker_PriorityResolution(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// First sub-engine: priority 100
	tracker.Merge(
		map[string]string{"KUBECONFIG": "/path1"},
		&forge.EnvPropagation{Priority: intPtr(100)},
		0,
	)

	// Second sub-engine: priority 0 (highest)
	tracker.Merge(
		map[string]string{"KUBECONFIG": "/path2"},
		&forge.EnvPropagation{Priority: intPtr(0)},
		1,
	)

	result := tracker.ToMap()
	if result["KUBECONFIG"] != "/path2" {
		t.Errorf("Expected KUBECONFIG=/path2 (priority 0 wins), got %s", result["KUBECONFIG"])
	}
}

func TestEnvSourceTracker_PriorityResolution_LowerWins(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// First sub-engine: priority 0 (highest)
	tracker.Merge(
		map[string]string{"KUBECONFIG": "/path1"},
		&forge.EnvPropagation{Priority: intPtr(0)},
		0,
	)

	// Second sub-engine: priority 100
	tracker.Merge(
		map[string]string{"KUBECONFIG": "/path2"},
		&forge.EnvPropagation{Priority: intPtr(100)},
		1,
	)

	result := tracker.ToMap()
	if result["KUBECONFIG"] != "/path1" {
		t.Errorf("Expected KUBECONFIG=/path1 (priority 0 wins over 100), got %s", result["KUBECONFIG"])
	}
}

func TestEnvSourceTracker_SamePriority_LaterWins(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// First sub-engine: priority 50, index 0
	tracker.Merge(
		map[string]string{"VAR1": "value1"},
		&forge.EnvPropagation{Priority: intPtr(50)},
		0,
	)

	// Second sub-engine: priority 50, index 1
	tracker.Merge(
		map[string]string{"VAR1": "value2"},
		&forge.EnvPropagation{Priority: intPtr(50)},
		1,
	)

	result := tracker.ToMap()
	if result["VAR1"] != "value2" {
		t.Errorf("Expected VAR1=value2 (later sub-engine wins), got %s", result["VAR1"])
	}
}

func TestEnvSourceTracker_SamePriority_EarlierWinsWhenSameIndex(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// First sub-engine: priority 50, index 0
	tracker.Merge(
		map[string]string{"VAR1": "value1"},
		&forge.EnvPropagation{Priority: intPtr(50)},
		0,
	)

	// Same sub-engine (index 0): priority 50
	tracker.Merge(
		map[string]string{"VAR1": "value2"},
		&forge.EnvPropagation{Priority: intPtr(50)},
		0,
	)

	result := tracker.ToMap()
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1 (earlier value kept when same index), got %s", result["VAR1"])
	}
}

func TestEnvSourceTracker_PerEnvPriorityOverride(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// First sub-engine: default priority 100, but KUBECONFIG has priority 0
	tracker.Merge(
		map[string]string{
			"KUBECONFIG":   "/path1",
			"REGISTRY_URL": "registry1",
		},
		&forge.EnvPropagation{
			Priority: intPtr(100),
			Envs: map[string]forge.EnvPropagationOverride{
				"KUBECONFIG": {Priority: intPtr(0)}, // Override to highest priority
			},
		},
		0,
	)

	// Second sub-engine: priority 50 for all
	tracker.Merge(
		map[string]string{
			"KUBECONFIG":   "/path2",
			"REGISTRY_URL": "registry2",
		},
		&forge.EnvPropagation{Priority: intPtr(50)},
		1,
	)

	result := tracker.ToMap()

	// KUBECONFIG should keep /path1 (priority 0 beats priority 50)
	if result["KUBECONFIG"] != "/path1" {
		t.Errorf("Expected KUBECONFIG=/path1 (priority 0 wins), got %s", result["KUBECONFIG"])
	}

	// REGISTRY_URL should use registry2 (priority 50 beats priority 100)
	if result["REGISTRY_URL"] != "registry2" {
		t.Errorf("Expected REGISTRY_URL=registry2 (priority 50 wins), got %s", result["REGISTRY_URL"])
	}
}

func TestEnvSourceTracker_Whitelist(t *testing.T) {
	tracker := NewEnvSourceTracker()

	tracker.Merge(
		map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		},
		&forge.EnvPropagation{
			Whitelist: []string{"VAR1", "VAR3"},
		},
		0,
	)

	result := tracker.ToMap()
	if len(result) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(result))
	}
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", result["VAR1"])
	}
	if result["VAR3"] != "value3" {
		t.Errorf("Expected VAR3=value3, got %s", result["VAR3"])
	}
	if _, exists := result["VAR2"]; exists {
		t.Errorf("VAR2 should be filtered by whitelist")
	}
}

func TestEnvSourceTracker_Blacklist(t *testing.T) {
	tracker := NewEnvSourceTracker()

	tracker.Merge(
		map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		},
		&forge.EnvPropagation{
			Blacklist: []string{"VAR2"},
		},
		0,
	)

	result := tracker.ToMap()
	if len(result) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(result))
	}
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", result["VAR1"])
	}
	if result["VAR3"] != "value3" {
		t.Errorf("Expected VAR3=value3, got %s", result["VAR3"])
	}
	if _, exists := result["VAR2"]; exists {
		t.Errorf("VAR2 should be filtered by blacklist")
	}
}

func TestEnvSourceTracker_DisabledPropagation(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// First merge
	tracker.Merge(
		map[string]string{"VAR1": "value1"},
		nil,
		0,
	)

	// Second merge with disabled propagation
	tracker.Merge(
		map[string]string{"VAR2": "value2"},
		&forge.EnvPropagation{Disabled: true},
		1,
	)

	result := tracker.ToMap()
	if len(result) != 1 {
		t.Errorf("Expected 1 var, got %d", len(result))
	}
	if result["VAR1"] != "value1" {
		t.Errorf("Expected VAR1=value1, got %s", result["VAR1"])
	}
	if _, exists := result["VAR2"]; exists {
		t.Errorf("VAR2 should not be merged when propagation is disabled")
	}
}

func TestEnvSourceTracker_ComplexScenario(t *testing.T) {
	tracker := NewEnvSourceTracker()

	// Sub-engine 0: testenv-kind exports KUBECONFIG with priority 0
	tracker.Merge(
		map[string]string{"KUBECONFIG": "/tmp/kubeconfig"},
		&forge.EnvPropagation{Priority: intPtr(0)},
		0,
	)

	// Sub-engine 1: testenv-lcr exports REGISTRY_URL with default priority
	tracker.Merge(
		map[string]string{"REGISTRY_URL": "localhost:5000"},
		nil,
		1,
	)

	// Sub-engine 2: testenv-helm-install tries to override KUBECONFIG (should fail)
	// and adds HELM_VERSION
	tracker.Merge(
		map[string]string{
			"KUBECONFIG":   "/tmp/other-kubeconfig",
			"HELM_VERSION": "v3.12.0",
		},
		&forge.EnvPropagation{Priority: intPtr(100)},
		2,
	)

	result := tracker.ToMap()

	// KUBECONFIG should remain from sub-engine 0 (priority 0)
	if result["KUBECONFIG"] != "/tmp/kubeconfig" {
		t.Errorf("Expected KUBECONFIG=/tmp/kubeconfig, got %s", result["KUBECONFIG"])
	}

	// REGISTRY_URL should be present
	if result["REGISTRY_URL"] != "localhost:5000" {
		t.Errorf("Expected REGISTRY_URL=localhost:5000, got %s", result["REGISTRY_URL"])
	}

	// HELM_VERSION should be added
	if result["HELM_VERSION"] != "v3.12.0" {
		t.Errorf("Expected HELM_VERSION=v3.12.0, got %s", result["HELM_VERSION"])
	}
}
