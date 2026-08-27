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
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// Declaring platforms is what makes an artifact public. A distribution
// build expands those entries once per platform and leaves everything else
// where it is - a repo's own tool never travels by accident. Live case:
// two repos each shipped their own cmd/docgen into one release because the
// dist step globbed cmd/* instead of reading a declaration.
func TestOnlyDeclaredPlatformsTravel(t *testing.T) {
	declared := forge.Build{
		{Name: "forge", Src: "./cmd/forge", Dest: "./build/dist",
			Engine: "forge://go-build", Platforms: []string{"linux/amd64", "linux/arm64"}},
		{Name: "docgen", Src: "./cmd/docgen", Dest: "./build/bin",
			Engine: "forge://go-build"},
	}

	specs := distSpecs(declared, []string{"linux/amd64", "linux/arm64"})

	if len(specs) != 2 {
		t.Fatalf("expected the two platforms of the one public command, got %d: %+v", len(specs), specs)
	}

	for _, spec := range specs {
		if spec.Name == "docgen" || spec.Name == "docgen_linux_amd64" {
			t.Fatalf("a command that declares no platform must never travel: %s", spec.Name)
		}
	}

	if specs[0].Name != "forge_linux_amd64" || specs[1].Name != "forge_linux_arm64" {
		t.Fatalf("a dist artifact travels as name_os_arch, got %s and %s", specs[0].Name, specs[1].Name)
	}

	env, ok := specs[0].Spec["env"].(map[string]any)
	if !ok {
		t.Fatalf("the platform must reach the engine as env, got %+v", specs[0].Spec)
	}

	if env["GOOS"] != "linux" || env["GOARCH"] != "amd64" {
		t.Fatalf("wrong target: %+v", env)
	}
}

// A host build is untouched by the declaration: every entry builds once,
// under its own name, for this machine.
func TestAHostBuildIgnoresPlatforms(t *testing.T) {
	declared := forge.Build{
		{Name: "forge", Engine: "forge://go-build", Platforms: []string{"linux/arm64"}},
		{Name: "docgen", Engine: "forge://go-build"},
	}

	specs := distSpecs(declared, nil)

	if len(specs) != 2 || specs[0].Name != "forge" || specs[1].Name != "docgen" {
		t.Fatalf("a host build builds what is declared, as declared: %+v", specs)
	}
}

// An entry only travels where IT says it can: asking for a platform it
// never declared builds nothing rather than guessing.
func TestAnUndeclaredPlatformIsNotBuilt(t *testing.T) {
	declared := forge.Build{
		{Name: "forge", Engine: "forge://go-build", Platforms: []string{"linux/amd64"}},
	}

	if specs := distSpecs(declared, []string{"darwin/arm64"}); len(specs) != 0 {
		t.Fatalf("expected nothing to build, got %+v", specs)
	}
}

func TestThePlatformsFlagIsParsedEitherWay(t *testing.T) {
	for _, args := range [][]string{
		{"--platforms", "linux/amd64,linux/arm64"},
		{"--platforms=linux/amd64,linux/arm64"},
	} {
		rest, platforms, err := parsePlatformsFlag(args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}

		if len(rest) != 0 {
			t.Fatalf("%v: the flag must not survive as an artifact name: %v", args, rest)
		}

		if len(platforms) != 2 || platforms[0] != "linux/amd64" || platforms[1] != "linux/arm64" {
			t.Fatalf("%v: got %v", args, platforms)
		}
	}

	if _, _, err := parsePlatformsFlag([]string{"--platforms"}); err == nil {
		t.Fatal("a flag with no list must fail loudly")
	}
}
