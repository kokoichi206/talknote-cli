// Package cli は cobra コマンド定義と入出力を担う。
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kokoichi206/talknote-cli/internal/config"
	"github.com/kokoichi206/talknote-cli/internal/output"
	"github.com/kokoichi206/talknote-cli/internal/talknote"
)

type Deps struct {
	Stdout         io.Writer
	Stderr         io.Writer
	Stdin          io.Reader
	ConfigPath     string
	StdoutIsTTY    bool
	StdinIsTTY     bool
	Version        string
	CurrentVersion string
	Home           string
	WorkDir        string
	Clipboard      func() (string, error)
	NewClient      func(account config.Account) *talknote.Client
}

type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

func usagef(format string, values ...any) error {
	return &UsageError{Err: fmt.Errorf(format, values...)}
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var usageErr *UsageError
	if errors.As(err, &usageErr) || strings.HasPrefix(err.Error(), "unknown command") {
		return 2
	}
	return 1
}

type app struct {
	deps Deps

	accountFlag string
	outputFlag  string

	project       *config.Project
	projectLoaded bool
	outputFormat  output.Format
	formatParsed  bool
}

func New(deps Deps) *cobra.Command {
	a := &app{deps: deps}
	root := &cobra.Command{
		Use:           "tn",
		Short:         "Talknote CLI for humans and AI agents",
		Long:          "tn operates Talknote as the selected signed-in browser session.\nRun `tn docs` for the complete reference.",
		Version:       deps.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return a.parseFormat()
		},
	}
	root.SetOut(deps.Stdout)
	root.SetErr(deps.Stderr)
	root.SetIn(nopReadCloser{deps.Stdin})
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &UsageError{Err: err} })
	root.PersistentFlags().StringVar(&a.accountFlag, "account", "", "account alias (default: configured default or the only account)")
	root.PersistentFlags().StringVarP(&a.outputFlag, "output", "o", "", "output format: json|table|text (default: table on TTY, json otherwise)")
	root.AddCommand(a.authCmd(), a.notesCmd(), a.dmsCmd(), a.searchCmd(), a.tasksCmd(), a.docsCmd(), a.agentCmd(), a.updateCmd())
	return root
}

type nopReadCloser struct{ io.Reader }

func (nopReadCloser) Close() error { return nil }

func (a *app) format() (output.Format, error) {
	if err := a.parseFormat(); err != nil {
		return "", err
	}
	return a.outputFormat, nil
}

func (a *app) parseFormat() error {
	if a.formatParsed {
		return nil
	}
	if a.outputFlag == "" {
		a.outputFormat = output.Default(a.deps.StdoutIsTTY)
		a.formatParsed = true
		return nil
	}
	format, err := output.Parse(a.outputFlag)
	if err != nil {
		return &UsageError{Err: err}
	}
	a.outputFormat = format
	a.formatParsed = true
	return nil
}

func (a *app) configPath() (string, error) {
	if a.deps.ConfigPath != "" {
		return a.deps.ConfigPath, nil
	}
	return config.Path()
}

func (a *app) loadConfig() (*config.Config, string, error) {
	path, err := a.configPath()
	if err != nil {
		return nil, "", err
	}
	if warning := config.PermWarning(path); warning != "" {
		fmt.Fprintln(a.deps.Stderr, warning)
	}
	cfg, err := config.Load(path)
	return cfg, path, err
}

func (a *app) newClient(account config.Account) *talknote.Client {
	if a.deps.NewClient != nil {
		return a.deps.NewClient(account)
	}
	return talknote.New(account.Origin, account.SID)
}

func (a *app) client() (*talknote.Client, error) {
	cfg, _, err := a.loadConfig()
	if err != nil {
		return nil, err
	}
	_, account, err := cfg.Resolve(a.accountFlag)
	if err != nil {
		return nil, &UsageError{Err: err}
	}
	return a.newClient(account), nil
}

func (a *app) loadProject() (*config.Project, error) {
	if a.projectLoaded {
		return a.project, nil
	}
	workDir := a.deps.WorkDir
	if workDir == "" {
		workDir = "."
	}
	home := a.deps.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}
	project, err := config.FindProject(workDir, home)
	if err != nil {
		return nil, err
	}
	a.project = project
	a.projectLoaded = true
	return project, nil
}

func (a *app) resolveNote(arg string) (string, error) {
	return a.resolveID(arg, func(project *config.Project) map[string]string { return project.Notes })
}

func (a *app) resolveDM(arg string) (string, error) {
	return a.resolveID(arg, func(project *config.Project) map[string]string { return project.DMs })
}

func (a *app) resolveID(arg string, values func(*config.Project) map[string]string) (string, error) {
	project, err := a.loadProject()
	if err != nil {
		return "", err
	}
	if project != nil {
		if id, ok := values(project)[arg]; ok {
			return id, nil
		}
	}
	if id, err := strconv.ParseUint(arg, 10, 64); err != nil || id == 0 {
		return "", usagef("%q is neither a configured alias nor a positive numeric ID", arg)
	}
	return arg, nil
}

func (a *app) readMessage(args []string, index int, file string) (string, error) {
	if file != "" && len(args) > index {
		return "", usagef("message argument and --file cannot be used together")
	}
	var message string
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		message = string(data)
	} else if len(args) <= index {
		return "", usagef("message is required (argument, `-` for stdin, or --file)")
	} else if args[index] == "-" {
		data, err := io.ReadAll(a.deps.Stdin)
		if err != nil {
			return "", err
		}
		message = string(data)
	} else {
		message = args[index]
	}
	message = strings.TrimRight(message, "\r\n")
	if message == "" {
		return "", usagef("message is empty")
	}
	return message, nil
}

func (a *app) confirm(action string, yes bool) error {
	if yes {
		return nil
	}
	if !a.deps.StdinIsTTY {
		return usagef("refusing to %s without confirmation; pass --yes in non-interactive mode", action)
	}
	fmt.Fprintf(a.deps.Stderr, "%s? [y/N]: ", action)
	line, err := bufio.NewReader(a.deps.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if answer := strings.ToLower(strings.TrimSpace(line)); answer == "y" || answer == "yes" {
		return nil
	}
	return fmt.Errorf("aborted")
}

func exactArgs(count int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != count {
			return usagef("expected %d argument(s): %s", count, usage)
		}
		return nil
	}
}

func rangeArgs(minimum, maximum int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) < minimum || len(args) > maximum {
			return usagef("expected %d-%d argument(s): %s", minimum, maximum, usage)
		}
		return nil
	}
}

func subcommandRequired(command string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		return usagef("subcommand is required: %s <command>", command)
	}
}

func validateLimit(limit int) error {
	if limit <= 0 {
		return usagef("--limit must be a positive integer")
	}
	return nil
}

func validateID(value, name string) error {
	if id, err := strconv.ParseUint(value, 10, 64); err != nil || id == 0 {
		return usagef("invalid %s %q: must be a positive integer", name, value)
	}
	return nil
}
