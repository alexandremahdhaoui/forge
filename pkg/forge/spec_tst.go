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

package forge

import "time"

type TestSpec struct {
	Name string `json:"name"`

	Testenv string `json:"testenv,omitempty"`

	Runner string `json:"runner"`

	Spec map[string]interface{} `json:"spec,omitempty"`

	EnvPropagation *EnvPropagation `json:"envPropagation,omitempty"`

	Manual bool `json:"manual,omitempty"`

	Needs []string `json:"needs,omitempty"`
}

func (ts *TestSpec) Validate() error {
	errs := NewValidationErrors()

	if err := ValidateRequired(ts.Name, "name", "TestSpec"); err != nil {
		errs.Add(err)
	}

	for _, need := range ts.Needs {
		if err := ValidateRequired(need, "needs", "TestSpec"); err != nil {
			errs.Add(err)
		}
	}

	if err := ValidateURI(ts.Runner, "TestSpec.runner"); err != nil {
		errs.Add(err)
	}

	if ts.Testenv != "" && ts.Testenv != "noop" {
		if err := ValidateURI(ts.Testenv, "TestSpec.testenv"); err != nil {
			errs.Add(err)
		}
	}

	return errs.ErrorOrNil()
}

type TestEnvironment struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Status string `json:"status"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`

	TmpDir string `json:"tmpDir,omitempty"`

	Files map[string]string `json:"files,omitempty"`

	ManagedResources []string `json:"managedResources"`

	Metadata map[string]string `json:"metadata,omitempty"`

	Env map[string]string `json:"env,omitempty"`

	Subengines []string `json:"subengines"`
}

const (
	TestStatusCreated          = "created"
	TestStatusRunning          = "running"
	TestStatusPassed           = "passed"
	TestStatusFailed           = "failed"
	TestStatusPartiallyDeleted = "partially_deleted"
)
