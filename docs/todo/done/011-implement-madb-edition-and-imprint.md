# MADBの版表示、単行本レーベル、シリーズ参照を実装

## 背景

- [TODO006](../archived/006-research-madb-edition-and-imprint.md)でMADBの版表示、
  単行本レーベル、マンガ単行本シリーズを調査した
- 現在のMADBクエリは `schema:version`、`schema:brand`、
  `schema:isPartOf` と参照先シリーズを取得していない
- 現在の `SeriesNames` は単行本へ直接記録された
  `ma:seriesName` だけを使用する
- 共通仕様へ `EditionStatements`、`Imprints`、`SeriesID`、
  `SeriesURL` を追加することが決まった

## 目的

- MADBから版表示、単行本レーベル、参照先シリーズを取得する
- 取得値を確定済みの共通モデルへ変換する
- 同じタイトルの異なる版を、推測せず取得元の根拠で識別できるようにする

## 非目的

- 版表示やレーベルを通常版、新装版、愛蔵版、完全版、文庫版へ分類する処理
- レーベル、タイトル、出版社、判型から欠落した版表示を補完する処理
- 著者役割と巻数の正規化
- 検索結果のソートとカーソルの変更
- シリーズ検索
- MADB以外のデータ取得元
- 公式サイトの画面表示との再照合

## 前提・制約

- `docs/spec.md` と `docs/pkg/madb/spec.md` の確定仕様に従う
- `api` にMADB固有のプロパティ名やRDF型を公開しない
- 単行本へ直接記録された値と参照先シリーズの値を混同しない
- `ja-hrkt` などの読みを共通モデルへ含めない
- 複数値を任意の1件へ絞らない
- 既存の検索条件、resource URI順、カーソル、エラー分類を変更しない
- rawレスポンスは受信本文を変更せず返す
- 新しい外部モジュールへ依存しない

## 対象範囲

### 対象

- `api/model.go` の `Book`
- `madb/query.go` の取得変数とOPTIONAL句
- `madb/response.go` の非公開型、binding集約、共通モデル変換
- クエリ、変換、公開API、実サービスのテスト
- README、CHANGELOG、仕様書

### 対象外

- `SearchBooksRequest` と `SearchBooksResult`
- カーソルpayloadとページ境界
- HTTP本文上限と自動リトライ
- `schema:brand` と `schema:version` の文字列内容に基づく品質判定
- 参照先シリーズのレーベルと版表示の公開

## 想定ユースケース

- 入力: 利用者が従来どおりタイトルと取得件数を指定する
- 実行: MADBクライアントが単行本と参照先シリーズの必要項目を取得する
- 出力: `Book` に版表示、単行本レーベル、シリーズ名、シリーズID、
  シリーズURLが設定される
- 失敗時の扱い: 既知bindingの型不正や0-1項目の競合は
  `invalid_response` とする

## 処理方針

1. 共通 `Book` に確定済みフィールドを追加する
2. 外側のSPARQLクエリで単行本の版表示とレーベルを取得する
3. `schema:isPartOf` の参照先を `class:MangaBookSeries` に限定し、
   シリーズ名、ID、URIを取得する
4. MADB固有の値を非公開型へ集約する
5. 複数値と欠落値の規則に従って共通 `Book` へ変換する
6. 既存の検索、rawレスポンス、ページングが変わらないことを検証する

## 実施項目

### 共通モデル

- [x] `api.Book` に `EditionStatements []string` を追加する
- [x] `api.Book` に `Imprints []string` を追加する
- [x] `api.Book` に `SeriesID string` と `SeriesURL string` を追加する
- [x] 新しいスライス項目を欠落時も空スライスにする
- [x] `madb.Book` の型エイリアスから新しい項目を利用できることを確認する

### SPARQLクエリ

- [x] 言語タグなしの `schema:version` を取得する
- [x] 言語タグなしの `schema:brand` を取得する
- [x] `schema:isPartOf` が参照する `class:MangaBookSeries` を取得する
- [x] 参照先シリーズの言語タグなし `schema:name` を取得する
- [x] 参照先シリーズの `schema:identifier` とリソースURIを取得する
- [x] 参照先シリーズの `schema:brand` と `schema:version` を取得しない
- [x] ページ対象を先に確定する既存サブクエリとresource URI順を維持する

### レスポンス変換

