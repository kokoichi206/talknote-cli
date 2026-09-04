package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/config"
	"github.com/kokoichi206/talknote-cli/internal/output"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

func (a *app) notesCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "notes", Aliases: []string{"note", "groups", "group"}, Short: "ノートと投稿を操作する", Args: exactArgs(0, "tn notes <command>"), RunE: subcommandRequired("tn notes")}
	cmd.AddCommand(a.notesListCmd(), a.notesReadCmd(), a.notesUnreadCmd(), a.notesSendCmd(), a.notesEditCmd(), a.notesDeleteCmd(), a.notesLikeCmd(true), a.notesLikeCmd(false), a.commentsCmd(), a.notesAliasesCmd())
	return cmd
}

func (a *app) notesListCmd() *cobra.Command {
	var filter string
	cmd := &cobra.Command{
		Use: "list", Short: "参加可能なノートを一覧表示する", Args: exactArgs(0, "tn notes list [--filter <substring>]"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := a.client()
			if err != nil {
				return err
			}
			groups, err := client.ListGroups(cmd.Context())
			if err != nil {
				return err
			}
			if filter != "" {
				filtered := groups[:0]
				needle := strings.ToLower(filter)
				for _, group := range groups {
					if strings.Contains(strings.ToLower(group.Data.Name), needle) {
						filtered = append(filtered, group)
					}
				}
				groups = filtered
			}
			return a.printGroups(groups)
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "", "ノート名の部分一致で絞り込む")
	return cmd
}

func (a *app) notesReadCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "read <note>", Short: "ノートの投稿を読む", Args: exactArgs(1, "tn notes read <note> [--limit N]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLimit(limit); err != nil {
				return err
			}
			noteID, err := a.resolveNote(args[0])
			if err != nil {
				return err
			}
			client, err := a.client()
			if err != nil {
				return err
			}
			feeds, err := client.ListFeeds(cmd.Context(), noteID, limit)
			if err != nil {
				return err
			}
			return a.printFeeds(feeds)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "取得件数")
	return cmd
}

func (a *app) notesUnreadCmd() *cobra.Command {
	return &cobra.Command{
		Use: "unread <note>", Short: "ノートの未読数を取得する", Args: exactArgs(1, "tn notes unread <note>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID, err := a.resolveNote(args[0])
			if err != nil {
				return err
			}
			client, err := a.client()
			if err != nil {
				return err
			}
			groups, err := client.ListGroups(cmd.Context())
			if err != nil {
				return err
			}
			for _, group := range groups {
				if string(group.Data.ID) == noteID {
					return a.printUnread(group.Reaction.YouUnreadCount)
				}
			}
			return fmt.Errorf("note %q not found", noteID)
		},
	}
}

func (a *app) notesSendCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use: "send <note> [message]", Short: "ノートへ投稿する", Args: rangeArgs(1, 2, "tn notes send <note> [message|-] [--file <path>]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID, err := a.resolveNote(args[0])
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
			feed, err := client.CreateFeed(cmd.Context(), noteID, message)
			if err != nil {
				return err
			}
			return a.printMutation("sent", "note", noteID, string(feed.Data.ID))
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "ファイルから本文を読む")
	return cmd
}

func (a *app) notesEditCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use: "edit <note> <post_id> [message]", Short: "自分の投稿を編集する", Args: rangeArgs(2, 3, "tn notes edit <note> <post_id> [message|-]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID, err := a.resolveNote(args[0])
			if err != nil {
				return err
			}
			if err := validateID(args[1], "post_id"); err != nil {
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
			if err := client.UpdateFeed(cmd.Context(), noteID, args[1], message); err != nil {
				return err
			}
			return a.printMutation("edited", "note", noteID, args[1])
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "ファイルから本文を読む")
	return cmd
}

func (a *app) notesDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use: "delete <note> <post_id>", Short: "自分の投稿を削除する", Args: exactArgs(2, "tn notes delete <note> <post_id> [--yes]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID, err := a.resolveNote(args[0])
			if err != nil {
				return err
			}
			if err := validateID(args[1], "post_id"); err != nil {
				return err
			}
			if err := a.confirm(fmt.Sprintf("delete post %s from note %s", args[1], noteID), yes); err != nil {
				return err
			}
			client, err := a.client()
			if err != nil {
				return err
			}
			if err := client.DeleteFeed(cmd.Context(), noteID, args[1]); err != nil {
				return err
			}
			return a.printMutation("deleted", "note", noteID, args[1])
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "確認を省略する")
	return cmd
}

func (a *app) notesLikeCmd(liked bool) *cobra.Command {
	name, action := "like", "liked"
	if !liked {
		name, action = "unlike", "unliked"
	}
	return &cobra.Command{
		Use: name + " <note> <post_id>", Short: "投稿のいいねを変更する", Args: exactArgs(2, "tn notes "+name+" <note> <post_id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			noteID, err := a.resolveNote(args[0])
			if err != nil {
				return err
			}
			if err := validateID(args[1], "post_id"); err != nil {
				return err
			}
			client, err := a.client()
			if err != nil {
				return err
			}
			if err := client.SetFeedLike(cmd.Context(), noteID, args[1], liked); err != nil {
				return err
			}
			return a.printMutation(action, "note", noteID, args[1])
		},
	}
}

