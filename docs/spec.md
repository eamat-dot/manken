# manken API仕様

## 1. 目的

`manken` は、複数のGoプロジェクトから漫画本の書誌情報を検索するためのライブラリである。

データ取得元ごとの通信方式やレスポンス形式を共通APIから分離し、利用側は取得元の詳細を
意識せずに検索条件と書籍モデルを扱えるようにする。

## 2. 関連仕様

データ取得元に固有の仕様は、パッケージごとの仕様書で定義する。

- [アーキテクチャ](../ARCHITECTURE.md)
- [MADBパッケージ仕様](pkg/madb/spec.md)
- [openBDパッケージ仕様](pkg/openbd/spec.md)
- [Google Booksパッケージ仕様](pkg/googlebooks/spec.md)

## 3. モジュールとパッケージ

モジュールパスは次のとおり。

```text
github.com/eamat-dot/manken
```

最低Goバージョンは1.26.0とし、外部モジュールへ依存しない。

提供するパッケージは次の4つである。

```text
github.com/eamat-dot/manken/api
github.com/eamat-dot/manken/madb
github.com/eamat-dot/manken/openbd
github.com/eamat-dot/manken/googlebooks
```

- `api`
  - データ取得元に依存しない検索条件、検索結果、書籍モデル、エラー分類を定義する
- `madb`
  - MADBへの問い合わせと、取得結果から共通モデルへの変換を担当する
  - 通常利用に必要な `api` の型と定数をエイリアスとして公開する
- `openbd`
  - openBDへのISBN問い合わせと、取得結果から共通モデルへの変換を担当する
  - 通常利用に必要な `api` の型と定数をエイリアスとして公開する
- `googlebooks`
  - Google Booksへの検索・ISBN問い合わせと、取得結果から共通モデルへの変換を担当する
  - 通常利用に必要な `api` の型と定数をエイリアスとして公開する

ルートパッケージと、取得元パッケージをまとめるファサードは提供しない。

### 3.1 取得元パッケージとの境界

`api` パッケージは、取得元に依存しない公開型と、その型が満たす仕様だけを
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

## 4. APIの対象

このライブラリは、漫画本の書誌条件による検索と、ISBNによる書籍参照を対象とする。

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

複数取得元をまたぐ検索・重複統合、汎用CLI、MCPサーバーは公開APIとして
提供しない。将来の実装候補は [Backlog](backlog.md) で管理する。

現行ライブラリはキャッシュ、自動再試行、クライアント側のレート制限を提供しない。
必要な場合は、呼び出し側が `http.Client` やその周辺処理で制御する。

## 5. 共通モデル

### 5.1 Source

`Source` は書誌情報の取得元を識別する文字列型である。`madb`、`openbd`、`googlebooks`、
`rakutenbooks` をそれぞれ `SourceMADB`、`SourceOpenBD`、`SourceGoogleBooks`、
`SourceRakutenBooks` として定義する。取得元パッケージは自身の定数をエイリアスとして公開し、
JSONではこの短い文字列を出力する。

### 5.2 Book

`Book` は検索またはISBN参照で得た1冊を表す。`Normalized` は通常参照する
共通書誌情報、`Sources` は書誌情報を取得した取得元への参照を保持する。

| Goフィールド | JSON項目 | 型 | 必須・省略 | 意味 |
| --- | --- | --- | --- | --- |
| `Normalized` | `normalized` | `NormalizedBook` | 常に出力 | 取得元に依存せず利用できる共通書誌情報。取得元応答との差分、変換履歴、項目ごとの出典は表さない。 |
| `Sources` | `sources` | `[]BookSource` | 常に出力 | 書誌情報を取得したサービスへの参照。取得元固有の本文や項目は含めない。 |

- `Normalized` には、取得元が返す書誌情報のうち、複数の取得元で共通の意味として
  扱える値だけを設定する
- `Normalized` は取得元応答との差分、変換履歴、項目ごとの出典を表さない
- `Sources` は取得元への参照だけを保持し、取得元固有の値を含めない
- 取得元固有の全レスポンスは `Book` に含めず、専用のRaw response用メソッドで返す
- 欠落項目や項目間の対応を推測で補完しない。取得元ごとの安全な変換規則は、
  各パッケージ仕様で定める
- 取得元が順序を提供する場合は維持する。順序を提供しない場合は結果を安定化し、
  その順序に意味がないことを取得元固有の仕様へ記載する

`NormalizedBook` の項目は次のとおり。

