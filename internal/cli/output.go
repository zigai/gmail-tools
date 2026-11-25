package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/spf13/cobra"
)

const tablePadding = 2

func writeTable(cmd *cobra.Command, rows [][]string) error {
	table := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, tablePadding, ' ', 0)

	for _, row := range rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = cleanCell(cell)
		}

		if _, err := fmt.Fprintln(table, strings.Join(cells, "\t")); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	if err := table.Flush(); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func printLine(w io.Writer, text string) error {
	if _, err := fmt.Fprintln(w, text); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func cleanCell(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, value)

	return cmpEmpty(strings.TrimSpace(value))
}

func cmpEmpty(value string) string {
	if value == "" {
		return "-"
	}

	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}
