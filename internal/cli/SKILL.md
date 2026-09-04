---
name: talknote-cli
description: Talknote のノート、コメント、DM、検索、タスクを tn CLI で操作する。
---

# Talknote CLI

`tn docs` で完全なリファレンスを確認する。

- 機械処理では常に `--output json` を付ける。
- ノート ID は `tn notes list --output json` で調べる。
- DM スレッド ID は `tn dms list --output json` で調べる。
- リポジトリ別名は `tn notes aliases --output json` と `tn dms aliases --output json` で確認する。
- 投稿、DM、コメント、いいね、編集、タスク変更は認証した本人として実行される。対象と内容をユーザーに確認してから実行する。
- 削除には `--yes` が必要。確認なしに付けない。
- 終了コードは 0=成功、1=API/実行時エラー、2=使い方エラー。
- `tn update --dry-run` で更新の有無だけを確認できる。
- skill 自体の再配置は `tn agent init --scope repo|user` を使う。
