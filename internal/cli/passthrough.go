package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/config"
	"github.com/zigai/gmail-tools/internal/gmailctl"
	"github.com/zigai/gmail-tools/internal/gog"
)

func newMailCommand(env environment) *cobra.Command {
	return newGogCommand(env, gog.GroupGmail, "mail", "Run a gog gmail command for an account")
}

func newDriveCommand(env environment) *cobra.Command {
	return newGogCommand(env, gog.GroupDrive, "drive", "Run a gog drive command for an account")
}

func newGogCommand(
	env environment,
	group gog.Group,
	use string,
	short string,
) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [--account NAME] <gog " + string(group) + " arguments>",
		Short: short,
		Long: short + ".\n\nArguments are passed to `gog " + string(group) + "`. " +
			"Run `gmail-tools " + use + " --help` for gog's own help.",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, rest, err := splitAccount(cmd, args)
			if err != nil {
				return err
			}

			account, err := loadAccount(env, name)
			if err != nil {
				return err
			}

			return env.gogRunner().Run(cmd.Context(), account.Email, group, rest)
		},
	}
}

func newGmailctlCommand(env environment, name string) *cobra.Command {
	return &cobra.Command{
		Use:                name + " [--account NAME] [gmailctl " + name + " arguments]",
		Short:              "Run gmailctl " + name + " with the account's config directory",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			accountName, rest, err := splitAccount(cmd, args)
			if err != nil {
				return err
			}

			account, err := loadAccount(env, accountName)
			if err != nil {
				return err
			}

			if account.GmailctlConfig == "" {
				return fmt.Errorf("%w: %s", config.ErrNoGmailctlConfig, account.Name)
			}

			runner := gmailctl.Runner{Binary: env.gmailctl, Streams: env.streams}

			return runner.Run(
				cmd.Context(),
				account.GmailctlConfig,
				append([]string{name}, rest...),
			)
		},
	}
}
