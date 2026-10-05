# ADR-0001：Javaによる性能学習基盤の技術選定

- 日付：2026-10-05
- 状態：採用（具体的なバージョンの固定は設計時に行う）
- 関連Issue：#11

## 要件・制約

- Javaを学びながら、DB・JVM・APIの性能改善と障害対応を実験する。
- 商品検索と在庫減算を題材に、同じ条件で改善前後を比較する。
- SQL、ロック、接続待ち、CPU、メモリ、GCを観測できる構成にする。
- ローカルで再現できる初期構成から始める。

## 選択肢

| 判断 | 選択肢 | 採用 |
| --- | --- | --- |
| 言語 | Java / TypeScript | Java |
| Web | Spring MVC / WebFlux | Spring MVC |
| ビルド | Maven / Gradle | Maven＋Wrapper |
| DBアクセス | JdbcClient / JPA・Hibernate | JdbcClient |
| DBテスト | 実PostgreSQL / 代替DB | TestcontainersのPostgreSQL |

## 採用内容と理由

| 項目 | 採用 | 理由 |
| --- | --- | --- |
| Java | Java 25 LTS | Java・JVMの学習基盤にする |
| Web | Spring Boot＋Spring MVC | DI・API・トランザクションと、スレッドを使う処理モデルを学ぶ |
| ビルド | Maven＋Wrapper | 環境間でビルド条件を揃える |
| DB | PostgreSQL | 実行計画・インデックス・更新競合を実験する |
| DBアクセス | JdbcClient | SQLと性能の関係を直接確認する |
| 接続プール | HikariCP | 接続数・接続待ちを観測する |
| スキーマ管理 | Flyway | テーブル・インデックス変更を再現する |
| テスト | JUnit＋Testcontainers | 実DBで正常系・競合時の整合性を確認する |
| 負荷試験 | k6 | 負荷条件をコードで管理する |
| メトリクス | Actuator・Micrometer＋Prometheus・Grafana | API・JVM・接続プールの変化を比較する |
| 詳細分析 | JFR・JDK Mission Control | CPU・メモリ・スレッドの原因を調べる |
| 実験環境 | Docker Compose | DBと計測環境を同じ条件で起動する |

## 代償

- JdbcClientではSQLと結果のマッピングを自分で書く。
- ORM固有の性能課題は、JPAを比較実験する段階で扱う。
- 実DBのテストと計測サービスには、実行時間・メモリ・コンテナ環境が必要になる。
- ローカル測定だけでは、複数ホストやクラウドの可用性を評価できない。
- バージョン互換性、教材との対応、コンテナの資源制限を別途確認する。

## Decisions

- Javaを採用
  > **人間**：javaの勉強も兼ねている
- 技術選定
  > **AI**：今回の主目的なら、JdbcClientから始めることを推します。
  >
  > **人間**：OKです。ADRとして残しておいてください。

## Refs

- [Spring Bootの対応環境](https://docs.spring.io/spring-boot/system-requirements.html)
- [SpringのSQLアクセス](https://docs.spring.io/spring-boot/reference/data/sql.html)
- [Springのメトリクス](https://docs.spring.io/spring-boot/reference/actuator/metrics.html)
- [TestcontainersのDBテスト](https://java.testcontainers.org/modules/databases/)
