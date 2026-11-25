// Package gmailctl runs gmailctl against one account's configuration directory.
package gmailctl

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/zigai/gmail-tools/internal/process"
)

var ErrUnsupportedCommand = errors.New("unsupported gmailctl command")

// Commands lists the gmailctl subcommands that can be run. The rest either open
// an editor or are not tied to one account's filters.
var Commands = []string{"init", "diff", "apply", "test", "debug", "download", "export"}

type Runner struct {
	Binary  string
	Streams process.Streams
}

func (r Runner) Run(ctx context.Context, configDir string, args []string) error {
	if len(args) == 0 || !slices.Contains(Commands, args[0]) {
		return fmt.Errorf(
			"%w %q (supported: %s)",
			ErrUnsupportedCommand,
			strings.Join(args[:min(1, len(args))], ""),
			strings.Join(Commands, ", "),
		)
	}

	full := append([]string{"--config=" + configDir}, args...)

	if err := process.Run(ctx, r.Streams, r.Binary, full...); err != nil {
		return fmt.Errorf("gmailctl %s: %w", args[0], err)
	}

	return nil
}
