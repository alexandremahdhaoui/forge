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

// Package runnerexec is the execution half of the generic run engine. It is
// shared by forge itself (which runs it in process, so exit codes and stdio
// pass through untouched) and by cmd/generic-runner (the standalone engine a
// forge:// URI resolves to).
package runnerexec

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type Spec struct {
	Command  string                 `json:"command,omitempty"`
	Args     []string               `json:"args,omitempty"`
	Env      map[string]string      `json:"env,omitempty"`
	Artifact string                 `json:"artifact"`
	Extra    []string               `json:"extra,omitempty"`
	Spec     map[string]interface{} `json:"spec,omitempty"`
}

// Run executes the spec attached to the caller's stdio and returns the exit
// code. A spec command wraps the artifact - the interpreted-language case -
// and the artifact path reaches it via the FORGE_ARTIFACT environment
// variable. No command means the artifact itself is the executable.
func Run(spec Spec) (int, error) {
	command, args := commandLine(spec)
	if command == "" {
		return 0, errors.New("nothing to execute: the spec names no command and no artifact")
	}

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = environment(spec)

	if dir, ok := spec.Spec["context"].(string); ok {
		cmd.Dir = dir
	}

	err := cmd.Run()
	if err == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}

	return 0, fmt.Errorf("running %s: %w", command, err)
}

func commandLine(spec Spec) (string, []string) {
	command := spec.Command
	if c, ok := spec.Spec["command"].(string); ok && c != "" {
		command = c
	}

	args := append([]string{}, spec.Args...)

	if raw, ok := spec.Spec["args"].([]interface{}); ok {
		for _, a := range raw {
			if s, ok := a.(string); ok {
				args = append(args, s)
			}
		}
	}

	if command == "" {
		return spec.Artifact, append(args, spec.Extra...)
	}

	return command, append(args, spec.Extra...)
}

func environment(spec Spec) []string {
	env := os.Environ()
	env = append(env, "FORGE_ARTIFACT="+spec.Artifact)

	for k, v := range spec.Env {
		env = append(env, k+"="+v)
	}

	if raw, ok := spec.Spec["env"].(map[string]interface{}); ok {
		for k, v := range raw {
			if s, ok := v.(string); ok {
				env = append(env, k+"="+s)
			}
		}
	}

	return env
}
