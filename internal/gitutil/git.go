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

package gitutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetCurrentCommitSHA returns the current Git commit SHA (full 40-character hash).
//
// Returns an error if:
//   - Git command fails to execute
//   - Not in a Git repository
//   - The returned SHA is empty
//
// Example usage:
//
//	sha, err := gitutil.GetCurrentCommitSHA()
//	if err != nil {
//	    return fmt.Errorf("failed to get git version: %w", err)
//	}
func GetCurrentCommitSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty git commit SHA")
	}

	return sha, nil
}
