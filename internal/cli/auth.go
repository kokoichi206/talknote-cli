package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/config"
	"github.com/kokoichi206/talknote-cli/internal/curlparse"
	"github.com/kokoichi206/talknote-cli/internal/output"
)

type authAccountView struct {
	Alias       string `json:"alias"`
	Origin      string `json:"origin"`
	NetworkID   string `json:"network_id"`
	NetworkName string `json:"network_name"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	IsDefault   bool   `json:"is_default"`
}

func (a *app) authCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Talknote のブラウザセッションを管理する", Args: exactArgs(0, "tn auth <command>"), RunE: subcommandRequired("tn auth")}
	cmd.AddCommand(a.authLoginCmd(), a.authListCmd(), a.authSetDefaultCmd(), a.authRemoveCmd(), a.authStatusCmd(), a.authGuideCmd())
	return cmd
}

func (a *app) authLoginCmd() *cobra.Command {
	var name string
	var fromClipboard bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Copy as cURL からログイン中のセッションを登録する",
		Args:  exactArgs(0, "tn auth login [--from-clipboard] [--name <alias>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			curlText, err := a.readCurl(fromClipboard)
			if err != nil {
				return err
			}
			parsed, err := curlparse.Parse(curlText)
			if err != nil {
				return &UsageError{Err: err}
			}
			account := config.Account{SID: parsed.SID, Origin: parsed.Origin}
			client := a.newClient(account)
			user, err := client.Me(cmd.Context())
			if err != nil {
				return fmt.Errorf("ブラウザセッションの検証に失敗しました: %w", err)
			}
			network, err := client.CurrentNetwork(cmd.Context())
			if err != nil {
				return fmt.Errorf("talknote ネットワークの取得に失敗しました: %w", err)
			}
			account.NetworkID = string(network.ID)
			account.NetworkName = network.Name
			account.UserID = string(user.ID)
			account.UserName = user.Name()

			cfg, path, err := a.loadConfig()
			if err != nil {
				return err
			}
			cfg.Accounts[name] = account
			if cfg.Default == "" {
				cfg.Default = name
			}
			if err := cfg.Save(path); err != nil {
				return err
			}
			return a.printAuthAccount(authAccountView{
				Alias: name, Origin: account.Origin, NetworkID: account.NetworkID,
				NetworkName: account.NetworkName, UserID: account.UserID,
				UserName: account.UserName, IsDefault: cfg.Default == name,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "default", "この Talknote アカウントのローカル名")
	cmd.Flags().BoolVar(&fromClipboard, "from-clipboard", false, "クリップボードから cURL を読む（対話モードでは既定）")
	return cmd
}

const maxCurlBytes = 1 << 20

func (a *app) readCurl(fromClipboard bool) (string, error) {
	if fromClipboard || a.deps.StdinIsTTY {
		text, err := a.clipboard()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) == "" {
			return "", usagef("クリップボードが空です。Talknote のリクエストを Copy as cURL してください")
		}
		return text, nil
	}
	data, err := io.ReadAll(io.LimitReader(a.deps.Stdin, maxCurlBytes))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", usagef("標準入力が空です。cURL を渡すか --from-clipboard を指定してください")
	}
	return string(data), nil
}

func (a *app) authListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "保存済みアカウントを一覧表示する", Args: exactArgs(0, "tn auth list"),
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, _, err := a.loadConfig()
			if err != nil {
				return err
			}
			aliases := make([]string, 0, len(cfg.Accounts))
			for alias := range cfg.Accounts {
				aliases = append(aliases, alias)
			}
			sort.Strings(aliases)
			views := make([]authAccountView, 0, len(aliases))
			for _, alias := range aliases {
				account := cfg.Accounts[alias]
				views = append(views, authAccountView{alias, account.Origin, account.NetworkID, account.NetworkName, account.UserID, account.UserName, alias == cfg.Default})
			}
			format, err := a.format()
			if err != nil {
				return err
			}
			if format == output.FormatJSON {
				return output.WriteJSON(a.deps.Stdout, struct {
					Accounts []authAccountView `json:"accounts"`
				}{views})
			}
			rows := make([][]string, 0, len(views))
			for _, view := range views {
				marker := ""
				if view.IsDefault {
					marker = "*"
				}
				rows = append(rows, []string{view.Alias, view.NetworkName, view.UserName, view.UserID, view.Origin, marker})
			}
			return output.WriteTable(a.deps.Stdout, []string{"ALIAS", "NETWORK", "USER", "USER_ID", "ORIGIN", "DEFAULT"}, rows)
		},
	}
}

func (a *app) authSetDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use: "set-default <alias>", Short: "既定のアカウントを設定する", Args: exactArgs(1, "tn auth set-default <alias>"),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, path, err := a.loadConfig()
			if err != nil {
				return err
			}
			if _, ok := cfg.Accounts[args[0]]; !ok {
				return usagef("account %q not found; run `tn auth list`", args[0])
			}
			cfg.Default = args[0]
			if err := cfg.Save(path); err != nil {
				return err
			}
			format, err := a.format()
			if err != nil {
				return err
			}
			if format == output.FormatJSON {
				return output.WriteJSON(a.deps.Stdout, struct {
					Default string `json:"default"`
				}{args[0]})
			}
			fmt.Fprintf(a.deps.Stdout, "default account: %s\n", args[0])
			return nil
		},
	}
}

func (a *app) authRemoveCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use: "remove <alias>", Short: "保存済みアカウントを削除する", Args: exactArgs(1, "tn auth remove <alias>"),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, path, err := a.loadConfig()
			if err != nil {
				return err
			}
			if _, ok := cfg.Accounts[args[0]]; !ok {
				return usagef("account %q not found; run `tn auth list`", args[0])
			}
			if err := a.confirm(fmt.Sprintf("remove account %q", args[0]), yes); err != nil {
				return err
			}
			delete(cfg.Accounts, args[0])
			if cfg.Default == args[0] {
				cfg.Default = ""
			}
			if err := cfg.Save(path); err != nil {
				return err
			}
			format, err := a.format()
			if err != nil {
				return err
			}
			if format == output.FormatJSON {
				return output.WriteJSON(a.deps.Stdout, struct {
					Removed string `json:"removed"`
				}{args[0]})
			}
			fmt.Fprintf(a.deps.Stdout, "removed account: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "確認を省略する")
	return cmd
}

func (a *app) authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use: "status", Short: "現在のブラウザセッションを検証する", Args: exactArgs(0, "tn auth status"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := a.client()
			if err != nil {
				return err
			}
			user, err := client.Me(cmd.Context())
			if err != nil {
				return err
			}
			network, err := client.CurrentNetwork(cmd.Context())
			if err != nil {
				return err
			}
			view := struct {
				Authenticated bool   `json:"authenticated"`
				NetworkID     string `json:"network_id"`
				NetworkName   string `json:"network_name"`
				UserID        string `json:"user_id"`
				UserName      string `json:"user_name"`
			}{true, string(network.ID), network.Name, string(user.ID), user.Name()}
			format, err := a.format()
			if err != nil {
				return err
			}
			if format == output.FormatJSON {
				return output.WriteJSON(a.deps.Stdout, view)
			}
			fmt.Fprintf(a.deps.Stdout, "authenticated as %s in %s (user_id=%s, network_id=%s)\n", view.UserName, view.NetworkName, view.UserID, view.NetworkID)
			return nil
		},
	}
}

func (a *app) authGuideCmd() *cobra.Command {
	return &cobra.Command{Use: "guide", Short: "ブラウザセッションの登録手順を表示する", Args: exactArgs(0, "tn auth guide"), Run: func(_ *cobra.Command, _ []string) { fmt.Fprintln(a.deps.Stdout, authGuide) }}
}

func (a *app) printAuthAccount(view authAccountView) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, view)
	}
	fmt.Fprintf(a.deps.Stdout, "authenticated as %s in %s (alias=%s)\n", view.UserName, view.NetworkName, view.Alias)
	return nil
}

const authGuide = `1. 使用する Chrome プロファイルで Talknote を開き、対象アカウントでログインしていることを確認します。
2. DevTools の Network を開いて Talknote を再読み込みします。
3. *.company.talknote.com へのリクエストを右クリックし、Copy > Copy as cURL を選びます。
4. 次を実行します。

   tn auth login --from-clipboard --name work

または pbpaste | tn auth login --name work を実行します。保存する Cookie は TALKNOTE_SID2 だけです。`
