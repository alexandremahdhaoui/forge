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

	if err := os.Remove(config.Kindenv.KubeconfigPath); err != nil {
		return err // TODO: wrap error
	}

	return nil
}
