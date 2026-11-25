package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
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

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
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

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}

	return string(runes[:width-1]) + "…"
}

func shortDate(iso string) string {
	parsed, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}

	return parsed.Format("2006-01-02 15:04")
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}
