# 共通型をapiパッケージへ移してmadbだけで利用できるようにする

## 背景

- 現在は、データ取得元に依存しない型をルートの `manken` パッケージに定義し、
  `madb` パッケージがルートをimportしている
- 利用者はMADBだけを検索する場合も、検索条件の生成とエラー判定のために
  `github.com/eamat-dot/manken` と `github.com/eamat-dot/manken/madb` の
  両方をimportする必要がある
- 将来ルートの `manken` パッケージから各データ取得元を呼び出すファサードを
  実装すると、現在の `madb -> manken` という依存方向では循環importになる
- 共通型は外部利用者との公開契約であるため、外部からのimportを禁止する
  `internal` パッケージには置かない
- まだリモートへpushしていないため、既存のルートパッケージとの互換性は
  維持しなくてよい

## 目的

- データ取得元に依存しない公開型を `github.com/eamat-dot/manken/api` へ移す
- `madb` からルートの `manken` パッケージへの依存を取り除く
- MADBの利用者が `github.com/eamat-dot/manken/madb` だけをimportして検索条件の生成、
  検索、結果の参照、分類済みエラーの判定を行えるようにする
- ルート直下を、将来の公開ファサード実装に使用できる状態にする

## 非目的

- ルートの `manken` パッケージへのファサード実装
- ルートパッケージへの互換用型エイリアス追加
- Google Books、openBD、Yahoo!ショッピング、楽天Books、楽天Koboなどの
  データ取得元追加
- APIキー、Application ID、Access Keyなどの認証設定設計
- 複数データ取得元の横断検索、重複統合、並び順、部分失敗の設計
- `Book` の項目、MADBの変換規則、検索条件、カーソル、エラー分類の仕様変更
- 共通検索APIが将来のすべてのデータ取得元に適用できるかの再設計

## 前提・制約

- `api` は外部からimportできる公開パッケージとする
- `internal` は共通型の定義元に使用しない
- `madb` の公開APIでは `api` の型を型エイリアスとして公開する
- 利用者が `api` を直接importしなくても、`madb` の公開名だけで通常利用を
  完結できるようにする
- 型エイリアスによる型変換、値のコピー、重複した型定義を発生させない
- ルートの既存公開型とのソース互換性は維持しない
- MADBへのHTTPリクエスト、レスポンス変換、rawレスポンス、ページングの
  実行時動作は変更しない

## 対象範囲

### 対象

- `manken.go` にある `Source`、`SourceMADB`、`Book`、
  `SearchBooksRequest`、`SearchBooksResult`
- `error.go` にある `ErrorKind`、エラー分類定数、`Error`
- 新しい `api` パッケージ
- `madb` パッケージのimport、公開型エイリアス、メソッドシグネチャ
- ルートの共通型定義削除
- テスト、サンプル、README、共通API仕様、MADBパッケージ仕様、構想文書

### 対象外

- MADB固有の非公開型とSPARQLレスポンス形式
- MADBの検索クエリ、取得項目、変換規則
- `Client.SearchBooksWithRawResponse` のrawレスポンス契約
- 既存の未完了TODOで調査している版、著者役割、巻数、ソート
- 新しい外部モジュール依存の追加

## 想定ユースケース

- 入力: 利用者がタイトル、取得件数、必要に応じてカーソルを指定する
- 実行: `madb.Client` がMADBを検索し、共通型へ変換した結果を返す
- 出力: 利用者は `madb.SearchBooksResult` として書籍と次のカーソルを参照する
- 失敗時の扱い: 利用者は `madb.Error` と `madb.ErrorKind` を使って
  分類済みエラーを判定する

利用例は次のimportだけで完結させる。

```go
import "github.com/eamat-dot/manken/madb"

result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
	Title: "動物のおしゃべり",
	Limit: 5,
})
```

## 処理方針

1. 共通型と共通エラーを新しい `api` パッケージへ移す
2. `madb` がルートではなく `api` をimportするように変更する
3. `madb` から通常利用に必要な共通型、定数、エラーを型エイリアスまたは
   定数として公開する
4. `Client.SearchBooks` と `Client.SearchBooksWithRawResponse` の
   シグネチャを `madb` 上の公開名で表す
5. サンプルと外部パッケージテストからルートの `manken` importを取り除く
6. ルートの既存共通型を削除し、ルートパッケージを将来のファサード用に予約する
7. 仕様書とREADMEを新しいパッケージ境界へ同期する

## 実施項目

### API構成

- [x] `api` パッケージへ移す型と定数を既存実装から確認する
- [x] `api` を共通の公開契約、`madb` を取得元固有の実装、
  ルートを将来のファサードとする依存方向を仕様書へ記載する
