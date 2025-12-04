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

package util

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// RunCmdWithStdPipes runs a command and pipes its stdout and stderr to the current process's stdout and stderr.
// It waits for the command to complete and returns an error if the command fails or if there is an error copying the output.
func RunCmdWithStdPipes(cmd *exec.Cmd) error {
	errChan := make(chan error, 2) // Buffered channel for 2 goroutines
	var wg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := io.Copy(os.Stdout, stdout); err != nil {
			errChan <- err
		}
	}()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if written, err := io.Copy(os.Stderr, stderr); err != nil {
			errChan <- err

			if written > 0 {
				errChan <- fmt.Errorf("%d bytes written to stderr", written) // TODO: wrap err
			}
		}
	}()

	// Start the command (don't use Run() as it closes pipes before goroutines finish)
	if err := cmd.Start(); err != nil {
		return err
	}

	// Wait for goroutines to finish reading from pipes
	wg.Wait()
	close(errChan)

	// Now wait for command to complete
	if err := cmd.Wait(); err != nil {
		return err
	}

	// Check for errors from goroutines
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
