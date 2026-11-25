package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/gmail"
)

const (
	defaultSearchMax = 20
	fromWidth        = 40
	subjectWidth     = 70
)

type accountMessage struct {
	gmail.Message

	Account string `json:"account"`
}

func newSearchCommand(env environment) *cobra.Command {
	var (
		limit  int
		asJSON bool
	)

	command := &cobra.Command{
		Use:   "search <gmail query>",
		Short: "Search mail in every account, newest first",
		Long: "Search mail in every account, or only in --account, and list the matches " +
			"newest first.\n\nRead a match with `gmail-tools mail --account NAME get ID " +
			"--sanitize-content`.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accounts, err := selectedAccounts(env, accountFlag(cmd))
			if err != nil {
				return err
			}

			query := strings.Join(args, " ")

			var found []accountMessage

			for _, account := range accounts {
				messages, err := env.gmail(account).Search(cmd.Context(), query, limit)
				if err != nil {
					return fmt.Errorf("account %s: %w", account.Name, err)
				}

				for _, message := range messages {
					found = append(found, accountMessage{Account: account.Name, Message: message})
				}
			}

			slices.SortStableFunc(found, func(a, b accountMessage) int {
				return cmp.Compare(b.Date, a.Date)
			})

			if asJSON {
				return writeJSON(cmd, found)
			}

			rows := [][]string{{"ACCOUNT", "DATE", "FROM", "SUBJECT", "ID"}}
			for _, message := range found {
				rows = append(rows, []string{
					message.Account,
					shortDate(message.Date),
					truncate(message.From, fromWidth),
					truncate(message.Subject, subjectWidth),
					message.ID,
				})
			}

			return writeTable(cmd, rows)
		},
	}

	command.Flags().
		IntVar(&limit, "max", defaultSearchMax, "Matches to read per account (0 reads all)")
	command.Flags().BoolVar(&asJSON, "json", false, "Print JSON")

	return command
}
