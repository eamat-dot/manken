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

初期APIは、漫画本のタイトル、ISBN、著者名による検索を対象とする。

次の機能を提供する。

- タイトルによる検索
- ISBN-10またはISBN-13による検索
- 著者名による検索
- 取得件数の指定
- カーソルによる続きの取得
- データ取得元固有の形式から共通の書籍モデルへの変換
- 入力エラー、外部サービスエラー、レスポンス解析エラーの分類

次の機能は初期APIに含めない。

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

検索で取得した1冊の漫画本は、利用者向けの共通書誌情報と、正規化に使用した
取得元別の値に分ける。既存の平坦な `Book` は移行期間を設けず破壊的に置き換える。

```go
type Book struct {
	Normalized NormalizedBook `json:"normalized"`
	Sources    []BookSource   `json:"sources"`
}

type NormalizedBook struct {
	Title             string             `json:"title,omitempty"`
	TitleKana         string             `json:"title_kana,omitempty"`
	Subtitle          string             `json:"subtitle,omitempty"`
	Series            []Series           `json:"series,omitempty"`
	Volume            Volume             `json:"volume,omitzero"`
	EditionStatements []string           `json:"edition_statements,omitempty"`
	IsFinalVolume     bool               `json:"is_final_volume,omitempty"`
	Authors           []string           `json:"authors,omitempty"`
	Contributors      []Contributor      `json:"contributors,omitempty"`
	Publishers        []string           `json:"publishers,omitempty"`
	Imprints          []string           `json:"imprints,omitempty"`
	Identifiers       []Identifier       `json:"identifiers,omitempty"`
	Dates             []BookDate         `json:"dates,omitempty"`
	Description       string             `json:"description,omitempty"`
	Languages         []string           `json:"languages,omitempty"`
	Subjects          []Subject          `json:"subjects,omitempty"`
	PageCount         *int               `json:"page_count,omitempty"`
	Medium            PublicationMedium  `json:"medium,omitempty"`
	PhysicalSize      *PhysicalSize      `json:"physical_size,omitempty"`
	Prices            []Price            `json:"prices,omitempty"`
	Images            []Image            `json:"images,omitempty"`
}

type BookSource struct {
	Source Source           `json:"source"`
	ID     string           `json:"id,omitempty"`
	URL    string           `json:"url,omitempty"`
	Values SourceBookValues `json:"values"`
}
```

#### 確定事項

- `Normalized` は変更差分ではなく、利用者が通常参照する共通書誌情報とする
- 加工不要な取得元値も、共通項目として採用できる場合は `Normalized` へコピーする
- `Sources` は正規化に使用した取得元、取得元内ID、参照URL、元値を保持する
- 取得元固有の全レスポンスは `Book` に含めず、Raw response用メソッドで返す
- `Title` は単数とし、分解できない場合は取得元タイトル全体を暫定値にする
- `TitleKana` は安全に分解できない場合に空にし、取得元の読みは `Sources` に残す
- `Subtitle` は単数とする
- `Authors` は利用者向けの著者名、`Contributors` は名前と複数の役割を保持する
- 取得元が寄与者の順序を提供する場合は、その順序を維持する
- 取得元が順序を提供しない場合は、取得元パッケージが結果を安定化し、
  その順序に意味がないことを取得元固有の仕様へ記載する
- 並列タイトルは初期の `NormalizedBook` に含めない
- `IsFinalVolume` は完結巻の肯定情報だけを表し、`完全版`から推測しない
- 初期実装では複数取得元のBookを自動統合しない

### 5.3 SourceBookValues

取得元の値は、文字列の内容を変更せず次の型へ保持する。取得元が同じ意味の値を
複数返せる項目は、正規化で1件を選んだ場合もスライスですべて残す。

