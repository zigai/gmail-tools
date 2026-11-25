package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zigai/gmail-tools/internal/config"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func TestLoadMissingFileNamesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")

	_, err := config.Load(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("Load() error = %v, want error naming %s", err, path)
	}
}

func TestLoadRejectsInvalidConfigs(t *testing.T) {
	cases := map[string]string{
		"account without email": "[accounts.home]\ngmailctl_config = \"/srv/home\"\n",
		"unknown default":       "default_account = \"work\"\n[accounts.home]\nemail = \"a@example.com\"\n",
		"relative gmailctl dir": "[accounts.home]\nemail = \"a@example.com\"\ngmailctl_config = \"rel/dir\"\n",
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := config.Load(writeConfig(t, body))
			if !errors.Is(err, config.ErrInvalidConfig) {
				t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	_, err := config.Load(
		writeConfig(t, "[accounts.home]\nemail = \"a@example.com\"\nemial = \"x\"\n"),
	)
	if err == nil {
		t.Fatal("Load() error = nil, want error for misspelled key")
	}
}

func TestLoadExpandsHomeInGmailctlConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := config.Load(
		writeConfig(
			t,
			"[accounts.home]\nemail = \"a@example.com\"\ngmailctl_config = \"~/.gmailctl-home\"\n",
		),
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := filepath.Join(home, ".gmailctl-home")
	if got := cfg.Accounts["home"].GmailctlConfig; got != want {
		t.Fatalf("GmailctlConfig = %q, want %q", got, want)
	}
}

func TestAccountSelection(t *testing.T) {
	body := "[accounts.home]\nemail = \"home@example.com\"\n[accounts.work]\nemail = \"work@example.com\"\n"

	noDefault, err := config.Load(writeConfig(t, body))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if _, err := noDefault.Account(""); !errors.Is(err, config.ErrNoAccount) {
		t.Fatalf("Account(\"\") error = %v, want ErrNoAccount", err)
	}

	if _, err := noDefault.Account("other"); !errors.Is(err, config.ErrUnknownAccount) {
		t.Fatalf("Account(other) error = %v, want ErrUnknownAccount", err)
	}

	withDefault, err := config.Load(writeConfig(t, "default_account = \"work\"\n"+body))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	account, err := withDefault.Account("")
	if err != nil || account.Email != "work@example.com" {
		t.Fatalf("Account(\"\") = %+v, %v, want work account", account, err)
	}

	account, err = withDefault.Account("home")
	if err != nil || account.Email != "home@example.com" {
		t.Fatalf("Account(home) = %+v, %v, want home account", account, err)
	}
}
