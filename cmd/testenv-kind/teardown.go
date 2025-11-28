package main

import (
	"os"
	"os/exec"

	"github.com/alexandremahdhaoui/forge/internal/util"
	"github.com/alexandremahdhaoui/forge/pkg/forge"
)

// ----------------------------------------------------- TEARDOWN --------------------------------------------------- //

func doTeardown(config forge.Spec, envs Envs) error {
	cmdName := envs.KindBinary
	args := []string{
		"delete",
		"cluster",
		"--name", config.Name,
	}

	if envs.KindBinaryPrefix != "" {
		cmdName = envs.KindBinaryPrefix
		args = append([]string{envs.KindBinary}, args...)
	}

	cmd := exec.Command(cmdName, args...)

	if err := util.RunCmdWithStdPipes(cmd); err != nil {
		return err // TODO: wrap error
	}

	// Only remove kubeconfig file if path is set
	// Path might be empty if cleanup is called without proper metadata
	if config.Kindenv.KubeconfigPath != "" {
		if err := os.Remove(config.Kindenv.KubeconfigPath); err != nil {
			// Log warning but don't fail - file might already be deleted
			// or cleanup might be called multiple times
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}
