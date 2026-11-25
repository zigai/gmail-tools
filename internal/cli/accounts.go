package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/config"
	"github.com/zigai/gmail-tools/internal/gmail"
	"github.com/zigai/gmail-tools/internal/gog"
)

var errMissingAccountValue = errors.New("--account needs a value")

func newAccountsCommand(env environment) *cobra.Command {
	return &cobra.Command{
		Use:   "accounts",
		Short: "List accounts with their sign-in state and unread inbox mail",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(env)
			if err != nil {
				return err
			}

			rows := [][]string{{"NAME", "EMAIL", "DEFAULT", "INBOX", "UNREAD", "STATUS"}}

			for _, name := range cfg.Names() {
				account := cfg.Accounts[name]
				total, unread, status := "-", "-", "ok"

				inbox, err := env.gmail(account).Inbox(cmd.Context())
				if err != nil {
					status = "error"
				} else {
					total, unread = strconv.Itoa(inbox.Total), strconv.Itoa(inbox.Unread)
				}

				rows = append(rows, []string{
					name, account.Email, yesNo(name == cfg.DefaultAccount), total, unread, status,
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

func selectedAccounts(env environment, name string) ([]config.Account, error) {
	if name != "" {
		account, err := loadAccount(env, name)
		if err != nil {
			return nil, err
		}

		return []config.Account{account}, nil
	}

	cfg, err := loadConfig(env)
	if err != nil {
		return nil, err
	}

	accounts := make([]config.Account, 0, len(cfg.Accounts))
	for _, name := range cfg.Names() {
		accounts = append(accounts, cfg.Accounts[name])
	}

	return accounts, nil
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

func (env environment) gmail(account config.Account) gmail.Client {
	return gmail.Client{Runner: env.gogRunner(), Email: account.Email}
}
