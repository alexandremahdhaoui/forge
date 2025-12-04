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

package cmdutil

// ExecuteInput contains the parameters for command execution.
type ExecuteInput struct {
	Command string            // Command to execute
	Args    []string          // Command arguments
	Env     map[string]string // Environment variables
	EnvFile string            // Path to environment file (optional)
	WorkDir string            // Working directory (optional)
}

// ExecuteOutput contains the result of command execution.
type ExecuteOutput struct {
	ExitCode int    // Command exit code
	Stdout   string // Standard output
	Stderr   string // Standard error
	Error    string // Error message if execution failed
}
