package localrepo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type commandRunner struct{}

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
