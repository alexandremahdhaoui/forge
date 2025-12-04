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

func TestNewS3Client(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		region   string
		wantErr  bool
	}{
		{
			name:     "valid endpoint and region",
			endpoint: "http://localhost:9000",
			region:   "us-east-1",
			wantErr:  false,
		},
		{
			name:     "valid https endpoint",
			endpoint: "https://s3.amazonaws.com",
			region:   "us-west-2",
			wantErr:  false,
		},
		{
			name:     "empty region defaults to us-east-1",
			endpoint: "http://localhost:9000",
			region:   "",
			wantErr:  false,
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			region:   "us-east-1",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewS3Client(tt.endpoint, tt.region)

			if tt.wantErr {
				if err == nil {
					t.Error("NewS3Client() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewS3Client() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("NewS3Client() returned nil client")
			}
		})
	}
}

func TestS3Client_ValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid http endpoint",
			endpoint: "http://localhost:9000",
			wantErr:  false,
		},
		{
			name:     "valid https endpoint",
			endpoint: "https://s3.amazonaws.com",
			wantErr:  false,
		},
		{
			name:     "endpoint without scheme",
			endpoint: "localhost:9000",
			wantErr:  true,
			errMsg:   "endpoint must start with http:// or https://",
		},
		{
			name:     "empty endpoint",
			endpoint: "",
			wantErr:  true,
			errMsg:   "endpoint is required",
		},
		{
			name:     "invalid scheme",
			endpoint: "ftp://example.com",
			wantErr:  true,
			errMsg:   "endpoint must start with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateS3Endpoint(tt.endpoint)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateS3Endpoint() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateS3Endpoint() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("validateS3Endpoint() unexpected error: %v", err)
			}
		})
	}
}

func TestS3Client_NormalizeRegion(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		wantRegion string
	}{
		{
			name:       "explicit region",
			region:     "us-west-2",
			wantRegion: "us-west-2",
		},
		{
			name:       "empty region defaults to us-east-1",
			region:     "",
			wantRegion: "us-east-1",
		},
		{
			name:       "whitespace region defaults to us-east-1",
			region:     "   ",
			wantRegion: "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeS3Region(tt.region)
			if got != tt.wantRegion {
				t.Errorf("normalizeS3Region() = %q, want %q", got, tt.wantRegion)
			}
		})
	}
}
