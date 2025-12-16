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
	"github.com/alexandremahdhaoui/forge/pkg/enginecli"
	"github.com/alexandremahdhaoui/forge/pkg/enginedocs"
	"github.com/alexandremahdhaoui/forge/pkg/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Name = "testenv-lcr"

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

// docsConfig is the configuration for the docs subcommand.
var docsConfig = &enginedocs.Config{
	EngineName:   Name,
	LocalDir:     "cmd/testenv-lcr/docs",
	BaseURL:      "https://raw.githubusercontent.com/alexandremahdhaoui/forge/refs/heads/main",
	RequiredDocs: []string{"usage", "schema"},
}

func main() {
	enginecli.Bootstrap(enginecli.Config{
		Name:           Name,
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
		DocsConfig:     docsConfig,
	})
}

func runMCPServer() error {
	server, err := SetupMCPServer(Name, Version, Create, Delete)
	if err != nil {
		return err
	}

	// Register additional tools specific to testenv-lcr
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "create-image-pull-secret",
		Description: "Create an image pull secret in a specific namespace for the local container registry",
	}, handleCreateImagePullSecretTool)

	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "list-image-pull-secrets",
		Description: "List all image pull secrets created by testenv-lcr across all namespaces or in a specific namespace",
	}, handleListImagePullSecretsTool)

	if err := enginedocs.RegisterDocsTools(server, *docsConfig); err != nil {
		return err
	}

	return server.RunDefault()
}
