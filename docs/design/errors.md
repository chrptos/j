# 共通エラー設計

状態：レビュー用の提案 / 関連Issue：#13

以下はレビュー用の提案。HTTPの分類とアプリケーションのコードを揃え、負荷試験でも結果を分ける。

## レスポンス形式

- RFC 9457のProblem Detailsを使い、Content-Typeを`application/problem+json`にする。
- typeはエラー種類を表す固定URIとする。例の`urn:j:problem:insufficient-stock`は識別子であり、取得用URLではない。
- statusは実際のHTTPステータスと一致させる。
- codeは集計用の固定コード、detailは利用者向けの説明、traceIdはログとの照合用とする。
- instanceは発生したエラーを識別するURIとし、traceIdから生成する。
- 入力不正ではerrorsにfield・code・messageを返す。SQL・接続先・スタックトレース・入力値は返さない。

```json
{
  "type": "urn:j:problem:insufficient-stock",
  "title": "在庫不足",
  "status": 409,
  "detail": "指定された数量の在庫がありません",
  "instance": "urn:uuid:ca326a98-8885-4b23-a456-7dd3ab5c27c4",
  "code": "INSUFFICIENT_STOCK",
  "traceId": "ca326a98-8885-4b23-a456-7dd3ab5c27c4"
}
```

## 分類

| 状況 | HTTP | code | 試験時の分類 |
| --- | --- | --- | --- |
| JSON・型・範囲・検索条件が不正 | 400 | INVALID_ARGUMENT | 入力不正 |
| 対象商品なし | 404 | PRODUCT_NOT_FOUND | 業務上の拒否 |
| URLなし | 404 | ENDPOINT_NOT_FOUND | 入力不正 |
| 許可されないHTTPメソッド | 405 | METHOD_NOT_ALLOWED | 入力不正 |
| 未対応のContent-Type | 415 | UNSUPPORTED_MEDIA_TYPE | 入力不正 |
| 在庫不足 | 409 | INSUFFICIENT_STOCK | 業務上の拒否 |
| 利用者単位のレート制限（導入時） | 429 | RATE_LIMITED | 負荷制御による拒否 |
| サービス全体の同時実行上限（導入時） | 503 | SERVICE_OVERLOADED | システム側の拒否 |
| DB接続不可・プール取得タイムアウト | 503 | DATABASE_UNAVAILABLE | システム障害 |
| DBクエリ・ロック待ちタイムアウト | 503 | DATABASE_TIMEOUT | システム障害 |
| デッドロック・直列化失敗で処理を中断 | 503 | TRANSACTION_ABORTED | システム障害 |
| 予期しない例外・データ整合性違反 | 500 | INTERNAL_ERROR | システム障害 |

- 入力検証後に在庫更新を行う。障害を在庫不足として扱わない。
- 商品が存在するのに在庫行が欠けている場合は整合性違反とする。
- 405ではAllowヘッダーを返す。
- 429・過負荷の503では、待機時間を設定できる場合にRetry-Afterを返す。初期版でレート制限・負荷制御を実装するかは別の設計判断とする。
- アプリが返せるエラーは共通形式にする。接続断・プロキシ側エラー・強制終了では形式や応答を保証できない。
- DBタイムアウトの具体的な時間と例外の対応はDB設計で固定する。

## 実装方針・ログ

- 入力不正と業務上の拒否は専用のエラー・結果型で表す。
- net/httpの共通応答処理でProblem Detailsへ変換する。入力検証とルーティング由来の400・404・405・415も揃える。
- DBエラーはSQLSTATE・接続エラー・contextの期限切れで原因を分類し、判定できないものは500にする。
- ログにtraceId、API、HTTPステータス、code、処理時間を記録する。
- 想定内の拒否はスタックトレースを付けず、障害は原因を記録する。負荷試験でログが性能を歪めないよう、出力量も固定する。
- リクエスト全体や接続情報はログへ出さない。
- HTTPステータスとcodeで集計する。traceIdはメトリクスのラベルに使わない。

## Refs

- [RFC 9457：Problem Details](https://www.rfc-editor.org/rfc/rfc9457.html)
- [RFC 9110：HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html)
