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

package version

import (
	"fmt"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
)

// Info holds version information for a tool.
type Info struct {
	// ToolName is the name of the tool
	ToolName string
	// Version is set via ldflags or from build info
	Version string
	// CommitSHA is set via ldflags or from build info
	CommitSHA string
	// BuildTimestamp is set via ldflags or from build info
	BuildTimestamp string
}

// Get returns version information, attempting to read from build info if not set via ldflags.
func (i *Info) Get() (version, commit, timestamp string) {
	version = i.Version
	commit = i.CommitSHA
	timestamp = i.BuildTimestamp

	// Try to get build info from Go modules (works with go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		// Use module version if available and we don't have a custom version
		if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}

		// Extract VCS information from build settings (requires Go 1.18+)
		var vcsRevision string
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				vcsRevision = setting.Value
				if commit == "unknown" && len(setting.Value) >= 7 {
					commit = setting.Value[:7] // Short commit hash
				}
			case "vcs.time":
				if timestamp == "unknown" {
					timestamp = setting.Value
				}
			}
		}

		// If version is still "dev" but we have VCS revision, use that as version
		// This handles cases where go run is used but we have git info
		if version == "dev" && vcsRevision != "" {
			if len(vcsRevision) >= 7 {
				version = vcsRevision[:7] // Use short commit hash as version
			} else {
				version = vcsRevision
			}
		}
	}

	// If version is still "dev", try to get it from git directly
	// This handles cases where go run is used without VCS build info
	if version == "dev" {
		if gitVersion := getGitVersion(); gitVersion != "" {
			version = gitVersion
		}
	}

	// If commit is still "unknown", try git
	if commit == "unknown" {
		if gitCommit := getGitCommit(); gitCommit != "" {
			commit = gitCommit
		}
	}

	return version, commit, timestamp
}

// getGitVersion attempts to get the version from git describe
func getGitVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitCommit attempts to get the short commit hash from git
func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Print outputs formatted version information to stdout.
func (i *Info) Print() {
	version, commit, timestamp := i.Get()
	fmt.Printf("%s version %s\n", i.ToolName, version)
	fmt.Printf("  commit:    %s\n", commit)
	fmt.Printf("  built:     %s\n", timestamp)
	fmt.Printf("  go:        %s\n", runtime.Version())
	fmt.Printf("  platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// String returns a one-line version string using the explicitly set Version field.
func (i *Info) String() string {
	return fmt.Sprintf("%s version %s", i.ToolName, i.Version)
}

// New creates a new Info with default values.
func New(toolName string) *Info {
	return &Info{
		ToolName:       toolName,
		Version:        "dev",
		CommitSHA:      "unknown",
		BuildTimestamp: "unknown",
	}
}
