package cli

import (
	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/output"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

func (a *app) searchCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use: "search <keyword>", Short: "ノート投稿と DM を横断検索する", Args: exactArgs(1, "tn search <keyword> [--limit N]"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLimit(limit); err != nil {
				return err
			}
			client, err := a.client()
			if err != nil {
				return err
			}
			results, err := client.Search(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			return a.printSearchResults(results)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "取得件数")
	return cmd
}

func (a *app) printSearchResults(results []talknote.SearchResult) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Results []talknote.SearchResult `json:"results"`
		}{results})
	}
	rows := make([][]string, 0, len(results))
	for _, result := range results {
		if result.GroupFeed != nil {
			feed := result.GroupFeed.Data
			rows = append(rows, []string{"note", string(feed.GroupID), string(feed.ID), formatMillis(feed.CreatedAt), output.Truncate(feed.PostedUser.Name(), 20), output.Truncate(feed.Content, 70)})
		}
		if result.DirectMessage != nil {
			message := result.DirectMessage.Data
			rows = append(rows, []string{"dm", string(message.ThreadID), string(message.ID), formatMillis(message.CreatedAt), output.Truncate(message.PostedUser.Name(), 20), output.Truncate(message.Content, 70)})
		}
	}
	return output.WriteTable(a.deps.Stdout, []string{"TYPE", "PARENT_ID", "ID", "CREATED_AT", "FROM", "CONTENT"}, rows)
}
