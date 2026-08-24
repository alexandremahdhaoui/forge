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
	"fmt"
	"go/format"
)

// GeneratedCLIFile is the cli kind's one generated file: the dispatcher and
// the main. The author writes the handlers next to it.
const GeneratedCLIFile = "zz_generated.cli.go"

// CLITemplateData feeds the cli template.
type CLITemplateData struct {
	Name         string
	Version      string
	PackageName  string
	HandlersFunc string
	Checksum     string
	Commands     []CommandConfig
}

// GenerateCLIFile renders the cli kind's dispatcher.
func GenerateCLIFile(config *Config, checksum string) ([]byte, error) {
	tmpl, err := parseTemplate("cli.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing cli template: %w", err)
	}

	handlersFunc := config.Generate.HandlersFunc
	if handlersFunc == "" {
		handlersFunc = "NewCLIHandlers"
	}

	data := CLITemplateData{
		Name:         config.Name,
		Version:      config.Version,
		PackageName:  config.Generate.PackageName,
		HandlersFunc: handlersFunc,
		Checksum:     checksum,
		Commands:     config.commands(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing cli template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), fmt.Errorf("formatting generated cli code: %w", err)
	}

	return formatted, nil
}
