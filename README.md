# talknote-cli

[日本語](README.ja.md) | English

`tn` is a Talknote CLI for humans and AI agents. It operates as the user signed in to Talknote in Chrome. It does not require Talknote's public API or an OAuth application.

- Notes, posts, comments, likes, editing, and deletion
- Direct-message threads, sending, editing, and deletion
- Search across note posts and direct messages
- Task listing, creation, completion, reopening, and deletion
- Multiple accounts, JSON output, and repository-local aliases

## Install

```sh
go install github.com/kokoichi206/talknote-cli/cmd/tn@latest
```

To install from a GitHub Release, the following script verifies the archive against `checksums.txt` before placing it at `~/.local/bin/tn`.

```sh
curl -fsSL https://raw.githubusercontent.com/kokoichi206/talknote-cli/main/scripts/install.sh | sh
```

## Authentication

Open Talknote in the intended Chrome profile and verify the signed-in account. In DevTools Network, right-click a Talknote request and choose Copy > Copy as cURL.

```sh
tn auth login --from-clipboard --name work
tn auth status
```

You can also pipe the copied request:

```sh
pbpaste | tn auth login --name work
```

Only the `TALKNOTE_SID2` cookie is stored in `~/.config/talknote-cli/accounts.json` with mode 0600. It is never included in CLI output.

## Usage

```sh
tn notes list --filter "daily" --output json
tn notes read 12345 --output json
tn notes send 12345 "Daily report"
tn notes comments list 12345 98765 --output json

tn dms list --filter "Tominaga" --output json
tn dms read 67890 --output json
tn dms send 67890 "Acknowledged"

tn search "project" --output json
tn tasks list --output json

tn docs
tn update --dry-run
```

Mutations run as the authenticated user. Deletion requires interactive confirmation or `--yes`.

## License

[MIT](LICENSE)
