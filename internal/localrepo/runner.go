package localrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type commandRunner struct{}

func (commandRunner) output(
	ctx context.Context,
	dir string,
	environment []string,
	name string,
	arguments ...string,
) ([]byte, error) {
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = dir
	command.Env = append(os.Environ(), environment...)

	output, err := command.Output()
	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}

		return nil, fmt.Errorf(
			"run %s %s: %w: %s",
			name,
			strings.Join(arguments, " "),
			err,
			stderr,
		)
	}

	return output, nil
}

func (commandRunner) run(
	ctx context.Context,
	dir string,
	environment []string,
	name string,
	arguments ...string,
) ([]byte, error) {
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = dir
	command.Env = append(os.Environ(), environment...)

	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"run %s %s: %w: %s",
			name,
			strings.Join(arguments, " "),
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return output, nil
}
