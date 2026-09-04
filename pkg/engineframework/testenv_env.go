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

package engineframework

import (
	"slices"
	"strings"

	"github.com/alexandremahdhaoui/forge/pkg/mcptypes"
)

func BuildTestEnv(input mcptypes.RunInput, specEnv map[string]string) map[string]string {
	env := propagatedTestenvEnv(input)

	if input.TestenvTmpDir != "" {
		env["FORGE_TESTENV_TMPDIR"] = input.TestenvTmpDir
	}

	for key, relPath := range input.ArtifactFiles {
		env["FORGE_ARTIFACT_"+NormalizeEnvKey(key)] = artifactPath(input.TestenvTmpDir, relPath)
	}

	for key, value := range input.TestenvMetadata {
		env["FORGE_METADATA_"+NormalizeEnvKey(key)] = value
	}

	for key, value := range input.Env {
		env[key] = value
	}

	for key, value := range specEnv {
		env[key] = value
	}

	return env
}

func propagatedTestenvEnv(input mcptypes.RunInput) map[string]string {
	env := make(map[string]string, len(input.TestenvEnv))
	policy := input.EnvPropagation

	if policy == nil {
		for key, value := range input.TestenvEnv {
			env[key] = value
		}

		return env
	}

	if policy.Disabled {
		return env
	}

	if len(policy.Whitelist) > 0 {
		for _, key := range policy.Whitelist {
			if value, ok := input.TestenvEnv[key]; ok {
				env[key] = value
			}
		}

		return env
	}

	for key, value := range input.TestenvEnv {
		if slices.Contains(policy.Blacklist, key) {
			continue
		}

		env[key] = value
	}

	return env
}

func artifactPath(testenvTmpDir, relPath string) string {
	if testenvTmpDir == "" {
		return relPath
	}

	return testenvTmpDir + "/" + relPath
}

func NormalizeEnvKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))

	for i := 0; i < len(key); i++ {
		b.WriteByte(normalizeEnvKeyByte(key[i]))
	}

	return b.String()
}

func normalizeEnvKeyByte(c byte) byte {
	switch {
	case c >= 'a' && c <= 'z':
		return c - 'a' + 'A'
	case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return c
	default:
		return '_'
	}
}
