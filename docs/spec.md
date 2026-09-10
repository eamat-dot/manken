# manken API仕様

## 1. 目的

`manken` は、複数のデータ取得元から漫画本を中心とした書誌情報・販売情報を検索・取得するGoライブラリである。

データ取得元ごとの通信方式やレスポンス形式は各providerパッケージで扱い、利用側は取得元固有のレスポンス型を
意識せずに検索条件と `Book` モデルを扱えるようにする。

## 2. 関連仕様

データ取得元に固有の仕様は、パッケージごとの仕様書で定義する。

- [アーキテクチャ](../ARCHITECTURE.md)
- [MADBパッケージ仕様](pkg/madb/spec.md)
- [openBDパッケージ仕様](pkg/openbd/spec.md)
- [Google Booksパッケージ仕様](pkg/googlebooks/spec.md)
- [楽天Booksパッケージ仕様](pkg/rakutenbooks/spec.md)
- [楽天Koboパッケージ仕様](pkg/rakutenkobo/spec.md)
- [NDLサーチパッケージ仕様](pkg/ndl/spec.md)
- [Yahoo!ショッピングパッケージ仕様](pkg/yahooshopping/spec.md)
- [DMMパッケージ仕様](pkg/dmm/spec.md)

## 3. モジュールとパッケージ

モジュールパスは次のとおり。

```text
github.com/eamat-dot/manken
```

最低Goバージョンは1.26.0とし、外部モジュールへ依存しない。

提供するパッケージは次の10個である。

```text
github.com/eamat-dot/manken
github.com/eamat-dot/manken/model
github.com/eamat-dot/manken/madb
github.com/eamat-dot/manken/openbd
github.com/eamat-dot/manken/googlebooks
github.com/eamat-dot/manken/rakutenbooks
github.com/eamat-dot/manken/rakutenkobo
github.com/eamat-dot/manken/ndl
github.com/eamat-dot/manken/yahooshopping
github.com/eamat-dot/manken/dmm
```

- `manken`
  - 利用側で生成したprovider Clientを登録し、明示した `Source` の検索またはISBN参照へそのまま委譲する
  - `model` の通常利用に必要な型、エラー型、定数をエイリアスまたは定数として公開する

- `model`
  - データ取得元に依存しない検索条件、検索結果、`Book` モデル、エラー分類を定義する
- `madb`
  - MADBへの問い合わせと、取得結果から `model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する
- `openbd`
  - openBDへのISBN問い合わせと、取得結果から `model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する
- `googlebooks`
  - Google Booksへの検索・ISBN問い合わせと、取得結果から `model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する
- `rakutenbooks`
  - 楽天Booksへの検索・ISBN問い合わせと、取得結果から `model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する
- `rakutenkobo`
  - 楽天Koboへの検索と、取得結果から `model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する
- `ndl`
  - 国立国会図書館サーチへの検索・ISBN参照と、DC-NDL v3から `model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する
- `yahooshopping`
  - Tower固定のYahoo!ショッピング紙書籍商品検索とISBN参照、`model.Book` への変換を担当する
- `dmm`
  - DMMブックス電子コミックのシリーズ探索と、指定シリーズ内の個別商品を `model.Book` へ限定的に変換する
  - 通常利用に必要な `model` の型と定数をエイリアスとして公開する

### 3.1 取得元パッケージとの境界

`model` パッケージは、取得元に依存しない公開型と、その型が満たす仕様だけを
定義する。取得元固有のレスポンス型、項目名、役割、欠落規則を扱わない。

利用側は `manken` のルートClientへprovider Clientを登録して検索またはISBN参照を呼び出すか、`madb` などの取得元パッケージを直接呼び出す。取得元パッケージは
通常利用に必要な `model` の型と定数をエイリアスとして公開するため、利用側が
`model` を直接インポートする必要はない。ルートClientはprovider Clientを生成せず、認証情報やprovider固有設定を保持しない。

各取得元パッケージは、外部サービスのレスポンスを固有の非公開型へ読み込み、
そのパッケージ内で `model.Book` へ変換してから返す。

```text
外部サービスのレスポンス
    ↓
取得元パッケージの非公開型
    ↓ 取得元パッケージ内で変換
model.Book
    ↓
利用側
```

