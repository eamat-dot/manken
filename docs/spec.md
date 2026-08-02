# manken API仕様

## 1. 目的

`manken` は、複数のGoプロジェクトから漫画本の書誌情報を検索するためのライブラリである。

データ取得元ごとの通信方式やレスポンス形式を共通APIから分離し、利用側は取得元の詳細を
意識せずに検索条件と書籍モデルを扱えるようにする。

## 2. 関連仕様

データ取得元に固有の仕様は、パッケージごとの仕様書で定義する。

- [アーキテクチャ](../ARCHITECTURE.md)
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

現在はルートパッケージと、取得元パッケージをまとめるファサードを提供しない。

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

取得元固有の型は公開APIへ含めない。複数の取得元に共通することが確認できた
変換後の概念だけを、`api` の共通モデルへ追加する。

## 4. 現在のAPIの対象

### 4.1 確定事項

現在のAPIは、漫画本の書誌条件による検索と、ISBNによる書籍参照を対象とする。

次の機能を提供する。

- タイトルによる検索
- ISBN-10またはISBN-13による参照
- 著者名による検索
- 主要な書誌項目を横断するフリーワード検索
- 主要な書誌項目に指定語を含む結果の除外
- 取得件数の指定
- カーソルによる続きの取得
- データ取得元固有の形式から共通の書籍モデルへの変換
- 入力エラー、外部サービスエラー、レスポンス解析エラーの分類

次の機能は現在のAPIに含めない。

- 漫画シリーズの検索
- 複数データ取得元の横断検索
- 検索結果の重複統合
- キャッシュ
- 自動リトライ
- クライアント側のレート制限
- 汎用CLIアプリケーション（`examples` は公開APIの動作確認用デモとする）
- MCPサーバー

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
	ParallelTitles    []string           `json:"parallel_titles,omitempty"`
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

- `Normalized` は変更差分や取得元の元レスポンスではなく、取得元の値から導いた、
  利用者が通常参照する共通書誌情報とする
- 加工不要な取得元値も、共通項目として採用できる場合は `Normalized` へコピーする
- `Sources` は正規化に使用した取得元、取得元内ID、参照URL、主要な元値を保持し、
  取得元固有の全レスポンスを表すものではない
- 取得元固有の全レスポンスは `Book` に含めず、専用のRaw response用メソッドで返す
- `Title` は単数とし、分解できない場合は取得元タイトル全体を暫定値にする
- `ParallelTitles` は、主タイトルと同じ内容を別の言語または文字体系で表した
  タイトルを、取得元が示した順序で保持する
- 取得元で並列タイトルと確認できない別題、略題、表紙題、原題は
  `ParallelTitles` に含めない
- 取得元タイトルを安全に分解できた場合も、完全な元タイトルは変更せず
  `Sources[].Values.Titles` に残す。分解できない場合は全体を `Title` に設定する
- `TitleKana` は安全に分解できない場合に空にし、取得元の読みは `Sources` に残す
- `Subtitle` は単数とする
- `Authors` は利用者向けの著者名、`Contributors` は名前と複数の役割を保持する
- 取得元が寄与者の順序を提供する場合は、その順序を維持する
- 取得元が順序を提供しない場合は、取得元パッケージが結果を安定化し、
  その順序に意味がないことを取得元固有の仕様へ記載する
- `IsFinalVolume` は完結巻の肯定情報だけを表し、`完全版`から推測しない
- 現在の実装では複数取得元のBookを自動統合しない

### 5.3 SourceBookValues

取得元の値は、文字列の内容を変更せず次の型へ保持する。これらは正規化に使用した
主要な元値であり、取得元固有の全レスポンスではない。取得元が同じ意味の値を
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

## 6. 検索・参照API

### 6.1 SearchBooksRequest

検索条件は、次の型で受け取る。

```go
type SearchBooksRequest struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}
```

各フィールドの共通の役割は次のとおり。

- `Title` はタイトルの検索条件を表す
- `Author` は著者名の検索条件を表す
- `FreeText` は複数の書誌項目を対象とする検索条件を表す
- `ExcludedText` は検索結果から除外する条件を表す
- `Limit` は1回に取得する最大件数を表す
- `Cursor` は続きの取得に使用する値を表す

検索対象の項目、複数条件の組み合わせ、入力値の検証、Limit、カーソルの規則は、
データ取得元パッケージごとの契約とする。現在のMADB検索では
[MADBパッケージ仕様](pkg/madb/spec.md#6-検索条件)と
[Limitとページング](pkg/madb/spec.md#8-limitとページング)で定義する。

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
- 合計件数は現在のAPIに含めない

### 6.3 ISBN参照

ISBNによる書籍参照は、検索条件、Limit、カーソルを持たない専用メソッドで行う。
取得元パッケージは次の共通結果型を返す。

```go
type ISBNLookupResult struct {
	Items []ISBNLookupItem `json:"items"`
}

type ISBNLookupItem struct {
	RequestedISBN string `json:"requested_isbn"`
	Books         []Book `json:"books"`
}
```

#### 確定事項

- 1件以上のISBNを必須とし、最大件数は取得元パッケージごとに定義する
- `RequestedISBN` は呼び出し側が指定した文字列を変更せず保持する
- `Items` は入力と同じ件数、同じ順序で返す
- 同じISBNを複数回指定した場合も入力位置ごとに要素を返す
- ISBN-10と対応するISBN-13を問い合わせ時に重複除去しても、元の入力位置へ展開する
- 該当なしはエラーとせず、非nilの空の `Books` を返す
- 同じISBNに複数書籍が対応する場合は、統合せず `Books` にすべて返す
- 1件でも不正なISBNがある場合は、外部通信せず呼び出し全体を `invalid_argument` にする

現在の取得元別上限はMADBが500件、将来実装するopenBDが1,000件とする。
この差は共通型へ埋め込まず、各クライアントが入力検証する。

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

データ取得元ごとのHTTPステータスと `ErrorKind` の対応は、各パッケージの仕様書で
定義する。未実装機能と着手時期を決めていない作業は [Backlog](backlog.md) で管理する。
