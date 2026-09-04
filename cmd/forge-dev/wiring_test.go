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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
	"github.com/stretchr/testify/require"
)

const smallWiring = `binary: songe-hello-node
ports:
  GreetingStore:
    default: sqlite
`

func writeWiringCell(t *testing.T, dir string) {
	t.Helper()

	writeKindFixture(t, dir, customKindYaml()+"wiring:\n  specPath: ./wiring.yaml\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wiring.yaml"), []byte(smallWiring), 0o644))
}

func TestTheWiringTextTravelsToTheGeneratorAsWiringSpec(t *testing.T) {
	dir := t.TempDir()
	writeWiringCell(t, dir)

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.Equal(t, smallWiring, stub.model.WiringSpec)
}

func TestACellWithNoWiringSendsNoWiringSpecKeyOnTheWire(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml())

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	_, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	wire, err := json.Marshal(stub.model)
	require.NoError(t, err)

	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(wire, &keys))
	require.NotContains(t, keys, "wiringSpec")
}

func TestTheWiringTextIsPartOfTheSourceChecksum(t *testing.T) {
	dir := t.TempDir()
	writeWiringCell(t, dir)

	stub := &stubGeneratorCaller{answer: generatedRustFile()}
	withStubGenerator(t, stub)

	first, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "wiring.yaml"), []byte(smallWiring+"drivers: {}\n"), 0o644))

	second, err := generate(context.Background(), mcptypes.BuildInput{Name: "fixture-gui", Src: dir, Engine: "forge://forge-dev"})
	require.NoError(t, err)

	require.NotEqual(t, first.Version, second.Version)
}

func TestConfigValidateWarnsWhenTheWiringFileIsNotThereYet(t *testing.T) {
	dir := t.TempDir()
	writeKindFixture(t, dir, customKindYaml()+"wiring:\n  specPath: ./wiring.yaml\n")

	output := validateForgeDevConfig(mcptypes.ConfigValidateInput{Spec: map[string]interface{}{"configPath": dir}})

	require.True(t, output.Valid, "%+v", output.Errors)

	var fields []string
	for _, w := range output.Warnings {
		fields = append(fields, w.Field)
	}
	require.Contains(t, fields, "wiring.specPath")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "wiring.yaml"), []byte(smallWiring), 0o644))

	output = validateForgeDevConfig(mcptypes.ConfigValidateInput{Spec: map[string]interface{}{"configPath": dir}})

	require.True(t, output.Valid, "%+v", output.Errors)
	require.Empty(t, output.Warnings)
}
