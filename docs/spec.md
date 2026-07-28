# manken API仕様

## 1. 目的

`manken` は、複数のGoプロジェクトから漫画本の書誌情報を検索するためのライブラリである。

データ取得元ごとの通信方式やレスポンス形式を共通APIから分離し、利用側は取得元の詳細を
意識せずに検索条件と書籍モデルを扱えるようにする。

将来はMCPサーバーから呼び出して使用する。MCP固有の処理は検索処理から分離し、
公開APIの入力と結果をMCPツールの構造化入出力へ変換できる設計にする。

## 2. 関連仕様

データ取得元に固有の仕様は、パッケージごとの仕様書で定義する。

- [MADBパッケージ仕様](pkg/madb/spec.md)

## 3. モジュールとパッケージ

### 3.1 確定事項

モジュールパスは次のとおり。

```text
github.com/eamat-dot/manken
```

初期パッケージは次の2つとする。

```text
github.com/eamat-dot/manken
github.com/eamat-dot/manken/madb
```

- `manken`
  - データ取得元に依存しない検索条件、検索結果、書籍モデル、エラー分類を定義する
- `madb`
  - MADBへの問い合わせと、取得結果から共通モデルへの変換を担当する

### 3.2 未確定事項

- MCP実装を同一モジュールの `mcp` パッケージにするか、独立モジュールにするか
- 将来MADB以外の取得元を追加するか

## 4. 初期APIの対象

### 4.1 確定事項

初期APIは、漫画本のタイトル検索を対象とする。

次の機能を提供する。

- タイトルによる検索
- 取得件数の指定
- カーソルによる続きの取得
- データ取得元固有の形式から共通の書籍モデルへの変換
- 入力エラー、外部サービスエラー、レスポンス解析エラーの分類

次の機能は初期APIに含めない。

- ISBN検索
- 著者名検索
- 漫画シリーズの検索
- 複数データ取得元の横断検索
- 検索結果の重複統合
- キャッシュ
- 自動リトライ
- クライアント側のレート制限
- CLI
- MCPサーバー

### 4.2 未確定事項

- 将来追加する検索条件
- 複数データ取得元を追加した場合の共通検索API

## 5. 共通モデル

### 5.1 Source

取得元は文字列を基底型とする専用型で表す予定である。

```go
type Source string
```

#### 確定事項

- 取得元ごとに `Source` の定数を定義する
- JSONへ変換した場合は取得元を示す短い文字列にする

#### 未確定事項

- 取得元定数をルートパッケージと各取得元パッケージのどちらに定義するか

### 5.2 Book

検索で取得した1冊の漫画本は、次の型で表す予定である。

```go
type Book struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle"`
	SeriesName    string   `json:"series_name"`
	VolumeNumber  string   `json:"volume_number"`
	Authors       []string `json:"authors"`
	Publishers    []string `json:"publishers"`
	ISBN10        string   `json:"isbn10"`
	ISBN13        string   `json:"isbn13"`
	PublishedDate string   `json:"published_date"`
	Source        Source   `json:"source"`
	SourceURL      string   `json:"source_url"`
}
```

#### 確定事項

- 欠落している文字列項目は空文字列にする
- `Authors` と `Publishers` は、値がない場合も空スライスにする
- 著者や出版者が複数存在する場合は、重複を除いてすべて返す
- 巻数は整数へ変換せず、取得元の表記を文字列として保持する
- 刊行日は `time.Time` へ変換せず、取得元の精度を保つ文字列として保持する
- タイトルやシリーズ名からISBN、巻数、刊行日を推測しない
- `SourceURL` には取得元の対象を参照できるURLを設定する

#### 未確定事項

- データ取得元ごとの識別子を共通の `ID` として扱う規則
- 著者、原作者、作画担当などを `Authors` へ含める範囲
- 複数値の共通の並び順
- 書籍モデルへ版、形式、言語などを追加するか

## 6. 検索API

### 6.1 SearchBooksRequest

検索条件は、次の型で受け取る予定である。

```go
type SearchBooksRequest struct {
	Title  string `json:"title"`
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}
```

#### 確定事項

- `Title` は必須とする
- 前後の空白を除いたタイトルが空の場合は入力エラーにする
- `Title` はデータ取得元のクエリ構文ではなく、通常の文字列として受け取る
- `Limit` が `0` の場合は既定値を使用する
- `Cursor` が空の場合は最初のページを取得する
- 続きを取得する場合は、直前の結果に含まれる `NextCursor` をそのまま指定する
- カーソルの内部形式は公開APIの契約に含めない

#### 未確定事項

- `Limit` の既定値と最大値
- カーソルへ検索条件を関連付ける共通規則
- 不正または期限切れのカーソルを判定する共通規則

### 6.2 SearchBooksResult

検索結果は、次の型で返す予定である。

```go
type SearchBooksResult struct {
	Books      []Book `json:"books"`
	NextCursor string `json:"next_cursor"`
}
```

#### 確定事項

- 該当する書籍がない場合は、エラーではなく空の `Books` を返す
- 続きがない場合は `NextCursor` を空文字列にする
- 合計件数は初期APIに含めない

#### 未確定事項

- 複数データ取得元で共通化できるページング規則

## 7. エラーAPI

利用側がエラー文字列を解析せずに原因を分類できるAPIを提供する予定である。

```go
type ErrorKind string

