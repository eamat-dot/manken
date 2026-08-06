# openBDの既存誤変換を除去する

## 背景

- openBDはISBN参照用の補助取得元であり、取得元が明示しない構造をタイトルやCollectionから推測して共通化しない
- 現行変換には、タイトル・巻数の分解、Collectionの分類、役割と刊行日の意味の取り違えが含まれる
- 022で、追加項目の実装ではなく既存の明らかな誤変換だけを除去する方針を確定した

## 目的

- openBDが明示したタイトル、人名、読み、役割、刊行日の意味を変更せずに共通モデルへ移す
- 取得元が明示しない並列タイトル、作品巻数、シリーズ、レーベル、役割を推測して設定しない

## 非目的

- 新しいONIX項目や書影情報を追加取得・変換すること
- openBDを主要な検索・正規化用データ取得元へ拡張すること
- MADBの変換、人物読み、クエリを変更すること
- 共通型を新設または変更すること

## 前提・制約

- 023が完了し、`TitleReading` と `Contributor.Reading` を利用できること
- 取得元が同じONIX要素で明示した値だけを共通型へ設定し、文字列や辞書から意味を推測しない
- `summary.cover` の既存処理は維持し、価格、商品階層 `PartNumber`、CollectionSequence、SupportingResourceは追加しない
- 未対応ONIX役割の人物は失わず、役割を空にして `Contributor` へ保持する

## 対象範囲

### 対象

- 推測的なタイトル・並列タイトル・巻数分解を廃止し、タイトル全体と読み全体を採用する
- Collectionを `Series` または `Imprints` へ変換しない
- Collectionの `PartNumber` と `summary.volume` を作品巻数に使用しない
- 人名の分割、書換え、生年などの注記削除を行わない
- `A38` を `original_creator` へ変換しない
- 未対応ONIX役割を近い共通役割へ丸めない
- `PublishingDateRole=11` を商品の出版日として使用しない
- 出版日候補をONIX `01`、`hanmoto.dateshuppan`、`summary.pubdate` の順に採用する

### 対象外

- ONIXの価格、商品階層 `PartNumber`、CollectionSequence、SupportingResourceの追加実装
- 新しいONIX項目の網羅的な取得・変換
- openBDを主要な検索・正規化用データ取得元にすること
- `summary.cover` 以外の書影取得
- MADBの人物読み、Agent情報、クエリ変更
- 共通型の定義変更（023で扱う）

## 変更対象

- `openbd/title.go`、`openbd/title_test.go`（削除）
- `openbd/response.go`、`openbd/response_test.go`、`openbd/client_test.go`
- `docs/pkg/openbd/spec.md`

`docs/spec.md`、`README.md`、`examples/` は023で定義済みの共通型・利用者向け説明を
変更しない。024で変わるopenBD固有の採用規則は `docs/pkg/openbd/spec.md` だけに記載する。

## 公開API・JSONへの影響

- 023実施後の `NormalizedBook.TitleReading` へ、`TitleText.content` と同じ要素の `collationkey` 全体を設定する
- 推測で設定していた `ParallelTitles`、`Volume`、`Series`、`Imprints` が空または省略になる場合がある
- `Contributor` の名前と読みは同じONIX要素の値を変更せず保持する。未対応役割は空の役割として保持する
- `Dates` から `PublishingDateRole=11` 由来の商品の出版日を除外する
- Raw responseの本文と `summary.cover` の既存処理は変更しない

## 想定ユースケース

- 入力: openBDからISBNで取得したONIX書誌データ
- 実行: 明示されたタイトル、人名、読み、既知役割、刊行日候補だけを共通型へ変換する
- 出力: 推測に依存していた並列タイトル、巻数、シリーズ、レーベル、役割、刊行日は設定しない
- 失敗時の扱い: 既存のISBN検証、エラー分類、Raw response返却の規則を変更しない

## 処理方針

1. `TitleText` と同じ要素の `collationkey` を全体のままタイトルと読みへ設定する
2. Collection、Collectionの `PartNumber`、`summary.volume` を作品シリーズ・レーベル・巻数へ変換しない
3. Contributorの文字列を変更せず、既知役割だけを変換する
4. 商品の出版日はONIX `01`、`hanmoto.dateshuppan`、`summary.pubdate` の順に選び、`11` を使わない

## 実施項目

### 調査

- [x] `openbd/title.go` と変換処理で推測・書換えを行う箇所を確認する
- [x] インラインのopenBD応答を使う既存テストから、変更前の期待値を整理する

### 実装

- [x] タイトル、並列タイトル、巻数の推測的分解を削除する
- [x] Collectionと関連する刊行番号を `Series`、`Imprints`、`Volume` へ設定しない
- [x] Contributorの名前・読み・役割の変換を022の方針へ更新する
- [x] 刊行日候補の選択順を更新し、`PublishingDateRole=11` を除外する
- [x] 未使用の `sourceBook.ISBNs` と初期化を削除する

### テスト

