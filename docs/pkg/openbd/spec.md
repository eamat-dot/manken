# openbd パッケージ仕様

## 1. 目的

`openbd` パッケージは、複数のISBNからopenBDの書誌情報を1回で参照し、
`model.Book` へ変換する。

本仕様書は、openBD固有のISBN参照、項目対応、HTTP処理、エラー分類を定義する。
`model` パッケージの型の仕様は [manken API仕様](../../spec.md)、パッケージの責務境界は
[アーキテクチャ](../../../ARCHITECTURE.md)で定義する。

## 2. 参照資料とサービスの状態

参照する公式資料は次のとおりである。

- [openBD公式サイト](https://openbd.jp/)
- [書誌APIデータ仕様](https://openbd.jp/spec/)
- [openBD API利用規約](https://openbd.jp/terms/)
- [openBD API（バージョン1）の提供終了について](https://openbd.jp/news/20230725.html)
- [代替書誌情報の提供開始について](https://openbd.jp/news/20230829.html)

従来のopenBD APIバージョン1は提供終了が告知されている。現在の同一エンドポイントは、
国立国会図書館の書誌情報を従来のONIX形式へ簡易変換した代替APIを含む。
プログラム上の互換性は維持されているが、項目の欠落と書影収録範囲の縮小がある。

## 3. パッケージとClient

インポートパスは次のとおり。

```text
github.com/eamat-dot/manken/openbd
```

公開APIは次の形とする。

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)

func WithEndpoint(endpoint string) Option

func (client *Client) LookupBooksByISBN(
	ctx context.Context,
	isbns []string,
) (openbd.ISBNLookupResult, error)

func (client *Client) LookupBooksByISBNWithRawResponse(
	ctx context.Context,
	isbns []string,
) (openbd.ISBNLookupResult, []byte, error)
```

通常利用する型とエラーの定義元は `model` パッケージである。`openbd` は必要な型と定数を
エイリアスとして公開するため、利用側は `openbd` だけをインポートして、ISBN参照、結果の参照、
分類済みエラーの判定を行える。

- `Option` は外部から独自実装できない不透明な設定型とする
- `Client` は接続設定だけを保持し、リクエストや結果を保持しない
- リクエストごとに新しい `http.Request` を作成する
- 呼び出し側の `http.Client` と `Transport` を変更しない
- `Client` は複数のgoroutineから安全に利用できるものとする
- ログ出力、goroutineの開始、自動リトライ、キャッシュを行わない
- `httpClient` が `nil` の場合は、タイムアウト60秒のクライアントを使用する

既定エンドポイントは次のとおり。

```text
https://api.openbd.jp/v1/get
```

`WithEndpoint` は絶対URLを受け付ける。スキームは `http` または `https`、
ホストは必須とし、ユーザー情報、クエリ、フラグメントを受け付けない。

## 4. ISBN入力と結果対応

ISBN参照は1件以上1,000件以下の入力を受け付ける。空入力、1,001件以上、
不正なISBNが1件でもある場合は外部通信せず `invalid_argument` とする。

各入力はISBNの検証・正規化処理を通し、問い合わせには対応するISBN-13を使用する。
ISBN-10と対応するISBN-13、同じISBNの重複入力は問い合わせ時に1件へまとめる。
問い合わせ順は、入力で正規化後のISBN-13が最初に現れた順序とする。

結果の `Items` は入力と同じ件数、同じ順序で返す。`RequestedISBN` は入力文字列を
変更せず保持する。該当なしを表す `null` は、対応する `Books` が `nil` ではない空スライスに
なる。同じ書誌を重複指定した場合は、問い合わせ結果を元の各入力位置へ展開する。

成功応答は問い合わせISBNと同じ件数、同じ順序のJSON配列でなければならない。
非 `null` 要素では、`onix.RecordReference`、ISBN-13を表す
`onix.ProductIdentifier.IDValue`、`summary.isbn` のうち存在する値を検証する。
少なくとも1つが問い合わせISBNと一致し、存在するISBN値が互いに矛盾しない場合だけ
有効とする。件数不一致、ISBN不一致、ISBNを確認できない要素は
`invalid_response` とする。

## 5. 項目対応

openBD応答は非公開型へ変換してから `Book` を組み立てる。未知のJSON項目は無視し、
欠落した任意項目は有効な応答として扱う。

| openBDの取得元                                                 | `openbd.Book`             |
| -------------------------------------------------------------- | ------------------------- |
| 検証済みISBN-13                                                | `Sources[0].ID`、`ISBN13` |
| ONIXの商品階層 `TitleText.content`、`summary.title`            | `Title`                   |
| `Title` に採用した同一ONIX要素の `TitleText.collationkey`      | `TitleReading`            |
| ONIXのSubtitle                                                 | `Subtitle`                |
| ONIXのContributor                                              | `Authors`、`Contributors` |
| ONIXのImprint、Publisher、`summary.publisher`                  | `Publishers`              |
| Collection階層の `TitleText.content`、`summary.series`         | `PublicationSeries[]`     |
| 同じCollection `TitleText` の `collationkey`                   | `Book` へ変換しない       |
| ONIXのPublishingDate、`hanmoto.dateshuppan`、`summary.pubdate` | `PublishedDate`           |
| `summary.cover`                                                | `CoverURL`                |
| 固定値                                                         | `Sources[0].Source`       |

`PriceType`が`01`または`02`、`CurrencyCode`が`JPY`、`PriceAmount`が非負のASCII整数である価格だけを変換する。`01`は税別、`02`は税込として`TaxIncluded`へ設定する。対応できる候補が同一の価格・税込情報だけなら重複を除いて採用し、異なる候補が複数ある場合は`ListPrice`を設定しない。`Source`は`openbd`、`ObservedAt`は空とし、`CurrentPrice`には設定しない。

`Sources[0].ID` は検証済みISBN-13とする。openBDは応答に書籍ごとの参照ページURLを
返さないため、`Sources[0].URL` は空にする。

### 5.1 採用順序

タイトル、タイトル読み、サブタイトル、寄与者、出版社、出版日は、
意味を確認できるONIX項目を優先する。ONIX項目がない場合だけ、対応する
`summary` または `hanmoto` の値を使用する。

- タイトルは `TitleType=01` の商品階層 `TitleElementLevel=01` を優先する
- タイトル読みは、採用したタイトルと同じ `TitleElement` の `TitleText.collationkey` を使う
- 発行元を表すONIXのImprint、発売元を表すPublisherを、この順で出版社候補にする
- ONIXの出版社候補がない場合だけ `summary.publisher` を使う
- 出版日は `PublishingDateRole=01`、`hanmoto.dateshuppan`、`summary.pubdate` の順で最初の値を使う
- 表紙画像は `summary.cover` だけを使用する

取得元の空文字列を候補から除き、完全に同じ値だけを重複除去する。文字列の空白、
句読点、人名表記、日付精度は変更しない。無加工の取得元本文が必要な場合は、
`WithRawResponse` 系メソッドを使用する。

### 5.2 タイトル、巻数、Collection

商品階層タイトルの `TitleText.content` は、区切り記号や末尾表記を解釈せず、全体を
`Title` に設定する。同じ `TitleElement` の `TitleText.collationkey` がある場合は、
全体を `TitleReading` に設定する。Titleは非破壊で保持したまま、安全なタイトル構文からVolume、Editions、IsFinalVolumeを補う場合がある。`=`を含む並列タイトルでは曖昧な独立末尾数値をVolumeへ使わず、明示的な巻表記だけを扱う。

ONIXのCollectionと `summary.series` は、`PublicationSeries` として名称を保持する。Collectionの
`TitleText.collationkey` は `Book` へ変換しない。同じ名称は完全一致で重複除去し、作品系列またはレーベルへ推測分類しない。Collection階層の
`PartNumber` とCollectionSequenceも作品巻数として使用しない。`summary.volume` はCollection階層の
刊行番号を返して作品巻数と一致しない実例があるため、名前だけを根拠に `Volume` へ変換しない。

### 5.3 寄与者

ONIXの `SequenceNumber` が数値として解釈できる寄与者を番号順にし、番号がない要素は
応答順で後ろに置く。人物名は `PersonName.content` を変更せず使用する。

対応する役割は次のとおり。

| ONIXコード          | `Contributor.Roles` |
| ------------------- | ------------------- |
| `A01`               | `著者`              |
| `A03`、`A14`、`A45` | `脚本`              |
| `A07`、`A12`、`A35` | `作画`              |
| `B01`               | `編集`              |
| `B06`               | `翻訳`              |

ONIXコードは上表の日本語の役割名へ変換して `Contributor.Roles` に設定する。
`PersonName` と同じ要素の `collationkey` がある場合は、そのまま同じ
`Contributor.Reading` に設定する。役割が空の場合は人物名を `Authors` に含め、
役割が空の `Contributor` も設定する。役割が1件以上ある場合は、対応できた役割だけを
`Contributors` に設定する。未知の役割しかない人物も、役割を空にして保持する。
同じ人物名が複数のONIX要素に現れる場合も、要素を統合せず応答順のまま返す。
`著者`、`脚本`、`作画` のいずれかの役割を持つ人物だけを
`Authors` に含める。未知の役割から既知の役割を推測しない。

ONIXの寄与者がない場合、空でない `summary.author` を分割せず1件の
`Authors` として使用する。

### 5.4 変換しない項目とタイトル由来の補完

ISBN、タイトル、タイトル読み、サブタイトル、寄与者、出版社、出版日、表紙画像を
`Book` へ設定する。

並列タイトル、作品系列、レーベル、説明、主題、言語、ページ数、判型、物理寸法、
CollectionSequence、商品階層 `PartNumber`、SupportingResourceは `Book` へ設定しない。
巻数や版表示もopenBD固有の項目からは直接設定せず、5.2のとおり、Titleを非破壊で保持したまま安全なタイトル構文から `Volume`、`Editions`、`IsFinalVolume` を補う場合がある。直接設定しない取得元固有の値はRaw responseから確認できる。

## 6. Raw response

`LookupBooksByISBNWithRawResponse` は、変換済み結果に加えてopenBDから受信した
成功レスポンス本文を変更せず `[]byte` で返す。JSON解析または結果変換に失敗した
場合も、上限内で読み込み済みの2xx本文を返す。

本文読込の失敗、本文上限の超過、2xx以外のHTTP応答、通信失敗では本文を返さない。
Raw responseを持たないメソッドも同じ通信と変換処理を使用する。

## 7. HTTP

ISBN-13をカンマ区切りにし、GETの `isbn` クエリパラメーターへ設定する。

```text
Accept: application/json
```

- 成功レスポンス本文の上限は64 MiBとする
- エラーレスポンス本文の読込上限は64 KiBとする
- 外部サービスのエラー本文を公開エラーメッセージへ含めない
- エラー本文を `openbd.Error` に保持しない
- 自動リトライとアクセス間隔制御は行わない

## 8. エラー

`openbd.Error.Operation` には次の値を使用する。

```text
openbd.NewClient
openbd.LookupBooksByISBN
```

| 状態                                         | `ErrorKind`        |
| -------------------------------------------- | ------------------ |
| ISBN、入力件数、Client設定の不正             | `invalid_argument` |
| HTTP 408、429、500から599                    | `unavailable`      |
| その他の成功以外のHTTPステータス             | `upstream`         |
| 通信失敗、タイムアウト                       | `unavailable`      |
| 本文上限超過、JSON、配列件数、ISBN対応の不正 | `invalid_response` |

`Retry-After` は秒数形式とHTTP-date形式を解析する。`context.Canceled` と
`context.DeadlineExceeded` は、ラップ後も `errors.Is` で判定できるようにする。

## 9. 利用条件とサービス変更

openBDのデータは本の販促・紹介目的に限って利用できる。利用者はopenBDの
利用規約へ同意し、削除要請やサービス変更へ対応する必要がある。

利用規約はデータの任意改変を禁止している。本パッケージは複数providerで同じ意味として扱える値だけを
`Book` へ設定し、無加工の成功レスポンス本文は `WithRawResponse` 系メソッドで返す。
具体的な利用方法が規約へ適合することは保証しないため、利用者が用途と表示方法を
確認する。

代替APIは従来のURLと応答形式を維持しているが、提供期間、収録範囲、項目、書影は
将来変更される可能性がある。
