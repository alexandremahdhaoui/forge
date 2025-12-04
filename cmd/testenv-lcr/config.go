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
	"path/filepath"

	"github.com/alexandremahdhaoui/forge/pkg/eventualconfig"
)

const (
	// TLS

	TLSCACert     = "tls-ca-cert"
	TLSCert       = "tls-cert"
	TLSKey        = "tls-key"
	TLSSecretName = "tls-secret-name"

	// Credential

	CredentialMount      = "credential-mount"
	CredentialSecretName = "credential-secret-name"
)

// Mount represents a file mount with a directory and filename.
type Mount struct {
	// Dir is the directory where the file is mounted.
	Dir string
	// Filename is the name of the mounted file.
	Filename string
}

// Path returns the full path of the mounted file.
func (m Mount) Path() string {
	return filepath.Join(m.Dir, m.Filename)
}

// NewEventualConfig creates a new EventualConfig for the local-container-registry tool.
func NewEventualConfig() eventualconfig.EventualConfig { //nolint:ireturn
	return eventualconfig.NewEventualConfig(
		// TLS
		TLSCACert,
		TLSCert,
		TLSKey,
		TLSSecretName,

		// Credential
		CredentialMount,
		CredentialSecretName,
	)
}
