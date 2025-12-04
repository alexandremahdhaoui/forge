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

// LocalContainerRegistry holds the configuration for the local-container-registry tool.
type LocalContainerRegistry struct {
	// Enabled indicates whether the local container registry is enabled.
	Enabled bool `json:"enabled"`
	// CredentialPath is the path to the credentials file for the local container registry.
	CredentialPath string `json:"credentialPath"`
	// CaCrtPath is the path to the CA certificate for the local container registry.
	CaCrtPath string `json:"caCrtPath"`
	// Namespace is the Kubernetes namespace where the local container registry is deployed.
	Namespace string `json:"namespace"`
	// ImagePullSecretNamespaces is a list of namespaces where image pull secrets should be automatically created.
	ImagePullSecretNamespaces []string `json:"imagePullSecretNamespaces,omitempty"`
	// ImagePullSecretName is the name of the image pull secret to create (defaults to "local-container-registry-credentials").
	ImagePullSecretName string `json:"imagePullSecretName,omitempty"`
}
