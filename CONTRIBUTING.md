# 開発ルール

## ブランチ戦略

Git Flowを採用する。

| ブランチ | 役割 | 分岐元 | マージ先 |
| --- | --- | --- | --- |
| `main` | リリース済みの状態 | — | — |
| `develop` | 次のリリースに向けた変更の統合 | — | — |
| `feature/...` | 機能追加・文書追加・通常の修正 | `develop` | `develop` |
| `release/...` | リリース準備 | `develop` | `main`・`develop` |
| `hotfix/...` | リリース済み内容の緊急修正 | `main` | `main`・`develop` |

## 作業ブランチの命名

通常の作業では、Issueを作成してから`develop`を元にブランチを作る。

Issueは共通テンプレートを使い、目的・対応内容・完了条件を記載する。

```text
feature/#<Issue番号>_<動作>_<内容>
```

動作には`add`・`update`・`fix`・`remove`を使う。内容は英語の小文字とハイフンで簡潔に表す。

```text
feature/#1_add_ec-requirements
feature/#2_update_order-rules
feature/#3_fix_stock-validation
feature/#4_remove_unused-docs
```

リリース準備と緊急修正では、次の形式を使う。

```text
release/<バージョン>
hotfix/#<Issue番号>_fix_<内容>
```

`#`を含むブランチ名は、シェルで引用符で囲む。

```sh
git switch -c 'feature/#1_add_ec-requirements' develop
```

## コミット・PR

- コミットには、そのIssueに関係する変更を含める。
- コミットメッセージに変更内容とIssue番号を記載する。例：`docs: add EC requirements (#1)`。
- 通常の作業は`develop`向けにPRを作成する。
- PRにはIssue・What・Why・How・Before / After・Otherを記載する。
- Otherにはテスト方法と結果、未確認事項などを記載する。
- PRの本文には共通テンプレートを使う。
- リリース時は`release/...`を`main`と`develop`へマージし、`main`にバージョンタグを付ける。
- 緊急修正は`hotfix/...`を`main`と`develop`へ反映する。進行中のリリースがある場合は、リリースにも修正を反映する。
