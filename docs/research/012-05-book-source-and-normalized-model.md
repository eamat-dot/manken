# 取得元情報と正規化書誌モデル調査

## 1. 目的

複数の書籍APIから取得した値を失わずに保持しながら、利用者が必要とする
正規化済みの漫画単行本情報を返すため、`Book` の構造を整理する。

この文書はモデルを検討するための調査案であり、確定仕様ではない。
`docs/spec.md` と現在の `api.Book` はまだ変更しない。

## 2. 背景

現在の `Titles []string` へ次の値をまとめて入れると、それぞれの意味を
区別できない。

```text
とんがり帽子のアトリエ（１６）特装版
トンガリボウシノアトリエ１６トクソウバン
とんがり帽子のアトリエ
```

これらは同じ種類のタイトル候補ではなく、次の異なる情報である。

- 取得元の商品タイトル
- 取得元の商品タイトル読み
- 正規化した作品タイトル
- 巻数
- 版表示

取得元の値を残す場所と、利用者向けに正規化した値を分ける必要がある。

## 3. 情報の三層

| 層 | 目的 | 加工 |
| --- | --- | --- |
| Raw response | APIから受信した全レスポンスを調査、診断、再確認に使う | 無加工の本文 |
| Source values | 正規化に使用した取得元の値と意味を保持する | 文字列の内容を変えない |
| Normalized | 利用者が検索、表示、比較に使う共通書誌情報を返す | 明確な規則で分解・正規化する |

Raw responseはページングや総件数など個別書籍以外の情報も含むため、
各 `Book` へ重複して埋め込まない。現在のMADBと同様に、
`SearchBooksWithRawResponse` のような別メソッドで返す。

`Source values` は取得元の全フィールドを再現するものではない。正規化に使用した
主な元値を、取得元での意味を変えずに保持する。取得元固有の全項目が必要な場合は
Raw responseを使用する。

## 4. 全体構造案

`Normalize` は処理を表す動詞に見えるため、フィールド名には処理済みの状態を示す
`Normalized` を候補とする。型名とフィールド名は未確定である。

```go
type Book struct {
	Normalized NormalizedBook
	Sources    []BookSource
}
```

紙書籍を楽天BooksやYahoo!ショッピングで発見し、ISBNを使ってopenBDから
詳細を取得する場合、`Sources` に発見元とopenBDの両方を保持できる。

## 5. `NormalizedBook` の項目案

```go
type NormalizedBook struct {
	Title             string
	TitleKana         string
	Subtitle          string
	Series            []Series
	Volume            Volume
	EditionStatements []string
	IsFinalVolume     bool

	Authors      []string
	Contributors []Contributor
	Publishers   []string
	Imprints     []string
	Identifiers  []Identifier
	Dates        []BookDate

	Description  string
	Languages    []string
	Subjects     []Subject
	PageCount    *int
	Medium       PublicationMedium
	PhysicalSize *PhysicalSize
	Prices       []Price
	Images       []Image
}
```

### 5.1 項目の意味

| 項目 | 収める情報 |
| --- | --- |
| `Title` | 共通Bookで採用した最良のタイトル。安全に分離できる場合は巻数、版表示などを除く |
| `TitleKana` | 正規化したタイトルの読み。取得元に根拠がある場合だけ設定する |
| `Subtitle` | タイトル本体と分離できる副題 |
| `Series` | シリーズ名と、それに対応する取得元内ID・参照URL |
| `Volume` | 数値化できる巻数と、「上」「下」など数値化できない巻表示 |
| `EditionStatements` | 特装版、完全版、新装版、電子限定版などの版表示 |
| `IsFinalVolume` | 取得元で、このBookが完結巻であることを明示している場合に `true` |
| `Authors` | 利用者向けに表示する著者名。取得元の表示順を維持する |
| `Contributors` | 著者、原作、作画、キャラクター原案などの名前と役割 |
| `Publishers` | 出版社、発行元 |
| `Imprints` | コミックレーベル、発行ブランド、叢書レーベル |
| `Identifiers` | ISBN-10、ISBN-13、JANなどの種別付き識別子 |
| `Dates` | 出版日、発売日、電子版配信開始日などの役割付き日付 |
| `Description` | 内容紹介、あらすじなど書誌情報として利用する文章 |
| `Languages` | 本文言語 |
| `Subjects` | 主題、読者対象、取得元のジャンルや分類 |
| `PageCount` | ページ数。未取得と0ページを区別する |
| `Medium` | 紙書籍または電子書籍 |
| `PhysicalSize` | 紙書籍の判型と高さ、幅、厚さ |
| `Prices` | 定価または取得時点の価格。単話版、無料試読版などの判定にも使う |
| `Images` | 表紙画像のURL、用途、画像サイズ |

