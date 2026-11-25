package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/zigai/gmail-tools/internal/config"
	"github.com/zigai/gmail-tools/internal/testutil"
)

const testConfig = `default_account = "home"

[accounts.home]
email = "home@example.com"
gmailctl_config = "/srv/gmailctl-home"

[accounts.work]
email = "work@example.com"
`

func testEnv(
	t *testing.T,
	gog testutil.FakeBinary,
	gmailctlBinary testutil.FakeBinary,
) environment {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(testConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return environment{configPath: path, gogBinary: gog.Path, gmailctl: gmailctlBinary.Path}
}

func TestMailSelectsAccount(t *testing.T) {
	cases := map[string]struct {
		args  []string
		email string
	}{
		"default account": {[]string{"mail", "search", "x"}, "home@example.com"},
		"flag before command": {
			[]string{"--account", "work", "mail", "search", "x"},
			"work@example.com",
		},
		"short flag before": {[]string{"-a", "work", "mail", "search", "x"}, "work@example.com"},
		"flag after command": {
			[]string{"mail", "--account=work", "search", "x"},
			"work@example.com",
		},
		"short flag after": {[]string{"mail", "-a", "work", "search", "x"}, "work@example.com"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			gog := testutil.NewFakeBinary(t)

			if _, _, err := executeWithEnv(testEnv(t, gog, gog), tc.args...); err != nil {
				t.Fatalf("execute error = %v", err)
			}

			got := gog.Args(t)
			if !slices.Contains(got, "--account="+tc.email) {
				t.Fatalf("gog args = %q, want account %s", got, tc.email)
			}

			if !slices.Equal(got[len(got)-3:], []string{"gmail", "search", "x"}) {
				t.Fatalf("gog args = %q, want to end with gmail search x", got)
			}
		})
	}
}

func TestDriveUsesDriveGroup(t *testing.T) {
	gog := testutil.NewFakeBinary(t)

	if _, _, err := executeWithEnv(testEnv(t, gog, gog), "drive", "ls"); err != nil {
		t.Fatalf("execute error = %v", err)
	}

	got := gog.Args(t)
	if !slices.Contains(got, "--enable-commands=drive") ||
		!slices.Equal(got[len(got)-2:], []string{"drive", "ls"}) {
		t.Fatalf("gog args = %q, want drive ls with only drive enabled", got)
	}
}

func TestFiltersRunsGmailctlWithAccountConfig(t *testing.T) {
	gmailctlBinary := testutil.NewFakeBinary(t)

	if _, _, err := executeWithEnv(
		testEnv(t, gmailctlBinary, gmailctlBinary),
		"filters",
		"apply",
		"--yes",
	); err != nil {
		t.Fatalf("execute error = %v", err)
	}

	want := []string{"--config=/srv/gmailctl-home", "apply", "--yes"}
	if got := gmailctlBinary.Args(t); !slices.Equal(got, want) {
		t.Fatalf("gmailctl args = %q, want %q", got, want)
	}
}

func TestFiltersRefusesAccountWithoutGmailctlConfig(t *testing.T) {
	for _, args := range [][]string{{"filters", "-a", "work", "diff"}, {"filters", "diff", "-a", "work"}} {
		gmailctlBinary := testutil.NewFakeBinary(t)

		_, _, err := executeWithEnv(testEnv(t, gmailctlBinary, gmailctlBinary), args...)
		if !errors.Is(err, config.ErrNoGmailctlConfig) {
			t.Fatalf("%q: execute error = %v, want ErrNoGmailctlConfig", args, err)
		}

		if calls := gmailctlBinary.Calls(t); len(calls) != 0 {
			t.Fatalf("gmailctl ran with %q", calls)
		}
	}
}

func TestFiltersRejectsEditorCommand(t *testing.T) {
	gmailctlBinary := testutil.NewFakeBinary(t)

	_, _, err := executeWithEnv(testEnv(t, gmailctlBinary, gmailctlBinary), "filters", "edit")
	if err == nil {
		t.Fatal("execute error = nil, want unknown command")
	}

	if calls := gmailctlBinary.Calls(t); len(calls) != 0 {
		t.Fatalf("gmailctl ran with %q", calls)
	}
}
