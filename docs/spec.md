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

最低Goバージョンは1.26.0とし、外部モジュールへ依存しない。

現在提供するパッケージは次の2つとする。

```text
github.com/eamat-dot/manken/api
github.com/eamat-dot/manken/madb
```

- `api`
  - データ取得元に依存しない検索条件、検索結果、書籍モデル、エラー分類を定義する
- `madb`
  - MADBへの問い合わせと、取得結果から共通モデルへの変換を担当する
  - 通常利用に必要な `api` の型と定数をエイリアスとして公開する

モジュールのルートは、将来各データ取得元を呼び出す公開ファサードに使用する。
現在はルートパッケージとファサードを提供しない。

### 3.2 取得元パッケージとの境界

`api` パッケージは、取得元に依存しない公開型と、その型が満たす契約だけを
定義する。取得元固有のレスポンス型、項目名、役割、欠落規則を扱わない。

利用側は `madb` などの取得元パッケージを直接呼び出す。取得元パッケージは
通常利用に必要な共通型と定数をエイリアスとして公開するため、利用側が
`api` を直接importする必要はない。取得元パッケージを呼び出す共通Clientや
ファサードは提供しない。

各取得元パッケージは、外部サービスのレスポンスを固有の非公開型へ読み込み、
そのパッケージ内で `api.Book` へ変換してから返す。

```text
外部サービスのレスポンス
    ↓
取得元パッケージの非公開型
    ↓ 取得元パッケージ内で変換
api.Book
    ↓
利用側
```

例えばMADBのcreatorは、`madb` パッケージ内では `Creators` として保持し、
MADB固有の役割表記を処理した後、`api.Book.Authors` へ設定する。
`api` パッケージは `Creators` を受け取って変換する関数を持たない。

将来別の取得元を追加する場合も、それぞれの取得元パッケージが同じ責務を持つ。

```text
madb   の非公開型 -> api.Book
google の非公開型 -> api.Book
kobo   の非公開型 -> api.Book
ndl    の非公開型 -> api.Book
```

取得元固有の型は公開APIへ含めない。複数の取得元に共通することが確認できた
変換後の概念だけを、`api` の共通モデルへ追加する。

### 3.3 未確定事項

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

取得元は文字列を基底型とする専用型で表す。

```go
type Source string

const SourceMADB Source = "madb"
```

#### 確定事項

- 取得元定数は `api` パッケージに定義する
- 取得元パッケージは、自身の通常利用に必要な取得元定数を公開する
- JSONへ変換した場合は取得元を示す短い文字列にする

### 5.2 Book

検索で取得した1冊の漫画本は、次の型で表す。

```go
type Book struct {
	ID                string   `json:"id"`
	Titles            []string `json:"titles"`
	Subtitles         []string `json:"subtitles"`
	SeriesNames       []string `json:"series_names"`
	SeriesID          string   `json:"series_id"`
	SeriesURL         string   `json:"series_url"`
	VolumeNumber      string   `json:"volume_number"`
	EditionStatements []string `json:"edition_statements"`
	Authors           []string `json:"authors"`
	Publishers        []string `json:"publishers"`
	Imprints          []string `json:"imprints"`
	ISBN10s           []string `json:"isbn10s"`
	ISBN13s           []string `json:"isbn13s"`
	PublishedDate     string   `json:"published_date"`
	Source            Source   `json:"source"`
	SourceURL         string   `json:"source_url"`
}
```

#### 確定事項

- 欠落している文字列項目は空文字列にする
- すべてのスライス項目は、値がない場合も空スライスにする
- 複数値は完全に同じ値だけを重複除去し、安定した昇順ですべて返す
- 取得元に正規値を示す情報がない場合、任意の1件を選ばず複数値として返す
- `Authors` は著者、原作者、作画担当などを役割で分けずに返す
- 取得元に役割情報があっても、共通モデルには役割を含めない
- 巻数は整数へ変換せず、取得元の表記を文字列として保持する
- 刊行日は `time.Time` へ変換せず、取得元の精度を保つ文字列として保持する
- `EditionStatements` は、取得元が版、バージョン、エディションとして
  提供する表示を分類せずに保持する
- `Imprints` は、出版者が出版物の形式や頒布計画に基づいて設定した
  レーベルまたはブランドの表示を保持する
- レーベル名から版を推測せず、版表示からレーベルを推測しない
- `SeriesID` には、取得元が対象のシリーズへ付与した識別子を設定する
- `SeriesURL` には、取得元のシリーズを参照できるURLを設定する
- タイトル、シリーズ名、出版社などからISBN、巻数、刊行日、版、
  レーベル、シリーズ識別子を推測しない
- `SourceURL` には取得元の対象を参照できるURLを設定する
- `ID` には取得元が対象へ付与した識別子を設定する

#### 未確定事項

- 書籍モデルへ形式、言語などを追加するか

## 6. 検索API

### 6.1 SearchBooksRequest

検索条件は、次の型で受け取る。

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
- 初期のMADB検索ではLimitの既定値を20、最大値を100とする
- カーソルは検索語とLimitに関連付け、一致しない場合は入力エラーにする
- カーソルの有効期間は設けない

#### 未確定事項

- 複数データ取得元で共通化した場合のLimitとカーソル規則

### 6.2 SearchBooksResult

検索結果は、次の型で返す。

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

利用側がエラー文字列を解析せずに原因を分類できるAPIを提供する。

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

func (err *Error) Error() string

func (err *Error) Unwrap() error
```

### 7.1 確定事項

- 入力値とカーソルの不正は `invalid_argument` とする
- データ取得元が返したエラー応答は `upstream` または `unavailable` とする
- タイムアウトや一時的な通信失敗は `unavailable` とする
- 成功応答を解析できない場合は `invalid_response` とする
- `context.Canceled` と `context.DeadlineExceeded` を `errors.Is` で判定可能にする
- `*api.Error` を `errors.As` で取得可能にする
- 取得元パッケージのエイリアスでも同じエラーを取得可能にする
- `Error` は `Unwrap` で原因を返し、標準のエラーチェーンを維持する
- 外部サービスのレスポンス本文を通常のエラーメッセージへそのまま含めない
- `Retry-After` は秒数形式とHTTP-date形式を扱う
- 外部サービスのエラー本文を `Error` に保持しない
- `Operation` の具体的な値は取得元パッケージの仕様で定義する

### 7.2 未確定事項

- HTTPステータスと `ErrorKind` の共通の対応

## 8. MCPから利用する場合の境界

### 8.1 確定事項

- `api` とデータ取得元パッケージはMCP SDKへ依存しない
- MCPのツール定義、JSON Schema、構造化結果、トランスポートはMCP側で扱う
- MCP側は `SearchBooksRequest` と `SearchBooksResult` を変換して使用する
- MCP側に必要な最小インターフェースを定義し、データ取得元のクライアントを受け取る
- MCP側がライブラリのエラーを利用者向けの安全なエラーへ変換する
- MCP仕様の変更が検索APIへ直接影響しない構成にする
- `Book` の文字列、文字列配列、`SearchBooksRequest`、
  `SearchBooksResult` はJSON Schemaで表現できる

MCP側のインターフェースは、次の形を想定する。

```go
type BookSearcher interface {
	SearchBooks(
		ctx context.Context,
		request api.SearchBooksRequest,
	) (api.SearchBooksResult, error)
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

1. 複数データ取得元を共通化する場合の検索条件とページング規則
2. HTTPエラーと `ErrorKind` の取得元横断の対応
3. MCP実装の配置、SDK、ツールSchema

データ取得元に固有の未確定事項は、各パッケージの仕様書へ記載する。
