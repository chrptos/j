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
- コミットメッセージは`<種別>: <変更内容> (#<Issue番号>)`の形式にする。種別は英語、変更内容は日本語で記載する。
- 種別には`feat`（機能追加）、`fix`（不具合修正）、`docs`（文書変更）、`refactor`（リファクタリング）、`test`（テスト変更）、`chore`（その他の保守作業）などを使う。
- 例：`docs: EC要件定義書を追加 (#1)`、`feat: 注文作成機能を追加 (#3)`、`fix: キャンセル時の在庫の二重返却を修正 (#4)`。
- 通常の作業は`develop`向けにPRを作成する。
- PRにはIssue・What・Why・How・Decisions・Evidence・Other・Refsを記載する。各項目の本文は箇条書きにし、箇条書きの項目間には空行を入れない。
- Decisionsには判断に関わる会話を、箇条書きの項目内に発言者付きの引用ブロックで記載する。実際の発言を引用し、省略は「…」、要約は要約と明記する。Slackなどの会話には参照可能な会話リンクを添える。未合意の提案は合意済みとして記載しない。
- Evidenceには画像・動画・入出力例・図などを記載する。必要に応じてUI・API・状態遷移のBefore / Afterを小見出しで整理し、該当する内容がなければ省略する。
- Otherにはテスト方法と結果、未確認事項などを記載する。
- Refsには参考にした資料やリンクを記載する。
- PRの本文には共通テンプレートを使う。
- リリース時は`release/...`を`main`と`develop`へマージし、`main`にバージョンタグを付ける。
- 緊急修正は`hotfix/...`を`main`と`develop`へ反映する。進行中のリリースがある場合は、リリースにも修正を反映する。
