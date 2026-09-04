package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/output"
)

//go:embed SKILL.md
var agentSkill string

func (a *app) agentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "agent", Short: "Install agent instructions for this CLI", Args: exactArgs(0, "tn agent <command>"), RunE: subcommandRequired("tn agent")}
	cmd.AddCommand(a.agentInitCmd())
	return cmd
}

func (a *app) agentInitCmd() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Install a Claude Code skill",
		Args:  exactArgs(0, "tn agent init [--scope repo|user]"),
		RunE: func(_ *cobra.Command, _ []string) error {
			var root string
			switch scope {
			case "repo":
				root = a.deps.WorkDir
				if root == "" {
					root = "."
				}
			case "user":
				root = a.deps.Home
				if root == "" {
					var err error
					root, err = os.UserHomeDir()
					if err != nil {
						return err
					}
				}
			default:
				return usagef("unknown scope %q (repo|user)", scope)
			}
			path := filepath.Join(root, ".claude", "skills", "talknote-cli", "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(agentSkill), 0o644); err != nil {
				return err
			}
			format, err := a.format()
			if err != nil {
				return err
			}
			if format == output.FormatJSON {
				return output.WriteJSON(a.deps.Stdout, struct {
					Path   string `json:"path"`
					Scope  string `json:"scope"`
					Status string `json:"status"`
				}{path, scope, "installed"})
			}
			fmt.Fprintln(a.deps.Stdout, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "repo", "install scope: repo or user")
	return cmd
}
