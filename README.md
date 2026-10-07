# j

Goで商品検索・在庫減算を実装し、性能改善を学ぶプロジェクト。

## 起動

DockerとComposeを用意して実行する。`docker-compose`を使う環境ではコマンドを読み替える。

```sh
docker compose up --build -d
curl --retry 10 --retry-connrefused --retry-delay 1 -fsS http://localhost:8080/health/live
curl -fsS http://localhost:8080/health/ready
```

- `/health/live`：APIの生存確認。正常時は200。
- `/health/ready`：DB接続確認。正常時は200、接続不可は503。
- DBはCompose内部から接続する。認証情報はローカル開発用。
- 停止は`docker compose down`。DBデータはボリュームに保持する。

## テスト

Go 1.26.5で実行する。

```sh
go test -race ./...
go vet ./...
```

## 起動確認

```sh
docker compose stop db
curl -i http://localhost:8080/health/ready
curl -i http://localhost:8080/health/live
docker compose start db
curl -fsS http://localhost:8080/health/ready
```

DB停止時はreadyが503、liveが200。DB起動後はreadyが200に戻る。

開発ルールは[CONTRIBUTING.md](CONTRIBUTING.md)、技術選定は[ADR-0002](docs/adr/0002-go-performance-learning-stack.md)を参照。
