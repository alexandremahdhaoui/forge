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
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// GeneratedRESTFile is the rest-api kind's one generated file: the typed
// handler set, the mux, and the main. The author writes the handlers next
// to it.
const GeneratedRESTFile = "zz_generated.rest.go"

// RESTOperation is one operation of the OpenAPI paths. The rest-api kind's
// surface is exactly this list: the spec declares it, nothing else does.
type RESTOperation struct {
	// Method is the HTTP method, upper case.
	Method string
	// Path is the OpenAPI path. Its {param} segments are Go 1.22 mux
	// patterns verbatim, so the route and the spec cannot drift.
	Path string
	// GoName is the operationId in Go form: the handler field name.
	GoName string
}

// RESTTemplateData feeds the rest template.
type RESTTemplateData struct {
	Name         string
	Version      string
	PackageName  string
	HandlersFunc string
	EnvPrefix    string
	Checksum     string
	Operations   []RESTOperation
}

// restOperations derives the operation list from the spec's paths, sorted
// by path then method so generation is deterministic. An operation with no
// operationId is an error: the id names the handler, so an anonymous
// operation has nothing to dispatch to.
func restOperations(spec *openapi3.T) ([]RESTOperation, error) {
	if spec.Paths == nil || spec.Paths.Len() == 0 {
		return nil, fmt.Errorf("reading the OpenAPI paths: a rest-api engine's surface is its paths, and there are none")
	}

	ops := []RESTOperation{}

	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			if op.OperationID == "" {
				return nil, fmt.Errorf("reading %s %s: operationId is required, it names the handler", method, path)
			}

			ops = append(ops, RESTOperation{
				Method: strings.ToUpper(method),
				Path:   path,
				GoName: toTitle(toCamelCase(op.OperationID)),
			})
		}
	}

	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}

		return ops[i].Method < ops[j].Method
	})

	return ops, nil
}

// GenerateRESTFile renders the rest-api kind's server skeleton.
func GenerateRESTFile(config *Config, spec *openapi3.T, checksum string) ([]byte, error) {
	operations, err := restOperations(spec)
	if err != nil {
		return nil, err
	}

	tmpl, err := parseTemplate("rest.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing rest template: %w", err)
	}

	handlersFunc := config.Generate.HandlersFunc
	if handlersFunc == "" {
		handlersFunc = "NewRESTHandlers"
	}

	data := RESTTemplateData{
		Name:         config.Name,
		Version:      config.Version,
		PackageName:  config.Generate.PackageName,
		HandlersFunc: handlersFunc,
		EnvPrefix:    envPrefix(config.Name),
		Checksum:     checksum,
		Operations:   operations,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing rest template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.Bytes(), fmt.Errorf("formatting generated rest code: %w", err)
	}

	return formatted, nil
}

// envPrefix turns rest-demo into REST_DEMO.
func envPrefix(name string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(name))
}
