package cli

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

const shortCommitLen = 7

var (
	version = "dev"
	commit  = "none"
	dirty   = ""
)

func Execute() {
	err := newRootCommand().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "gmail-tools",
		Short: "Gmail and Drive for AI agents across several Google accounts",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			showVersion, err := cmd.Flags().GetBool("version")
			if err != nil {
				return fmt.Errorf("read version flag: %w", err)
			}

			if showVersion {
				_, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s\n",
					versionText(),
				)
				if err != nil {
					return fmt.Errorf("print version: %w", err)
				}

				return nil
			}

			if err := cmd.Help(); err != nil {
				return fmt.Errorf("show help: %w", err)
			}

			return nil
		},
	}
	rootCommand.Flags().BoolP("version", "v", false, "Print version")

	return rootCommand
}

func versionText() string {
	v, c, modified := version, commit, dirty

	if info, ok := debug.ReadBuildInfo(); ok {
		if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}

		for _, setting := range info.Settings {
			switch {
			case setting.Key == "vcs.revision" && c == "none":
				c = setting.Value[:min(shortCommitLen, len(setting.Value))]
			case setting.Key == "vcs.modified" && modified == "":
				modified = setting.Value
			}
		}
	}

	if modified == "true" {
		c += ", dirty"
	}

	return fmt.Sprintf("gmail-tools %s (%s)", v, c)
}
