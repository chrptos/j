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
- Composeが`.env.local`を読み込み、Goはcaarlos0/envで型変換・必須チェックする。
- 設定は`.env.local`で変更する。タイムアウトは`1s`・`250ms`形式。公開ポートを変えた場合はcurlのポートも合わせる。`.env.example`は共有用、`.env.local`はGit管理対象外。変更後はComposeの`up -d`で反映する。
- DBはCompose内部から接続する。認証情報はローカル開発用。既存ボリュームのDB名・ユーザー・パスワードは環境変数だけでは変更されない。
- 停止は`docker compose --env-file .env.local down`。DBデータはボリュームに保持する。

## マイグレーション

```sh
docker compose --env-file .env.local run --build --rm migrate up
```

商品・カテゴリ・在庫のテーブルを作成する。適用済みの変更は再実行しない。

直前の1件を取り消す場合は、`up`を`down`へ変更する。初期マイグレーションの取り消しは3テーブルとデータを削除する。

- SQLは`migrations/`で管理し、ビルド時に実行ファイルへ埋め込む。
- 商品と在庫の作成は同じトランザクションで行う。外部キーだけでは在庫行の存在は保証されない。

## 商品検索

```sh
curl -fsS 'http://localhost:8080/products?keyword=Go&sort=price_asc&limit=20&offset=0'
```

- カテゴリ・価格の絞り込みは`categoryId`・`minPrice`・`maxPrice`で指定する。
- 対象がない場合は`items: []`。試験データは別途投入する。
- 接続取得・DB処理の上限は`DB_ACQUIRE_TIMEOUT`・`DB_QUERY_TIMEOUT`で設定する。

## 在庫減算

```sh
curl -i -X POST 'http://localhost:8080/products/1/stock/decrements' \
  -H 'Content-Type: application/json' -d '{"quantity":1}'
```

- 商品と在庫の試験データを投入して実行する。
- 成功時は商品IDと更新時点の残数を返す。商品なしは404、在庫不足は409。
- 再送は別の減算になるため、自動再試行しない。

## テスト

Go 1.26.5で実行する。

```sh
go test -race ./...
go vet ./...
# Dockerが必要なDB統合テスト
go test -race -tags=integration ./internal/product ./migrations
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