### 5.2 タイトルの例

取得元の値:

```text
Title:      とんがり帽子のアトリエ（１６）特装版
TitleKana:  トンガリボウシノアトリエ１６トクソウバン
SeriesName: とんがり帽子のアトリエ
```

正規化結果:

```text
Title:             とんがり帽子のアトリエ
TitleKana:         トンガリボウシノアトリエ
Volume.Number:     16
Volume.Label:      16
EditionStatements: [特装版]
```

タイトル読みはタイトルとは独立して解析する。巻数と版表示を安全に分離できない場合、
`TitleKana` は空にする。取得元の読みは `BookSource.Values.TitleKana` に残る。

タイトルを安全に分解できない場合は、一部だけを推測で削らず、取得元タイトル全体を
`Normalized.Title` の暫定値として採用する。取得元にタイトルがある限り、加工できなかった
ことだけを理由に `Normalized.Title` を空にしない。

## 6. `BookSource` の項目案

```go
type BookSource struct {
	Source Source
	ID     string
	URL    string
	Values SourceBookValues
}
```

| 項目 | 収める情報 |
| --- | --- |
| `Source` | `madb`、`openbd`、`rakuten_books`、`rakuten_kobo`などの取得元 |
| `ID` | 取得元が書誌レコードまたは商品へ付与したID |
| `URL` | 取得元のレコードを確認できる参照URL |
| `Values` | 正規化に使用した取得元の値 |

### 6.1 `SourceBookValues`

取得元ごとに値の有無や型が異なるため、次は共通して保持する候補の一覧である。

```go
type SourceBookValues struct {
	Titles        []string
	TitleKana     string
	Subtitles     []string
	SeriesNames   []string
	Volume        string
	Editions      []string
	Authors       []string
	Publishers    []string
	Imprints      []string
	ISBNs         []string
	PublishedDate string
	Description   string
	GenreIDs      []string
	PageCount     *int
	Size          string
	Prices        []SourcePrice
}
```

| 項目 | 収める情報 |
| --- | --- |
| `Titles` | 取得元が返した商品名または書名。複数値を返せる取得元ではすべて保持する |
| `TitleKana` | 取得元が返したタイトル読み |
| `Subtitles` | 取得元が副題として返した値 |
| `SeriesNames` | 取得元がシリーズ名として返した値 |
| `Volume` | 取得元が巻数として返した文字列 |
| `Editions` | 取得元が版表示として返した文字列 |
| `Authors` | 取得元が返した著者表示。元の順序を維持する |
| `Publishers` | 取得元が出版社として返した値 |
| `Imprints` | 取得元がレーベルとして返した値 |
| `ISBNs` | 取得元がISBNとして返した値。検証前の値もここへ残す |
| `PublishedDate` | 取得元が出版日または発売日として返した文字列 |
| `Description` | 取得元が返した内容紹介または商品説明 |
| `GenreIDs` | 取得元が返したジャンルID。複数ジャンルを排他的に扱わない |
| `PageCount` | 取得元が明示したページ数 |
| `Size` | 取得元が返した判型または寸法の文字列 |
| `Prices` | 取得元が返した価格種別の元表記、金額、通貨、税込区分 |

```go
type SourcePrice struct {
	Type        string
	Amount      int64
	Currency    string
	TaxIncluded *bool
}
```

`SourcePrice.Type` は `listPrice`、`itemPrice`など取得元における価格種別の
元表記を保持する。共通の `list`、`current`への分類は `Normalized.Prices`で行う。

この共通構造で意味を保てない項目を無理に追加しない。取得元固有の値は
取得元パッケージのレスポンス型とRaw responseに残す。

## 7. 補助型案

### 7.1 巻数

```go
type Volume struct {
	Number *int
	Label  string
}
```

| 元表記 | `Number` | `Label` |
| --- | ---: | --- |
| `16巻` | `16` | `16` |
| `上巻` | `nil` | `上` |
| `下巻` | `nil` | `下` |
| `前編` | `nil` | `前編` |
| 不明 | `nil` | 空文字列 |
| `0巻` | `0` | `0` |
| `中巻` | `nil` | `中` |
| `後編` | `nil` | `後編` |
| `1.5巻` | `nil` | `1.5` |
| `別巻` | `nil` | `別巻` |
| `外伝` | `nil` | `外伝` |
| `番外編` | `nil` | `番外編` |

