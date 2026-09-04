// Package config は Talknote ブラウザセッションの複数アカウント設定を読み書きする。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Account struct {
	SID         string `json:"sid"`
	Origin      string `json:"origin"`
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
}

type Config struct {
	Default  string             `json:"default,omitempty"`
	Accounts map[string]Account `json:"accounts"`
}

func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "talknote-cli", "accounts.json"), nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{Accounts: map[string]Account{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Accounts == nil {
		cfg.Accounts = map[string]Account{}
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".accounts-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func PermWarning(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return fmt.Sprintf("warning: %s has loose permissions %#o; run: chmod 600 %s", path, perm, path)
	}
	return ""
}

func (c *Config) Resolve(alias string) (string, Account, error) {
	if alias != "" {
		account, ok := c.Accounts[alias]
		if !ok {
			return "", Account{}, fmt.Errorf("account %q not found; run `tn auth list`", alias)
		}
		return alias, account, nil
	}
	if c.Default != "" {
		account, ok := c.Accounts[c.Default]
		if !ok {
			return "", Account{}, fmt.Errorf("default account %q not found; run `tn auth set-default`", c.Default)
		}
		return c.Default, account, nil
	}
	if len(c.Accounts) == 1 {
		for name, account := range c.Accounts {
			return name, account, nil
		}
	}
	if len(c.Accounts) == 0 {
		return "", Account{}, errors.New("no accounts configured; copy a Talknote request as cURL and run `tn auth login` (see `tn auth guide`)")
	}
	return "", Account{}, fmt.Errorf("multiple accounts (%s); pass --account or run `tn auth set-default`", joinNames(c.Accounts))
}

func joinNames(accounts map[string]Account) string {
	names := make([]string, 0, len(accounts))
	for name := range accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
