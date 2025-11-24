package testenvutil

import (
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// MergeEnv merges environment variables from a sub-engine into accumulated environment.
//
// Priority resolution logic:
//   - If envPropagation.Disabled=true, returns accumulated without merging newEnv
//   - Applies whitelist filtering: only merge vars in whitelist (if non-empty)
//   - Applies blacklist filtering: exclude vars in blacklist (if non-empty)
//   - For each env var: calculate effective priority using CalculateEffectivePriority
//   - Lower priority number = higher priority (wins conflict)
//   - When priorities equal, later sub-engine wins (higher subengineIndex)
//
// IMPORTANT: This function is stateless and treats all accumulated values as having
// default priority (65536). For proper priority tracking across multiple merges,
// the caller must maintain priority state separately.
//
// Parameters:
//   - accumulated: Environment accumulated from previous sub-engines
//   - newEnv: Environment from current sub-engine
//   - envPropagation: Optional propagation configuration (nil = default behavior)
//   - subengineIndex: Index of current sub-engine (used as tiebreaker)
//
// Returns:
//   - Merged environment map (new map, does not modify input maps)
func MergeEnv(accumulated map[string]string, newEnv map[string]string, envPropagation *forge.EnvPropagation, subengineIndex int) map[string]string {
	// Create result map - start with copy of accumulated
	result := make(map[string]string)
	for k, v := range accumulated {
		result[k] = v
	}

	// If propagation is disabled, return accumulated only
	if envPropagation != nil && envPropagation.Disabled {
		return result
	}

	// Get whitelist and blacklist from envPropagation
	var whitelist []string
	var blacklist []string
	if envPropagation != nil {
		whitelist = envPropagation.Whitelist
		blacklist = envPropagation.Blacklist
	}

	// Merge newEnv into result with filtering
	for envKey, newValue := range newEnv {
		// Apply whitelist filtering
		if len(whitelist) > 0 {
			if !contains(whitelist, envKey) {
				continue // Skip: not in whitelist
			}
		}

		// Apply blacklist filtering
		if len(blacklist) > 0 {
			if contains(blacklist, envKey) {
				continue // Skip: in blacklist
			}
		}

		// Add/override in result
		// Note: Since we don't track priorities in the map itself,
		// we use simple override semantics: later wins
		// For proper priority tracking, caller should use EnvSourceTracker
		result[envKey] = newValue
	}

	return result
}

// EnvSource tracks the source and priority of an environment variable value.
// This enables proper priority-based conflict resolution when merging environments.
//
// Fields:
//   - Value: The environment variable value
//   - Priority: Effective priority (lower number = higher priority, 0=highest, 65536=default)
//   - SubengineIndex: Sub-engine index in the chain (used as tiebreaker when priorities equal)
type EnvSource struct {
	Value          string
	Priority       int
	SubengineIndex int
}

// EnvSourceTracker tracks environment variable sources with their priorities.
// Use this for proper priority-based merging across multiple sub-engines.
type EnvSourceTracker struct {
	sources map[string]EnvSource
}

// NewEnvSourceTracker creates a new environment source tracker.
func NewEnvSourceTracker() *EnvSourceTracker {
	return &EnvSourceTracker{
		sources: make(map[string]EnvSource),
	}
}

// Merge merges newEnv into the tracker with priority-based conflict resolution.
func (t *EnvSourceTracker) Merge(newEnv map[string]string, envPropagation *forge.EnvPropagation, subengineIndex int) {
	// If propagation is disabled, skip merge
	if envPropagation != nil && envPropagation.Disabled {
		return
	}

	// Get whitelist and blacklist from envPropagation
	var whitelist []string
	var blacklist []string
	if envPropagation != nil {
		whitelist = envPropagation.Whitelist
		blacklist = envPropagation.Blacklist
	}

	// Merge newEnv with filtering and priority resolution
	for envKey, newValue := range newEnv {
		// Apply whitelist filtering
		if len(whitelist) > 0 {
			if !contains(whitelist, envKey) {
				continue // Skip: not in whitelist
			}
		}

		// Apply blacklist filtering
		if len(blacklist) > 0 {
			if contains(blacklist, envKey) {
				continue // Skip: in blacklist
			}
		}

		// Calculate effective priority for new value
		newPriority := CalculateEffectivePriority(envKey, envPropagation)

		// Check if env var already exists
		if existing, exists := t.sources[envKey]; exists {
			// Conflict resolution: compare priorities
			// Lower priority number = higher priority
			if newPriority < existing.Priority {
				// New value has higher priority - replace
				t.sources[envKey] = EnvSource{
					Value:          newValue,
					Priority:       newPriority,
					SubengineIndex: subengineIndex,
				}
			} else if newPriority == existing.Priority {
				// Same priority - later sub-engine wins (higher subengineIndex)
				if subengineIndex > existing.SubengineIndex {
					t.sources[envKey] = EnvSource{
						Value:          newValue,
						Priority:       newPriority,
						SubengineIndex: subengineIndex,
					}
				}
				// Otherwise keep existing (earlier or same subengine)
			}
			// Otherwise: existing has higher priority - keep existing
		} else {
			// New env var - add it
			t.sources[envKey] = EnvSource{
				Value:          newValue,
				Priority:       newPriority,
				SubengineIndex: subengineIndex,
			}
		}
	}
}

// ToMap returns the current environment as a simple map.
func (t *EnvSourceTracker) ToMap() map[string]string {
	result := make(map[string]string)
	for k, source := range t.sources {
		result[k] = source.Value
	}
	return result
}

// CalculateEffectivePriority calculates the effective priority for an environment variable.
//
// Priority resolution order:
//  1. Per-env override priority (envPropagation.Envs[envKey].Priority)
//  2. Default priority (envPropagation.Priority via GetEffectivePriority())
//  3. System default (65536) if envPropagation is nil
//
// CRITICAL: Must handle nil pointers correctly:
//   - nil envPropagation: return 65536
//   - nil Priority: return 65536 via GetEffectivePriority()
//   - Explicit 0: return 0 (highest priority, NOT converted to default)
//
// Parameters:
//   - envKey: The environment variable name
//   - envPropagation: Optional propagation configuration
//
// Returns:
//   - Effective priority (0=highest, 65536=default, 99999=lowest)
func CalculateEffectivePriority(envKey string, envPropagation *forge.EnvPropagation) int {
	// If no envPropagation, return default
	if envPropagation == nil {
		return 65536
	}

	// Check for per-env override
	if override, exists := envPropagation.Envs[envKey]; exists {
		if override.Priority != nil {
			return *override.Priority // Return explicit priority (including 0)
		}
	}

	// Return default priority via GetEffectivePriority()
	return envPropagation.GetEffectivePriority()
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
