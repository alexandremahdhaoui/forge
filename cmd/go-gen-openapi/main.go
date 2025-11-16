package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexandremahdhaoui/forge/internal/mcpserver"
	"github.com/alexandremahdhaoui/forge/internal/util"
	"github.com/alexandremahdhaoui/forge/internal/version"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

var versionInfo *version.Info

func init() {
	versionInfo = version.New("go-gen-openapi")
	versionInfo.Version = Version
	versionInfo.CommitSHA = CommitSHA
	versionInfo.BuildTimestamp = BuildTimestamp
}

const (
	sourceFileTemplate  = "%s.%s.yaml"
	zzGeneratedFilename = "zz_generated.oapi-codegen.go"

	clientTemplate = `---
package: %[1]s
output: %[2]s
generate:
  client: true
  models: true
  embedded-spec: true
output-options:
  # to make sure that all types are generated
  skip-prune: true
`

	serverTemplate = `---
package: %[1]s
output: %[2]s
generate:
  embedded-spec: true
  models: true
  std-http-server: true
  strict-server: true
output-options:
  skip-prune: true
`
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--mcp":
			if err := runMCPServer(); err != nil {
				log.Printf("MCP server error: %v", err)
				os.Exit(1)
			}
			return
		case "version", "--version", "-v":
			versionInfo.Print()
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	// Direct invocation is no longer supported
	fmt.Fprintf(os.Stderr, "Error: Direct invocation is no longer supported.\n")
	fmt.Fprintf(os.Stderr, "go-gen-openapi must be invoked through the forge build system.\n")
	fmt.Fprintf(os.Stderr, "See docs/migration-go-gen-openapi.md for migration instructions.\n")
	os.Exit(1)
}

func printUsage() {
	fmt.Println(`go-gen-openapi - Generate OpenAPI client and server code

Usage:
  go-gen-openapi --mcp        Run as MCP server
  go-gen-openapi version      Show version information
  go-gen-openapi help         Show this help message

Environment Variables:
  OAPI_CODEGEN_VERSION            Version of oapi-codegen to use (default: v2.3.0)

Configuration:
  Must be invoked through forge build system.
  See docs/migration-go-gen-openapi.md for configuration details.`)
}

func runMCPServer() error {
	v, _, _ := versionInfo.Get()
	server := mcpserver.New("go-gen-openapi", v)

	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "build",
		Description: "Generate OpenAPI client and server code",
	}, handleBuild)

	return server.RunDefault()
}

func handleBuild(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input mcptypes.BuildInput,
) (*mcp.CallToolResult, any, error) {
	log.Printf("🚀 REFACTORED go-gen-openapi: Generating OpenAPI code for: %s", input.Name)

	// Validate required fields
	if result := mcputil.ValidateRequiredWithPrefix("Build failed", map[string]string{
		"name":   input.Name,
		"engine": input.Engine,
	}); result != nil {
		return result, nil, nil
	}

	// Extract OpenAPI config from BuildInput.Spec
	config, err := extractOpenAPIConfigFromInput(input)
	if err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Build failed: %v", err)), nil, nil
	}

	// Get oapi-codegen version and build executable command
	oapiCodegenVersion := os.Getenv("OAPI_CODEGEN_VERSION")
	if oapiCodegenVersion == "" {
		oapiCodegenVersion = "v2.3.0"
	}

	executable := fmt.Sprintf("go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@%s", oapiCodegenVersion)

	// Call existing generation logic, passing RootDir for relative path resolution
	if err := doGenerate(executable, *config, input.RootDir); err != nil {
		return mcputil.ErrorResult(fmt.Sprintf("Build failed: %v", err)), nil, nil
	}

	// Create artifact with CORRECT values
	artifact := forge.Artifact{
		Name:      input.Name,                            // Use input.Name, NOT hardcoded
		Type:      "generated",                           // Fixed type
		Location:  config.Specs[0].DestinationDir,        // ACTUAL resolved destination directory
		Timestamp: time.Now().UTC().Format(time.RFC3339), // UTC timestamp
		// NO Version field - generated code is versioned by source spec
	}

	result, returnedArtifact := mcputil.SuccessResultWithArtifact(
		fmt.Sprintf("Generated OpenAPI code for: %s", input.Name),
		artifact,
	)
	return result, returnedArtifact, nil
}

