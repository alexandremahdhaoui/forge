//go:build unit

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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/require"
)

const smallProto = `syntax = "proto3";
package hello.v1;
service Hello { rpc Greet (GreetRequest) returns (GreetReply); }
message GreetRequest { string name = 1; }
message GreetReply { string greeting = 1; }
`

func protoOnlyCellYaml() string {
	return `name: hello-grpc
kind: grpc
generator: forge://example.com/org/codegen/cmd/grpc-rust-tonic
version: 0.1.0
language: rust
proto:
  specPath: ./hello.v1.proto
generate:
  packageName: main
  docsBaseURL: https://example.com/raw
`
}

func writeProtoCell(t *testing.T, dir, forgeDevYaml string) {
	t.Helper()

	createRequiredDocs(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "forge-dev.yaml"), []byte(forgeDevYaml), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.v1.proto"), []byte(smallProto), 0o644))
}

func generatedRustFile() map[string]interface{} {
	return map[string]interface{}{
		"files": []interface{}{
			map[string]interface{}{"path": "zz_generated_grpc.rs", "content": "// grpc\n"},
		},
	}
}

func TestAProtoOnlyCellHandsTheGeneratorTheRawProtoAndLoadsNoOpenAPI(t *testing.T) {
	dir := t.TempDir()
	writeProtoCell(t, dir, protoOnlyCellYaml())

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "hello-grpc", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.Equal(t, smallProto, stub.model.ProtoSpec)
	require.Empty(t, stub.model.OpenAPISpec)
	require.FileExists(t, filepath.Join(dir, "zz_generated_grpc.rs"))
	require.FileExists(t, filepath.Join(dir, GeneratedRunnableFile))
}

func TestACellThatDeclaresBothSpecsHandsTheGeneratorBoth(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml()+"proto:\n  specPath: ./hello.v1.proto\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.v1.proto"), []byte(smallProto), 0o644))

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.Equal(t, smallProto, stub.model.ProtoSpec)
	require.Contains(t, stub.model.OpenAPISpec, "greeting")
}

func TestAnOpenAPICellStillCarriesNoProtoSpec(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.Empty(t, stub.model.ProtoSpec)
	require.Contains(t, stub.model.OpenAPISpec, "greeting")
}

func TestTheProtoTextIsPartOfTheSourceChecksum(t *testing.T) {
	dir := t.TempDir()
	writeProtoCell(t, dir, protoOnlyCellYaml())

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	first, err := generate(context.Background(), mcptypes.BuildInput{Name: "hello-grpc", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.v1.proto"), []byte(smallProto+"// changed\n"), 0o644))

	second, err := generate(context.Background(), mcptypes.BuildInput{Name: "hello-grpc", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.NotEqual(t, first.Version, second.Version)
}

func TestAProtoOnlyCellWithoutAGeneratorFailsValidationNamingTheGenerator(t *testing.T) {
	config := &Config{
		Name:     "hello-grpc",
		Kind:     "grpc",
		Version:  "0.1.0",
		Language: "rust",
		Proto:    ProtoConfig{SpecPath: "./hello.v1.proto"},
		Generate: GenerateConfig{PackageName: "main"},
	}

	errs := ValidateConfig(config)

	var fields []string
	for _, e := range errs {
		fields = append(fields, e.Field)
	}
	require.Contains(t, fields, "generator")
	require.NotContains(t, fields, "openapi.specPath")
}

func TestACellWithNeitherSpecStillFailsOnTheOpenAPIPath(t *testing.T) {
	config := &Config{
		Name:     "plain",
		Kind:     KindBinary,
		Version:  "0.1.0",
		Generate: GenerateConfig{PackageName: "main"},
	}

	errs := ValidateConfig(config)

	var fields []string
	for _, e := range errs {
		fields = append(fields, e.Field)
	}
	require.Contains(t, fields, "openapi.specPath")
}

func TestConfigValidateAcceptsAProtoOnlyCellAndWarnsWhenTheProtoIsNotThereYet(t *testing.T) {
	dir := t.TempDir()
	createRequiredDocs(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "forge-dev.yaml"), []byte(protoOnlyCellYaml()), 0o644))

	output := validateForgeDevConfig(mcptypes.ConfigValidateInput{Spec: map[string]interface{}{"configPath": dir}})

	require.True(t, output.Valid, "%+v", output.Errors)
	require.Len(t, output.Warnings, 1)
	require.Equal(t, "proto.specPath", output.Warnings[0].Field)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.v1.proto"), []byte(smallProto), 0o644))

	output = validateForgeDevConfig(mcptypes.ConfigValidateInput{Spec: map[string]interface{}{"configPath": dir}})

	require.True(t, output.Valid, "%+v", output.Errors)
	require.Empty(t, output.Warnings)
}
