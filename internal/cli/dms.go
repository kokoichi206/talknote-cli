package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/config"
	"github.com/kokoichi206/talknote-cli/internal/output"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

func (a *app) dmsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "dms", Aliases: []string{"dm", "threads", "thread"}, Short: "ダイレクトメッセージを操作する", Args: exactArgs(0, "tn dms <command>"), RunE: subcommandRequired("tn dms")}
	cmd.AddCommand(a.dmsListCmd(), a.dmsReadCmd(), a.dmsUnreadCmd(), a.dmsSendCmd(), a.dmsEditCmd(), a.dmsDeleteCmd(), a.dmsAliasesCmd())
	return cmd
}

func (a *app) dmsListCmd() *cobra.Command {
	var filter string
	cmd := &cobra.Command{Use: "list", Short: "DM スレッドを一覧表示する", Args: exactArgs(0, "tn dms list [--filter TEXT]"), RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := a.client()
		if err != nil {
			return err
		}
		threads, err := client.ListThreads(cmd.Context())
		if err != nil {
			return err
		}
		if filter != "" {
			filtered := threads[:0]
			needle := strings.ToLower(filter)
			for _, thread := range threads {
				if strings.Contains(strings.ToLower(thread.Session.DisplayTitle), needle) || strings.Contains(strings.ToLower(memberNames(thread.Data.Member10)), needle) {
					filtered = append(filtered, thread)
				}
			}
			threads = filtered
		}
		return a.printThreads(threads)
	}}
	cmd.Flags().StringVar(&filter, "filter", "", "表示名またはメンバー名で絞り込む")
	return cmd
}

func (a *app) dmsReadCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{Use: "read <dm>", Short: "DM のメッセージを読む", Args: exactArgs(1, "tn dms read <dm> [--limit N]"), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateLimit(limit); err != nil {
			return err
		}
		dmID, err := a.resolveDM(args[0])
		if err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		messages, err := client.ListMessages(cmd.Context(), dmID, limit)
		if err != nil {
			return err
		}
		return a.printMessages(messages)
	}}
	cmd.Flags().IntVar(&limit, "limit", 20, "取得件数")
	return cmd
}

func (a *app) dmsUnreadCmd() *cobra.Command {
	return &cobra.Command{Use: "unread <dm>", Short: "DM の未読数を取得する", Args: exactArgs(1, "tn dms unread <dm>"), RunE: func(cmd *cobra.Command, args []string) error {
		dmID, err := a.resolveDM(args[0])
		if err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		threads, err := client.ListThreads(cmd.Context())
		if err != nil {
			return err
		}
		for _, thread := range threads {
			if string(thread.Data.ID) == dmID {
				return a.printUnread(thread.Reaction.YouUnreadCount)
			}
		}
		return fmt.Errorf("dm %q not found", dmID)
	}}
}

func (a *app) dmsSendCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{Use: "send <dm> [message]", Short: "DM を送信する", Args: rangeArgs(1, 2, "tn dms send <dm> [message|-]"), RunE: func(cmd *cobra.Command, args []string) error {
		dmID, err := a.resolveDM(args[0])
		if err != nil {
			return err
		}
		message, err := a.readMessage(args, 1, file)
		if err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		created, err := client.CreateMessage(cmd.Context(), dmID, message)
		if err != nil {
			return err
		}
		return a.printMutation("sent", "dm", dmID, string(created.Data.ID))
	}}
	cmd.Flags().StringVar(&file, "file", "", "ファイルから本文を読む")
	return cmd
}

func (a *app) dmsEditCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{Use: "edit <dm> <message_id> [message]", Short: "自分の DM を編集する", Args: rangeArgs(2, 3, "tn dms edit <dm> <message_id> [message|-]"), RunE: func(cmd *cobra.Command, args []string) error {
		dmID, err := a.resolveDM(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "message_id"); err != nil {
			return err
		}
		message, err := a.readMessage(args, 2, file)
		if err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.UpdateMessage(cmd.Context(), dmID, args[1], message); err != nil {
			return err
		}
		return a.printMutation("edited", "dm", dmID, args[1])
	}}
	cmd.Flags().StringVar(&file, "file", "", "ファイルから本文を読む")
	return cmd
}

func (a *app) dmsDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <dm> <message_id>", Short: "自分の DM を削除する", Args: exactArgs(2, "tn dms delete <dm> <message_id> [--yes]"), RunE: func(cmd *cobra.Command, args []string) error {
		dmID, err := a.resolveDM(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "message_id"); err != nil {
			return err
		}
		if err := a.confirm(fmt.Sprintf("delete message %s from dm %s", args[1], dmID), yes); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.DeleteMessage(cmd.Context(), dmID, args[1]); err != nil {
			return err
		}
		return a.printMutation("deleted", "dm", dmID, args[1])
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "確認を省略する")
	return cmd
}

func (a *app) dmsAliasesCmd() *cobra.Command {
	return &cobra.Command{Use: "aliases", Short: "リポジトリ設定の DM 別名を表示する", Args: exactArgs(0, "tn dms aliases"), RunE: func(_ *cobra.Command, _ []string) error {
		project, err := a.loadProject()
		if err != nil {
			return err
		}
		if project == nil {
			return fmt.Errorf("no .config/%s found in this directory or its parents", config.ProjectConfigName)
		}
		return a.printAliases(project.Dir, "dms", project.DMs, project.DMAliases())
	}}
}

func (a *app) printThreads(threads []talknote.Thread) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Threads []talknote.Thread `json:"threads"`
		}{threads})
	}
	rows := make([][]string, 0, len(threads))
	for _, thread := range threads {
		rows = append(rows, []string{string(thread.Data.ID), output.Truncate(thread.Session.DisplayTitle, 36), output.Truncate(memberNames(thread.Data.Member10), 50), fmt.Sprint(thread.Reaction.YouUnreadCount), formatMillis(thread.Data.LastPostedAt)})
	}
	return output.WriteTable(a.deps.Stdout, []string{"DM_ID", "TITLE", "MEMBERS", "UNREAD", "LAST_POSTED_AT"}, rows)
}

func memberNames(users []talknote.User) string {
	names := make([]string, 0, len(users))
	for _, user := range users {
		names = append(names, user.Name())
	}
	return strings.Join(names, ", ")
}
