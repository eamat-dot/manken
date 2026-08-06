# `SourceBookValues` を削除して共通書籍モデルを整理する

## 背景

- `SourceBookValues` は完全なRaw responseでも共通化済みの値でもなく、人物名と読み、役割、識別子種別、画像用途などの対応を失う不完全な複製になっている
- 完全な取得元本文は、既存の `WithRawResponse` 系メソッドがリクエスト単位で返せる
- 初回リリース前のため、公開APIとJSONを互換維持する必要はない

## 目的

- 取得元参照と共通書籍情報の責務を分け、`SourceBookValues` を削除する
- 人物名、読み、既知役割を同じ `Contributor` 要素で表せる共通モデルへ整理する
- MADBとopenBDが新しい共通型へ変換できる状態にする

## 非目的

- MADBの人物読み、Agent URI、人物詳細クエリを実装すること
- Raw responseをBook単位へ複製または再構成すること
- 取得元をまたぐBook統合と項目ごとの出典管理を設計すること
- openBDの既存誤変換を修正すること

## 前提・制約

- 初回リリース前の破壊的変更として実施し、旧フィールドの互換コードを残さない
- `BookSource.Source` は必須、`ID` と `URL` は取得元で利用できる場合だけ設定する
- 人物名と読みは同じ取得元要素で対応が明示される場合だけ同じ `Contributor` に設定する
- MADBは人物名と読みの対応を確認できないため、`Contributor.Reading` を設定しない
- Raw responseは既存の `WithRawResponse` 系メソッドで、リクエスト単位の無加工本文として返す

## 対象範囲

### 対象

- `SourceBookValues` を初回リリース前に削除し、互換用フィールド・型エイリアスを残さない
- `BookSource` を必須の `Source` と、取得元で利用できる場合だけ設定する `ID`、`URL` へ縮小する
- `TitleKana` を `TitleReading` へ変更する
- `Contributor.Reading` を追加し、役割不明の人物も `Contributor` に保持する
- `Authors []string` を表示・簡易利用向け一覧として残す
- MADBでは人物読みを設定しない
- MADBとopenBDの変換コード、テスト、共通仕様・パッケージ仕様を新しい共通型へ追従させる
- `WithRawResponse` 系メソッドによるリクエスト単位のRaw response返却を維持する

### 対象外

- MADBの人物読み、Agent URI、人物詳細クエリの実装
- Raw responseを各 `Book` または `BookSource` へ埋め込むこと
- MADB bindingから1冊分のRaw JSONを再構成すること
- 複数取得元を統合する際の項目ごとの出典管理
- openBD固有の既存誤変換の修正（024で扱う）

## 変更対象

- `api/model.go`、`api/model_test.go`
- `madb/api.go`、`madb/response.go`、`madb/response_test.go`、`madb/integration_test.go`
- `openbd/api.go`、`openbd/response.go`、`openbd/response_test.go`、`openbd/client_test.go`
- `docs/spec.md`、`docs/pkg/madb/spec.md`、`docs/pkg/openbd/spec.md`
- `examples/` 配下のJSON出力テストまたは保存例（該当する場合）

## 公開API・JSONへの影響

- `BookSource.Values` と `SourceBookValues` を削除する破壊的変更とする
- `NormalizedBook.TitleKana` を削除し、`title_reading` のJSON項目を持つ `TitleReading` へ変更する
- `Contributor` に `reading` のJSON項目を追加する
- `Contributors` は役割が空の人物も含み得る。`Authors` は文字列一覧として維持する
- `BookSource.Source` は必須とし、`id` と `url` は利用できる取得元だけに出力する
- Raw responseのメソッド名、戻り値の本文単位、無加工本文を返す規則は変更しない

## 想定ユースケース

- 入力: 利用者がMADBまたはopenBDで書誌検索・ISBN参照を実行する
- 実行: 取得元は共通 `Book` を返し、必要な場合だけ`WithRawResponse` 系メソッドを選ぶ
- 出力: `Sources` は取得元参照、`Normalized` は共通化済み書誌情報、人物詳細は `Contributors` として返る
- 失敗時の扱い: 既存のエラー分類とRaw response返却条件を変更しない