- [x] インラインのopenBD応答を使い、タイトル、人物、Collection、役割、刊行日の回帰テストを更新する
- [x] 追加対象外のONIX項目を読込・変換しないことを確認する
- [x] JSON出力とRaw response返却の退行を確認する
- [x] 対応済み・未対応ONIX役割を表駆動テストで確認する
- [x] 人名の記号、生年注記、典拠形を変更・分割せず、同じ要素の読みを保持することを確認する
- [x] 同名でも別のContributor要素を統合しないことを確認する
- [x] 商品階層 `PartNumber`、CollectionSequence、SupportingResource、価格を含む応答を変換しないことを確認する

### ドキュメント

- [x] `docs/pkg/openbd/spec.md` を実装済みの変換規則へ更新する
- [x] 023で更新する共通仕様との重複がないことを確認する

## 実施順

1. 023で導入済みの `TitleReading` と `Contributor.Reading` を使用する
2. タイトル分解とCollection・巻数の変換を削除する
3. Contributor役割と人物文字列の変換を修正する
4. 刊行日候補の選択順を修正する
5. インラインのopenBD応答を使った回帰テストと仕様を更新する

実装時は、タイトルと読みを別々の候補リストへ収集しない。同じ商品階層
`TitleElement` を選んだ時点で、その `TitleText.content` と `collationkey` を対にして
保持する。これにより、空文字列の重複除去や候補順の違いでタイトルと読みの対応を失わない。

## テストケース

- `TitleText.content` 全体を主タイトルとし、` = ` や末尾数字から並列タイトル・巻数を作らない
- 同じTitleText要素の `collationkey` 全体を `TitleReading` に設定する
- Collection名、Collectionの `PartNumber`、`summary.volume` から `Series`、`Imprints`、`Volume` を設定しない
- `／` を含む人名、生年注記、コンマ、中黒を含む人名を変更・分割しない
- 同じContributor要素の名前と読みを同じ `Contributor` に保持する
- `A38`、`A36`、`A46`、`A47` を近い共通役割へ変換しない
- `A01`、`A03`、`A14`、`A45`、`A07`、`A12`、`A35`、`B01`、`B06` の既定の対応を退行させない
- `PublishingDateRole=01` を優先し、ない場合だけ `hanmoto.dateshuppan`、次に `summary.pubdate` を使い、`11` を使わない
- 価格、商品階層 `PartNumber`、CollectionSequence、SupportingResourceの読込・変換を追加しない

## 検証

- [x] `gofmt` を実行する
- [x] `go test ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] `golangci-lint run` を実行する
- [x] `go test -run '^TestIntegrationLookupBooksByISBN$' -v -tags=integration ./openbd` を実行する
- [x] `git diff --check` を実行する

## 受け入れ条件

- [x] openBDの変換がタイトル、人名、Collection、役割、刊行日を推測して設定しない
- [x] 取得元が同じ要素で明示したタイトル読み・人物読みを変更せず保持する
- [x] `A38` と未対応ONIX役割を誤った共通役割へ変換しない
- [x] `PublishingDateRole=11` が商品の出版日として返らない
- [x] 追加対象外のONIX項目や書影処理を増やしていない
- [x] 023の共通モデルに沿ったテスト、仕様、JSON出力になっている
- [x] `go test ./...`、`go build ./...`、プロジェクトで採用しているlint、`git diff --check` が成功する

## 依存関係

- 022の調査結論に依存する
- 023の後に実施する

## リスク・懸念

- 推測で設定されていた値が空または省略になり、利用者の表示や並べ替えが変わる
- ONIX要素の取り違えにより、同じ要素の名前と読みの対応を失う可能性がある
- `PublishingDateRole=11` を使わなくなることで、商品の出版日候補がない書籍が増える可能性がある

## 未決事項

- なし。対象外のONIX項目は具体的な利用要件が生じた場合に別TODOで検討する

## 着手時の確認結果（2026年8月6日）

- 現行の `parseSourceTitle` と `normalizeTitleKana` が、タイトルの区切りと末尾表記を
  解釈して `ParallelTitles`、`Volume`、タイトル読みを変更している。024では両関数と
  専用テストを削除する
- `collectSourceBook` がONIX `Collection` と `summary.series` を `Series` に設定している。
  024ではこれらの読込みと変換を削除し、`Imprints` にも設定しない
- `mapContributorRole` は `A36`、`A38`、`A46`、`A47` を近い共通役割へ対応付けている。
  024では対応表から除外し、人物要素は役割なしで保持する
- `selectPublishedDate` は `01` と `11` を同じ出版日候補としている。024では `01` だけを
  ONIX候補とし、その後に `hanmoto.dateshuppan`、`summary.pubdate` を使う

## レビュー後の追加作業（2026年8月6日）

- 対応済みONIX役割 `A01`、`A03`、`A14`、`A45`、`A07`、`A12`、`A35`、`B01`、`B06` と、
  未対応 `A36`、`A38`、`A46`、`A47` の対応表を表駆動テストで固定する
- `／`、生年注記、コンマ、中黒を含む人物名と同じ要素の読みを変更せず、分割しないことを
  テストする。別Contributor要素の同名人物を統合しないことも別ケースで保持する
- 対象外のONIX項目を含む応答でも、既存の公開結果へ追加項目を設定しないことを確認する
- 023で不要になった `sourceBook.ISBNs` を削除する