- [x] `sourceBook` へ `Versions`、`Brands`、`SeriesID`、
  `SeriesResourceURI` を追加する
- [x] 版表示とレーベルを空値除外、完全一致重複除去、文字列昇順で集約する
- [x] 単行本と参照先シリーズの名前を `SeriesNames` へまとめる
- [x] 参照先シリーズIDを `SeriesID`、URIを `SeriesURL` へ変換する
- [x] 異なる複数のシリーズ参照、ID、URIを `invalid_response` とする
- [x] レーベル名や版表示の内容から値を除外、補完、分類しない
- [x] 参照先シリーズの値で単行本の版表示とレーベルを補完しない
- [x] 未知のbinding変数を無視する既存契約を維持する

### テスト

- [x] クエリに新しい変数とOPTIONAL句が含まれることを検証する
- [x] レーベルの読みを除外するFILTERを検証する
- [x] 版表示とレーベルの欠落、単一値、複数値を検証する
- [x] 参照先シリーズの欠落と正常値を検証する
- [x] 異なる複数のシリーズ参照、ID、URIを拒否することを検証する
- [x] `ma:seriesName` と参照先シリーズ名の重複除去を検証する
- [x] 参照先シリーズと単行本で異なるレーベルを混同しないことを検証する
- [x] 新しいスライス項目がJSONで `null` にならないことを検証する
- [x] `madb` だけをimportする公開APIテストで新しい項目を参照する
- [x] rawレスポンスが受信本文とバイト単位で一致する既存テストを維持する
- [x] カーソルによる2ページ目取得の既存テストを維持する

### 実サービス

- [x] 通常版、新装版、愛蔵版、完全版、文庫版の代表IDを確認する
- [x] `M292127`、`M292128`、`M292129` のシリーズ参照を確認する
- [x] `M335551` または `M346749` で複数の版表示を確認する
- [x] `M1079781` で単行本とシリーズのレーベルを混同しないことを確認する
- [x] 100冊と次ページ判定用1冊の応答が4 MiB上限内に収まることを確認する

### ドキュメント

- [x] READMEのJSON例と項目説明を更新する
- [x] CHANGELOGへ利用者に見える追加項目を記載する
- [x] `docs/spec.md` と `docs/pkg/madb/spec.md` を実装結果へ同期する
- [x] 実サービスで確認したIDと結果をMADB仕様書へ記録する

## 検証

- [x] `gofmt` または `goimports` を実行する
- [x] `go test -v ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] `golangci-lint run` を実行する
- [x] `task all` を実行する
- [x] 実サービスで変換前後の値を確認する
- [x] `git diff --check` を実行する

## 受け入れ条件

- [x] `EditionStatements` と `Imprints` が複数値を失わず返される
- [x] `SeriesNames`、`SeriesID`、`SeriesURL` に参照先シリーズが反映される
- [x] 版表示、レーベル、シリーズ参照の欠落を正常に扱える
- [x] 読み、参照先シリーズのレーベル、参照先シリーズの版表示が混入しない
- [x] タイトルやレーベルから版を推測しない
- [x] 既存の検索順、カーソル、rawレスポンス、エラー分類が変わらない
- [x] `madb` だけをimportする利用者が新しい項目を参照できる
- [x] README、CHANGELOG、仕様書が実装と一致する
- [x] テスト、ビルド、lint、実サービス確認が成功する

## リスク・懸念

- OPTIONAL句の追加により1冊当たりのbinding数とレスポンスサイズが増える
- `schema:brand` にはレーベルらしくない値が混在する
- 版表示とレーベルが欠落する単行本では、版を特定できない
- 参照先シリーズ名の追加により、既存の `SeriesNames` の内容が変わる
- `Book` の追加フィールドにより、位置指定の複合リテラルとJSON完全一致比較へ
  影響する
- MADBのデータとスキーマは将来変更される可能性がある

## 未決事項

- なし

## AIへの入力メモ

- 会話で決まった前提: `Imprints`、`EditionStatements`、
  `SeriesID`、`SeriesURL` を共通モデルへ追加する
- 会話で決まった前提: 読みは共通モデルへ追加しない
- 会話で決まった前提: 単行本へ直接記録された版表示とレーベルだけを返す
- 優先度: TODO006の公式サイト画面再確認とは独立して実装可能
- 先に決めたい論点: なし
