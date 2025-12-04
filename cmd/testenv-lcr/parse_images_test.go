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
)

func TestParseImagesFromSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    map[string]any
		wantLen int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no images key",
			spec:    map[string]any{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "nil spec",
			spec:    nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "valid local image",
			spec: map[string]any{
				"images": []any{
					map[string]any{"name": "local://myapp:v1"},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "valid remote image with auth",
			spec: map[string]any{
				"images": []any{
					map[string]any{
						"name": "quay.io/example/img:v1",
						"basicAuth": map[string]any{
							"username": map[string]any{"literal": "user"},
							"password": map[string]any{"envName": "PASS"},
						},
					},
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "invalid - empty name",
			spec: map[string]any{
				"images": []any{
					map[string]any{"name": ""},
				},
			},
			wantErr: true,
			errMsg:  "name must not be empty",
		},
		{
			name: "invalid - duplicate images",
			spec: map[string]any{
				"images": []any{
					map[string]any{"name": "local://myapp:v1"},
					map[string]any{"name": "local://myapp:v1"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := parseImagesFromSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if len(images) != tt.wantLen {
					t.Errorf("expected %d images, got %d", tt.wantLen, len(images))
				}
			}
		})
	}
}
