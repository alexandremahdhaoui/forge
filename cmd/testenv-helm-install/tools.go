package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ToolValidator provides dependency injection for tool validation
type ToolValidator struct {
	checkToolFn        func(name string, args []string) error
	checkHelmVersionFn func(output string) error
	getHelmVersionFn   func() (string, error)
}

// NewToolValidator creates a validator with production dependencies
func NewToolValidator() *ToolValidator {
	return &ToolValidator{
		checkToolFn:        checkTool,
		checkHelmVersionFn: checkHelmVersion,
		getHelmVersionFn:   getHelmVersion,
	}
}

// validateTools checks that all required tools are available and meet version requirements
// nolint:unused // Reserved for future engine initialization
func validateTools() error {
	v := NewToolValidator()
	return v.ValidateTools()
}

// ValidateTools checks that all required tools are available and meet version requirements
func (v *ToolValidator) ValidateTools() error {
	var missing []string

	// Check helm version >= 3.8.0 (required for OCI support)
	if err := v.checkToolFn("helm", []string{"version", "--short"}); err != nil {
		missing = append(missing, "helm (>=3.8.0)")
	} else {
		// Get full version output to parse version
		out, err := v.getHelmVersionFn()
		if err != nil {
			missing = append(missing, "helm (>=3.8.0)")
		} else if err := v.checkHelmVersionFn(out); err != nil {
			return fmt.Errorf("helm version check failed: %w", err)
		}
	}

	// Check git is available
	if err := v.checkToolFn("git", []string{"--version"}); err != nil {
		missing = append(missing, "git")
	}

	// Check kubectl is available
	if err := v.checkToolFn("kubectl", []string{"version", "--client"}); err != nil {
		missing = append(missing, "kubectl")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}

// getHelmVersion retrieves the helm version output
func getHelmVersion() (string, error) {
	cmd := exec.Command("helm", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// checkTool verifies a tool is available by running it with the specified args
func checkTool(name string, args []string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s not available: %w", name, err)
	}
	return nil
}

// checkHelmVersion parses helm version output and verifies it's >= 3.8.0
func checkHelmVersion(versionOutput string) error {
	// Parse version from output like: version.BuildInfo{Version:"v3.10.1", ...}
	// or from --short output like: v3.10.1+g18e6ce3
	re := regexp.MustCompile(`[vV]ersion[:"]+v?(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionOutput)

	if len(matches) < 4 {
		return fmt.Errorf("unable to parse helm version from: %s", versionOutput)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("invalid major version %q: %w", matches[1], err)
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return fmt.Errorf("invalid minor version %q: %w", matches[2], err)
	}

	// Check >= 3.8.0
	if major < 3 || (major == 3 && minor < 8) {
		return fmt.Errorf("helm version %d.%d.x is too old, requires >= 3.8.0 for OCI support", major, minor)
	}

	return nil
}
