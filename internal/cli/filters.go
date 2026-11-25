package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/gmailctl"
)

func newFiltersCommand(env environment) *cobra.Command {
	command := &cobra.Command{
		Use:   "filters",
		Short: "Run gmailctl with an account's config directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.Help(); err != nil {
				return fmt.Errorf("show help: %w", err)
			}

			return nil
		},
	}

	for _, name := range gmailctl.Commands {
		command.AddCommand(newGmailctlCommand(env, name))
	}

	return command
}
