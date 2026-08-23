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
	"github.com/alexandremahdhaoui/forge/internal/engineresolver"
)

// parseEngine parses an engine URI and returns the engine type, command, and args for execution.
// Supports forge:// and alias:// protocols; see engineresolver.ParseEngineURI.
//
func parseEngine(engineURI, forgeVersion string) (engineType string, command string, args []string, err error) {
	return engineresolver.ParseEngineURI(engineURI, forgeVersion)
}
