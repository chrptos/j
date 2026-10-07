# ADR-0002：Goによる性能学習基盤の技術選定

- 日付：2026-10-07
- 状態：採用（ADR-0001を置き換える）
- 関連Issue：#17

## 背景

- 実装言語をGoへ変更し、DB・APIの性能改善と障害対応を学ぶ。
- 商品検索・在庫減算の要件、API契約、DB構造、6種類の性能試験を引き継ぐ。

## 採用内容

| 項目 | 技術 | 理由 |
| --- | --- | --- |
| 言語・ビルド | Go・Go Modules | 標準ツールでビルドと依存管理を行う |
| HTTP | net/http・ServeMux | 標準ライブラリで2つのAPIを実装する |
| DB | PostgreSQL | SQL・実行計画・更新競合を実験する |
| DBアクセス・プール | pgx/v5・pgxpool | SQLとトランザクションを明示し、接続待ちを観測する |
| マイグレーション | golang-migrate/migrate | SQLファイルでスキーマ変更を再現する |
| テスト | testing・httptest・Testcontainers for Go | HTTPと実PostgreSQLの正常系・競合を検証する |
| 負荷試験 | k6 | 同じ条件で6種類の試験を実行する |
| メトリクス | prometheus/client_golang・Prometheus・Grafana | API・Goランタイム・接続プールを観測する |
| 詳細分析 | pprof | CPU・ヒープ・goroutine・待ちを分析する |
| 実験環境 | Docker Compose | DBと計測サービスの起動条件を揃える |

## 実装方針

- SQLと結果のマッピングは手書きする。トランザクションの開始・コミット・ロールバックを明示する。
- 入力検証とProblem Detailsの生成は共通処理にまとめる。
- HTTP処理時間・件数を計測し、Goランタイム指標とpgxpool.Statの接続・待ち指標を公開する。
- /metricsとヘルスチェックを用意する。pprofは詳細分析時に管理用ポートで有効にする。
- 初期構成はGo 1.26.5、pgx v5.11.0、PostgreSQL 18.3を使う。Go ModulesとDockerイメージで版を固定する。
- 計測・マイグレーション・DB統合テストの依存は、それぞれの実装時に版を固定する。

## 代償

- 入力検証・エラー変換・接続プールの計測を自分で実装する。
- DB統合テストにはDockerが必要になる。
- ローカルの測定だけでは複数ホストの可用性を評価できない。

## Decisions

- Goへの変更
  > **人間**：技術なのですが、golangで進めるように方向転換します。
  >
  > **人間**：OKです。進めてください。
- 個別ライブラリの選定はPR #18でレビュー・マージ済み。

## Refs

- [net/http](https://pkg.go.dev/net/http)
- [pgx・pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Testcontainers for Go](https://golang.testcontainers.org/modules/postgres/)
- [Prometheus Goクライアント](https://prometheus.io/docs/guides/go-application/)
- [pprof](https://pkg.go.dev/net/http/pprof)
