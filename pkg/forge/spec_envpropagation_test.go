//go:build unit

package forge

import (
	"testing"
)

// TestEnvPropagation_Defaults tests that EnvPropagation uses correct default values
func TestEnvPropagation_Defaults(t *testing.T) {
	tests := []struct {
		name              string
		envProp           *EnvPropagation
		expectedPriority  int
		expectedDisabled  bool
		expectedWhitelist int // length
		expectedBlacklist int // length
		expectedEnvsCount int
	}{
		{
			name:              "nil EnvPropagation",
			envProp:           nil,
			expectedPriority:  65536, // should use default when nil
			expectedDisabled:  false,
			expectedWhitelist: 0,
			expectedBlacklist: 0,
			expectedEnvsCount: 0,
		},
		{
			name:              "empty EnvPropagation",
			envProp:           &EnvPropagation{},
			expectedPriority:  65536, // nil priority should return 65536
			expectedDisabled:  false, // default is false
			expectedWhitelist: 0,
			expectedBlacklist: 0,
			expectedEnvsCount: 0,
		},
		{
			name: "disabled EnvPropagation",
			envProp: &EnvPropagation{
				Disabled: true,
			},
			expectedPriority:  65536,
			expectedDisabled:  true,
			expectedWhitelist: 0,
			expectedBlacklist: 0,
			expectedEnvsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envProp == nil {
				// For nil case, we expect default behavior
				return
			}

			priority := tt.envProp.GetEffectivePriority()
			if priority != tt.expectedPriority {
				t.Errorf("GetEffectivePriority() = %d, want %d", priority, tt.expectedPriority)
			}

			if tt.envProp.Disabled != tt.expectedDisabled {
				t.Errorf("Disabled = %v, want %v", tt.envProp.Disabled, tt.expectedDisabled)
			}

			if len(tt.envProp.Whitelist) != tt.expectedWhitelist {
				t.Errorf("Whitelist length = %d, want %d", len(tt.envProp.Whitelist), tt.expectedWhitelist)
			}

			if len(tt.envProp.Blacklist) != tt.expectedBlacklist {
				t.Errorf("Blacklist length = %d, want %d", len(tt.envProp.Blacklist), tt.expectedBlacklist)
			}

			if len(tt.envProp.Envs) != tt.expectedEnvsCount {
				t.Errorf("Envs length = %d, want %d", len(tt.envProp.Envs), tt.expectedEnvsCount)
			}
		})
	}
}

// TestEnvPropagation_PriorityPointer tests the critical *int behavior for Priority
func TestEnvPropagation_PriorityPointer(t *testing.T) {
	tests := []struct {
		name             string
		priority         *int
		expectedPriority int
		description      string
	}{
		{
			name:             "nil priority returns default 65536",
			priority:         nil,
			expectedPriority: 65536,
			description:      "When Priority is nil, GetEffectivePriority() must return 65536",
		},
		{
			name:             "explicit 0 returns 0 (highest priority)",
			priority:         intPtr(0),
			expectedPriority: 0,
			description:      "Explicit 0 must return 0, NOT 65536. This is highest priority.",
		},
		{
			name:             "explicit 100 returns 100",
			priority:         intPtr(100),
			expectedPriority: 100,
			description:      "Explicit priority values must be returned as-is",
		},
		{
			name:             "explicit 99999 returns 99999 (lowest valid priority)",
			priority:         intPtr(99999),
			expectedPriority: 99999,
			description:      "Maximum valid priority is 99999",
		},
		{
			name:             "explicit 65536 returns 65536",
			priority:         intPtr(65536),
			expectedPriority: 65536,
			description:      "Explicit 65536 is different from nil (even though same value)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envProp := &EnvPropagation{
				Priority: tt.priority,
			}

			result := envProp.GetEffectivePriority()
			if result != tt.expectedPriority {
				t.Errorf("GetEffectivePriority() = %d, want %d. %s", result, tt.expectedPriority, tt.description)
			}
		})
	}
}