例えばMADBのcreatorは、`madb` パッケージ内では `Creators` として保持し、
MADB固有の役割表記を処理した後、`model.Book.Authors` へ設定する。
`model` パッケージは `Creators` を受け取って変換する関数を持たない。

取得元固有の型は `model` パッケージの公開型へ含めない。取得元パッケージは、
`model` パッケージへ一般化しない固有の検索条件や設定を、そのパッケージ固有の公開型として提供できる。
複数の取得元に共通することが確認できた変換後の概念だけを、`model` パッケージの型へ追加する。

## 4. APIの対象

このライブラリは、漫画本の書誌条件による検索と、ISBNによる書籍参照を対象とする。

次の機能を提供する。

- タイトルによる検索
- ISBN-10またはISBN-13による参照
- 著者名による検索
- 出版社名による検索
- 主要な書誌項目を横断するフリーワード検索
- 主要な書誌項目に指定語を含む結果の除外
- 取得件数の指定
- カーソルによる続きの取得
- データ取得元固有の形式から `Book` モデルへの変換
- 入力エラー、外部サービスエラー、レスポンス解析エラーの分類

複数取得元をまたぐ検索・重複統合、汎用CLI、MCPサーバーは公開APIとして
提供しない。

ルート `manken.Client` は、`WithMADBClient`、`WithOpenBDClient`、`WithGoogleBooksClient`、`WithRakutenBooksClient`、`WithRakutenKoboClient`、`WithNDLClient`、`WithYahooShoppingClient` で完成済みprovider Clientを1つ以上登録して生成する。
provider Clientを指定しない場合は `ErrorKindInvalidArgument` となり、同じOptionを複数指定した場合は後のClientを使用する。
検索はMADB、Google Books、楽天Books、楽天Kobo、NDL、Yahoo!ショッピングに対応する。ISBN参照はMADB、openBD、Google Books、楽天Books、NDL、Yahoo!ショッピングに対応する。

`SearchBooks`、`SearchBooksWithRawResponse`、`LookupBooksByISBN`、`LookupBooksByISBNWithRawResponse` は、`Source` を引数で受け取る。
選択したproviderの結果、Raw response、エラーは変更せず返し、フォールバック、追加通信、再試行、並行実行、並べ替え、統合、重複除去、ローカル絞り込みを行わない。
未初期化Client、`nil` 設定、`nil` provider Client、未知または未登録の `Source`、非対応操作は `ErrorKindInvalidArgument` の `*Error` として返す。

現行ライブラリはキャッシュ、自動再試行、クライアント側のレート制限を提供しない。
必要な場合は、呼び出し側が `http.Client` やその周辺処理で制御する。

## 5. `model` パッケージの型

### 5.1 Source

`Source` は書誌情報の取得元を識別する文字列型である。`madb`、`openbd`、`googlebooks`、
`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` をそれぞれ `SourceMADB`、`SourceOpenBD`、`SourceGoogleBooks`、
`SourceRakutenBooks`、`SourceRakutenKobo`、`SourceNDL`、`SourceYahooShopping`、`SourceDMM` として定義する。取得元パッケージは自身の定数をエイリアスとして公開し、
JSONではこの短い文字列を出力する。

### 5.2 Book

`Book` モデルは検索またはISBN参照で得た1冊を表す。定義元は `model.Book` であり、
ルートパッケージと各providerパッケージの `Book` はそのtype aliasである。
タイトル、著者、ISBNなどの書誌情報に加え、価格や表紙画像URLも `Book` の直下に保持し、
`Sources` は情報を取得したサービスと参照先を保持する。

