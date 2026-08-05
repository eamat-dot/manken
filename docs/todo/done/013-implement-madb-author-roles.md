# MADBのcreator役割と寄与者変換を実装

## 状態

2026年8月1日完了。

## 背景

- 現状はAgent名が1件でもあれば全Agent名を `Normalized.Authors` と
  `Sources[0].Values.Authors` に設定する
- `M292132` では著者と解説者のAgent名が区別されず、解説者も著者に含まれる
- 現在の `removeCreatorRoles` は先頭の角括弧を繰り返し除去するため、
  `[作画][田辺節雄]` のような実データでは人物名まで削除する
- TODO007でcreator役割、Agentとの関係、角括弧表記、順序を調査し、変換規則を確定した

## 目的

- MADBのcreator文字列から、確定した役割だけを共通 `ContributorRole` へ変換する
- 主要な創作者だけを `Normalized.Authors` に含める
- 役割を確定できる人物を `Normalized.Contributors` に設定する
- 未知、不正、対応不能な値を推測せず取得元値またはRaw responseに残す

## 根拠

- [`../../research/007-madb-author-roles.md`](../../research/007-madb-author-roles.md)
- [`../../spec.md`](../../spec.md)
- [`../../pkg/madb/spec.md`](../../pkg/madb/spec.md)

## 非目的

- 人名の異体字、別名、読み、空白差による同一人物判定
- creator文字列とAgent URIの推測による対応付け
- 未知のMADB役割の自動分類
- 取得元にない寄与者順序の推測
- 著者名検索
- MADB以外の取得元の変換実装
- ONIXコードを公開APIへ追加すること

## 前提・制約

- Go 1.26.0と標準ライブラリだけを使用する
- `api` を共通型の唯一の定義箇所とし、`madb` は必要な型と定数を再公開する
- `SourceBookValues` の公開フィールドは追加しない
- MADBのSPARQL、検索条件、ページング、カーソルを変更しない
- creator文字列とAgent名は `sourceBook` で別々に保持する
- 既存のJSON構造を維持するが、`authors` の内容と `contributors` の出力は
  確定仕様に従って変更する

## 対象範囲

### 対象

- `api.ContributorRole` の共通役割定数
- `madb` パッケージからの共通役割定数の再公開
- MADB役割から共通役割への明示的な対応
- creator文字列の保守的な解析
- `Authors`、`Contributors`、取得元の `Authors` の組み立て
- 単体テスト、実サービス統合テスト、仕様との整合確認

### 対象外

- `Contributor` または `SourceBookValues` の型変更
- Agent IDまたはAgent URLの公開
- 役割対応表の外部設定化
- 他の書籍APIとの寄与者統合

## 処理方針

1. `api` に確定済みの `ContributorRole` 定数を追加し、`madb` から再公開する
2. creator文字列とAgent名の両方を現在どおり取得し、別々に集約する
3. creator文字列がある場合は、変更前の全creator文字列を取得元の `Authors` に残す
4. creator文字列がない場合だけ、Agent名を取得元の `Authors` と
   `Normalized.Authors` に使用する
5. 先頭役割を対応表で完全一致させ、既知の役割だけを変換する
6. 2つ目以降の角括弧を役割として反復除去せず、最初の既知役割より後ろを
   人物名として解析する
7. `・` 区切りの複合役割は、全構成要素が既知の場合だけ複数役割へ変換する
8. 主要な創作者役割を持つ人物だけを `Authors` に含める
9. 既知役割と人物名を確定できた場合だけ `Contributors` を作る
10. creatorの文字列昇順を安定化順として使い、名前と役割を完全一致で重複除去する

## 実施項目

### 公開API仕様

- [x] `api.ContributorRole` に確定済みの11定数を追加する
- [x] `madb` から同じ役割定数を再公開する
- [x] 各公開定数へ日本語のGo Docコメントを追加する

### MADB変換

