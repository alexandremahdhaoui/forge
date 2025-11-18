package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/alexandremahdhaoui/forge/internal/util"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
	"github.com/caarlos0/env/v11"
)

// ----------------------------------------------------- CONFIG ----------------------------------------------------- //

// Envs holds the environment variables required by the kindenv tool.
type Envs struct {
	// KindBinary is the path to the kind binary.
	KindBinary string `env:"KIND_BINARY,required"`
	// KindBinaryPrefix is a prefix to add to the kind binary command (e.g., sudo).
	KindBinaryPrefix string `env:"KIND_BINARY_PREFIX"`

	// TODO: make use of the below variables.
	ContainerRegistryBaseURL string `env:"CONTAINER_REGISTRY_BASE_URL"`
	ContainerEngineBinary    string `env:"CONTAINER_ENGINE_BINARY"`
	HelmBinary               string `env:"HELM_BINARY"`
}

// readEnvs reads the environment variables required by the kindenv tool.
func readEnvs() (Envs, error) {
	out := Envs{} //nolint:exhaustruct // unmarshal

	if err := env.Parse(&out); err != nil {
		return Envs{}, err // TODO: wrap err
	}

	return out, nil
}

// ----------------------------------------------------- SETUP ------------------------------------------------------ //

func doSetup(pCfg forge.Spec, envs Envs) error {
	// 1. Allow prefixing kind binary with "sudo".
	cmdName := envs.KindBinary
	args := []string{
		"create",
		"cluster",
		"--name", pCfg.Name,
		"--kubeconfig", pCfg.Kindenv.KubeconfigPath,
		"--wait", "5m",
	}

	if envs.KindBinaryPrefix != "" {
		cmdName = envs.KindBinaryPrefix
		args = append([]string{envs.KindBinary}, args...)
	}

	// 2. kind create cluster and wait.
	cmd := exec.Command(cmdName, args...)

	if err := util.RunCmdWithStdPipes(cmd); err != nil {
		return err // TODO: wrap error
	}

	// 3. chown kubeconfig
	if envs.KindBinaryPrefix == "sudo" { // TODO: Make this a bit more robust (e.g. use which or something)
		chownCmd := exec.Command(
			envs.KindBinaryPrefix,
			"chown",
			fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
			pCfg.Kindenv.KubeconfigPath,
		)

		if err := util.RunCmdWithStdPipes(chownCmd); err != nil {
			return err // TODO: wrap err
		}
	}

	// 3. TODO: setup communication towards local-registry.

	// 4. TODO: setup communication towards any provided registry (e.g. required if users wants to install some apps into their kind cluster). It can be any OCI registry. (to support helm chart)

	// 5. TODO: setup communication CONTAINER_ENGINE login & HELM login.

	return nil
}