| Goフィールド         | JSON項目               | 型                  | 省略条件                       | 意味                                                                                                                 |
| -------------------- | ---------------------- | ------------------- | ------------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| `Title`              | `title`                | `string`            | 空文字列                       | 取得元から採用した完全なタイトル文字列。巻数や版表示を別項目へ抽出しても、この値から除去しない。                     |
| `TitleReading`       | `title_reading`        | `string`            | 空文字列                       | タイトルの読み。取得元がタイトルとの対応を示し、安全に採用できる場合だけ設定する。                                   |
| `Subtitle`           | `subtitle`             | `string`            | 空文字列                       | 副題。取得元から安全に分離できる場合だけ設定する。                                                                   |
| `BookSeries`         | `book_series`          | `[]BookSeries`      | 空スライス                     | 作品または個別Bookの系列。名称、取得元内ID、URL、取得元を同じ要素へ保持する。                                        |
| `PublicationSeries`  | `publication_series`   | `[]string`          | 空スライス                     | 叢書、刊行シリーズ、コミックレーベル、Publisher collection等の出版側グループ表示名。取得元が返す読みは共通化しない。 |
| `Volume`             | `volume`               | `Volume`            | `Number` と `Label` がともに空 | 巻数。整数化できる場合は `Number`、正規化済み表示は `Label` に設定する。                                             |
| `Editions`           | `editions`             | `[]string`          | 空スライス                     | 特装版、新装版など、取得元から安全に抽出できる版表示。                                                               |
| `IsFinalVolume`      | `is_final_volume`      | `bool`              | `false`                        | 完結巻であることを取得元が肯定している場合だけ `true` にする。                                                       |
| `Authors`            | `authors`              | `[]string`          | 空スライス                     | 表示や簡易利用向けの著者名一覧。含める役割、順序、重複除去は取得元仕様に従う。                                       |
| `Contributors`       | `contributors`         | `[]Contributor`     | 空スライス                     | 人物単位の寄与者情報。読みや共通役割がない人物も保持できる。                                                         |
| `Publishers`         | `publishers`           | `[]string`          | 空スライス                     | 出版社。取得元仕様で定めた安全な整形だけを行う。                                                                     |
| `ISBN10`             | `isbn10`               | `[]string`          | 空スライス                     | 検証済みのISBN-10。                                                                                                  |
| `ISBN13`             | `isbn13`               | `[]string`          | 空スライス                     | 検証済みのISBN-13。                                                                                                  |
| `JAN`                | `jan`                  | `[]string`          | 空スライス                     | ISBNとして扱わないJANコード。                                                                                        |
| `PublishedDate`      | `published_date`       | `string`            | 空文字列                       | 出版日。`YYYY`、`YYYY-MM`、`YYYY-MM-DD` など取得元の精度を保つ。                                                     |
| `ReleaseDate`        | `release_date`         | `string`            | 空文字列                       | 発売日。取得元の精度を保つ。                                                                                         |
| `DigitalReleaseDate` | `digital_release_date` | `string`            | 空文字列                       | 電子版配信開始日。取得元がその意味を明示する場合だけ設定する。                                                       |
| `Description`        | `description`          | `string`            | 空文字列                       | 説明文。取得元が共通の意味で提供できる場合だけ設定する。                                                             |
| `Languages`          | `languages`            | `[]string`          | 空スライス                     | 言語。取得元の値から推測しない。                                                                                     |
| `Subjects`           | `subjects`             | `[]Subject`         | 空スライス                     | 取得元の分類体系に基づく主題またはジャンル。                                                                         |
| `PageCount`          | `page_count`           | `*int`              | `nil`                          | ページ数。0が取得元の明示値である場合は保持できる。                                                                  |
| `Medium`             | `medium`               | `PublicationMedium` | 空文字列                       | `print` または `digital`。判定できない場合は設定しない。                                                             |
| `Size`               | `size`                 | `string`            | 空文字列                       | 取得元が書籍サイズとして返す文字列表現。寸法や判型へ独自に変換しない。                                               |
| `ListPrice`          | `list_price`           | `*Price`            | `nil`                          | 定価・通常価格として扱える価格。価格0円と欠落を区別する。                                                            |
| `CurrentPrice`       | `current_price`        | `*Price`            | `nil`                          | API取得時点の商品価格。価格0円と欠落を区別する。                                                                     |
| `CoverURL`           | `cover_url`            | `string`            | 空文字列                       | 取得元の利用可能な表紙画像候補から採用した1件のURL。複数サイズがある場合は最大候補を優先する。                       |
| `Sources`            | `sources`              | `[]BookSource`      | 常に出力                       | 情報を取得したサービスと、そのサービス上の参照先。                                                                   |

