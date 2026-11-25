package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func fields(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

func outputLines(stdout string) []string {
	return strings.Split(strings.TrimSpace(stdout), "\n")
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

func TestSearchMergesAccountsNewestFirst(t *testing.T) {
	gog := testutil.NewFakeBinary(t,
		testutil.Response{Pattern: "--account=home@example.com", Stdout: `{"messages": [
			{"id": "h1", "internalDateIso": "2026-03-01T10:00:00Z", "from": "a@example.com", "subject": "older"}
		]}`},
		testutil.Response{Pattern: "--account=work@example.com", Stdout: `{"messages": [
			{"id": "w1", "internalDateIso": "2026-03-02T10:00:00Z", "from": "b@example.com", "subject": "newer"}
		]}`},
	)

	stdout, _, err := executeWithEnv(testEnv(t, gog, gog), "search", "invoice")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}

	lines := outputLines(stdout)
	if len(lines) != 3 || !strings.HasPrefix(fields(lines[1]), "work 2026-03-02") ||
		!strings.HasPrefix(fields(lines[2]), "home 2026-03-01") {
		t.Fatalf("stdout = %q, want work's newer match before home's", stdout)
	}
}

func TestSearchWithAccountReadsOnlyThatAccount(t *testing.T) {
	gog := testutil.NewFakeBinary(t, testutil.Response{Stdout: `{"messages": []}`})

	if _, _, err := executeWithEnv(
		testEnv(t, gog, gog),
		"search",
		"-a",
		"work",
		"invoice",
	); err != nil {
		t.Fatalf("execute error = %v", err)
	}

	calls := gog.Calls(t)
	if len(calls) != 1 || !slices.Contains(calls[0], "--account=work@example.com") {
		t.Fatalf("gog calls = %q, want one call for work", calls)
	}
}

func TestAccountsShowsInboxCountsAndFailures(t *testing.T) {
	gog := testutil.NewFakeBinary(
		t,
		testutil.Response{
			Pattern: "--account=home@example.com",
			Stdout:  `{"label": {"messagesTotal": 120, "messagesUnread": 7}}`,
		},
		testutil.Response{Pattern: "--account=work@example.com", Code: 4},
	)

	stdout, _, err := executeWithEnv(testEnv(t, gog, gog), "accounts")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}

	lines := outputLines(stdout)
	if len(lines) != 3 || fields(lines[1]) != "home home@example.com yes 120 7 ok" ||
		fields(lines[2]) != "work work@example.com no - - error" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestSendersPrintsCountsForQuery(t *testing.T) {
	gog := testutil.NewFakeBinary(t, testutil.Response{Stdout: `{"messages": [
		{"from": "Shop <deals@shop.example>", "labels": ["INBOX", "UNREAD"]},
		{"from": "deals@shop.example", "labels": ["INBOX", "UNREAD"]},
		{"from": "friend@example.net", "labels": ["INBOX"]}
	]}`})

	stdout, _, err := executeWithEnv(testEnv(t, gog, gog), "senders", "--top", "1", "newer_than:7d")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}

	lines := outputLines(stdout)
	if len(lines) != 2 || !strings.HasPrefix(fields(lines[1]), "2 2 deals@shop.example") {
		t.Fatalf("stdout = %q, want header and the top sender", stdout)
	}

	got := gog.Args(t)
	if got[len(got)-1] != "newer_than:7d" || slices.Contains(got, "--wrap-untrusted") {
		t.Fatalf("gog args = %q, want the query last and no untrusted markers", got)
	}
}

func filtersGog(t *testing.T) testutil.FakeBinary {
	t.Helper()

	return testutil.NewFakeBinary(
		t,
		testutil.Response{Pattern: " filters list ", Stdout: `{"filters": [
			{"id": "busy", "criteria": {"from": "busy@example.com"}, "action": {"removeLabelIds": ["INBOX"]}},
			{"id": "dead", "criteria": {"subject": "gone"}, "action": {"addLabelIds": ["Label_1"]}},
			{"id": "lots", "criteria": {"from": "lots@example.com"}, "action": {"addLabelIds": ["Label_1"]}}
		]}`},
		testutil.Response{
			Pattern: " labels list ",
			Stdout:  `{"labels": [{"id": "Label_1", "name": "Old"}]}`,
		},
		testutil.Response{Pattern: "from:(busy@example.com)", Stdout: `{"totalMatches": 12}`},
		testutil.Response{Pattern: "subject:(gone)", Stdout: `{"totalMatches": 0}`},
		testutil.Response{
			Pattern: "from:(lots@example.com)",
			Stdout:  `{"totalMatchesAtLeast": 500}`,
		},
	)
}

func TestFiltersAuditListsLeastUsedFirst(t *testing.T) {
	gog := filtersGog(t)

	stdout, _, err := executeWithEnv(testEnv(t, gog, gog), "filters", "audit", "--since", "30d")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}

	lines := outputLines(stdout)
	want := []string{
		"0 label Old subject:(gone)",
		"12 archive from:(busy@example.com)",
		"500+ label Old from:(lots@example.com)",
	}

	if len(lines) != 4 {
		t.Fatalf("stdout = %q", stdout)
	}

	for i, w := range want {
		if fields(lines[i+1]) != w {
			t.Fatalf("line %d = %q, want %q", i+1, fields(lines[i+1]), w)
		}
	}

	for _, call := range gog.Calls(t) {
		if slices.Contains(call, "--count") &&
			!strings.HasSuffix(call[len(call)-1], " newer_than:30d") {
			t.Fatalf("count query %q does not use the window", call[len(call)-1])
		}
	}
}

func TestFiltersAuditUnusedShowsOnlyZeroMatches(t *testing.T) {
	gog := filtersGog(t)

	stdout, _, err := executeWithEnv(testEnv(t, gog, gog), "filters", "audit", "--unused")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}

	lines := outputLines(stdout)
	if len(lines) != 2 || fields(lines[1]) != "0 label Old subject:(gone)" {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestFiltersSuggestSkipsQuietSenders(t *testing.T) {
	gog := testutil.NewFakeBinary(t, testutil.Response{Stdout: `{"messages": [
		{"from": "deals@shop.example", "labels": ["INBOX"]},
		{"from": "deals@shop.example", "labels": ["INBOX"]},
		{"from": "friend@example.net", "labels": ["INBOX"]}
	]}`})

	stdout, _, err := executeWithEnv(testEnv(t, gog, gog), "filters", "suggest", "--min", "2")
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}

	if !strings.Contains(stdout, "deals@shop.example") ||
		strings.Contains(stdout, "friend@example.net") {
		t.Fatalf("stdout = %q, want only the sender with 2 messages", stdout)
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
