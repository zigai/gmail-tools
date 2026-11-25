// Package gog runs the gog CLI for one account with sending, sharing, deletion,
// and mail-settings commands blocked.
package gog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/zigai/gmail-tools/internal/process"
)

const (
	GroupGmail Group = "gmail"
	GroupDrive Group = "drive"
)

var ErrForbiddenFlag = errors.New("flag is controlled by gmail-tools")

// blockedCommands lists gog command paths, including hidden aliases, that agents
// must not reach: anything that sends mail, shares files, deletes permanently,
// or changes forwarding, delegation, send-as, vacation, or filter settings.
var blockedCommands = []string{
	"gmail.send",
	"gmail.import",
	"gmail.reply",
	"gmail.reply-all",
	"gmail.forward",
	"gmail.autoreply",
	"gmail.track",
	"gmail.drafts.send",
	"gmail.batch.delete",
	"gmail.watch",
	"gmail.autoforward",
	"gmail.delegates",
	"gmail.forwarding",
	"gmail.sendas",
	"gmail.vacation",
	"gmail.filters.create",
	"gmail.filters.delete",
	"gmail.settings.watch",
	"gmail.settings.autoforward",
	"gmail.settings.delegates",
	"gmail.settings.forwarding",
	"gmail.settings.sendas",
	"gmail.settings.vacation",
	"gmail.settings.filters.create",
	"gmail.settings.filters.delete",
	"drive.share",
	"drive.unshare",
	"drive.bulk",
	"drive.delete",
}

var forbiddenFlags = []string{
	"--account",
	"--acct",
	"--access-token",
	"--client",
	"--home",
	"--enable-commands",
	"--enable-commands-exact",
	"--disable-commands",
	"--gmail-no-send",
}

type Group string

type Runner struct {
	Binary  string
	Streams process.Streams
}

// Run passes args to `gog <group>` for the given account, streaming its output.
// Text fetched from mail or files is wrapped in gog's untrusted-content markers.
func (r Runner) Run(ctx context.Context, email string, group Group, args []string) error {
	if err := checkArgs(args); err != nil {
		return err
	}

	full := append([]string{"--wrap-untrusted"}, buildArgs(email, group, args)...)

	if err := process.Run(ctx, r.Streams, r.Binary, full...); err != nil {
		return fmt.Errorf("gog %s: %w", group, err)
	}

	return nil
}

// Output runs `gog <group>` like Run without untrusted-content markers and returns
// its standard output, for callers that parse it.
func (r Runner) Output(
	ctx context.Context,
	email string,
	group Group,
	args []string,
) ([]byte, error) {
	if err := checkArgs(args); err != nil {
		return nil, err
	}

	var stdout bytes.Buffer

	streams := r.Streams
	streams.Stdout = &stdout

	if err := process.Run(ctx, streams, r.Binary, buildArgs(email, group, args)...); err != nil {
		return nil, fmt.Errorf("gog %s: %w", group, err)
	}

	return stdout.Bytes(), nil
}

func buildArgs(email string, group Group, args []string) []string {
	fixed := []string{
		"--account=" + email,
		"--gmail-no-send",
		"--no-input",
		"--enable-commands=" + string(group),
		"--disable-commands=" + strings.Join(blockedCommands, ","),
		string(group),
	}

	return slices.Concat(fixed, args)
}

func checkArgs(args []string) error {
	for _, arg := range args {
		if arg == "--" {
			return nil
		}

		if isShortCluster(arg) && strings.ContainsRune(arg, 'a') {
			return fmt.Errorf("%w: %s (use gmail-tools --account)", ErrForbiddenFlag, arg)
		}

		name, _, _ := strings.Cut(arg, "=")
		for _, flag := range forbiddenFlags {
			if name == flag {
				return fmt.Errorf("%w: %s", ErrForbiddenFlag, flag)
			}
		}
	}

	return nil
}

func isShortCluster(arg string) bool {
	return len(arg) > 1 && arg[0] == '-' && arg[1] != '-'
}