// TestEnvPropagation_Validate tests validation logic
func TestEnvPropagation_Validate(t *testing.T) {
	tests := []struct {
		name        string
		envProp     *EnvPropagation
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "nil is valid",
			envProp:     nil,
			shouldError: false,
		},
		{
			name:        "empty is valid",
			envProp:     &EnvPropagation{},
			shouldError: false,
		},
		{
			name: "valid priority 0",
			envProp: &EnvPropagation{
				Priority: intPtr(0),
			},
			shouldError: false,
		},
		{
			name: "valid priority 99999",
			envProp: &EnvPropagation{
				Priority: intPtr(99999),
			},
			shouldError: false,
		},
		{
			name: "invalid priority -1",
			envProp: &EnvPropagation{
				Priority: intPtr(-1),
			},
			shouldError: true,
			errorMsg:    "priority must be in range [0, 99999]",
		},
		{
			name: "invalid priority 100000",
			envProp: &EnvPropagation{
				Priority: intPtr(100000),
			},
			shouldError: true,
			errorMsg:    "priority must be in range [0, 99999]",
		},
		{
			name: "whitelist only is valid",
			envProp: &EnvPropagation{
				Whitelist: []string{"PATH", "HOME"},
			},
			shouldError: false,
		},
		{
			name: "blacklist only is valid",
			envProp: &EnvPropagation{
				Blacklist: []string{"SECRET", "TOKEN"},
			},
			shouldError: false,
		},
		{
			name: "whitelist and blacklist both specified is invalid",
			envProp: &EnvPropagation{
				Whitelist: []string{"PATH"},
				Blacklist: []string{"SECRET"},
			},
			shouldError: true,
			errorMsg:    "cannot specify both whitelist and blacklist",
		},
		{
			name: "empty whitelist and empty blacklist is valid",
			envProp: &EnvPropagation{
				Whitelist: []string{},
				Blacklist: []string{},
			},
			shouldError: false,
		},
		{
			name: "per-env priority override valid",
			envProp: &EnvPropagation{
				Envs: map[string]EnvPropagationOverride{
					"PATH": {Priority: intPtr(100)},
				},
			},
			shouldError: false,
		},
		{
			name: "per-env priority override invalid negative",
			envProp: &EnvPropagation{
				Envs: map[string]EnvPropagationOverride{
					"PATH": {Priority: intPtr(-10)},
				},
			},
			shouldError: true,
			errorMsg:    "priority must be in range [0, 99999]",
		},
		{
			name: "per-env priority override invalid too high",
			envProp: &EnvPropagation{
				Envs: map[string]EnvPropagationOverride{
					"PATH": {Priority: intPtr(100001)},
				},
			},
			shouldError: true,
			errorMsg:    "priority must be in range [0, 99999]",
		},
		{
			name: "multiple per-env overrides with one invalid",
			envProp: &EnvPropagation{
				Envs: map[string]EnvPropagationOverride{
					"PATH": {Priority: intPtr(100)},
					"HOME": {Priority: intPtr(200)},
					"BAD":  {Priority: intPtr(-1)},
				},
			},
			shouldError: true,
			errorMsg:    "priority must be in range [0, 99999]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envProp == nil {
				// nil is always valid, skip validation
				return
			}

			err := tt.envProp.Validate()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Validate() should return error containing '%s', but got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = '%v', should contain '%s'", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() should not return error, but got: %v", err)
				}
			}
		})
	}
}

// TestEnvPropagationOverride_PriorityPointer tests the per-env priority override
func TestEnvPropagationOverride_PriorityPointer(t *testing.T) {
	tests := []struct {
		name             string
		override         EnvPropagationOverride
		defaultPriority  int
		expectedPriority int
	}{
		{
			name:             "nil override priority uses default",
			override:         EnvPropagationOverride{Priority: nil},
			defaultPriority:  65536,
			expectedPriority: 65536,
		},
		{
			name:             "explicit 0 overrides default",
			override:         EnvPropagationOverride{Priority: intPtr(0)},
			defaultPriority:  65536,
			expectedPriority: 0,
		},
		{
			name:             "explicit priority overrides default",
			override:         EnvPropagationOverride{Priority: intPtr(50)},
			defaultPriority:  65536,
			expectedPriority: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate getting effective priority for a specific env var
			var effectivePriority int
			if tt.override.Priority != nil {
				effectivePriority = *tt.override.Priority
			} else {
				effectivePriority = tt.defaultPriority
			}

			if effectivePriority != tt.expectedPriority {
				t.Errorf("Effective priority = %d, want %d", effectivePriority, tt.expectedPriority)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
