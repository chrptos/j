# APIスモークテスト

k6とDocker Composeを用意する（検証版：k6 2.3.0）。負荷試験前のAPI動作確認として、1 VU・1 iterationで12要求を順に送る。大量データ・ウォームアップは使わない。

## 準備

リポジトリのルートで実行する。`.env.local`はREADMEの手順で用意する。以下の関数は専用のComposeプロジェクト・DB・ポートを使う。`docker-compose`を使う環境では読み替える。

```sh
smoke_compose() {
  POSTGRES_DB=j_smoke API_HOST_PORT=18080 docker compose --env-file .env.local -p j-smoke "$@"
}
smoke_compose up -d db
smoke_compose run --build --rm migrate up
smoke_compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < tests/smoke/seed.sql
smoke_compose up --build -d api
curl --retry 10 --retry-connrefused --retry-delay 1 -fsS http://127.0.0.1:18080/health/ready
```

- seedは空の`j_smoke` DBにのみ投入する。商品3件・カテゴリ2件・在庫10／10／0を用意する。

## 実行

```sh
smoke_compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < tests/smoke/reset.sql
k6 run tests/smoke/api.js
smoke_compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < tests/smoke/verify.sql
```

- 再実行前にresetで在庫を10／10／0へ戻す。試験は並行実行しない。
- 検索・ページネーション・空配列・減算・入力不正・商品不存在・在庫不足を確認する。
- 24チェックがすべて成功するとk6が終了コード0を返す。要求の自動再試行は行わない。
- verifyはDBの最終在庫7／10／0を確認し、違えば非ゼロで終了する。
- 接続先の変更は`BASE_URL`で指定する。標準は`http://127.0.0.1:18080`。

## 停止

```sh
smoke_compose down
```

DBデータは保持される。次回はseedを省略し、APIの起動とresetから実行する。

## Refs

- [試験設計](../../docs/design/measurement.md)
- [k6 checksと終了判定](https://grafana.com/docs/k6/latest/using-k6/checks/)
