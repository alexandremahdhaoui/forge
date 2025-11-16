//go:build unit

package main

import (
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func TestExtractOpenAPIConfigFromInput(t *testing.T) {
	tests := []struct {
		name      string
		input     mcptypes.BuildInput
		want      *forge.GenerateOpenAPIConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid sourceFile pattern with client",
			input: mcptypes.BuildInput{
				Name:   "example-api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile":     "./api/example-api.v1.yaml",
					"destinationDir": "./pkg/generated",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "exampleclient",
					},
				},
			},
			want: &forge.GenerateOpenAPIConfig{
				Defaults: forge.GenerateOpenAPIDefaults{
					DestinationDir: "./pkg/generated",
				},
				Specs: []forge.GenerateOpenAPISpec{
					{
						Source:         "./api/example-api.v1.yaml",
						DestinationDir: "./pkg/generated",
						Versions:       []string{}, // Empty - new design has no versions array
						Client: forge.GenOpts{
							Enabled:     true,
							PackageName: "exampleclient",
						},
						Server: forge.GenOpts{
							Enabled: false,
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "valid sourceDir+name+version pattern with server",
			input: mcptypes.BuildInput{
				Name:   "products-api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceDir":      "./api",
					"name":           "products-api",
					"version":        "v1",
					"destinationDir": "./pkg/generated",
					"server": map[string]interface{}{
						"enabled":     true,
						"packageName": "productsserver",
					},
				},
			},
			want: &forge.GenerateOpenAPIConfig{
				Defaults: forge.GenerateOpenAPIDefaults{
					DestinationDir: "./pkg/generated",
				},
				Specs: []forge.GenerateOpenAPISpec{
					{
						Source:         "api/products-api.v1.yaml", // Pre-templated (filepath.Join strips ./)
						DestinationDir: "./pkg/generated",
						Versions:       []string{}, // Empty - new design
						Client: forge.GenOpts{
							Enabled: false,
						},
						Server: forge.GenOpts{
							Enabled:     true,
							PackageName: "productsserver",
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "both client and server enabled",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile":     "./api/api.v1.yaml",
					"destinationDir": "./pkg/generated",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "apiclient",
					},
					"server": map[string]interface{}{
						"enabled":     true,
						"packageName": "apiserver",
					},
				},
			},
			want: &forge.GenerateOpenAPIConfig{
				Defaults: forge.GenerateOpenAPIDefaults{
					DestinationDir: "./pkg/generated",
				},
				Specs: []forge.GenerateOpenAPISpec{
					{
						Source:         "./api/api.v1.yaml",
						DestinationDir: "./pkg/generated",
						Versions:       []string{}, // Empty - new design
						Client: forge.GenOpts{
							Enabled:     true,
							PackageName: "apiclient",
						},
						Server: forge.GenOpts{
							Enabled:     true,
							PackageName: "apiserver",
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "default destinationDir applied",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/api.v1.yaml",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "apiclient",
					},
				},
			},
			want: &forge.GenerateOpenAPIConfig{
				Defaults: forge.GenerateOpenAPIDefaults{
					DestinationDir: "./pkg/generated",
				},
				Specs: []forge.GenerateOpenAPISpec{
					{
						Source:         "./api/api.v1.yaml",
						DestinationDir: "./pkg/generated",
						Versions:       []string{}, // Empty - new design
						Client: forge.GenOpts{
							Enabled:     true,
							PackageName: "apiclient",
						},
						Server: forge.GenOpts{
							Enabled: false,
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "error - missing sourceFile and sourceDir",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "apiclient",
					},
				},
			},
			wantError: true,
			errorMsg:  "must provide either 'sourceFile' or all of 'sourceDir', 'name', and 'version'",
		},
		{
			name: "error - missing name in templated pattern",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceDir": "./api",
					"version":   "v1",
					"client": map[string]interface{}{
						"enabled":     true,
						"packageName": "apiclient",
					},
				},
			},
			wantError: true,
			errorMsg:  "must provide either 'sourceFile' or all of 'sourceDir', 'name', and 'version'",
		},
		{
			name: "error - client enabled but no packageName",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/api.v1.yaml",
					"client": map[string]interface{}{
						"enabled": true,
					},
				},
			},
			wantError: true,
			errorMsg:  "client.packageName is required when client.enabled=true",
		},
		{
			name: "error - server enabled but no packageName",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/api.v1.yaml",
					"server": map[string]interface{}{
						"enabled": true,
					},
				},
			},
			wantError: true,
			errorMsg:  "server.packageName is required when server.enabled=true",
		},
		{
			name: "error - both client and server disabled",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/api.v1.yaml",
					"client": map[string]interface{}{
						"enabled": false,
					},
					"server": map[string]interface{}{
						"enabled": false,
					},
				},
			},
			wantError: true,
			errorMsg:  "at least one of client.enabled or server.enabled must be true",
		},
		{
			name: "error - neither client nor server configured",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/api.v1.yaml",
				},
			},
			wantError: true,
			errorMsg:  "at least one of client.enabled or server.enabled must be true",
		},
		{
			name: "error - invalid enabled type",
			input: mcptypes.BuildInput{
				Name:   "api-v1",
				Engine: "go://go-gen-openapi",
				Spec: map[string]interface{}{
					"sourceFile": "./api/api.v1.yaml",
					"client": map[string]interface{}{
						"enabled":     "yes",
						"packageName": "apiclient",
					},
				},
			},
			wantError: true,
			errorMsg:  "client.enabled must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractOpenAPIConfigFromInput(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("extractOpenAPIConfigFromInput() expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("extractOpenAPIConfigFromInput() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("extractOpenAPIConfigFromInput() unexpected error = %v", err)
				return
			}

			if got == nil {
				t.Fatal("extractOpenAPIConfigFromInput() returned nil config")
			}

			// Compare the result
			if len(got.Specs) != 1 {
				t.Errorf("extractOpenAPIConfigFromInput() specs length = %d, want 1", len(got.Specs))
				return
			}

			gotSpec := got.Specs[0]
			wantSpec := tt.want.Specs[0]

			// Verify Versions array
			if len(gotSpec.Versions) != len(wantSpec.Versions) {
				t.Errorf("Versions length = %d, want %d", len(gotSpec.Versions), len(wantSpec.Versions))
			}
			// CRITICAL: New design should have empty Versions array
			if len(gotSpec.Versions) != 0 {
				t.Errorf("Versions should be empty for new design, got %v", gotSpec.Versions)
			}

			if gotSpec.Source != wantSpec.Source {
				t.Errorf("Source = %v, want %v", gotSpec.Source, wantSpec.Source)
			}
			if gotSpec.SourceDir != wantSpec.SourceDir {
				t.Errorf("SourceDir = %v, want %v", gotSpec.SourceDir, wantSpec.SourceDir)
			}
			if gotSpec.Name != wantSpec.Name {
				t.Errorf("Name = %v, want %v", gotSpec.Name, wantSpec.Name)
			}
			if gotSpec.DestinationDir != wantSpec.DestinationDir {
				t.Errorf("DestinationDir = %v, want %v", gotSpec.DestinationDir, wantSpec.DestinationDir)
			}
			if gotSpec.Client.Enabled != wantSpec.Client.Enabled {
				t.Errorf("Client.Enabled = %v, want %v", gotSpec.Client.Enabled, wantSpec.Client.Enabled)
			}
			if gotSpec.Client.PackageName != wantSpec.Client.PackageName {
				t.Errorf("Client.PackageName = %v, want %v", gotSpec.Client.PackageName, wantSpec.Client.PackageName)
			}
			if gotSpec.Server.Enabled != wantSpec.Server.Enabled {
				t.Errorf("Server.Enabled = %v, want %v", gotSpec.Server.Enabled, wantSpec.Server.Enabled)
			}
			if gotSpec.Server.PackageName != wantSpec.Server.PackageName {
				t.Errorf("Server.PackageName = %v, want %v", gotSpec.Server.PackageName, wantSpec.Server.PackageName)
			}
		})
	}
}
