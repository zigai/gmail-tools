package cli

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestRootCommandShowsHelpByDefault(t *testing.T) {
	stdout, stderr, err := executeRootCommand()
	if err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("stdout = %q, want Usage section", stdout)
	}

	if !strings.Contains(stdout, "gmail-tools [flags]") {
		t.Fatalf("stdout = %q, want command usage", stdout)
	}
}

func TestRootCommandShowsVersion(t *testing.T) {
	stdout, stderr, err := executeRootCommand("--version")
	if err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	pattern := regexp.MustCompile(`^gmail-tools \S+ \([^(),\s]+(?:, dirty)?\)\n$`)
	if !pattern.MatchString(stdout) {
		t.Fatalf("unexpected version: %q", stdout)
	}
}

func executeRootCommand(args ...string) (string, string, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	rootCommand := newRootCommand()
	rootCommand.SetOut(&stdout)
	rootCommand.SetErr(&stderr)
	rootCommand.SetArgs(args)

	err := rootCommand.Execute()

	return stdout.String(), stderr.String(), err
}

func TestVersionTextMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDirty := version, commit, dirty

	t.Cleanup(func() {
		version, commit, dirty = oldVersion, oldCommit, oldDirty
	})

	version, commit = "v1.2.3", "abcdef0"

	for _, modified := range []string{"false", "true"} {
		dirty = modified

		want := "gmail-tools v1.2.3 (abcdef0)"
		if modified == "true" {
			want = "gmail-tools v1.2.3 (abcdef0, dirty)"
		}

		if got := versionText(); got != want {
			t.Fatalf("versionText() = %q, want %q", got, want)
		}
	}
}
