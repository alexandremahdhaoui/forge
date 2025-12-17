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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

// outputFormat represents the desired output format
type outputFormat string

const (
	outputFormatTable outputFormat = "table"
	outputFormatJSON  outputFormat = "json"
	outputFormatYAML  outputFormat = "yaml"
)

// parseOutputFormat extracts the output format flag from args
// Supports: -o json, -ojson, -o yaml, -oyaml, -o table, -otable, --format=<value>
// Returns: format, remaining args
func parseOutputFormat(args []string) (outputFormat, []string) {
	format := outputFormatTable // default
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-o" && i+1 < len(args) {
			switch args[i+1] {
			case "json":
				format = outputFormatJSON
			case "yaml":
				format = outputFormatYAML
			case "table":
				format = outputFormatTable
			default:
				remaining = append(remaining, arg)
				continue
			}
			i++ // skip next arg
		} else if arg == "-ojson" {
			format = outputFormatJSON
		} else if arg == "-oyaml" {
			format = outputFormatYAML
		} else if arg == "-otable" {
			format = outputFormatTable
		} else if strings.HasPrefix(arg, "--format=") {
			value := strings.TrimPrefix(arg, "--format=")
			switch value {
			case "json":
				format = outputFormatJSON
			case "yaml":
				format = outputFormatYAML
			case "table":
				format = outputFormatTable
			}
		} else {
			remaining = append(remaining, arg)
		}
	}

	return format, remaining
}

// printJSON prints a value as formatted JSON to stdout.
func printJSON(v any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

// printYAML prints a value as YAML to stdout.
func printYAML(v any) {
	data, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding YAML: %v\n", err)
		return
	}
	fmt.Print(string(data))
}
