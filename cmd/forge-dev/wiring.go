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
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

type WiringConfig struct {
	SpecPath string `yaml:"specPath"`
}

func (c *Config) declaresWiring() bool {
	return c.Wiring.SpecPath != ""
}

func (c *Config) wiringSpecPath(srcDir string) string {
	return filepath.Join(srcDir, c.Wiring.SpecPath)
}

func (c *Config) wiringSpecText(srcDir string) (string, error) {
	if !c.declaresWiring() {
		return "", nil
	}

	return readSpecText(c.wiringSpecPath(srcDir))
}

func warnMissingWiring(config *Config, configPath string, warnings []mcptypes.ValidationWarning) []mcptypes.ValidationWarning {
	if !config.declaresWiring() {
		return warnings
	}

	if _, statErr := os.Stat(config.wiringSpecPath(configPath)); !os.IsNotExist(statErr) {
		return warnings
	}

	return append(warnings, mcptypes.ValidationWarning{
		Field: "wiring.specPath",
		Message: fmt.Sprintf(
			"%s does not exist yet; the build's spec resolution materializes it, so its content is validated at build time",
			config.Wiring.SpecPath),
	})
}