数値化できない場合を0にすると、実在する0巻と区別できないため `*int` を使う。
`Label` は取得元の元表記ではなく、共通表示に使う正規化済みの巻表示とする。
元表記は `BookSource.Values.Volume` または取得元タイトルに残す。

`上`、`中`、`下`や`前編`、`後編`の並べ替え順は `Volume` へIndexとして
保持せず、必要な箇所で内部的に導出する。小数巻は `float64` へ変換しない。

### 7.2 識別子

```go
type Identifier struct {
	Type  IdentifierType
	Value string
}
```

ISBN-10、ISBN-13、JANなどを種類と値の組で保持する。楽天Koboの
`itemNumber` のように取得元内だけで意味を持つIDは `BookSource.ID` に置く。

### 7.3 寄与者

```go
type Contributor struct {
	Name  string
	Roles []ContributorRole
}
```

同じ人物が複数の役割を持てる。`Authors` と `Contributors` は、取得元が順序を
示す場合にその順序を維持し、役割別に並べ替えない。

### 7.4 日付

```go
type BookDate struct {
	Type  BookDateType
	Value string
}
```

出版日、発売日、電子版配信開始日を区別する。`Value` は `2026`、`2026-07`、
`2026-07-31` など取得元の精度を保つ。

### 7.5 紙・電子

```go
type PublicationMedium string

const (
	PublicationMediumUnknown PublicationMedium = ""
	PublicationMediumPrint   PublicationMedium = "print"
	PublicationMediumDigital PublicationMedium = "digital"
)
```

`Format` は判型、ファイル形式、版表示と混同しやすいため使用しない。

### 7.6 判型・寸法

```go
type PhysicalSize struct {
	Name        string
	HeightMM    *int
	WidthMM     *int
	ThicknessMM *int
}
```

`Name` にはB6判、新書判、A5判などを入れる。判型名と実寸は別々に保持する。
電子書籍では `PhysicalSize` を設定しない。

### 7.7 価格

```go
type PriceType string

const (
	PriceTypeList    PriceType = "list"
	PriceTypeCurrent PriceType = "current"
)

type Price struct {
	Type        PriceType
	Amount      int64
	Currency    string
	TaxIncluded *bool
	Source      Source
	ObservedAt  string
}
```

`list` は定価、`current` はAPI取得時点の販売価格を表す。楽天Booksと
楽天Koboの `itemPrice` は、定価と同額に見える場合も `current` として扱い、
定価と推測しない。

`Type`、`Amount`、`Currency`、`Source` は必須とする。`TaxIncluded` は
取得元で確認できる場合だけ設定する。変動する `current` では `ObservedAt` を
必須とし、`list` では省略できる。

価格は購入機能のためではなく、
単話版、無料試読版などを判定する補助情報として `NormalizedBook` に保持する。

在庫、ポイント、送料、配送予定などは `Price` に含めない。

### 7.8 主題・ジャンル

```go
type Subject struct {
	Scheme string
	Code   string
	Name   string
}
```

取得元の分類体系、分類コード、表示名を保持する。一般、BL、TLのジャンルが
併記されていても除外や排他的分類には使わない。

## 8. Raw response

Raw responseは共通 `Book` へ入れず、別の戻り値として返す。

```go
SearchBooks(...)
SearchBooksWithRawResponse(...)
```

複数取得元を使う検索では、取得元と本文の組を複数返す構造が必要になる可能性が
ある。具体的な戻り値は検索APIの設計時に決める。

### 8.1 欠落項目のJSON表現

`normalized` と `sources` は常に出力する。その内側の任意項目は、値が欠落して
いる場合にJSON項目自体を省略する。

- 空文字列、空または `nil` のスライス、`nil` ポインターは省略する
- `IsFinalVolume` は `true` の場合だけ出力する
- 実在する0巻、価格0円など、意味のある0は省略しない
- `TaxIncluded` の `false` は明示値として出力し、未取得の `nil` と区別する
- `Volume` 全体が空の場合は `volume` を省略する

Goではスカラーのポインター、`omitempty`、値型構造体の `omitzero` を項目の
意味に応じて使い分ける。

## 9. 正規化の原則

- 取得元の値を変更せず `BookSource.Values` に残す。
- `NormalizedBook` は変更差分ではなく、利用者が通常参照する共通書誌情報の
  完成形とする。
- 取得元の値がそのまま共通項目として採用できる場合も、`NormalizedBook` へ
  単純コピーする。
