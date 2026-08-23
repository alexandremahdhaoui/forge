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

// generic-runner is the run twin of generic-builder: it executes one
// command attached to the caller's stdio and propagates the exit code.
// forge run invokes it as: generic-runner exec '<json>'.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexandremahdhaoui/forge/internal/runnerexec"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "exec" {
		fmt.Fprintln(os.Stderr, "usage: generic-runner exec '<json>'")
		os.Exit(2)
	}

	var spec runnerexec.Spec
	if err := json.Unmarshal([]byte(os.Args[2]), &spec); err != nil {
		fmt.Fprintf(os.Stderr, "generic-runner: decoding the exec spec: %v\n", err)
		os.Exit(1)
	}

	code, err := runnerexec.Run(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generic-runner: %v\n", err)
		os.Exit(1)
	}

	os.Exit(code)
}
