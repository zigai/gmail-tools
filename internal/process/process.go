// Package process runs external commands with caller-supplied standard streams.
package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExitError reports a command that exited with a non-zero status.
type ExitError struct {
	Command string
	Code    int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("%s exited with status %d", e.Command, e.Code)
}

func Run(ctx context.Context, streams Streams, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = streams.Stdin
	cmd.Stdout = streams.Stdout
	cmd.Stderr = streams.Stderr

	err := cmd.Run()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return &ExitError{Command: binary, Code: exitErr.ExitCode()}
	}

	if err != nil {
		return fmt.Errorf("run %s: %w", binary, err)
	}

	return nil
}
