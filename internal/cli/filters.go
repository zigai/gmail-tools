package cli

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/filters"
	"github.com/zigai/gmail-tools/internal/gmail"
	"github.com/zigai/gmail-tools/internal/gmailctl"
	"github.com/zigai/gmail-tools/internal/senders"
)

const (
	defaultAuditWindow    = "180d"
	auditConcurrency      = 8
	queryWidth            = 80
	defaultSuggestTop     = 20
	defaultSuggestMinimum = 5
)

type filterUsage struct {
	ID      string `json:"id"`
	Query   string `json:"query"`
	Actions string `json:"actions"`
	Matches int    `json:"matches"`
	Exact   bool   `json:"exact"`
}

func newFiltersCommand(env environment) *cobra.Command {
	command := &cobra.Command{
		Use:   "filters",
		Short: "Audit Gmail filters, draft new ones, and apply them with gmailctl",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.Help(); err != nil {
				return fmt.Errorf("show help: %w", err)
			}

			return nil
		},
	}

	command.AddCommand(newFiltersAuditCommand(env), newFiltersSuggestCommand(env))

	for _, name := range gmailctl.Commands {
		command.AddCommand(newGmailctlCommand(env, name))
	}

	return command
}

func newFiltersAuditCommand(env environment) *cobra.Command {
	var (
		window     string
		unusedOnly bool
		asJSON     bool
	)

	command := &cobra.Command{
		Use:   "audit",
		Short: "Count the messages each Gmail filter matched, least used first",
		Long: "Count the messages each Gmail filter matched in the --since window, least " +
			"used first. Filters with no matches are candidates for removal.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			account, err := loadAccount(env, accountFlag(cmd))
			if err != nil {
				return err
			}

			client := env.gmail(account)

			list, err := client.Filters(cmd.Context())
			if err != nil {
				return fmt.Errorf("account %s: %w", account.Name, err)
			}

			labelNames, err := client.LabelNames(cmd.Context())
			if err != nil {
				return fmt.Errorf("account %s: %w", account.Name, err)
			}

			usage, err := countFilterMatches(cmd.Context(), client, list, labelNames, window)
			if err != nil {
				return err
			}

			if unusedOnly {
				usage = slices.DeleteFunc(usage, func(u filterUsage) bool { return u.Matches > 0 })
			}

			if asJSON {
				return writeJSON(cmd, usage)
			}

			rows := [][]string{{"MATCHES", "ACTIONS", "QUERY"}}

			for _, u := range usage {
				matches := strconv.Itoa(u.Matches)
				if !u.Exact {
					matches += "+"
				}

				rows = append(rows, []string{matches, u.Actions, truncate(u.Query, queryWidth)})
			}

			return writeTable(cmd, rows)
		},
	}

	command.Flags().
		StringVar(&window, "since", defaultAuditWindow, "Window to count matches in, as in newer_than")
	command.Flags().BoolVar(&unusedOnly, "unused", false, "Show only filters with no matches")
	command.Flags().BoolVar(&asJSON, "json", false, "Print JSON")

	return command
}

func countFilterMatches(
	ctx context.Context,
	client gmail.Client,
	list []gmail.Filter,
	labelNames map[string]string,
	window string,
) ([]filterUsage, error) {
	usage := make([]filterUsage, len(list))
	errs := make([]error, len(list))
	slots := make(chan struct{}, auditConcurrency)

	var wg sync.WaitGroup

	for i, filter := range list {
		query := filters.Query(filter.Criteria)
		usage[i] = filterUsage{
			ID:      filter.ID,
			Query:   query,
			Actions: filters.Describe(filter.Action, labelNames),
			Matches: 0,
			Exact:   true,
		}

		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()

			count, err := client.Count(ctx, query+" newer_than:"+window)
			if err != nil {
				errs[i] = fmt.Errorf("filter %s: %w", filter.ID, err)

				return
			}

			usage[i].Matches, usage[i].Exact = count.Total, count.Exact
		})
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	slices.SortStableFunc(usage, func(a, b filterUsage) int {
		return cmp.Compare(a.Matches, b.Matches)
	})

	return usage, nil
}

func newFiltersSuggestCommand(env environment) *cobra.Command {
	var (
		window      string
		limit       int
		top         int
		minMessages int
	)

	command := &cobra.Command{
		Use:   "suggest",
		Short: "Draft gmailctl rules for the senders that fill the inbox",
		Long: "Draft gmailctl rules that archive and label mail from the busiest inbox " +
			"senders and mailing lists. Review the rules, add them to the account's " +
			"config.jsonnet, then run `gmail-tools filters diff` and `filters apply`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			account, err := loadAccount(env, accountFlag(cmd))
			if err != nil {
				return err
			}

			messages, err := env.gmail(account).
				Search(cmd.Context(), inboxQuery(window), limit, senders.ListIDHeader)
			if err != nil {
				return fmt.Errorf("account %s: %w", account.Name, err)
			}

			counts := topSenders(senders.Count(messages), top, minMessages)

			_, err = fmt.Fprint(cmd.OutOrStdout(), filters.Suggest(counts))
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}

			return nil
		},
	}

	command.Flags().
		StringVar(&window, "since", defaultSendersWindow, "Inbox window, as in newer_than")
	command.Flags().IntVar(&limit, "max", defaultSendersMax, maxHelp)
	command.Flags().IntVar(&top, "top", defaultSuggestTop, "Rules to draft")
	command.Flags().
		IntVar(&minMessages, "min", defaultSuggestMinimum, "Skip senders with fewer messages")

	return command
}