| Goフィールド | JSON項目 | 型 | 省略条件 | 意味 |
| --- | --- | --- | --- | --- |
| `Title` | `title` | `string` | 空文字列 | 主タイトル。複数候補の選択や取得元タイトルの分解規則は取得元仕様に従う。 |
| `ParallelTitles` | `parallel_titles` | `[]string` | 空スライス | 主タイトルと同じ内容を別言語または別文字体系で表すタイトル。別題、略題、表紙題、原題は含めない。 |
| `TitleReading` | `title_reading` | `string` | 空文字列 | 主タイトルの読み。取得元が主タイトルとの対応を示し、安全に採用できる場合だけ設定する。 |
| `Subtitle` | `subtitle` | `string` | 空文字列 | 副題。複数候補から1件を選ぶ規則は取得元仕様に従う。 |
| `Series` | `series` | `[]Series` | 空スライス | シリーズ名と、取得元が提供できるID、URL、取得元。順序と重複除去は取得元仕様に従う。 |
| `Volume` | `volume` | `Volume` | `Number` と `Label` がともに空 | 巻数。整数化できる場合は `Number`、正規化済み表示は `Label` に設定する。 |
| `EditionStatements` | `edition_statements` | `[]string` | 空スライス | 版表示。通常版、新装版などの独自分類へ変換しない。 |
| `IsFinalVolume` | `is_final_volume` | `bool` | `false` | 完結巻であることを取得元が肯定している場合だけ `true` にする。 |
| `Authors` | `authors` | `[]string` | 空スライス | 表示や簡易利用向けの著者名一覧。含める役割、順序、重複除去は取得元仕様に従う。 |
| `Contributors` | `contributors` | `[]Contributor` | 空スライス | 人物単位の寄与者情報。読みや共通役割がない人物も保持できる。 |
| `Publishers` | `publishers` | `[]string` | 空スライス | 出版社。取得元仕様で定めた安全な整形だけを行う。 |
| `Imprints` | `imprints` | `[]string` | 空スライス | レーベル。取得元の表記から独自分類を推測しない。 |
| `Identifiers` | `identifiers` | `[]Identifier` | 空スライス | ISBN-10、ISBN-13、JANなど、種類を明示した識別子。 |
| `Dates` | `dates` | `[]BookDate` | 空スライス | 出版日、発売日、電子版配信開始日を区別した日付。取得元の精度を保つ。 |
| `Description` | `description` | `string` | 空文字列 | 説明文。取得元が共通の意味で提供できる場合だけ設定する。 |
| `Languages` | `languages` | `[]string` | 空スライス | 言語。取得元の値から推測しない。 |
| `Subjects` | `subjects` | `[]Subject` | 空スライス | 取得元の分類体系に基づく主題またはジャンル。 |
| `PageCount` | `page_count` | `*int` | `nil` | ページ数。0が取得元の明示値である場合は保持できる。 |
| `Medium` | `medium` | `PublicationMedium` | 空文字列 | `print` または `digital`。判定できない場合は設定しない。 |
| `PhysicalSize` | `physical_size` | `*PhysicalSize` | `nil` | 紙書籍の判型名とミリメートル単位の寸法。 |
| `Prices` | `prices` | `[]Price` | 空スライス | 種類と出典を明示した価格。価格0円は欠落ではない。 |
| `Images` | `images` | `[]Image` | 空スライス | 表紙など書籍に関係する画像。 |

### 5.3 BookSource

| Goフィールド | JSON項目 | 型 | 必須・省略 | 意味 |
| --- | --- | --- | --- | --- |
| `Source` | `source` | `Source` | 必須 | 書誌情報を取得したサービス。空文字列の `Source` を持つ要素は作らない。 |
| `ID` | `id` | `string` | 空文字列なら省略 | 取得元内の書籍または商品を識別する値。 |
| `URL` | `url` | `string` | 空文字列なら省略 | 取得元が提供する書籍参照URL。 |

`BookSource` は取得元への参照だけを表し、取得元固有の本文や項目を含めない。
無加工の取得元本文が必要な場合は、各パッケージの `WithRawResponse` 系メソッドを
使用する。

### 5.4 巻数と識別子

- `Number` は整数化できる場合だけ設定し、実在する0巻を保持できるよう `*int` とする
- `Label` は `16`、`上`、`前編`、`1.5`、`別巻`など正規化済みの表示を保持する
- 並べ替え用Indexは公開モデルへ追加しない
- ISBN-10、ISBN-13、JANは `IdentifierType` で区別する
- 取得元内だけで意味を持つ商品IDは `BookSource.ID` に置く

### 5.5 寄与者

`Contributor` の項目は次のとおり。