## 処理方針

1. 公開モデルから `SourceBookValues` と `BookSource.Values` を削除し、JSONタグと型エイリアスを更新する
2. タイトル読みと人物読みを、対応関係を保てる `TitleReading` と `Contributor.Reading` へ移す
3. 各取得元の変換を新しい共通型へ追従させ、MADBの人物読みは空のままにする
4. テストと仕様を更新して、取得元参照とRaw responseの責務を検証する

## 実施項目

### 調査

- [x] `SourceBookValues`、`BookSource.Values`、`TitleKana` の全参照を確認する
- [x] JSON出力、型エイリアス、保存例、仕様書への影響を整理する

### 実装

- [x] `api` の公開型とJSONタグを変更する
- [x] MADBとopenBDの変換を新しい共通型へ更新する
- [x] MADBが人物読みを設定しないことを明示する
- [x] `WithRawResponse` 系メソッドのRaw response返却を維持する

### テスト

- [x] 共通型とJSON出力の単体テストを更新する
- [x] MADBとopenBDの変換テスト、必要な結合テスト、保存例を更新する
- [x] Raw responseの本文単位と無加工性に関する退行テストを維持する

### ドキュメント

- [x] 共通仕様とMADB・openBDパッケージ仕様を実装済みの公開API仕様へ更新する
- [x] JSON出力例または利用者向け文書を必要に応じて更新する

## 実施順

1. `api` の公開型、JSONタグ、型エイリアスへの影響を変更する
2. MADBとopenBDの変換を新しい型へ追従させ、MADBの人物読みを空のままにする
3. 単体・結合テストと保存例を新しいJSONへ更新する
4. 共通仕様と各パッケージ仕様を実装済みの動作として更新する
5. 024を実施する

## テストケース

- `BookSource` が`Source`、`ID`、`URL` だけをJSONへ出力し、`values` を出力しない
- `Source` が未設定のBookSourceを生成しない
- `TitleReading` がJSONへ出力され、`title_kana` を出力しない
- `Contributor` が名前、読み、既知役割を同一要素で保持し、役割なしの人物も保持できる
- `Authors` が従来どおり文字列一覧として利用できる
- MADBが人物読みを設定せず、既存の人名一覧を維持する
- openBDが同一人物要素の名前と読みを保持できる
- `WithRawResponse` 系メソッドが受信本文を変更せずリクエスト単位で返す
- 外部サービスに接続しない通常テストで変換とJSONを検証する

## 検証

- [x] `gofmt` または `goimports` を実行する
- [x] `go test -v ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] プロジェクトで採用しているlintを実行する
- [x] 必要なMADB・openBD結合テストを実行する
- [x] `git diff --check` を実行する

## 受け入れ条件

- `SourceBookValues`、`BookSource.Values`、互換用の旧項目が公開API・JSONから削除されている
- `BookSource`、`TitleReading`、`Contributor.Reading` の意味と省略規則が共通仕様に記載されている
- 役割不明の人物を失わず、`Authors` を簡易一覧として維持している
- MADBの人物読みを推測または設定していない
- MADBとopenBDの変換、テスト、仕様が新しい共通型と一致している
- Raw responseの返却単位を変更していない
- `go test ./...`、`go build ./...`、プロジェクトで採用しているlint、`git diff --check` が成功する

## 依存関係

- 022の調査結論に依存する
- 024より先に実施する。024はこのTODOで導入する `TitleReading` と `Contributor.Reading` を使用する

## リスク・懸念

- 公開型とJSONの破壊的変更により、既存利用コードと保存済みJSONが互換にならない
- `Contributors` に役割不明の人物を含めるため、役割が常に存在すると仮定する利用コードが影響を受ける
- 取得元変換と仕様の一部だけを更新すると、JSON、テスト、ドキュメントの説明が食い違う

## 未決事項

- なし。MADBの人物読みは022の結論に従い実装しない
