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
	"context"
	"testing"

	"github.com/alexandremahdhaoui/forge/pkg/engineframework"
)

func TestDeleteSucceedsWhenTheStubEnvironmentWasNeverCreated(t *testing.T) {
	err := deleteStubEnv(context.Background(), engineframework.DeleteInput{
		TestID:   "test-integration-00000000-deadbeef",
		Metadata: map[string]string{},
	})
	if err != nil {
		t.Fatalf("expected delete to succeed on a stub environment that is not there, got %v", err)
	}
}
