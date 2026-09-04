package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/output"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

func (a *app) tasksCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "tasks", Aliases: []string{"task"}, Short: "タスクを操作する", Args: exactArgs(0, "tn tasks <command>"), RunE: subcommandRequired("tn tasks")}
	cmd.AddCommand(a.tasksListCmd(), a.tasksCreateCmd(), a.tasksCompleteCmd(), a.tasksRevertCmd(), a.tasksDeleteCmd())
	return cmd
}

func (a *app) tasksListCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{Use: "list", Short: "自分のタスクを一覧表示する", Args: exactArgs(0, "tn tasks list [--status incomplete|complete]"), RunE: func(cmd *cobra.Command, _ []string) error {
		apiStatus := "INCOMPLETE"
		switch status {
		case "incomplete":
		case "complete":
			apiStatus = "COMPLETE"
		default:
			return usagef("invalid --status %q (incomplete|complete)", status)
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		tasks, err := client.ListTasks(cmd.Context(), apiStatus)
		if err != nil {
			return err
		}
		return a.printTasks(tasks)
	}}
	cmd.Flags().StringVar(&status, "status", "incomplete", "incomplete または complete")
	return cmd
}

func (a *app) tasksCreateCmd() *cobra.Command {
	var assigneeIDs []string
	var deadline string
	cmd := &cobra.Command{Use: "create <content>", Short: "タスクを作成する", Args: exactArgs(1, "tn tasks create <content> --assignee <user_id> [--deadline YYYY-MM-DD]"), RunE: func(cmd *cobra.Command, args []string) error {
		if len(assigneeIDs) == 0 {
			return usagef("--assignee <user_id> is required")
		}
		ids := make([]talknote.ID, len(assigneeIDs))
		for i, id := range assigneeIDs {
			parsed, err := strconv.ParseUint(id, 10, 64)
			if err != nil || parsed == 0 {
				return usagef("invalid --assignee %q: user_id must be a positive integer", id)
			}
			ids[i] = talknote.ID(id)
		}
		parsedDeadline, err := parseDeadline(deadline)
		if err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.CreateTask(cmd.Context(), args[0], ids, parsedDeadline); err != nil {
			return err
		}
		return a.printAction("created", "task")
	}}
	cmd.Flags().StringSliceVar(&assigneeIDs, "assignee", nil, "担当者の user_id（複数指定可）")
	cmd.Flags().StringVar(&deadline, "deadline", "", "期限: YYYY-MM-DD または Unix ミリ秒")
	return cmd
}

func parseDeadline(value string) (*int64, error) {
	if value == "" {
		return nil, nil
	}
	deadline, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err == nil {
		milliseconds := deadline.UnixMilli()
		return &milliseconds, nil
	}
	if len(value) != 13 {
		return nil, usagef("invalid --deadline %q: use YYYY-MM-DD or 13-digit Unix milliseconds", value)
	}
	milliseconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || milliseconds <= 0 {
		return nil, usagef("invalid --deadline %q: use YYYY-MM-DD or 13-digit Unix milliseconds", value)
	}
	return &milliseconds, nil
}

func (a *app) tasksCompleteCmd() *cobra.Command {
	return &cobra.Command{Use: "complete <task_id>", Aliases: []string{"done"}, Short: "タスクを完了する", Args: exactArgs(1, "tn tasks complete <task_id>"), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateID(args[0], "task_id"); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.CompleteTask(cmd.Context(), args[0]); err != nil {
			return err
		}
		return a.printMutation("completed", "task", "", args[0])
	}}
}

func (a *app) tasksRevertCmd() *cobra.Command {
	return &cobra.Command{Use: "revert <task_id>", Aliases: []string{"reopen"}, Short: "完了したタスクを未完了へ戻す", Args: exactArgs(1, "tn tasks revert <task_id>"), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateID(args[0], "task_id"); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.RevertTask(cmd.Context(), args[0]); err != nil {
			return err
		}
		return a.printMutation("reverted", "task", "", args[0])
	}}
}

func (a *app) tasksDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <task_id>", Short: "タスクを削除する", Args: exactArgs(1, "tn tasks delete <task_id> [--yes]"), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateID(args[0], "task_id"); err != nil {
			return err
		}
		if err := a.confirm(fmt.Sprintf("delete task %s", args[0]), yes); err != nil {
			return err
		}
		client, err := a.client()
		if err != nil {
			return err
		}
		if err := client.DeleteTask(cmd.Context(), args[0]); err != nil {
			return err
		}
		return a.printMutation("deleted", "task", "", args[0])
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "確認を省略する")
	return cmd
}

func (a *app) printTasks(tasks []talknote.Task) error {
	format, err := a.format()
	if err != nil {
		return err
	}
	if format == output.FormatJSON {
		return output.WriteJSON(a.deps.Stdout, struct {
			Tasks []talknote.Task `json:"tasks"`
		}{tasks})
	}
	rows := make([][]string, 0, len(tasks))
	for _, task := range tasks {
		deadline := ""
		if task.Deadline != nil {
			deadline = formatMillis(*task.Deadline)
		}
		rows = append(rows, []string{string(task.ID), task.Status, output.Truncate(task.DisplayAssignee.Name(), 20), deadline, output.Truncate(task.Content, 60)})
	}
	return output.WriteTable(a.deps.Stdout, []string{"TASK_ID", "STATUS", "ASSIGNEE", "DEADLINE", "CONTENT"}, rows)
}
