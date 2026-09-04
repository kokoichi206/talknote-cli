# talknote-cli

[English](README.md) | 日本語

`tn` は、人間と AI agent の双方で使う Talknote CLI です。Chrome でログイン中の Talknote セッションを使い、本人として操作します。公開 API の有効化や OAuth アプリは不要です。

- ノートの一覧、投稿、コメント、いいね、編集、削除
- DM スレッドの一覧、メッセージ送信、編集、削除
- ノート投稿と DM の横断検索
- タスクの一覧、作成、完了、再開、削除
- 複数アカウント、JSON 出力、リポジトリ別名

## インストール

```sh
go install github.com/kokoichi206/talknote-cli/cmd/tn@latest
```

GitHub Release から入れる場合は、次のスクリプトがアーカイブを `checksums.txt` で検証してから `~/.local/bin/tn` へ配置します。

```sh
curl -fsSL https://raw.githubusercontent.com/kokoichi206/talknote-cli/main/scripts/install.sh | sh
```

## 認証

使う Chrome プロファイルで Talknote を開き、対象アカウントでログインしていることを確認します。DevTools の Network で Talknote のリクエストを右クリックし、Copy > Copy as cURL を選んでください。

```sh
tn auth login --from-clipboard --name work
tn auth status
```

または次のように標準入力へ渡せます。

```sh
pbpaste | tn auth login --name work
```

`TALKNOTE_SID2` Cookie だけを `~/.config/talknote-cli/accounts.json` に 0600 で保存し、CLI 出力には表示しません。

## 使い方

```sh
tn notes list --filter "日報" --output json
tn notes read 12345 --output json
tn notes send 12345 "日報です"
tn notes comments list 12345 98765 --output json

tn dms list --filter "冨永" --output json
tn dms read 67890 --output json
tn dms send 67890 "確認しました"

tn search "案件名" --output json
tn tasks list --output json

tn docs
tn update --dry-run
```

投稿・DM・コメント・リアクション・編集・タスク変更は、認証した本人として実行されます。削除には対話確認または `--yes` が必要です。

## ライセンス

[MIT](LICENSE)