func doGenerate(executable string, config forge.GenerateOpenAPIConfig, rootDir string) error {
	cmdName, args := parseExecutable(executable)
	errChan := make(chan error, 100) // Buffered to avoid goroutine leaks
	wg := &sync.WaitGroup{}

	for i := range config.Specs {
		i := i

		// Handle new design: empty Versions array means single BuildSpec per version
		// Source path is already fully resolved in the Spec.Source field
		versions := config.Specs[i].Versions
		if len(versions) == 0 {
			// New design: Source is already resolved, no need to loop over versions
			sourcePath := config.Specs[i].Source

			// Generate client if enabled
			if config.Specs[i].Client.Enabled {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := generatePackage(cmdName, args, config, i, "", config.Specs[i].Client, clientTemplate, sourcePath, rootDir); err != nil {
						errChan <- err
					}
				}()
			}

			// Generate server if enabled
			if config.Specs[i].Server.Enabled {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := generatePackage(cmdName, args, config, i, "", config.Specs[i].Server, serverTemplate, sourcePath, rootDir); err != nil {
						errChan <- err
					}
				}()
			}
		} else {
			// Old design (backward compatibility): loop over versions
			for _, version := range versions {
				version := version

				sourcePath := templateSourcePath(config, i, version)

				// Generate client if enabled
				if config.Specs[i].Client.Enabled {
					wg.Add(1)
					go func() {
						defer wg.Done()
						if err := generatePackage(cmdName, args, config, i, version, config.Specs[i].Client, clientTemplate, sourcePath, rootDir); err != nil {
							errChan <- err
						}
					}()
				}

				// Generate server if enabled
				if config.Specs[i].Server.Enabled {
					wg.Add(1)
					go func() {
						defer wg.Done()
						if err := generatePackage(cmdName, args, config, i, version, config.Specs[i].Server, serverTemplate, sourcePath, rootDir); err != nil {
							errChan <- err
						}
					}()
				}
			}
		}
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect all errors
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("generation failed: %s", strings.Join(errors, "; "))
	}

	fmt.Fprintln(os.Stderr, "✅ Successfully generated OpenAPI code")
	return nil
}

func generatePackage(cmdName string, baseArgs []string, config forge.GenerateOpenAPIConfig, specIndex int, version string, opts forge.GenOpts, template string, sourcePath string, rootDir string) error {
	outputPath := templateOutputPath(config, specIndex, opts.PackageName)
	templatedConfig := fmt.Sprintf(template, opts.PackageName, outputPath)

	path, cleanup, err := writeTempCodegenConfig(templatedConfig)
	if err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	defer cleanup()

	// Create output directory, handling both relative and absolute paths
	// If outputPath is relative and we have a rootDir, resolve it from rootDir
	actualOutputPath := outputPath
	if rootDir != "" && !filepath.IsAbs(outputPath) {
		actualOutputPath = filepath.Join(rootDir, outputPath)
	}

	if err := os.MkdirAll(filepath.Dir(actualOutputPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	args := append(baseArgs, "--config", path, sourcePath)
	cmd := exec.Command(cmdName, args...)

	// Set working directory to rootDir so relative paths work correctly
	// rootDir is where forge.yaml is located, making relative paths in spec work
	if rootDir != "" {
		cmd.Dir = rootDir
	}

	if err := util.RunCmdWithStdPipes(cmd); err != nil {
		return fmt.Errorf("oapi-codegen failed for %s: %w", opts.PackageName, err)
	}

	return nil
}

func parseExecutable(executable string) (string, []string) {
	split := strings.Split(executable, " ")
	return split[0], split[1:]
}

func writeTempCodegenConfig(templatedConfig string) (string, func(), error) {
	tempFile, err := os.CreateTemp("", "oapi-codegen-*.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tempFile.Name())
	}

	if _, err := tempFile.WriteString(templatedConfig); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), cleanup, nil
}

func templateOutputPath(config forge.GenerateOpenAPIConfig, index int, packageName string) string {
	destDir := config.Defaults.DestinationDir
	if config.Specs[index].DestinationDir != "" {
		destDir = config.Specs[index].DestinationDir
	}

	return filepath.Join(destDir, packageName, zzGeneratedFilename)
}

func templateSourcePath(config forge.GenerateOpenAPIConfig, index int, version string) string {
	if source := config.Specs[index].Source; source != "" {
		return source
	}

	sourceFile := fmt.Sprintf(sourceFileTemplate, config.Specs[index].Name, version)

	sourceDir := config.Defaults.SourceDir
	if config.Specs[index].SourceDir != "" {
		sourceDir = config.Specs[index].SourceDir
	}

	return filepath.Join(sourceDir, sourceFile)
}