- `Title` は取得元タイトルを非破壊で保持する。`Volume`、`Editions` などを設定するためにタイトル文字列を短縮しない
- 取得元の明示 `Volume` / `Editions` を優先する。欠落時は、安全なタイトル構文から `Volume`、`Editions`、安全な巻表示に隣接する完結表示を補う場合がある。完結表示だけから `IsFinalVolume` は推測しない。タイトルから `Subtitle`、系列、著者などは推測しない
- タイトルから補う完結表示は、安全な巻表示へ隣接する `完`、`（完）`、`(完)`、`＜完＞`、`<完>` に限る
- 複数providerで同じ意味として扱えない取得元固有項目を `Book` へ複製しない。必要な場合は各パッケージのRaw response用メソッドを使用する
- 欠落項目や項目間の対応を推測で補完しない。取得元ごとの安全な変換規則は各パッケージ仕様で定める
- 取得元が順序を提供する場合は維持する。順序を提供しない場合は結果を安定化し、その順序に意味がないことを取得元仕様へ記載する

### 5.3 BookSource

| Goフィールド   | JSON項目        | 型       | 必須・省略       | 意味                                                                   |
| -------------- | --------------- | -------- | ---------------- | ---------------------------------------------------------------------- |
| `Source`       | `source`        | `Source` | 必須             | 書籍情報を取得したサービス。空文字列の `Source` を持つ要素は作らない。 |
| `ID`           | `id`            | `string` | 空文字列なら省略 | 取得元内の書籍または商品を識別する値。                                 |
| `URL`          | `url`           | `string` | 空文字列なら省略 | 取得元が提供する通常の書籍・商品参照URL。                              |
| `AffiliateURL` | `affiliate_url` | `string` | 空文字列なら省略 | 同じ取得元が提供するアフィリエイト用URL。通常URLとは別に保持する。     |

`BookSource` は取得元と標準化した参照先情報を表し、取得元固有の本文や未共通化項目を含めない。
アフィリエイトURLを取得できる場合も `URL` を置き換えず、`AffiliateURL` に分離する。
無加工の取得元本文が必要な場合は、各パッケージの `WithRawResponse` 系メソッドを
使用する。

### 5.4 巻数と識別子

- `Number` は整数化できる場合だけ設定し、実在する0巻を保持できるよう `*int` とする
- `Label` は `16`、`上`、`前編`、`1.5`、`別巻`など正規化済みの表示を保持する
- 並べ替え用Indexは公開モデルへ追加しない
- ISBN-10、ISBN-13、JANはそれぞれ `ISBN10`、`ISBN13`、`JAN` に保持する
- 取得元内だけで意味を持つ商品IDは `BookSource.ID` に置く

### 5.5 寄与者

`Contributor` の項目は次のとおり。

| Goフィールド | JSON項目  | 型         | 必須・省略         | 意味                                                                                   |
| ------------ | --------- | ---------- | ------------------ | -------------------------------------------------------------------------------------- |
| `Name`       | `name`    | `string`   | 必須               | 取得元が人物単位で示した名前。空文字列の要素は作らない。                               |
| `Reading`    | `reading` | `string`   | 空文字列なら省略   | 同じ人物との対応が取得元から確認できる読み。別要素から対応を推測しない。               |
| `Roles`      | `roles`   | `[]string` | 空スライスなら省略 | 安全に対応付けられた共通の日本語役割名。未知roleを既知の役割へ推測分類しない。         |

- `Roles` は取得元固有の表記やONIXコードをそのまま公開せず、`著者`、`原作`、`作画` などmankenで使用する日本語の役割名へ変換する
- 公開する役割一覧をenumとして固定しない。取得元で安全に意味を確認できた役割だけを変換する
- `Contributor` は取得元から人物単位で取得できた情報を保持する
- `Authors` は表示や簡易利用向けの名前一覧であり、どの人物を含めるかは取得元ごとの仕様で定める
- `Authors` と `Contributors` は用途が異なり、一方から他方を完全には再構成できない。同じ人物が両方に含まれる場合もある
- 同じ人物を推測して統合しない。寄与者の順序、同名要素の統合、役割の重複除去は取得元ごとの仕様で定める

### 5.6 シリーズ、日付、紙・電子、サイズ

`BookSeries` は取得元が作品または個別の `Book` の系列として明示する値であり、名称、ID、URL、取得元を同じ要素へ保持する。`PublicationSeries` は叢書、刊行シリーズ、コミックレーベル、Publisher collection等の出版側グループ表示名を保持する。取得元が返す読みをprovider間で統一せず、文字列内容から両者を相互に推測しない。

