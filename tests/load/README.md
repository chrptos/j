# 通常負荷テスト

k6・Python 3・Docker Composeを使う。計測の動作確認とAPIスモークテストを終えてから実行する。

## データ準備

リポジトリのルートで実行する。`docker-compose`を使う環境では読み替える。

```sh
export LOAD_PROJECT=j-load-100k-uniform
export PRODUCTS=100000 SEED=42 STOCK=1000000 DISTRIBUTION=uniform
load_compose() {
  POSTGRES_DB=j_load API_HOST_PORT=18082 PROMETHEUS_HOST_PORT=19092 GRAFANA_HOST_PORT=13002 \
    docker compose --env-file .env.local -p "$LOAD_PROJECT" "$@"
}
load_psql() {
  load_compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"' sh "$@"
}
mkdir -p /tmp/j-load-results
load_compose up -d db
load_compose run --build --rm migrate up
load_psql -v "products=$PRODUCTS" -v "seed=$SEED" -v "distribution=$DISTRIBUTION" -v "stock=$STOCK" \
  < tests/load/seed.sql > /tmp/j-load-results/seed-100k-uniform.txt
load_compose --profile monitoring up --build -d
curl --retry 10 --retry-connrefused --retry-delay 1 -fsS http://127.0.0.1:18082/health/ready
```

- seedは空の専用DB`j_load`にのみ投入する。カテゴリ100件、商品と在庫を同じトランザクションで作成し、ANALYZEする。
- 商品名はIDとseed由来のグループ00〜99。`グループ-00`の部分一致は約1%。価格はseed由来の0〜100,000円。
- `uniform`は100カテゴリへ均等、`hot`はカテゴリ1へ80%を割り当てる。実際の分布・価格範囲・ヒット件数はseed出力へ保存する。
- 100万件は`PRODUCTS=1000000`。集中分布は`DISTRIBUTION=hot`。条件ごとにLOAD_PROJECTとseed出力先を変え、別ボリュームで比較する。

## 実行

```sh
WORKLOAD=search OUTPUT_DIR=/tmp/j-load-results/100k-uniform-search tests/load/run.sh
```

- 標準は10 RPS。2分のウォームアップ→在庫リセット→30秒で0から10 RPSへ増加→5分維持を3回実施する。
- 1 iterationで1要求。負荷増加はramping-arrival-rate、一定負荷はconstant-arrival-rate。再試行しない。
- 検索は`WORKLOAD=search`、減算は`decrement`、90%検索・10%減算は`mixed`。
- 減算は数量1。`ACCESS=uniform`は商品を分散、`hot`は減算5件中4件を商品1へ集中する。
- 同じseed・データ・負荷条件を使う。STOCKは全予定減算数を超える値を指定し、売り切れを防ぐ。
- 試験終了後に全区間の成功減算件数とDB在庫を照合し、ready確認後に在庫を初期化する。

| 設定 | 標準 | 内容 |
| --- | --- | --- |
| RPS | 10 | 到着率 |
| SEARCH_QUERY | グループ-00、ID順、limit20、offset0 | URLエンコード済み検索条件 |
| VUS | 100 | 各シナリオで事前確保するVU数 |
| BASE_URL | http://127.0.0.1:18082 | API接続先 |
| K6_BIN | k6 | k6実行ファイル |
| ENV_FILE | .env.local | Compose環境設定 |
| OUTPUT_DIR | /tmp/j-load-results/UTC日時 | 結果保存先 |

検索条件の例：

```sh
SEARCH_QUERY='categoryId=1&minPrice=1000&maxPrice=50000&sort=price_asc&limit=20&offset=1000' \
  WORKLOAD=search OUTPUT_DIR=/tmp/j-load-results/category-price-offset1000 tests/load/run.sh
```

スクリプト自体の短時間確認は`WARMUP_SECONDS=2 RAMP_SECONDS=2 HOLD_SECONDS=5`で行える。基準測定とは区別する。

## 記録

- 各試行の設定・生データ・集計JSON、SQL統計の前後、在庫照合、コンテナ資源推移を保存する。
- k6のCPU・RSSを1秒ごとに記録する。CPUはpsによるプロセス累積平均、RSSはKiB。OS・Docker VM資源・版・コンテナ制限も保存する。ホストのCPU・メモリとDockerの割当設定も記録する。
- 集計は応答完了時刻で増加区間・測定区間・終了後の処理待ちを分ける。成功RPSは測定区間の成功応答数÷維持秒数。
- 測定区間のHTTP・コード別件数、API・結果別と全体のp95/p99、システム・通信障害率、業務拒否率、タイムアウト率を記録する。
- p95/p99は生データのHTTP処理時間から線形補間で算出する。接続失敗時の時間とタイムアウト率も併せて確認する。
- dropped_iterations、チェック失敗、要求の未完了、在庫不一致があれば非ゼロで終了する。応答を失う障害試験には別途監査が必要。
- 初期10 RPSの結果を確認して通常負荷を決める。改善前後は同じ条件で比較する。

## 停止

```sh
load_compose --profile monitoring down
```

データは保持される。再実行時はseedを省略する。

## Refs

- [試験設計](../../docs/design/measurement.md)
- [計測手順](../../monitoring/README.md)
- [k6到着率方式](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/ramping-arrival-rate/)