| Goフィールド | JSON項目 | 型 | 必須・省略 | 意味 |
| --- | --- | --- | --- | --- |
| `Name` | `name` | `string` | 必須 | 取得元が人物単位で示した名前。空文字列の要素は作らない。 |
| `Reading` | `reading` | `string` | 空文字列なら省略 | 同じ取得元要素で名前との対応が示された読み。別要素の値から対応を推測しない。 |
| `Roles` | `roles` | `[]ContributorRole` | 空スライスなら省略 | 安全に対応付けられた共通役割。空の場合は役割表記がないか、共通役割へ対応できなかったことを表す。 |

- `ContributorRole` は取得元固有の表記やONIXコードをそのまま公開せず、
  複数の取得元で意味が通じる一般名を使用する
- `Contributor` は、取得元から人物単位で取得できた情報を保持する
- `Authors` は表示や簡易利用向けの名前一覧である。どの人物を含めるかは、
  取得元ごとの仕様で定める
- `Authors` と `Contributors` は用途が異なり、一方から他方を完全には再構成できない。
  同じ人物が両方に含まれる場合もある
- 同じ人物を推測して統合しない。同一人物を複数の要素で表す取得元の扱いは、
  取得元ごとの仕様で定める
- 寄与者の順序、同名要素の統合、役割の重複除去は取得元ごとの仕様で定める

### 5.6 シリーズ、日付、紙・電子

`Series` は名前と、それに対応する取得元内ID、URL、取得元を同じ要素へ保持する。
`BookDate` は出版日、発売日、電子版配信開始日を区別し、日付文字列の精度を保つ。
`PublicationMedium` は `print`、`digital`、不明の空文字列を取る。紙書籍だけが
`PhysicalSize` に判型名とミリメートル単位の寸法を設定する。

### 5.7 価格

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

各フィールドの共通の役割は次のとおり。

- `Title` はタイトルの検索条件を表す
- `Author` は著者名の検索条件を表す
- `FreeText` は複数の書誌項目を対象とする検索条件を表す
- `ExcludedText` は検索結果から除外する条件を表す
- `Limit` は1回に取得する最大件数を表す
- `Cursor` は続きの取得に使用する値を表す

検索対象の項目、複数条件の組み合わせ、入力値の検証、Limit、カーソルの規則は、
データ取得元パッケージごとの仕様とする。MADB検索では
[MADBパッケージ仕様](pkg/madb/spec.md#6-検索条件)と
[Limitとページング](pkg/madb/spec.md#8-limitとページング)、Google Books検索では
[Google Booksパッケージ仕様](pkg/googlebooks/spec.md#3-検索)で定義する。Google Booksでも
`ExcludedText` を受け付けるが、取得元固有の除外構文へ安全に変換する。

### 6.2 SearchBooksResult

`SearchBooksResult.Books` は該当した書籍を返す。該当がなければ非nilの空スライスを返す。

- 続きがない場合は `NextCursor` を空文字列にする
- 合計件数は公開APIに含めない

### 6.3 ISBN参照

ISBNによる書籍参照は、検索条件、Limit、カーソルを持たない専用メソッドで行う。
取得元パッケージは、入力ISBNごとに `RequestedISBN` と対応する `Books` を持つ
`ISBNLookupItem` の配列を `ISBNLookupResult.Items` として返す。

- 1件以上のISBNを必須とし、最大件数は取得元パッケージごとに定義する
- `RequestedISBN` は呼び出し側が指定した文字列を変更せず保持する
- `Items` は、その取得元が受け付けた入力と同じ件数、同じ順序で返す
- 複数ISBNを受け付ける取得元では、同じISBNを複数回指定した場合も入力位置ごとに要素を返す
- 複数ISBNを受け付ける取得元では、ISBN-10と対応するISBN-13を問い合わせ時に重複除去しても元の入力位置へ展開する
- 該当なしはエラーとせず、非nilの空の `Books` を返す
- 同じISBNに複数書籍が対応する場合は、統合せず `Books` にすべて返す
- 1件でも不正なISBNがある場合は、外部通信せず呼び出し全体を `invalid_argument` にする

取得元別上限はMADBが500件、openBDが1,000件、Google Booksが1件とする。
この差は共通型へ埋め込まず、各クライアントが入力検証する。

## 7. エラーAPI

利用側がエラー文字列を解析せずに原因を分類できるAPIを提供する。

`Error.Kind` は `invalid_argument`、`upstream`、`unavailable`、`invalid_response` の
いずれかである。`Operation` は失敗した公開操作、`StatusCode` はHTTP応答を得た場合の
状態コード、`RetryAfter` は取得元が提示した待機時間、`Err` は原因を保持する。

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