- [x] MADB役割から共通役割への非公開対応表を実装する
- [x] 既知の単一役割と複合役割を解析する
- [x] 最初の既知役割より後ろを人物名として安全に解析する
- [x] 役割なしcreatorとAgentだけのフォールバックを実装する
- [x] 主要な創作者役割から `Authors` を組み立てる
- [x] 同じ人物の既知役割を1つの `Contributor` へまとめる
- [x] creator文字列を優先して取得元の `Authors` に残す
- [x] `removeCreatorRoles` を新しい解析処理へ置き換える
- [x] 未知、不正、空の値を推測で正規化しない

### 単体テスト

- [x] `[著]`、`[原作]`、`[作画]` をAuthorsとContributorsへ変換する
- [x] `[解説]`、`[監修]`、`[訳]`、`[編集]`、デザイン系をContributorsだけへ変換する
- [x] `[原作・監修]` と `[作・画]` を複数役割へ変換する
- [x] 未知の要素を含む複合役割を全体として正規化しない
- [x] `[作画][田辺節雄]`、`[作][ジョン・ブッセマン]`、
  `[画][葛飾]北斎`、`[原作][スタン・リー]` から人物名を失わない
- [x] `[[著]]近江のこ` を推測で正規化しない
- [x] 役割なしcreatorをAuthorsだけへ設定する
- [x] creatorだけ、Agentだけ、両方、どちらもないケースを確認する
- [x] `M292132` 相当の入力で佐々木倫子だけがAuthorsに入り、
  藤原新也はcommentatorとしてContributorsに残る
- [x] 同じ人物の複数役割と重複bindingを1件へまとめる
- [x] creator文字列を取得元値へ変更せず残す
- [x] AuthorsとContributorsの安定化順を確認する

### 統合確認

- [x] 実サービスで `M292132` を含む検索結果を確認する
- [x] creatorだけ、Agentだけ、役割なし、不正角括弧の代表IDを可能な範囲で確認する
- [x] 実サービスの確認結果を `docs/pkg/madb/spec.md` の確認例へ追記する

### ドキュメント

- [x] 実装後の型名、定数名、対応表が確定仕様と一致することを確認する
- [x] 調査文書と仕様書へ実装上の新しい判断を混在させない
- [x] TODO完了時にこの文書を `docs/todo/done/` へ移動する

## 検証

- [x] `go test -v ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] `golangci-lint run` を実行する
- [x] `go test -tags=integration ./madb` を実行する
- [x] デモCLIで `authors`、`contributors`、取得元の `authors` を確認する
- [x] 外部モジュールから `madb` だけをimportし、役割定数を参照できることを確認する
- [x] `git diff --check` を実行する

## 受け入れ条件

- [x] `M292132` で解説者がAuthorsへ混入しない
- [x] 既知のcreator役割と人物名がContributorsへ保持される
- [x] 角括弧付き人名を役割として削除しない
- [x] 未知、不正、対応不能な値を推測で正規化しない
- [x] creator文字列が取得元値またはRaw responseから失われない
- [x] Agent参照の有無にかかわらず確定規則を適用できる
- [x] 結果順が決定的であり、意味のある取得元順として扱われていない
- [x] 通常テスト、ビルド、lint、実サービス統合テストが成功する

## リスク・懸念

- 対応表にない実在役割はContributorsへ出力されない
- 既知役割の表記揺れが増えた場合は、実データを確認して対応表を更新する必要がある
- 人物名自体に角括弧が必要な例では、人物名候補の解析規則を再検討する可能性がある
- Authorsの内容が変わるため、現在の出力に依存する利用側には挙動変更となる

## 未決事項

- なし

## 実装結果

- 共通役割11種類を `api` に追加し、`madb` から再公開した
- creator文字列を保守的に解析し、主要な創作者とその他の寄与者を分離した
- creator文字列がない場合だけAgent名をAuthorsへ使用するよう変更した
- 未知、不正、対応不能な値を推測で正規化せず、取得元値へ残した
- `M292132` の著者と解説者を単体テスト、実サービス統合テスト、デモCLIで確認した
- 通常テスト、ビルド、lint、外部利用と同じ `madb_test` および一時的な外部モジュールからの
  import検証が成功した
