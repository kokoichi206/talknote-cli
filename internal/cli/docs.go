package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *app) docsCmd() *cobra.Command {
	return &cobra.Command{Use: "docs", Short: "CLI リファレンスを表示する", Args: exactArgs(0, "tn docs"), Run: func(_ *cobra.Command, _ []string) { fmt.Fprint(a.deps.Stdout, referenceDocs) }}
}

const referenceDocs = `# Talknote CLI reference

tn は Chrome でログイン中の Talknote セッションを TALKNOTE_SID2 Cookie 経由で使用します。

## 認証

    tn auth guide
    tn auth login --from-clipboard --name work
    pbpaste | tn auth login --name work
    tn auth status --output json
    tn auth list --output json
    tn auth set-default work
    tn auth remove work --yes

Cookie は ~/.config/talknote-cli/accounts.json に 0600 で保存します。出力には表示しません。
複数アカウントは --account で選択できます。

## ノート

    tn notes list [--filter TEXT] --output json
    tn notes read <note_id|alias> [--limit N] --output json
    tn notes unread <note_id|alias>
    tn notes send <note_id|alias> "本文" --output json
    tn notes edit <note_id|alias> <post_id> "本文"
    tn notes delete <note_id|alias> <post_id> --yes
    tn notes like <note_id|alias> <post_id>
    tn notes unlike <note_id|alias> <post_id>
    tn notes aliases --output json

    tn notes comments list <note_id|alias> <post_id> [--limit N] --output json
    tn notes comments send <note_id|alias> <post_id> "本文"
    tn notes comments edit <note_id|alias> <post_id> <comment_id> "本文"
    tn notes comments delete <note_id|alias> <post_id> <comment_id> --yes
    tn notes comments like <note_id|alias> <post_id> <comment_id>
    tn notes comments unlike <note_id|alias> <post_id> <comment_id>

## ダイレクトメッセージ

    tn dms list [--filter TEXT] --output json
    tn dms read <dm_id|alias> [--limit N] --output json
    tn dms unread <dm_id|alias>
    tn dms send <dm_id|alias> "本文"
    tn dms edit <dm_id|alias> <message_id> "本文"
    tn dms delete <dm_id|alias> <message_id> --yes
    tn dms aliases --output json

## 検索とタスク

    tn search <keyword> [--limit N] --output json
    tn tasks list [--status incomplete|complete] --output json
    tn tasks create "内容" --assignee <user_id> [--deadline YYYY-MM-DD|Unixミリ秒]
    tn tasks complete <task_id>
    tn tasks revert <task_id>
    tn tasks delete <task_id> --yes

tasks create は Talknote Web API が ID を返さないため、成功時は action と kind のみを出力します。
作成後の task_id は tn tasks list --output json で確認してください。

## リポジトリ別名

.config/talknote-cli.json に次を保存します。

    {
      "notes": {"main": "12345"},
      "dms": {"owner": "67890"}
    }

## 入出力

- 本文は引数、- による標準入力、--file のいずれかで渡せます。
- --output json|table|text。TTY では table、パイプでは json が既定です。
- 終了コードは 0=成功、1=API/実行時エラー、2=使い方エラーです。
- 投稿・DM・コメント・リアクション・編集は認証した本人として実行されます。
- delete と auth remove は対話確認または --yes が必要です。

## エージェント連携と更新

    tn agent init --scope repo
    tn agent init --scope user
    tn update --dry-run
    tn update

agent init は tn の利用手順を Claude Code skill としてインストールします。
update は GitHub Release の checksums.txt で検証してから実行ファイルを更新します。
`
