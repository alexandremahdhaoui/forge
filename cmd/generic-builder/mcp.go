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
	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

// runMCPServer starts the generic-builder MCP server with stdio transport.
func runMCPServer() error {
	server := mcpserver.New(Name, Version)

	// Configure builder with engineframework
	config := engineframework.BuilderConfig{
		Name:      Name,
		Version:   Version,
		BuildFunc: build,
	}

	// Register builder tools (registers both 'build' and 'buildBatch')
	if err := engineframework.RegisterBuilderTools(server, config); err != nil {
		return err
	}

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	// Run the MCP server
	return server.RunDefault()
}

// build is the core business logic for executing a shell command as a build step.
// It implements engineframework.BuilderFunc.
func build(ctx context.Context, input mcptypes.BuildInput) (*forge.Artifact, error) {
	log.Printf("Executing command: %s %v (workDir: %s)", input.Command, input.Args, input.WorkDir)

	// Validate required fields
	if input.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Process templated arguments
	processedArgs, err := processTemplatedArgs(input.Args, input)
	if err != nil {
		return nil, fmt.Errorf("template processing failed: %w", err)
	}

	// Convert BuildInput to ExecuteInput
	execInput := ExecuteInput{
		Command: input.Command,
		Args:    processedArgs,
		Env:     input.Env,
		EnvFile: input.EnvFile,
		WorkDir: input.WorkDir,
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
	location := input.WorkDir
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
		Version:   fmt.Sprintf("%s-exit%d", input.Command, output.ExitCode),
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
