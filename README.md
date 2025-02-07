# Henceforth

指定された時間・指定されたチャンネルに指定されたメッセージを送る traQ Bot 「Scheduled-Messenger」を LLM で強化したバージョン

## 環境変数

- `DEV_MODE`
  開発モード (default: false)
- `BOT_ID`
  ボットの ID (default: "")
- `VERIFICATION_TOKEN`
  Bot へのリクエストの認証トークン (default: "")
- `BOT_ACCESS_TOKEN`
  Bot からのアクセストークン (default: "")
- `LOG_CHAN_ID`
  エラーログを送信するチャンネルの ID (default: "")
- `NS_MARIADB_HOSTNAME`
  DB のホスト (default: "mariadb")
- `NS_MARIADB_DATABASE`
  DB の DB 名 (default: "SchMes")
- `NS_MARIADB_USER`
  DB のユーザー名 (default: "root")
- `NS_MARIADB_PASSWORD`
  DB のパスワード (default: "password")

## ローカルで動かすときのサンプル

シェルスクリプトを使いましょう。
ディレクトリ内に`env.sh`を作り、下のコードをコピーして環境変数を設定した後、`sh env.sh`で実行します。

```sh *.sh
#!/bin/sh

export DEV_MODE=
export BOT_ID=
export VERIFICATION_TOKEN=
export BOT_ACCESS_TOKEN=
export LOG_CHAN_ID=
export NS_MARIADB_HOSTNAME=
export NS_MARIADB_DATABASE=
export NS_MARIADB_USER=
export NS_MARIADB_PASSWORD=
export GEMINI_API_KEY=

go run ./*.go
```

MariaDB が`{NS_MARIADB_HOSTNAME}:3306`(デフォルトのポート)で立っていることを確認してください。
ポート`8080`でサーバーが立つので、`localhost:8080`のエンドポイントにリクエストを送り、レスポンスを確かめてください。