- [x] `madb` だけで通常利用とエラー判定に必要な公開名を列挙する
- [x] ルートパッケージに互換用エイリアスを残さないことを確認する

### 実装

- [x] `api` パッケージを作成し、共通型、定数、共通エラーを移す
- [x] `madb` の内部実装が `api` の型を使用するように変更する
- [x] `madb` に `Source`、`Book`、`SearchBooksRequest`、
  `SearchBooksResult`、`ErrorKind`、`Error` の型エイリアスを追加する
- [x] `madb` に `SourceMADB` と既存のエラー分類定数を公開する
- [x] `madb.Client` の公開メソッドを `madb` 上の型名で記述する
- [x] ルートにある既存の共通型と共通エラーの定義を削除する
- [x] サンプルとテストから不要になったルートパッケージのimportを削除する

### テスト

- [x] `madb` だけをimportする外部パッケージテストを追加または更新する
- [x] `madb.SearchBooksRequest` で検索し、`madb.SearchBooksResult` を
  受け取れることを検証する
- [x] `errors.As` で `*madb.Error` を取得できることを検証する
- [x] `madb.ErrorKind` と公開したエラー分類定数を比較できることを検証する
- [x] `madb.SourceMADB` と検索結果の取得元を比較できることを検証する
- [x] `Client.SearchBooksWithRawResponse` の既存動作が変わらないことを検証する
- [x] ルートの `manken` パッケージへのimportが実装とテストに残っていないことを
  検索して確認する

### ドキュメント

- [x] READMEのインストール先と利用例を `madb` 単独importへ変更する
- [x] READMEの共通型と取得元パッケージの説明を新しい依存方向へ変更する
- [x] `docs/spec.md` のモジュール、パッケージ、公開型の定義元を更新する
- [x] `docs/pkg/madb/spec.md` の型名とパッケージ境界を更新する
- [x] `docs/concept.md` の責務分離を `api`、取得元パッケージ、
  将来のルートファサードに分ける
- [x] ファサードと複数データ取得元の設計が未実装であることを明記する

## 検証

- [x] `gofmt` または `goimports` を実行する
- [x] `go test -v ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] `golangci-lint run` を実行する
- [x] `task all` を実行する
- [x] `rg 'github.com/eamat-dot/manken"' --glob '*.go'` で、
  ルートパッケージへのimportが残っていないことを確認する
- [x] `git diff --check` を実行する

## 受け入れ条件

- [x] `madb` がルートの `manken` パッケージをimportしていない
- [x] MADBの利用例が `github.com/eamat-dot/manken/madb` だけのimportで
  コンパイルできる
- [x] 利用者が `madb` の公開名だけで検索条件、検索結果、取得元、
  分類済みエラーを扱える
- [x] 共通型の実体が `api` に一度だけ定義され、`madb` では型エイリアスとして
  公開されている
- [x] 共通型が `internal` パッケージに置かれていない
- [x] ルートに既存共通型の互換用エイリアスが残っていない
- [x] MADB検索、ページング、rawレスポンス、エラー分類の動作が変わっていない
- [x] READMEと仕様書が実装後のパッケージ構成と一致している
- [x] テスト、ビルド、lintが成功する

## リスク・懸念

- `SearchBooksRequest` の `Cursor` は現在のMADBページング仕様の影響を受けており、
  将来のすべてのデータ取得元に共通化できるとは限らない
- `api` にデータ取得元固有の型や設定を追加すると、将来のファサードと
  取得元パッケージの責務が再び混在する
- `madb` で必要な型や定数のエイリアスが不足すると、利用者が通常操作の途中で
  `api` の追加importを必要とする
- ルートパッケージを削除するため、ローカルにある既存利用コードは
  importと型名の変更が必要になる

## 未決事項

- `SearchBooksRequest`、`SearchBooksResult`、共通エラーが将来の複数データ取得元でも
  共通契約として使えるかは、横断検索と能力別インターフェースの設計時に再検討する
- 将来のルートファサードで、取得元の選択、部分失敗、カーソル、重複統合を
  どのように表すかは別TODOで決める

## AIへの入力メモ

- 会話で決まった前提: まだリモートへpushしていないため互換性は考慮しない
- 会話で決まった前提: 共通型は公開 `api` パッケージへ置き、`internal` には置かない
- 会話で決まった前提: ルート直下は将来の公開ファサード用に予約する
- 優先度: 将来のデータ取得元追加とファサード設計より先に実施する
- 先に決めたい論点: なし
