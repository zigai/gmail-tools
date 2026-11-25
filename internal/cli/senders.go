package cli

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/senders"
)

const (
	defaultSendersWindow = "90d"
	defaultSendersTop    = 30
	// Gmail allows each user about 3000 message reads a minute.
	defaultSendersMax = 1000
	maxHelp           = "Newest messages to read (0 reads all; Gmail allows about 3000 a minute)"
)

func newSendersCommand(env environment) *cobra.Command {
	var (
		window string
		limit  int
		top    int
		asJSON bool
	)

	command := &cobra.Command{
		Use:   "senders [gmail query]",
		Short: "Rank the senders and mailing lists that fill the inbox",
		Long: "Count inbox messages per sender and mailing list, most messages first.\n\n" +
			"Without a query it reads the inbox for the --since window: the mail that no " +
			"filter archives, which is where new filters help most.",
		RunE: func(cmd *cobra.Command, args []string) error {
			account, err := loadAccount(env, accountFlag(cmd))
			if err != nil {
				return err
			}

			query := strings.Join(args, " ")
			if query == "" {
				query = inboxQuery(window)
			}

			messages, err := env.gmail(account).
				Search(cmd.Context(), query, limit, senders.ListIDHeader)
			if err != nil {
				return fmt.Errorf("account %s: %w", account.Name, err)
			}

			counts := topSenders(senders.Count(messages), top, 1)

			if asJSON {
				return writeJSON(cmd, counts)
			}

			rows := [][]string{{"MESSAGES", "UNREAD", "SENDER", "LIST"}}
			for _, sender := range counts {
				rows = append(rows, []string{
					strconv.Itoa(sender.Messages),
					strconv.Itoa(sender.Unread),
					sender.Address,
					cmp.Or(sender.ListID, "-"),
				})
			}

			return writeTable(cmd, rows)
		},
	}

	command.Flags().
		StringVar(&window, "since", defaultSendersWindow, "Inbox window, as in newer_than (7d, 3m, 1y)")
	command.Flags().IntVar(&limit, "max", defaultSendersMax, maxHelp)
	command.Flags().IntVar(&top, "top", defaultSendersTop, "Rows to show (0 shows all)")
	command.Flags().BoolVar(&asJSON, "json", false, "Print JSON")

	return command
}

func inboxQuery(window string) string {
	return "in:inbox newer_than:" + window
}

func topSenders(counts []senders.Sender, top int, minMessages int) []senders.Sender {
	kept := counts[:0:0]
	for _, sender := range counts {
		if sender.Messages >= minMessages {
			kept = append(kept, sender)
		}
	}

	if top > 0 && len(kept) > top {
		return kept[:top]
	}

	return kept
}
