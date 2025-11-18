package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alexandremahdhaoui/forge/internal/cli"
	"github.com/alexandremahdhaoui/forge/internal/util"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// Version information (set via ldflags during build)
var (
	Version        = "dev"
	CommitSHA      = "unknown"
	BuildTimestamp = "unknown"
)

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
	cli.Bootstrap(cli.Config{
		Name:           "go-gen-openapi",
		Version:        Version,
		CommitSHA:      CommitSHA,
		BuildTimestamp: BuildTimestamp,
		RunMCP:         runMCPServer,
	})
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
