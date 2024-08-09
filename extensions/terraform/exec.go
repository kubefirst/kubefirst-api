/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package terraform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/rs/zerolog/log"
)

type outputLogger struct {
	prefix  string
	isError bool
}

func (l outputLogger) Write(p []byte) (n int, err error) {
	if l.isError {
		log.Error().Msgf("%s %s", l.prefix, string(p))
	} else {
		log.Info().Msgf("%s %s", l.prefix, string(p))
	}
	return len(p), nil
}

// ExecShellWithVars Exec shell actions supporting:
//   - On-the-fly logging of result
//   - Map of Vars loaded
func ExecShellWithVars(osvars map[string]string, command string, args ...string) error {
	allvars := os.Environ()
	for k, v := range osvars {
		allvars = append(allvars, k+"="+v)
		log.Info().Msgf("adding %s=%q to environment", k, strings.Repeat("*", len(v)))
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = &outputLogger{prefix: command, isError: false}
	cmd.Stderr = &outputLogger{prefix: command, isError: true}
	cmd.Env = allvars

	if err := cmd.Run(); err != nil {
		if exitError := new(exec.ExitError); errors.As(err, &exitError) {
			return fmt.Errorf("command %s failed with exit code %d", command, exitError.ExitCode())
		}

		return fmt.Errorf("command %s failed: %w", command, err)
	}

	return nil
}
