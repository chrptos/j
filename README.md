# j

Goで商品検索・在庫減算を実装し、性能改善を学ぶプロジェクト。

## 起動

DockerとComposeを用意して実行する。`docker-compose`を使う環境ではコマンドを読み替える。

```sh
cp .env.example .env.local
docker compose --env-file .env.local up --build -d
curl --retry 10 --retry-connrefused --retry-delay 1 -fsS http://localhost:8080/health/live
curl -fsS http://localhost:8080/health/ready
```

- `/health/live`：APIの生存確認。正常時は200。
- `/health/ready`：DB接続確認。正常時は200、接続不可は503。
- 設定は`.env.local`で変更する。タイムアウトは`1s`・`250ms`形式。公開ポートを変えた場合はcurlのポートも合わせる。`.env.example`は共有用、`.env.local`はGit管理対象外。変更後はComposeの`up -d`で反映する。
- DBはCompose内部から接続する。認証情報はローカル開発用。既存ボリュームのDB名・ユーザー・パスワードは環境変数だけでは変更されない。
- 停止は`docker compose --env-file .env.local down`。DBデータはボリュームに保持する。

## テスト

Go 1.26.5で実行する。

```sh
go test -race ./...
go vet ./...
```

## 起動確認

```sh
docker compose --env-file .env.local stop db
curl -i http://localhost:8080/health/ready
curl -i http://localhost:8080/health/live
docker compose --env-file .env.local start db
curl -fsS http://localhost:8080/health/ready
```

DB停止時はreadyが503、liveが200。DB起動後はreadyが200に戻る。

開発ルールは[CONTRIBUTING.md](CONTRIBUTING.md)、技術選定は[ADR-0002](docs/adr/0002-go-performance-learning-stack.md)を参照。
