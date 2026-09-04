package cli

import (
	"fmt"
	"time"

	"github.com/kokoichi206/talknote-cli/internal/output"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

func (a *app) printFeeds(feeds []talknote.Feed) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Posts []talknote.Feed `json:"posts"`
		}{feeds})
	}
	if format == output.FormatText {
		for _, feed := range feeds {
			fmt.Fprintf(a.deps.Stdout, "--- %s | %s | id=%s\n%s\n", formatMillis(feed.Data.CreatedAt), output.StripControl(feed.Data.PostedUser.Name()), feed.Data.ID, output.StripControl(feed.Data.Content))
		}
		return nil
	}
	rows := make([][]string, 0, len(feeds))
	for _, feed := range feeds {
		rows = append(rows, []string{string(feed.Data.ID), formatMillis(feed.Data.CreatedAt), output.Truncate(feed.Data.PostedUser.Name(), 20), output.Truncate(feed.Data.Content, 60), fmt.Sprint(feed.Data.CommentCount), fmt.Sprint(feed.Data.LikeCount)})
	}
	return output.WriteTable(a.deps.Stdout, []string{"POST_ID", "CREATED_AT", "FROM", "CONTENT", "COMMENTS", "LIKES"}, rows)
}

func (a *app) printComments(comments []talknote.Comment) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Comments []talknote.Comment `json:"comments"`
		}{comments})
	}
	if format == output.FormatText {
		for _, comment := range comments {
			fmt.Fprintf(a.deps.Stdout, "--- %s | %s | id=%s\n%s\n", formatMillis(comment.Data.CreatedAt), output.StripControl(comment.Data.PostedUser.Name()), comment.Data.ID, output.StripControl(comment.Data.Content))
		}
		return nil
	}
	rows := make([][]string, 0, len(comments))
	for _, comment := range comments {
		rows = append(rows, []string{string(comment.Data.ID), formatMillis(comment.Data.CreatedAt), output.Truncate(comment.Data.PostedUser.Name(), 20), output.Truncate(comment.Data.Content, 70), fmt.Sprint(comment.Data.LikeCount)})
	}
	return output.WriteTable(a.deps.Stdout, []string{"COMMENT_ID", "CREATED_AT", "FROM", "CONTENT", "LIKES"}, rows)
}

func (a *app) printMessages(messages []talknote.Message) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Messages []talknote.Message `json:"messages"`
		}{messages})
	}
	if format == output.FormatText {
		for _, message := range messages {
			fmt.Fprintf(a.deps.Stdout, "--- %s | %s | id=%s\n%s\n", formatMillis(message.Data.CreatedAt), output.StripControl(message.Data.PostedUser.Name()), message.Data.ID, output.StripControl(message.Data.Content))
		}
		return nil
	}
	rows := make([][]string, 0, len(messages))
	for _, message := range messages {
		rows = append(rows, []string{string(message.Data.ID), formatMillis(message.Data.CreatedAt), output.Truncate(message.Data.PostedUser.Name(), 20), output.Truncate(message.Data.Content, 70)})
	}
	return output.WriteTable(a.deps.Stdout, []string{"MESSAGE_ID", "CREATED_AT", "FROM", "CONTENT"}, rows)
}

func (a *app) printUnread(unread int) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			UnreadCount int `json:"unread_count"`
		}{unread})
	}
	fmt.Fprintln(a.deps.Stdout, unread)
	return nil
}

func (a *app) printMutation(action, kind, parentID, id string) error {
	result := struct {
		Action   string `json:"action"`
		Kind     string `json:"kind"`
		ParentID string `json:"parent_id"`
		ID       string `json:"id"`
	}{action, kind, parentID, id}
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, result)
	}
	fmt.Fprintf(a.deps.Stdout, "%s: kind=%s parent_id=%s id=%s\n", action, kind, parentID, id)
	return nil
}

func (a *app) printAction(action, kind string) error {
	result := struct {
		Action string `json:"action"`
		Kind   string `json:"kind"`
	}{action, kind}
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, result)
	}
	fmt.Fprintf(a.deps.Stdout, "%s: kind=%s\n", action, kind)
	return nil
}

func (a *app) printAliases(dir, kind string, aliases map[string]string, names []string) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Dir     string            `json:"dir"`
			Aliases map[string]string `json:"aliases"`
			Kind    string            `json:"kind"`
		}{dir, aliases, kind})
	}
	rows := make([][]string, 0, len(aliases))
	for _, name := range names {
		rows = append(rows, []string{name, aliases[name]})
	}
	return output.WriteTable(a.deps.Stdout, []string{"ALIAS", "ID"}, rows)
}

func formatMillis(milliseconds int64) string {
	if milliseconds == 0 {
		return ""
	}
	return time.UnixMilli(milliseconds).Local().Format(time.RFC3339)
}