日付は `PublishedDate`、`ReleaseDate`、`DigitalReleaseDate` に用途を分けて保持する。値は文字列とし、取得元が返す `YYYY`、`YYYY-MM`、`YYYY-MM-DD` などの精度を保つ。日付の意味を取得元の根拠なく別用途へ読み替えない。

`PublicationMedium` は `print`、`digital`、不明の空文字列を取る。`Size` は取得元が書籍サイズとして返す文字列表現を保持し、高さ・幅・厚さや判型へ独自に構造化しない。

### 5.7 価格

`Price` は金額、通貨、税込情報、取得元、観測時刻を保持する。価格の用途は `Book.ListPrice` と `Book.CurrentPrice` のフィールド名で区別する。NDLの書誌に記録された販売価格は税込情報不明の `ListPrice` として、openBDの確認済みONIX推奨小売価格はONIXの `PriceType` に応じた税込情報付き `ListPrice` として扱う。どちらも `CurrentPrice` には入れない。

- `ListPrice` は定価または通常価格として安全に扱える場合に設定する
- `CurrentPrice` はAPI取得時点の商品価格として安全に扱える場合に設定する
- `Amount` は整数値、`Currency` は通貨、`Source` は価格の取得元を表す
- `ObservedAt` は取得時点価格で観測時刻を記録する取得元が設定する
- `TaxIncluded` は取得元で確認できる場合だけ設定する
- `*Price` を使うため、価格0円と価格欠落を区別できる
- 在庫、送料、ポイント、会員価格は `Book` に含めない

### 5.8 JSONの欠落項目

- `sources` は常に出力する
- その他の空文字列、空スライス、`nil` ポインター、`IsFinalVolume=false` は省略する
- 0巻、価格0円、明示された `TaxIncluded=false` は省略しない
- 空の `Volume` 全体は省略する
- `Book` のJSONはタイトルや価格などのフィールドを直下に出力し、`normalized` 階層を持たない

## 6. 検索・参照API

### 6.1 SearchRequest

`SearchRequest` の各フィールドの基本的な意味は次のとおり。

- `Title` はタイトルの検索条件を表す
- `Author` は著者名の検索条件を表す
- `Publisher` は出版社名の検索条件を表す
- `Query` は複数の書誌項目を対象とする検索条件を表す
- `Exclude` は検索結果から除外する条件を表す
- `DateFrom` と `DateTo` は、対応する取得元の時期で結果を絞る開始と終了の条件を表す
- `Limit` は1回に取得する最大件数を表す
- `Cursor` は続きの取得に使用する値を表す

`DateFrom` と `DateTo` は `YYYY`、`YYYY-MM`、`YYYY-MM-DD` を受け付ける。両方を指定する場合は同じ精度とし、開始が終了より後の値は受け付けない。開始は指定精度の期間開始、終了は期間終端を含む。何の時期を検索するかはproviderごとに異なり、対応しないproviderは入力エラーにする。JSONでは `Query`、`Exclude`、`DateFrom`、`DateTo` をそれぞれ `query`、`exclude`、`date_from`、`date_to` として出力する。

