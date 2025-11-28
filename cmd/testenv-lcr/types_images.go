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
