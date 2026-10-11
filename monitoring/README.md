# 計測

APIの`/metrics`をPrometheusが5秒間隔で取得する。DBは専用ユーザー`j_metrics`でpostgres_exporterから取得する。

## 起動

`.env.example`の計測サービス用設定を`.env.local`へ追加し、リポジトリのルートで実行する。

```sh
docker compose --env-file .env.local --profile monitoring up --build -d
```

- Prometheus：<http://127.0.0.1:9090/targets>。`api`・`postgres`がUPになることを確認する。
- Grafana：<http://127.0.0.1:3000/d/j-performance>。`.env.local`のユーザー・パスワードでログインする。
- ポートは`PROMETHEUS_HOST_PORT`・`GRAFANA_HOST_PORT`で変更する。ログイン情報は初回作成時に適用される。
- DBはpg_stat_statementsを事前ロードして起動する。setupが既存DBにも拡張とpg_monitor権限を設定する。
- 通常の起動では計測サービスを起動しない。APIのメトリクスとDBの事前ロードは共通とする。

## 指標

| 対象 | 内容 |
| --- | --- |
| API | ルート・メソッド・HTTPステータス別の件数と時間、p95・p99 |
| Go | ヒープ・goroutine・GC停止時間・CPU |
| pgxpool | 使用中・アイドル・最大接続数、取得待ち回数・時間、取得キャンセル |
| DB | 接続状態、ロック待ち、デッドロック、ブロック読込、SQL呼出数・実行時間 |

- 商品ID・検索語・traceIdはラベルに使わない。`/metrics`への要求はAPI指標に数えない。
- 価格や商品IDを含むSQL本文はラベルにしない。SQLはqueryidで識別する。
- pgxpoolの待ち時間は成功した取得の累計。タイムアウトはキャンセル回数で確認する。
- APIのp95・p99はヒストグラムによる推定。k6の応答時間も併せて記録する。
- SQLダッシュボードはexporterが選ぶ最大100件。厳密な前後差分はstatements.sqlの結果を保存して比較する。
- コンテナ資源は`docker stats`で別途記録する。

## スモークテスト

[APIスモークテスト](../tests/smoke/README.md)の専用環境でも、`smoke_compose --profile monitoring up --build -d`で計測サービスを起動できる。

- k6の24チェックとDB照合を確認する。
- Grafanaで検索・減算・400・404・409の件数と時間が表示されることを確認する。
- API／DBの指標取得が成功し、`pg_up=1`・`pg_exporter_last_scrape_error=0`になることを確認する。

## 記録・停止

```sh
# 負荷試験の前後で保存し、queryidごとの差分を取る
mkdir -p /tmp/j-measurements
docker compose --env-file .env.local exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -A -F , -f -' < monitoring/statements.sql > /tmp/j-measurements/statements-before.csv
# 試験後は出力先をstatements-after.csvに変更する
docker stats --no-stream --format json > /tmp/j-measurements/containers.json
docker compose --env-file .env.local --profile monitoring down
```

データはボリュームに保持する。

## Refs

- [試験設計](../docs/design/measurement.md)
- [client_golang](https://github.com/prometheus/client_golang)
- [postgres_exporter](https://github.com/prometheus-community/postgres_exporter)
- [pg_stat_statements](https://www.postgresql.org/docs/current/pgstatstatements.html)
- [Grafana provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)
