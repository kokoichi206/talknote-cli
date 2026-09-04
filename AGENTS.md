# talknote-cli — agent guide

This repository builds `tn`, a Talknote CLI for humans and AI agents.

## Using the CLI

- Run `tn docs` for the complete reference.
- Pass `--output json` for stable machine-readable output.
- IDs are strings in CLI JSON, even when Talknote sends numeric JSON IDs.
- Mutations act as the authenticated human account. Confirm target and content before running them.
- Exit codes: 0 success, 1 API/runtime error, 2 usage error.

## Working on the codebase

- `internal/talknote`: signed-in browser session Web API contract; no CLI concerns.
- `internal/cli`: cobra commands and rendering.
- `internal/config`: accounts and repository aliases.
- `internal/output`: JSON/table/text rendering.
- Unit tests use `httptest`. Live verification must use the exact user-specified Chrome profile and account.
- If command behavior changes, update `internal/cli/docs.go`, `internal/cli/SKILL.md`, and the README files.
