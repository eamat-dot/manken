# Google Booksパッケージ仕様

## 1. 目的

`googlebooks` パッケージは、Google Books Volumes APIから書誌候補を検索し、
`model` パッケージの共通書籍モデルへ変換する。Google Booksは漫画専用の取得元ではないため、
`printType=books` を指定しても小説などが結果に含まれる。

共通モデルは [manken API仕様](../../spec.md)、パッケージ構成は
[アーキテクチャ](../../../ARCHITECTURE.md)を参照する。

## 2. Clientと認証

```go
client, err := googlebooks.NewClient(nil, googlebooks.WithAPIKey(apiKey))
```

import pathは `github.com/eamat-dot/manken/googlebooks` である。`NewClient` はAPIキーを必須とし、
空白だけのキー、nil Option、不正なendpointは通信前に `invalid_argument` を返す。
通常の認証付きendpointはHTTPSを使用する。`WithEndpoint` は絶対URLを設定し、HTTPSを受け付ける。
HTTPを受け付けるのはテスト用の`localhost`、`127.0.0.0/8`、`::1`だけであり、名前解決によって
外部hostをloopbackとして扱わない。ユーザー情報、query、fragmentは使えない。

`httpClient` がnilの場合は60秒タイムアウトのHTTPクライアントを使う。Clientはリクエスト状態を
保持せず、複数goroutineから安全に使用できる。APIキーは `key` queryへだけ設定し、エラー、
カーソル、Raw responseへ含めない。

## 3. 検索

```go
result, err := client.SearchBooks(ctx, googlebooks.SearchRequest{
    Title: "動物のお医者さん",
    Limit: 20,
})
```

`Title`、`Author`、`Publisher`、`Query` の少なくとも1つを指定する。各値はUnicode空白で分割し、
利用者入力を検索演算子として解釈しない引用済みの語へ変換する。

`DateFrom` と `DateTo` はGoogle Booksが範囲検索条件を提供しないため、通信前に `invalid_argument` を返す。取得後の絞り込みは行わない。

- `Title` は各語を `intitle:` 条件にする
- `Author` は各語を `inauthor:` 条件にする
- `Publisher` は各語を `inpublisher:` 条件にする
- `Query` は通常検索語にする
- `Exclude` は各語をGoogle Booksの除外構文 `-term` 相当へ変換し、正条件の後ろへ追加する
- Google Booksの除外はfull-text query全体に作用し、タイトルだけを対象にしない。説明文など検索対象の別情報に除外語が含まれる結果も除外される場合がある
- `Exclude` だけの検索は許可せず、`Title`、`Author`、`Publisher`、`Query` の少なくとも1つを必須とする

リクエストには `printType=books`、`orderBy=relevance`、`projection=full` を設定する。
`langRestrict` と電子書籍filterは設定しない。Google Booksの応答順を変更しない。

`Limit` の0は20、1から40は指定値、範囲外は `invalid_argument` となる。`NextCursor` は
次の `startIndex`、実効Limit、正規化済み検索式のハッシュを保持する不透明な値である。
別の条件またはLimit、壊れた値、未知のバージョンで使うと `invalid_argument` となる。
続きがない場合と空結果では空文字列となる。

`SearchBooksWithRawResponse` は、同じ検索結果と1回の2xx本文を無加工で返す。JSON解析または
変換に失敗した場合も読み込み済み本文を返す。通信失敗、非2xx、本文読込失敗、16 MiB超過では
Raw responseを返さない。

## 4. ISBN参照

```go
result, err := client.LookupBooksByISBN(ctx, []string{"4088466365"})
result, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{"4088466365"})
```

Google BooksのISBN参照は1回の呼び出しにつき1件だけ受け付ける。ISBN-10またはISBN-13を
`internal/isbn` で検証し、問い合わせ用のISBN-13へ正規化する。0件または2件以上は外部通信前に
`invalid_argument` とする。

問い合わせは `isbn:<ISBN-13>`、`printType=books`、`projection=full`、`maxResults=40`、
`startIndex=0` を使う1回のHTTPリクエストとする。応答Volumeの検証済みISBNが要求ISBNと一致する
ものだけを返し、40件を超える追加ページは取得しない。`RequestedISBN` は元の入力文字列を保持し、
該当なしの `Books` は非nilの空スライスとする。

