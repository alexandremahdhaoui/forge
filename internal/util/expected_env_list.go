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

package util

import (
	"fmt"
	"reflect"
	"strings"
)

// ----------------------------------------------------- FormatExpectedEnvList -------------------------------------- //

// FormatExpectedEnvList formats a list of expected environment variables from a struct.
// It uses reflection to read the `env` tag of the struct fields.
// The output is a formatted string with the environment variable name, and whether it is required or optional.
func FormatExpectedEnvList[T any]() string {
	optionalEnvs := make([]string, 0)
	requiredEnvs := make([]string, 0)

	observedMaxStrLen := 0

	rt := reflect.TypeFor[T]()
	for i := range rt.NumField() {
		field := rt.Field(i)
		val, ok := field.Tag.Lookup("env")
		if !ok {
			continue
		}

		substr := strings.Split(val, ",")
		switch len(substr) {
		case 0:
			continue
		case 1:
			optionalEnvs = append(optionalEnvs, substr[0])
		default:
			requiredEnvs = append(requiredEnvs, substr[0])
		}

		if envStrLen := len(substr[0]); envStrLen > observedMaxStrLen {
			observedMaxStrLen = envStrLen
		}
	}

	envs := ""
	for _, s := range requiredEnvs {
		envs = fmt.Sprintf("%s- %s %s[Required]\n", envs, s, fmtSpaces(s, observedMaxStrLen))
	}

	for _, s := range optionalEnvs {
		envs = fmt.Sprintf("%s- %s %s[Optional]\n", envs, s, fmtSpaces(s, observedMaxStrLen))
	}

	return envs
}

func fmtSpaces(s string, maxLen int) string {
	return strings.Repeat(" ", maxLen-len(s))
}
