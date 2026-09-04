package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/kokoichi206/talknote-cli/internal/cli"
	"github.com/kokoichi206/talknote-cli/internal/version"
)

func main() {
	cmd := cli.New(cli.Deps{
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Stdin:          os.Stdin,
		StdoutIsTTY:    term.IsTerminal(int(os.Stdout.Fd())),
		StdinIsTTY:     term.IsTerminal(int(os.Stdin.Fd())),
		Version:        version.String(),
		CurrentVersion: version.Version,
	})
	if err := cmd.Execute(); err != nil {
		var usageErr *cli.UsageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr, "error:", usageErr)
		} else {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(cli.ExitCode(err))
	}
}
