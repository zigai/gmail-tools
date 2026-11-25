package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/config"
	"github.com/zigai/gmail-tools/internal/gog"
)

var errMissingAccountValue = errors.New("--account needs a value")

func newAccountsCommand(env environment) *cobra.Command {
	return &cobra.Command{
		Use:   "accounts",
		Short: "List configured accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(env)
			if err != nil {
				return err
			}

			rows := [][]string{{"NAME", "EMAIL", "DEFAULT", "GMAILCTL CONFIG"}}

			for _, name := range cfg.Names() {
				account := cfg.Accounts[name]
				rows = append(rows, []string{
					name, account.Email, yesNo(name == cfg.DefaultAccount), account.GmailctlConfig,
				})
			}

			return writeTable(cmd, rows)
		},
	}
}

func loadConfig(env environment) (config.Config, error) {
	cfg, err := config.Load(env.configPath)
	if err != nil {
		return config.Config{}, fmt.Errorf("load accounts: %w", err)
	}

	return cfg, nil
}

func loadAccount(env environment, name string) (config.Account, error) {
	cfg, err := loadConfig(env)
	if err != nil {
		return config.Account{}, err
	}

	account, err := cfg.Account(name)
	if err != nil {
		return config.Account{}, fmt.Errorf("select account: %w", err)
	}

	return account, nil
}

func accountFlag(cmd *cobra.Command) string {
	return cmd.Flag("account").Value.String()
}

func splitAccount(cmd *cobra.Command, args []string) (string, []string, error) {
	// Cobra skips flag parsing for commands that pass their arguments on, so an
	// --account given before or after the command name arrives here as raw arguments.
	name := accountFlag(cmd)

	for len(args) > 0 {
		arg := args[0]

		switch {
		case arg == "--account" || arg == "-a":
			value := args[1:]
			if len(value) == 0 {
				return "", nil, errMissingAccountValue
			}

			name, args = value[0], value[1:]
		case strings.HasPrefix(arg, "--account="):
			name, args = strings.TrimPrefix(arg, "--account="), args[1:]
		default:
			return name, args, nil
		}
	}

	return name, args, nil
}

func (env environment) gogRunner() gog.Runner {
	return gog.Runner{Binary: env.gogBinary, Streams: env.streams}
}