`LookupBooksByISBNWithRawResponse` は、その1回の2xx成功レスポンス本文を変更せず返す。
JSON解析または共通モデル変換に失敗した場合も、読み込み済み本文を返す。通信失敗、2xx以外、
本文読込失敗、本文上限超過ではRaw responseを返さない。

## 5. 結果変換

| Google Books                              | 変換先                                                          |
| ----------------------------------------- | --------------------------------------------------------------- |
| `id`                                      | `Sources[0].ID`                                                 |
| `canonicalVolumeLink`、`infoLink`         | 有効なHTTP(S) URLを優先して `Sources[0].URL`                    |
| `title`、`subtitle`                       | `Title`、`Subtitle`                                             |
| `authors[]`                               | `Authors` と役割なしの `Contributors`                           |
| `publisher`、`publishedDate`              | `Publishers`、`PublishedDate`                                   |
| 検証済み `ISBN_10`、`ISBN_13`             | `ISBN10`、`ISBN13`                                              |
| `description`、`language`、`categories[]` | `Description`、`Languages`、`Subjects`                          |
| 正の `pageCount`                          | `PageCount`。0以下は不明値として未設定にする                    |
| `imageLinks`                              | 利用可能な最大候補を `CoverURL` に設定                          |
| `saleInfo.listPrice`                      | 条件を満たす場合だけ `ListPrice`                                |
| `saleInfo.retailPrice`                    | 条件を満たす場合だけ `CurrentPrice`                             |
| `saleInfo.isEbook`                        | `true` の場合だけ `Medium: digital`。`false` または欠落は未設定 |

著者と編集者の役割は区別できないため推測しない。`volumeInfo.title` は非破壊でTitleへ保持し、明確な巻表示、確認済み版表示、巻表示へ隣接する完結表示だけをVolume、Editions、IsFinalVolumeへ補う場合がある。subtitle、シリーズ、著者はタイトルから推測しない。
`saleInfo` の価格は、`country` がJP、`currencyCode` がJPYであり、`amount` が存在し、有限で負でなく、
小数部のない `int64` 範囲内の値である場合だけ変換する。文字列の大文字小文字は区別しない。小数を
丸めたり切り捨てたりせず、条件外の価格は設定しない。明示された0円は価格として設定する。

`listPrice` は `Source: googlebooks`、`Currency: JPY` の `ListPrice` として設定する。`retailPrice` は同じ情報に
加え、1回の正常なAPI応答の変換で共通となるUTC RFC3339Nano形式の `ObservedAt` を持つ `CurrentPrice` として
設定する。Google Booksの資料から消費税の扱いは確定できないため、`TaxIncluded` は設定しない。

国別の適用範囲を共通モデルへ誤って広げないため、JP以外の `saleInfo` は価格を設定しない。
`saleInfo.isEbook` は国や販売可否にかかわらず、`true` の場合だけ `PublicationMediumDigital` とする。
`false` または項目欠落は紙書籍とみなさず、`PublicationMediumUnknown` のままとする。ISBN、
`volumeInfo.printType`、`accessInfo` のEPUB/PDF availability、価格、購入URLなどから媒体を推測しない。

`saleInfo.country`、`saleability`、`buyLink`、`onSaleDate`、`accessInfo`、物理サイズ、rating、
`searchInfo`、`contentVersion`は共通モデルへ設定しない。特に`searchInfo.textSnippet`は検索語に依存する本文断片であるため、`volumeInfo.description`が欠落しても`Description`へfallbackしない。Volume IDの欠落は `invalid_response`、
その他の任意項目の欠落は正常である。

## 6. HTTPとエラー

既定endpointは `https://www.googleapis.com/books/v1/volumes` で、GETと
`Accept: application/json` を使う。成功本文は16 MiB、エラー本文は64 KiBまで読み込む。
エラー本文と完全なリクエストURLは公開エラーへ含めない。

| 状態                                               | `ErrorKind`        |
| -------------------------------------------------- | ------------------ |
| 入力、APIキー、Client設定、カーソルの不正          | `invalid_argument` |
| HTTP 408、429、500〜599、通信失敗、タイムアウト    | `unavailable`      |
| その他の非2xx                                      | `upstream`         |
| 成功本文の上限超過、JSON、必須ID、ページングの不正 | `invalid_response` |

`Retry-After` は秒数形式とHTTP-date形式を解釈する。`context.Canceled` と
`context.DeadlineExceeded` は `errors.Is` で判定できる。自動リトライ、ログ、キャッシュ、
内部並行実行は行わない。
