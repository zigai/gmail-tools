package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/zigai/gmail-tools/internal/config"
	"github.com/zigai/gmail-tools/internal/process"
)

const shortCommitLen = 7

var (
	version = "dev"
	commit  = "none"
	dirty   = ""
)

type environment struct {
	configPath string
	gogBinary  string
	gmailctl   string
	streams    process.Streams
}

func Execute() {
	configPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gmail-tools:", err)
		os.Exit(1)
	}

	env := environment{
		configPath: configPath,
		gogBinary:  "gog",
		gmailctl:   "gmailctl",
		streams:    process.Streams{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr},
	}

	err = newRootCommand(env).Execute()
	if exitErr, ok := errors.AsType[*process.ExitError](err); ok {
		os.Exit(exitErr.Code)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "gmail-tools:", err)
		os.Exit(1)
	}
}

func newRootCommand(env environment) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "gmail-tools",
		Short: "Gmail and Drive for AI agents across several Google accounts",
		Long: "Search, triage, and filter Gmail across several Google accounts.\n\n" +
			"Accounts are read from " + env.configPath + ".\n" +
			"mail and drive run gog with sending, sharing, permanent deletion, and changes\n" +
			"to forwarding, delegation, send-as, and vacation settings blocked. Filters\n" +
			"change only through gmailctl, so its config stays the source of truth.",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			showVersion, err := cmd.Flags().GetBool("version")
			if err != nil {
				return fmt.Errorf("read version flag: %w", err)
			}

			if showVersion {
				return printLine(cmd.OutOrStdout(), versionText())
			}

			if err := cmd.Help(); err != nil {
				return fmt.Errorf("show help: %w", err)
			}

			return nil
		},
	}
	rootCommand.Flags().BoolP("version", "v", false, "Print version")
	rootCommand.PersistentFlags().
		StringP("account", "a", "", "Account name from the config file (default: default_account)")
	rootCommand.SetIn(env.streams.Stdin)
	rootCommand.SetOut(env.streams.Stdout)
	rootCommand.SetErr(env.streams.Stderr)

	rootCommand.AddCommand(
		newAccountsCommand(env),
		newFiltersCommand(env),
		newMailCommand(env),
		newDriveCommand(env),
	)

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