const (
	ErrorKindInvalidArgument ErrorKind = "invalid_argument"
	ErrorKindUpstream        ErrorKind = "upstream"
	ErrorKindUnavailable     ErrorKind = "unavailable"
	ErrorKindInvalidResponse ErrorKind = "invalid_response"
)

type Error struct {
	Kind       ErrorKind
	Operation  string
	StatusCode int
	RetryAfter time.Duration
	Err        error
}
```

### 7.1 確定事項

- 入力値とカーソルの不正は `invalid_argument` とする
- データ取得元が返したエラー応答は `upstream` または `unavailable` とする
- タイムアウトや一時的な通信失敗は `unavailable` とする
- 成功応答を解析できない場合は `invalid_response` とする
- `context.Canceled` と `context.DeadlineExceeded` を `errors.Is` で判定可能にする
- `*manken.Error` を `errors.As` で取得可能にする
- 外部サービスのレスポンス本文を通常のエラーメッセージへそのまま含めない

### 7.2 未確定事項

- HTTPステータスと `ErrorKind` の共通の対応
- `Retry-After` の対応形式
- エラー本文を診断情報として保持するか
- `Operation` に使用する値

## 8. MCPから利用する場合の境界

### 8.1 確定事項

- `manken` とデータ取得元パッケージはMCP SDKへ依存しない
- MCPのツール定義、JSON Schema、構造化結果、トランスポートはMCP側で扱う
- MCP側は `SearchBooksRequest` と `SearchBooksResult` を変換して使用する
- MCP側に必要な最小インターフェースを定義し、データ取得元のクライアントを受け取る
- MCP側がライブラリのエラーを利用者向けの安全なエラーへ変換する
- MCP仕様の変更が検索APIへ直接影響しない構成にする

MCP側のインターフェースは、次の形を想定する。

```go
type BookSearcher interface {
	SearchBooks(
		ctx context.Context,
		request manken.SearchBooksRequest,
	) (manken.SearchBooksResult, error)
}
```

初期のMCPツール名は、次を候補とする。

```text
search_manga_books
```

### 8.2 未確定事項

- MCPパッケージまたはモジュールの配置
- 使用するGo向けMCP SDK
- ツールの正式名称と説明
- MCPの入力Schemaと出力Schema
- stdioとStreamable HTTPのどちらを提供するか

## 9. 現時点の未確定事項一覧

1. `Source` 定数を定義するパッケージ
2. 共通の識別子、複数値、ページングの規則
3. Limitの既定値と最大値
4. HTTPエラーと `ErrorKind` の共通の対応
5. MCP実装の配置、SDK、ツールSchema

データ取得元に固有の未確定事項は、各パッケージの仕様書へ記載する。
