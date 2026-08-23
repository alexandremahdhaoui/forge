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

type RunSpec struct {
	Name string `json:"name"`

	Src string `json:"src"`

	Engine string `json:"engine,omitempty"`

	Factory string `json:"factory"`

	FactoryRevision string `json:"factoryRevision,omitempty"`

	Spec map[string]interface{} `json:"spec,omitempty"`
}

func (r *RunSpec) Validate() error {
	errs := NewValidationErrors()

	if err := ValidateRequired(r.Name, "name", "RunSpec"); err != nil {
		errs.Add(err)
	}

	if err := ValidateRequired(r.Src, "src", "RunSpec"); err != nil {
		errs.Add(err)
	}

	if err := ValidateRequired(r.Factory, "factory", "RunSpec"); err != nil {
		errs.Add(err)
	}

	if r.Engine != "" {
		if err := ValidateURI(r.Engine, "RunSpec.engine"); err != nil {
			errs.Add(err)
		}
	}

	return errs.ErrorOrNil()
}
