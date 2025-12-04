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
	"fmt"
)

// EnvPropagation configures how environment variables are propagated between testenv sub-engines.
//
// Environment variables can be propagated from one sub-engine to the next in the chain,
// with configurable filtering and priority to resolve conflicts.
//
// Fields:
//   - Disabled: When true, this sub-engine does not propagate any environment variables
//   - Priority: Controls resolution when multiple sub-engines set the same variable
//   - nil (default): Uses priority 65536 (middle priority)
//   - 0: Highest priority (overrides all others)
//   - 99999: Lowest priority (can be overridden by others)
//   - Lower numbers = higher priority in conflict resolution
//   - Whitelist: If non-empty, only these environment variables are propagated
//   - Blacklist: If non-empty, these environment variables are excluded from propagation
//   - Envs: Per-environment-variable overrides (e.g., specific priority for PATH)
//
// Validation rules:
//   - Priority must be in range [0, 99999] when non-nil
//   - Whitelist and Blacklist are mutually exclusive (cannot both be non-empty)
//   - Per-env Priority overrides must also be in valid range [0, 99999]
//
// Example:
//
//	envProp := &EnvPropagation{
//	    Priority: intPtr(100),  // High priority
//	    Whitelist: []string{"KUBECONFIG", "TESTENV_LCR_FQDN"},
//	    Envs: map[string]EnvPropagationOverride{
//	        "KUBECONFIG": {Priority: intPtr(0)}, // Highest priority for KUBECONFIG
//	    },
//	}
type EnvPropagation struct {
	// Disabled when true prevents any environment variable propagation
	Disabled bool `json:"disabled"`

	// Priority controls conflict resolution (nil=65536, 0=highest, 99999=lowest)
	// CRITICAL: Must be *int to distinguish nil (default) from explicit 0
	Priority *int `json:"priority,omitempty"`

	// Whitelist specifies which env vars to propagate (empty=disabled, mutually exclusive with Blacklist)
	Whitelist []string `json:"whitelist,omitempty"`

	// Blacklist specifies which env vars to exclude (empty=disabled, mutually exclusive with Whitelist)
	Blacklist []string `json:"blacklist,omitempty"`

	// Envs provides per-environment-variable configuration overrides
	Envs map[string]EnvPropagationOverride `json:"envs,omitempty"`
}

// EnvPropagationOverride provides per-environment-variable configuration.
//
// This allows fine-grained control over specific environment variables.
// For example, KUBECONFIG might need higher priority than other variables.
//
// Fields:
//   - Priority: Overrides the default Priority for this specific environment variable
//   - nil: Use the default Priority from EnvPropagation
//   - 0-99999: Override with this specific priority value
type EnvPropagationOverride struct {
	// Priority overrides the default priority for this specific env var
	Priority *int `json:"priority,omitempty"`
}

// GetEffectivePriority returns the effective priority value.
//
// This method handles the critical nil vs 0 distinction:
//   - nil Priority: Returns 65536 (default middle priority)
//   - Explicit 0: Returns 0 (highest priority, overrides all)
//   - Any other value: Returns that value as-is
//
// This ensures that:
//   - Unspecified priority (nil) gets reasonable default
//   - Explicit 0 is respected as highest priority (not converted to default)
//
// Returns:
//   - 65536 if Priority is nil (default)
//   - *Priority value if Priority is non-nil (including 0)
func (ep *EnvPropagation) GetEffectivePriority() int {
	if ep.Priority == nil {
		return 65536 // Default priority
	}
	return *ep.Priority
}

// Validate validates the EnvPropagation configuration.
//
// Validation checks:
//   - Priority is in valid range [0, 99999] when non-nil
//   - Whitelist and Blacklist are not both non-empty (mutually exclusive)
//   - All per-env Priority overrides are in valid range [0, 99999] when non-nil
//
// Returns:
//   - nil if configuration is valid
//   - ValidationErrors if any validation rules are violated
func (ep *EnvPropagation) Validate() error {
	errs := NewValidationErrors()

	// Validate priority range
	if ep.Priority != nil {
		if *ep.Priority < 0 || *ep.Priority > 99999 {
			errs.Add(fmt.Errorf("EnvPropagation.priority must be in range [0, 99999], got %d", *ep.Priority))
		}
	}

	// Validate whitelist and blacklist are mutually exclusive
	hasWhitelist := len(ep.Whitelist) > 0
	hasBlacklist := len(ep.Blacklist) > 0
	if hasWhitelist && hasBlacklist {
		errs.Add(fmt.Errorf("EnvPropagation cannot specify both whitelist and blacklist"))
	}

	// Validate per-env priority overrides
	for envName, override := range ep.Envs {
		if override.Priority != nil {
			if *override.Priority < 0 || *override.Priority > 99999 {
				errs.Add(fmt.Errorf("EnvPropagation.envs[%s].priority must be in range [0, 99999], got %d", envName, *override.Priority))
			}
		}
	}

	return errs.ErrorOrNil()
}
