package forge

import "fmt"

// Build holds the list of artifacts to build
type Build []BuildSpec

// BuildSpec represents a single artifact to build
type BuildSpec struct {
	// Name of the artifact to build
	Name string `json:"name"`
	// Path to the source, e.g.:
	// - ./cmd/<NAME>
	// - ./containers/<NAME>/Containerfile
	Src string `json:"src"`
	// The destination of the artifact, e.g.:
	// - "./build/bin/<NAME>"
	// - can be left empty for container images
	Dest string `json:"dest,omitempty"`
	// Engine that will build this artifact, e.g.:
	// - go://container-build (go://github.com/alexandremahdhaoui/forge/cmd/container-build)
	// - go://go-build        (go://github.com/alexandremahdhaoui/forge/cmd/go-build)
	Engine string `json:"engine"`
	// Spec contains engine-specific configuration (free-form)
	// Supports fields like: command, args, env, envFile, workDir
	// For container-build engine, also supports:
	//   - dependsOn: []DependsOnSpec - list of dependency detectors to run
	// The exact fields supported depend on the engine being used
	Spec map[string]interface{} `json:"spec,omitempty"`
}

// DependsOnSpec defines a dependency detector configuration
type DependsOnSpec struct {
	// Engine is the URI of the dependency-detector engine (e.g., "go://go-dependency-detector")
	Engine string `json:"engine"`

	// Spec contains engine-specific configuration for the dependency detector
	Spec map[string]interface{} `json:"spec,omitempty"`
}

// Validate validates the BuildSpec
func (bs *BuildSpec) Validate() error {
	errs := NewValidationErrors()

	// Validate required fields
	if err := ValidateRequired(bs.Name, "name", "BuildSpec"); err != nil {
		errs.Add(err)
	}
	if err := ValidateRequired(bs.Src, "src", "BuildSpec"); err != nil {
		errs.Add(err)
	}

	// Validate engine URI
	if err := ValidateURI(bs.Engine, "BuildSpec.engine"); err != nil {
		errs.Add(err)
	}

	return errs.ErrorOrNil()
}

// ParseDependsOn extracts and validates dependsOn field from BuildSpec.Spec
// Returns nil slice if dependsOn not present (no error)
// Returns error if dependsOn present but invalid structure
func ParseDependsOn(spec map[string]interface{}) ([]DependsOnSpec, error) {
	dependsOnRaw, ok := spec["dependsOn"]
	if !ok {
		return nil, nil // not present, not an error
	}

	dependsOnSlice, ok := dependsOnRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("dependsOn must be an array, got %T", dependsOnRaw)
	}

	result := make([]DependsOnSpec, 0, len(dependsOnSlice))
	for i, item := range dependsOnSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("dependsOn[%d] must be an object, got %T", i, item)
		}

		engine, ok := itemMap["engine"].(string)
		if !ok || engine == "" {
			return nil, fmt.Errorf("dependsOn[%d].engine is required and must be a string", i)
		}

		specMap, _ := itemMap["spec"].(map[string]interface{})

		result = append(result, DependsOnSpec{
			Engine: engine,
			Spec:   specMap,
		})
	}

	return result, nil
}