- 値がない、不正である、意味を確定できない場合だけ、対応する正規化項目を空にする。
- 推測できない値は空文字列、空スライスまたは `nil` にする。
- 取得元にない読みを機械的に生成しない。
- 巻数や版表示として確実に識別できる部分だけタイトルから分離する。
- タイトルを安全に分解できない場合は、取得元タイトル全体を暫定的に採用する。
- ISBNとJANは形式とチェックディジットを検証してから正規化する。
- `Authors` と `Contributors` の順序を変更しない。
- 一般、BL、TLを相互排他的な分類として正規化しない。
- タイトル検索では、明示された巻数と版表示を必須条件として扱う。
- 単話版、分冊版、試読版、全巻セットなど検索対象外の商品は、`Book` へ
  変換する前に除外する。
- 合本版は通常のタイトル検索から除外する。クエリで `合本版` を明示した場合は
  合本版だけを採用し、ISBN検索で取得した場合は除外しない。

並列タイトルは初期の `NormalizedBook` へ追加しない。取得元タイトルへ結合されて
いる場合は `BookSource.Values.Titles`、取得元の独立項目として必要になった場合は
取得元型またはRaw responseに保持する。

初期実装では1回の検索につき1つの取得元を使用し、1つの `Book` に原則として
1つの `BookSource` を設定する。複数取得元の自動統合と競合解決は将来課題とする。

各取得元の実例と期待する分解候補は
[`012-06-title-volume-edition-examples.md`](012-06-title-volume-edition-examples.md)
に分離する。

## 10. 現在の `api.Book` からの変更点

| 現在 | 変更案 | 理由 |
| --- | --- | --- |
| `ID`、`Source`、`SourceURL` | `Sources []BookSource` | 発見元と詳細取得元を両方保持する |
| `Titles []string` | `Normalized.Title`、`Normalized.TitleKana`、`Sources[].Values.Titles` | タイトル、読み、元の商品名を分ける |
| `Subtitles []string` | `Normalized.Subtitle`と取得元の副題 | 正規化値と元値を分ける |
| `SeriesNames`、`SeriesID`、`SeriesURL` | `Normalized.Series []Series` | 名前、ID、URLの対応を保つ |
| `VolumeNumber string` | `Normalized.Volume`と取得元の巻数文字列 | 数値巻と「上」「下」などを区別する |
| `EditionStatements` | `Normalized.EditionStatements`と取得元の版表示 | 抽出結果の根拠を残す |
| `Authors` | `Normalized.Authors`と`Contributors` | 表示順と役割の両方を保持する |
| `ISBN10s`、`ISBN13s` | `Normalized.Identifiers` | 種類付き識別子へ一般化する |
| `PublishedDate` | `Normalized.Dates`と取得元の日付文字列 | 出版日、発売日、配信開始日を区別する |
| 該当項目なし | 説明、言語、主題、ページ数、紙・電子、判型、価格、画像 | 複数の取得元から利用できる情報を保持する |

現在の仕様にある「すべての複数値を安定した昇順で返す」という規則は、
`Authors` と `Contributors` には適用しない。順序に意味がない項目だけを
重複除去または整列する。

## 11. `Book` に含めない情報

- 在庫状況
- 発送予定
- 送料
- ポイント還元
- 会員限定価格
- アフィリエイトURL
- ストア評価
- レビュー件数、平均点
- 検索カーソル
- 一般、BL、TLの検索指定
- 検索結果から除外した理由
- Raw response本体

## 12. 初期公開契約の方針

- `Book.Normalized`、`Book.Sources`、`BookSource.Values` の名称を使用する
- JSON項目名は `normalized`、`sources`、`values` とする
- `Subtitle` は単数の文字列とする
- 現在の `api.Book` は移行期間を設けず、破壊的に置き換える
- 欠落した任意項目はJSONから省略する
- 価格は定価と取得時点価格を区別し、取得時点価格には取得日時を付ける

`SourceBookValues` は、正規化に使用する取得元の主要項目だけを共通化する。
MADBが複数値を返せるタイトル、副題、シリーズ名、版表示、著者、出版社、
レーベル、ISBNはスライスですべて保持する。共通の意味を保てない取得元固有項目と、
複数取得元のRaw response契約は初期実装に含めない。

複数取得元の優先順位は初期実装の対象外とする。将来統合する場合は、同じISBNまたは
取得元商品IDで同一商品と確認できることを前提とし、専用項目もタイトルなどとの
内部整合性を検証してから採用する。

初期実装の範囲と実施項目は
[`012-redesign-common-book-model.md`](../todo/done/012-redesign-common-book-model.md)
に分離する。