```go
type SourceBookValues struct {
	Titles        []string      `json:"titles,omitempty"`
	TitleKana     []string      `json:"title_kana,omitempty"`
	Subtitles     []string      `json:"subtitles,omitempty"`
	SeriesNames   []string      `json:"series_names,omitempty"`
	Volume        string        `json:"volume,omitempty"`
	Editions      []string      `json:"editions,omitempty"`
	Authors       []string      `json:"authors,omitempty"`
	Publishers    []string      `json:"publishers,omitempty"`
	Imprints      []string      `json:"imprints,omitempty"`
	ISBNs         []string      `json:"isbns,omitempty"`
	PublishedDate string        `json:"published_date,omitempty"`
	Description   string        `json:"description,omitempty"`
	GenreIDs      []string      `json:"genre_ids,omitempty"`
	PageCount     *int          `json:"page_count,omitempty"`
	Size          string        `json:"size,omitempty"`
	Prices        []SourcePrice `json:"prices,omitempty"`
}
```

`TitleKana` は取得元が同じ書籍へ複数の読みを返す場合があるため、全候補を保持する。
取得元固有で共通の意味を保てない値はこの型へ追加せず、取得元パッケージの
非公開レスポンス型またはRaw responseに残す。

### 5.4 巻数と識別子

```go
type Volume struct {
	Number *int   `json:"number,omitempty"`
	Label  string `json:"label,omitempty"`
}

type Identifier struct {
	Type  IdentifierType `json:"type"`
	Value string         `json:"value"`
}
```

- `Number` は整数化できる場合だけ設定し、実在する0巻を保持できるよう `*int` とする
- `Label` は `16`、`上`、`前編`、`1.5`、`別巻`など正規化済みの表示を保持する
- 元の巻表示は `BookSource.Values.Volume` に残す
- 並べ替え用Indexは公開モデルへ追加しない
- ISBN-10、ISBN-13、JANは `IdentifierType` で区別する
- 取得元内だけで意味を持つ商品IDは `BookSource.ID` に置く

### 5.5 寄与者

```go
type ContributorRole string

const (
	ContributorRoleAuthor            ContributorRole = "author"
	ContributorRoleOriginalCreator   ContributorRole = "original_creator"
	ContributorRoleWriter            ContributorRole = "writer"
	ContributorRoleArtist            ContributorRole = "artist"
	ContributorRoleCharacterCreator  ContributorRole = "character_creator"
	ContributorRoleCharacterDesigner ContributorRole = "character_designer"
	ContributorRoleEditor            ContributorRole = "editor"
	ContributorRoleTranslator        ContributorRole = "translator"
	ContributorRoleSupervisor        ContributorRole = "supervisor"
	ContributorRoleCommentator       ContributorRole = "commentator"
	ContributorRoleDesigner          ContributorRole = "designer"
)

type Contributor struct {
	Name  string            `json:"name"`
	Roles []ContributorRole `json:"roles,omitempty"`
}
```

- `ContributorRole` は取得元固有の表記やONIXコードをそのまま公開せず、
  複数の取得元で意味が通じる一般名を使用する
- `Authors` には、著者、原作者、構成または脚本担当、作画担当、
  キャラクター原案、キャラクターデザインを含める
- 編集、翻訳、監修、解説、装丁またはデザイン担当は `Authors` に含めず、
  役割を確定できる場合だけ `Contributors` に含める
- 役割がない著者表示は `Authors` に含めるが、推測した役割を持つ
  `Contributor` は作らない
- 同じ人物に複数の役割がある場合は1つの `Contributor` にまとめる
- 取得元の役割を共通役割へ確実に対応付けられない場合は、
  `Authors` または `Contributors` へ推測で分類しない
- 対応できなかった取得元値は `Sources` またはRaw responseに残す

### 5.6 シリーズ、日付、紙・電子

`Series` は名前と、それに対応する取得元内ID、URL、取得元を同じ要素へ保持する。
`BookDate` は出版日、発売日、電子版配信開始日を区別し、日付文字列の精度を保つ。
`PublicationMedium` は `print`、`digital`、不明の空文字列を取る。紙書籍だけが
`PhysicalSize` に判型名とミリメートル単位の寸法を設定する。