func (a *app) commentsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "comments", Aliases: []string{"comment"}, Short: "投稿のコメントを操作する", Args: exactArgs(0, "tn notes comments <command>"), RunE: subcommandRequired("tn notes comments")}
	cmd.AddCommand(a.commentsListCmd(), a.commentsSendCmd(), a.commentsEditCmd(), a.commentsDeleteCmd(), a.commentsLikeCmd(true), a.commentsLikeCmd(false))
	return cmd
}

func (a *app) commentsListCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{Use: "list <note> <post_id>", Short: "コメントを一覧表示する", Args: exactArgs(2, "tn notes comments list <note> <post_id>"), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateLimit(limit); err != nil {
			return err
		}
		noteID, err := a.resolveNote(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "post_id"); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		comments, err := client.ListComments(cmd.Context(), noteID, args[1], limit)
		if err != nil {
			return err
		}
		return a.printComments(comments)
	}}
	cmd.Flags().IntVar(&limit, "limit", 100, "取得件数")
	return cmd
}

func (a *app) commentsSendCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{Use: "send <note> <post_id> [message]", Short: "コメントを投稿する", Args: rangeArgs(2, 3, "tn notes comments send <note> <post_id> [message|-]"), RunE: func(cmd *cobra.Command, args []string) error {
		noteID, err := a.resolveNote(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "post_id"); err != nil {
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
		comment, err := client.CreateComment(cmd.Context(), noteID, args[1], message)
		if err != nil {
			return err
		}
		return a.printMutation("sent", "comment", args[1], string(comment.Data.ID))
	}}
	cmd.Flags().StringVar(&file, "file", "", "ファイルから本文を読む")
	return cmd
}

func (a *app) commentsEditCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{Use: "edit <note> <post_id> <comment_id> [message]", Short: "自分のコメントを編集する", Args: rangeArgs(3, 4, "tn notes comments edit <note> <post_id> <comment_id> [message|-]"), RunE: func(cmd *cobra.Command, args []string) error {
		noteID, err := a.resolveNote(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "post_id"); err != nil {
			return err
		}
		if err := validateID(args[2], "comment_id"); err != nil {
			return err
		}
		message, err := a.readMessage(args, 3, file)
		if err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.UpdateComment(cmd.Context(), noteID, args[1], args[2], message); err != nil {
			return err
		}
		return a.printMutation("edited", "comment", args[1], args[2])
	}}
	cmd.Flags().StringVar(&file, "file", "", "ファイルから本文を読む")
	return cmd
}

func (a *app) commentsDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <note> <post_id> <comment_id>", Short: "自分のコメントを削除する", Args: exactArgs(3, "tn notes comments delete <note> <post_id> <comment_id> [--yes]"), RunE: func(cmd *cobra.Command, args []string) error {
		noteID, err := a.resolveNote(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "post_id"); err != nil {
			return err
		}
		if err := validateID(args[2], "comment_id"); err != nil {
			return err
		}
		if err := a.confirm(fmt.Sprintf("delete comment %s", args[2]), yes); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.DeleteComment(cmd.Context(), noteID, args[1], args[2]); err != nil {
			return err
		}
		return a.printMutation("deleted", "comment", args[1], args[2])
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "確認を省略する")
	return cmd
}

func (a *app) commentsLikeCmd(liked bool) *cobra.Command {
	name, action := "like", "liked"
	if !liked {
		name, action = "unlike", "unliked"
	}
	return &cobra.Command{Use: name + " <note> <post_id> <comment_id>", Short: "コメントのいいねを変更する", Args: exactArgs(3, "tn notes comments "+name+" <note> <post_id> <comment_id>"), RunE: func(cmd *cobra.Command, args []string) error {
		noteID, err := a.resolveNote(args[0])
		if err != nil {
			return err
		}
		if err := validateID(args[1], "post_id"); err != nil {
			return err
		}
		if err := validateID(args[2], "comment_id"); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.SetCommentLike(cmd.Context(), noteID, args[1], args[2], liked); err != nil {
			return err
		}
		return a.printMutation(action, "comment", args[1], args[2])
	}}
}

func (a *app) notesAliasesCmd() *cobra.Command {
	return &cobra.Command{Use: "aliases", Short: "リポジトリ設定のノート別名を表示する", Args: exactArgs(0, "tn notes aliases"), RunE: func(_ *cobra.Command, _ []string) error {
		project, err := a.loadProject()
		if err != nil {
			return err
		}
		if project == nil {
			return fmt.Errorf("no .config/%s found in this directory or its parents", config.ProjectConfigName)
		}
		return a.printAliases(project.Dir, "notes", project.Notes, project.NoteAliases())
	}}
}

func (a *app) printGroups(groups []talknote.Group) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Notes []talknote.Group `json:"notes"`
		}{groups})
	}
	rows := make([][]string, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, []string{string(group.Data.ID), output.Truncate(group.Data.Name, 40), fmt.Sprint(group.Reaction.YouUnreadCount), fmt.Sprint(group.Data.MemberCount)})
	}
	return output.WriteTable(a.deps.Stdout, []string{"NOTE_ID", "NAME", "UNREAD", "MEMBERS"}, rows)
}
