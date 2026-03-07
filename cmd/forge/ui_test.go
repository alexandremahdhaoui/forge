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
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge/internal/cmdutil"
)

func TestResolveUIBinary_UserOverride(t *testing.T) {
	cfg := cmdutil.UserConfig{
		Tools: cmdutil.ToolsConfig{
			UI: "/usr/local/bin/my-custom-ui",
		},
	}

	binary, err := resolveUIBinary(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if binary != "/usr/local/bin/my-custom-ui" {
		t.Errorf("expected /usr/local/bin/my-custom-ui, got: %s", binary)
	}
}

func TestResolveUIBinary_NotFound(t *testing.T) {
	cfg := cmdutil.UserConfig{}

	_, err := resolveUIBinary(cfg)
	if err == nil {
		t.Fatal("expected error when binary not found, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "forge-ui-tui") {
		t.Errorf("error message should mention forge-ui-tui, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "config.yaml") {
		t.Errorf("error message should mention config.yaml, got: %s", errMsg)
	}
}