### 5.7 価格

```go
type Price struct {
	Type        PriceType `json:"type"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	TaxIncluded *bool     `json:"tax_included,omitempty"`
	Source      Source    `json:"source"`
	ObservedAt  string    `json:"observed_at,omitempty"`
}
```

- `list` は定価、`current` はAPI取得時点の販売価格を表す
- `Type`、`Amount`、`Currency`、`Source` は必須とする
- `current` は `ObservedAt` を必須とし、`list` では省略できる
- `TaxIncluded` は取得元で確認できる場合だけ設定する
- 価格0円を欠落扱いしない
- 在庫、送料、ポイント、会員価格は `Book` に含めない

### 5.8 JSONの欠落項目

- `normalized` と `sources` は常に出力する
- その内側の欠落した任意項目はJSONから省略する
- 空文字列、空スライス、`nil` ポインター、`IsFinalVolume=false`は省略する
- 0巻、価格0円、明示された `TaxIncluded=false` は省略しない
- 空の `Volume` 全体は省略する

## 6. 検索API

### 6.1 SearchBooksRequest

検索条件は、次の型で受け取る。

```go
type SearchBooksRequest struct {
	Title        string `json:"title"`
	ISBN         string `json:"isbn"`
	Author       string `json:"author"`
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}
```

#### 確定事項

- `Title`、`ISBN`、`Author`、`FreeText` の少なくとも1つを正条件として必須とする
- 複数の検索条件を指定した場合はAND条件にする
- 前後の空白を除いたすべての正条件が空の場合は入力エラーにする
- `Title` はデータ取得元のクエリ構文ではなく、通常の文字列として受け取る
- `Title` はUnicode空白で検索語に分け、すべての語を含むタイトルを検索する
- `Title` の検索語数にライブラリ独自の上限を設けない
- `ISBN` はASCIIハイフンとUnicode空白を除き、末尾の小文字 `x` を大文字にする
- `ISBN` はISBN-10または978/979で始まるISBN-13のチェックディジットを検証する
- ISBN-10と978で始まるISBN-13は相互変換し、どちらか一方だけを持つ取得元も検索する
- 979で始まるISBN-13からISBN-10は生成しない
- 不正なISBNは外部サービスへ送信せず入力エラーにする
- `Author` はデータ取得元のクエリ構文ではなく、通常の文字列として受け取る
- `Author` はUnicode空白で検索語に分け、すべての語を含む著者名を検索する
- `Author` の検索語数にライブラリ独自の上限を設けない
- `FreeText` は取得元固有の主要な書誌項目を横断して検索する
- `FreeText` はUnicode空白で検索語に分け、複数の項目をまたいですべての語に
  一致する書籍を検索する
- `FreeText` の検索語数にライブラリ独自の上限を設けない
- `ExcludedText` は取得元固有の主要な書誌項目を横断し、指定語を含む書籍を除外する
- `ExcludedText` はUnicode空白で検索語に分け、どれか1語でも含む書籍を除外する
- `ExcludedText` だけの検索は入力エラーにする
- `ExcludedText` の検索語数にライブラリ独自の上限を設けない
- Agent参照だけに存在する著者名を漏れなく検索する場合は `Author` を使用する
- Agent参照だけに存在する著者名は `ExcludedText` の対象外とする
- 大文字小文字、Unicode正規化、表記揺れはライブラリ側で変換しない
- `Limit` が `0` の場合は既定値を使用する
- `Cursor` が空の場合は最初のページを取得する
- 続きを取得する場合は、直前の結果に含まれる `NextCursor` をそのまま指定する
- カーソルの内部形式は公開APIの契約に含めない
- 初期のMADB検索ではLimitの既定値を20、最大値を100とする
- カーソルは正規化済みの全検索条件とLimitに関連付け、一致しない場合は入力エラーにする
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
