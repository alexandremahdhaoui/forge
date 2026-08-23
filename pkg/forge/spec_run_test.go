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

package forge

import (
	"strings"
	"testing"
)

func TestRunSpecValidate(t *testing.T) {
	t.Parallel()

	valid := RunSpec{
		Name:    "my-tool",
		Src:     "./cmd/my-tool",
		Factory: "git@github.com:x/some-factory.git",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("a minimal runnable with a factory must validate: %v", err)
	}

	withEngine := valid
	withEngine.Engine = "forge://generic-runner"
	if err := withEngine.Validate(); err != nil {
		t.Fatalf("an engine URI must validate: %v", err)
	}

	noFactory := valid
	noFactory.Factory = ""
	err := noFactory.Validate()
	if err == nil {
		t.Fatal("a runnable without a factory must fail validation")
	}
	if !strings.Contains(err.Error(), "factory") {
		t.Fatalf("the error must name the factory key, got: %v", err)
	}

	badEngine := valid
	badEngine.Engine = "not-a-uri"
	if err := badEngine.Validate(); err == nil {
		t.Fatal("an engine without a scheme must fail validation")
	}
}

func TestSpecValidatesItsRunTargets(t *testing.T) {
	t.Parallel()

	s := Spec{
		Name:              "p",
		ArtifactStorePath: "a",
		EnvFile:           ".envrc",
		Run:               []RunSpec{{Name: "broken", Src: "./cmd/broken"}},
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("a spec carrying an invalid runnable must fail")
	}
	if !strings.Contains(err.Error(), "run[0] (broken)") {
		t.Fatalf("the error must locate the runnable, got: %v", err)
	}
}