検索対象の項目、複数条件の組み合わせ、入力値の検証、Limit、カーソルの規則は、
データ取得元パッケージごとの仕様とする。MADB検索では
[MADBパッケージ仕様](pkg/madb/spec.md#6-検索条件)と
[Limitとページング](pkg/madb/spec.md#8-limitとページング)、Google Books検索では
[Google Booksパッケージ仕様](pkg/googlebooks/spec.md#3-検索)で定義する。Google Booksでも
`Exclude` を受け付けるが、取得元固有の除外構文へ安全に変換する。楽天Books検索では
[楽天Booksパッケージ仕様](pkg/rakutenbooks/spec.md#5-searchbooks)、楽天Kobo検索では
[楽天Koboパッケージ仕様](pkg/rakutenkobo/spec.md#4-searchbooks)で定義する。

### 6.2 SearchBooksResult

`SearchBooksResult.Books` は該当した書籍を返す。該当がなければ `nil` ではない空スライスを返す。

- 続きがない場合は `NextCursor` を空文字列にする
- `Attributions` は、その取得元が定型の出典・クレジット情報を提供する場合に設定する。成功した検索では0件でも設定し、未対応の取得元では空とする
- JSONでは `Attributions` を `attributions` として出力し、空の場合は省略する
- 合計件数は公開APIに含めない。取得元ごとに件数の意味、上限、提供可否が異なるため、APIを揃えるためだけの追加COUNTリクエストや推測を行わない

### 6.3 ISBN参照

ISBNによる書籍参照は、検索条件、Limit、カーソルを持たない専用メソッドで行う。
取得元パッケージは、入力ISBNごとに `RequestedISBN` と対応する `Books` を持つ
`ISBNLookupItem` の配列を `ISBNLookupResult.Items` として返す。
`ISBNLookupResult.Attributions` は検索結果と同じ出典・クレジット情報を保持し、JSONでは空の場合に省略する。

- 1件以上のISBNを必須とし、最大件数は取得元パッケージごとに定義する
- `RequestedISBN` は呼び出し側が指定した文字列を変更せず保持する
- `Items` は、その取得元が受け付けた入力と同じ件数、同じ順序で返す
- 複数ISBNを受け付ける取得元では、同じISBNを複数回指定した場合も入力位置ごとに要素を返す
- 複数ISBNを受け付ける取得元では、ISBN-10と対応するISBN-13を問い合わせ時に重複除去しても元の入力位置へ展開する
- 該当なしはエラーとせず、`nil` ではない空の `Books` を返す
- 同じISBNに複数書籍が対応する場合は、統合せず `Books` にすべて返す
- 1件でも不正なISBNがある場合は、外部通信せず呼び出し全体を `invalid_argument` にする。`ISBNLookupItem` は入力ごとのエラーを表現しないため、無効入力だけを除いた部分通信は行わない

取得元別上限はMADBが500件、openBDが1,000件、Google Books、楽天Books、NDLサーチ、
Yahoo!ショッピングがそれぞれ1件とする。この差は `ISBNLookupResult` などの公開型へ埋め込まず、各クライアントが入力検証する。
取得元が複数ISBNを1回の問い合わせで
受け付けない場合、各providerの呼び出し方を揃えるためだけに内部で複数HTTPリクエストへ展開しない。
1回の公開呼び出しに伴う通信回数、レート制限、部分失敗の境界を暗黙に変えないためである。

### 6.4 出典・クレジット情報

`Attribution` は取得元に関する表示・保存用のメタデータであり、個々の `Book` には複製しない。

- `Source` は対象の取得元を表す
- `Scope` は `service` または `data` とする。`service` はAPIやサービスを利用していること自体の表示、`data` は取得データの出典やライセンスに関する情報を表す
- `Text` は利用側が表示・保存に使えるクレジット文、`URL` はその取得元またはサービスの参照先を表す
- `License` と `LicenseURL` は、適用するライセンスを安全に特定できる場合だけ設定する
- `RequirementsURL` は、追加の表示方法や利用条件を確認する公式ページを表す
- JSONでは `Source`、`Scope` を `source`、`scope` として常に出力し、その他の空文字列は省略する

`Attribution` が返ることや `Text` と `URL` を表示することだけで、取得元の利用条件への適合を保証しない。ロゴ、指定HTML、配置、個別書籍へのリンクなど別の条件がある場合は、利用側が `RequirementsURL` と各取得元の最新条件を確認する。

定型情報を公開する取得元パッケージは、外部通信を行わない `Attributions()` から同じ情報を取得できる。返却スライスは呼び出しごとに独立し、利用側の変更が後続呼び出しへ影響しない。現在の対応取得元と具体的な値は各パッケージ仕様で定義する。

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
- `*model.Error` を `errors.As` で取得可能にする
- 取得元パッケージのエイリアスでも同じエラーを取得可能にする
- `Error` は `Unwrap` で原因を返し、標準のエラーチェーンを維持する
- 外部サービスのレスポンス本文を通常のエラーメッセージへそのまま含めない
- `Retry-After` は秒数形式とHTTP-date形式を扱う
- 外部サービスのエラー本文を `Error` に保持しない
- `Operation` の具体的な値は取得元パッケージの仕様で定義する

データ取得元ごとのHTTPステータスと `ErrorKind` の対応は、各パッケージの仕様書で
定義する。
