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

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/cmdutil"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the generic-builder MCP server with stdio transport.
func runMCPServer() error {
	server, err := SetupMCPServer(Name, Version, build)
	if err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	// Run the MCP server
	return server.RunDefault()
}

// build is the core business logic for executing a shell command as a build step.
// It implements the BuildFunc signature defined in zz_generated.mcp.go.
func build(ctx context.Context, input mcptypes.BuildInput, spec *Spec) (*forge.Artifact, error) {
	// Use spec values, falling back to input values for compatibility
	command := spec.Command
	if command == "" {
		command = input.Command
	}

	args := spec.Args
	if len(args) == 0 {
		args = input.Args
	}

	env := spec.Env
	if len(env) == 0 {
		env = input.Env
	}

	envFile := spec.EnvFile
	if envFile == "" {
		envFile = input.EnvFile
	}

	workDir := spec.WorkDir
	if workDir == "" {
		workDir = input.WorkDir
	}

	log.Printf("Executing command: %s %v (workDir: %s)", command, args, workDir)

	// Validate required fields
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Process templated arguments
	processedArgs, err := processTemplatedArgs(args, input)
	if err != nil {
		return nil, fmt.Errorf("template processing failed: %w", err)
	}

	// Convert to ExecuteInput
	execInput := ExecuteInput{
		Command: command,
		Args:    processedArgs,
		Env:     env,
		EnvFile: envFile,
		WorkDir: workDir,
	}

	// Execute command
	output := cmdutil.ExecuteCommand(execInput)

	// Check if command failed
	if output.ExitCode != 0 {
		errorMsg := fmt.Sprintf("command failed with exit code %d", output.ExitCode)
		if output.Error != "" {
			errorMsg += fmt.Sprintf(": %s", output.Error)
		}
		if output.Stderr != "" {
			errorMsg += fmt.Sprintf(" (stderr: %s)", output.Stderr)
		}
		return nil, fmt.Errorf("%s", errorMsg)
	}

	// Log output
	if output.Stdout != "" {
		log.Printf("Stdout: %s", output.Stdout)
	}
	if output.Stderr != "" {
		log.Printf("Stderr: %s", output.Stderr)
	}

	// Determine location (use WorkDir if specified, otherwise Src or ".")
	location := workDir
	if location == "" {
		location = input.Src
	}
	if location == "" {
		location = "."
	}

	// Create artifact
	artifact := &forge.Artifact{
		Name:      input.Name,
		Type:      "command-output",
		Location:  location,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   fmt.Sprintf("%s-exit%d", command, output.ExitCode),
	}

	return artifact, nil
}

// processTemplatedArgs processes arguments with Go template syntax.
// Supports: {{ .Name }}, {{ .Src }}, {{ .Dest }}, {{ .Version }}
func processTemplatedArgs(args []string, data mcptypes.BuildInput) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}

	result := make([]string, len(args))
	for i, arg := range args {
		// Parse the template
		tmpl, err := template.New(fmt.Sprintf("arg%d", i)).Parse(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template in arg[%d]: %w", i, err)
		}

		// Execute the template
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute template in arg[%d]: %w", i, err)
		}

		result[i] = buf.String()
	}

	return result, nil
}
