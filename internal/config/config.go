// Package config loads the account list that maps short account names to Google
// accounts and gmailctl configuration directories.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var (
	ErrNoAccount         = errors.New("no account selected")
	ErrUnknownAccount    = errors.New("unknown account")
	ErrInvalidConfig     = errors.New("invalid config")
	ErrNoGmailctlConfig  = errors.New("account has no gmailctl_config")
	errRelativeDirectory = errors.New("path must be absolute or start with ~/")
)

type Account struct {
	Name           string
	Email          string
	GmailctlConfig string
}

type Config struct {
	DefaultAccount string
	Accounts       map[string]Account
}

type fileConfig struct {
	DefaultAccount string                 `toml:"default_account"`
	Accounts       map[string]fileAccount `toml:"accounts"`
}

type fileAccount struct {
	Email          string `toml:"email"`
	GmailctlConfig string `toml:"gmailctl_config"`
}

// DefaultPath returns the config file location under the user's config directory.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}

	return filepath.Join(dir, "gmail-tools", "config.toml"), nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var raw fileConfig

	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("find home directory: %w", err)
	}

	cfg := Config{DefaultAccount: raw.DefaultAccount, Accounts: map[string]Account{}}

	for name, account := range raw.Accounts {
		if strings.TrimSpace(account.Email) == "" {
			return Config{}, fmt.Errorf("%w: account %q has no email", ErrInvalidConfig, name)
		}

		gmailctlConfig := ""
		if account.GmailctlConfig != "" {
			gmailctlConfig, err = expandHome(account.GmailctlConfig, home)
			if err != nil {
				return Config{}, fmt.Errorf(
					"%w: account %q gmailctl_config: %w",
					ErrInvalidConfig,
					name,
					err,
				)
			}
		}

		cfg.Accounts[name] = Account{
			Name:           name,
			Email:          account.Email,
			GmailctlConfig: gmailctlConfig,
		}
	}

	if cfg.DefaultAccount != "" {
		if _, ok := cfg.Accounts[cfg.DefaultAccount]; !ok {
			return Config{}, fmt.Errorf(
				"%w: default_account %q is not in [accounts]",
				ErrInvalidConfig,
				cfg.DefaultAccount,
			)
		}
	}

	return cfg, nil
}

// Account returns the named account, or the default account when name is empty.
func (c Config) Account(name string) (Account, error) {
	if name == "" {
		name = c.DefaultAccount
	}

	if name == "" {
		return Account{}, fmt.Errorf(
			"%w: pass --account or set default_account (accounts: %s)",
			ErrNoAccount,
			strings.Join(c.Names(), ", "),
		)
	}

	account, ok := c.Accounts[name]
	if !ok {
		return Account{}, fmt.Errorf(
			"%w %q (accounts: %s)",
			ErrUnknownAccount,
			name,
			strings.Join(c.Names(), ", "),
		)
	}

	return account, nil
}

func (c Config) Names() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

func expandHome(path string, home string) (string, error) {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, rest), nil
	}

	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: %q", errRelativeDirectory, path)
	}

	return filepath.Clean(path), nil
}
