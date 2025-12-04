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

// ImageSource defines a container image to push to the local registry.
type ImageSource struct {
	Name      string     `json:"name"`                // Image reference (local://name:tag or registry/path:tag)
	BasicAuth *BasicAuth `json:"basicAuth,omitempty"` // Optional credentials for private images
}

// BasicAuth provides credentials for pulling private images from remote registries.
type BasicAuth struct {
	Username ValueFrom `json:"username"` // Username credential
	Password ValueFrom `json:"password"` // Password credential
}

// ValueFrom specifies how to obtain a credential value.
type ValueFrom struct {
	EnvName string `json:"envName,omitempty"` // Get from environment variable
	Literal string `json:"literal,omitempty"` // Direct literal value
}
