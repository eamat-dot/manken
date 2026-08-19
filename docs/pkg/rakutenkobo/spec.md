# 楽天Koboパッケージ仕様

## 1. 概要

`rakutenkobo` パッケージは、楽天Kobo電子書籍検索APIを使って電子書籍を検索し、
取得結果を `model.Book` へ変換する。

import path:

```text
github.com/eamat-dot/manken/rakutenkobo
```

楽天Koboは電子書籍の商品検索APIであり、ISBN参照は提供しない。Kobo固有の商品番号、
取得時点価格、商品URL、アフィリエイトURLなど、取得元が明示する情報だけを安全に共通化する。
単話、分冊、合本、無料版を商品タイトルから推測・分類しない。titleは非破壊で保持し、通常巻と安全に判断できる明確な巻表示、確認済み版表示、巻表示へ隣接する完結表示だけをVolume、Editions、IsFinalVolumeへ補う場合がある。分冊、単話、合本、無料試読、セット、Vol.NはVolumeへ推測しない。

## 2. Client

### 2.1 NewClient

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)
```

`httpClient` が `nil` の場合は、60秒のTimeoutを持つ `http.Client` を使用する。
呼び出し側から渡された `http.Client` とTransportは変更しない。リダイレクトは自動追従しない。

Client作成にはApplication IDとAccess Keyが必要である。Affiliate IDは任意である。
ライブラリ本体は環境変数を読み込まず、Optionへ渡された値だけを使用する。

### 2.2 Options

```go
WithApplicationID(applicationID string)
WithAccessKey(accessKey string)
WithAffiliateID(affiliateID string)
WithComicGenre(genre ComicGenre)
WithEndpoint(endpoint string)
```

| Option              | 必須 | 内容                                         |
| ------------------- | ---- | -------------------------------------------- |
| `WithApplicationID` | 必須 | 楽天ウェブサービスのApplication ID           |
| `WithAccessKey`     | 必須 | 楽天ウェブサービスのAccess Key               |
| `WithAffiliateID`   | 任意 | アフィリエイトURL生成に使用するAffiliate ID  |
| `WithComicGenre`    | 任意 | 検索対象の漫画区分。省略時は一般コミック     |
| `WithEndpoint`      | 任意 | 通常は使用しない。HTTPSの絶対URLを受け付ける |

空文字列または空白だけの必須認証値、空のAffiliate IDを明示設定した場合、未対応の
`ComicGenre`、`nil` Optionは `invalid_argument` となる。Access Keyに制御文字を含められない。

通常endpointはHTTPSを使用する。テスト用には`localhost`、loopback IPv4、`::1`へのHTTPも
受け付ける。`WithEndpoint` はuser information、query、fragmentを含むURLを受け付けない。

## 3. 漫画区分

`ComicGenre` は検索1回で楽天Koboへ指定する漫画区分を表す。

| 定数                | 値        | `koboGenreId` |
| ------------------- | --------- | ------------- |
| `ComicGenreGeneral` | `general` | `101904`      |
| `ComicGenreBL`      | `bl`      | `101940002`   |
| `ComicGenreTL`      | `tl`      | `101940011`   |

Clientの既定値は `ComicGenreGeneral` である。1回の `SearchBooks` で複数区分を横断しない。
複数区分を内部で別々のAPIリクエストへ展開すると、Limit、page、取得元の返却順を単一の検索結果と
Cursorで表せなくなるためである。これらのジャンルIDは2026年8月10日にKoboジャンル検索APIで再確認している。

## 4. SearchBooks

```go
func (client *Client) SearchBooks(
    ctx context.Context,
    request SearchRequest,
) (SearchBooksResult, error)
```

`SearchBooksWithRawResponse` は同じ検索結果に加えて、受信した2xx成功レスポンス本文を返す。

```go
func (client *Client) SearchBooksWithRawResponse(
    ctx context.Context,
    request SearchRequest,
) (SearchBooksResult, []byte, error)
```

### 4.1 対応する共通検索条件

`DateFrom` と `DateTo` は楽天Koboが範囲検索条件を提供しないため、通信前に `invalid_argument` を返す。取得後の絞り込みは行わない。

| `SearchRequest` | 楽天Kobo        | 動作                   |
| -------------------- | --------------- | ---------------------- |
| `Title`              | `title`         | 対応                   |
| `Author`             | `author`        | 対応                   |
| `Publisher`          | `publisherName` | 対応                   |
| `Query`           | `keyword`       | 対応                   |
| `Exclude`       | `NGKeyword`     | 条件付きで対応         |
| `Limit`              | `hits`          | 対応                   |
| `Cursor`             | `page`          | 不透明Cursor経由で対応 |

`Title`、`Author`、`Publisher`、`Query` の少なくとも1つが必要である。各文字列の前後の
空白を除いて送信する。

楽天Kobo APIは `NGKeyword` を指定する場合に `keyword` も要求する。この制約に対して
`rakutenkobo` は次の規則を適用する。

- `Exclude` がある場合は、`Query`、`Title`、`Author`、`Publisher` の順で最初の非空値を `keyword` として使用する
- 専用条件は常に対応する取得元パラメーターへ送る。補完する `keyword` は利用者が別途指定した検索語ではなく、`NGKeyword` の取得元要件を満たすため既存の正条件と同じ値を送る
- `Exclude` だけでは検索できない

この補完は2026年8月11日に取得元へ送信して確認した挙動に基づく回避策であり、`NGKeyword` が検索対象とする項目の範囲は保証しない。固定の除外語は追加しない。利用者が指定した `Exclude` だけを `NGKeyword` へ送る。

### 4.2 固定パラメータ

検索では次を設定する。

```text
format=json
formatVersion=2
sort=+releaseDate
language=JA
field=0
orFlag=0
koboGenreId=<Clientの漫画区分>
```

`sort=+releaseDate` は発売日の古い順である。古い商品から候補を確認しやすくし、楽天BooksやNDLサーチと
固定既定の考え方を揃えるため明示指定する。版違い、分冊、合本等が混在し得るため巻数順は保証しない。
`language=JA` で日本語商品に限定し、`field=0` は楽天Koboの広い検索対象範囲を使用する。
複数キーワードはAND条件を維持する。
楽天Koboが返した順序を維持し、ライブラリ内で独自に並べ替えない。

## 5. Limitとページング

`Limit == 0` は20件として扱う。有効範囲は1〜30件である。

楽天Koboの1始まり `page` は公開せず、`SearchBooksResult.NextCursor` に不透明なCursorとして
保持する。Cursorは次の情報へ関連付けられる。

- Cursor形式のバージョン
- 次のページ番号
- 実効Limit
- Title
- Author
- Publisher
- Query
- Exclude
- Clientの漫画区分

Cursorを異なる検索条件、Limit、漫画区分で再利用した場合は `invalid_argument` となる。
Cursor内へApplication ID、Access Key、Affiliate IDは保存しない。

次のいずれかでは `NextCursor` を返さない。

- 結果が0件
- `pageCount == 0`
- 現在ページが `pageCount` 以上
- 現在ページが100

楽天Koboの上限に合わせ、100ページを超えるCursorを生成しない。2026年8月10日の実API確認では、
0件時はHTTP 200で `count=0`、`page=0`、`pageCount=0`、`hits=0`、空の`Items`が返った。

## 6. 共通モデルへの変換

### 6.1 変換する項目

| 楽天Kobo        | 共通モデル                | 規則                                                                   |
| --------------- | ------------------------- | ---------------------------------------------------------------------- |
| `itemNumber`    | `BookSource.ID`           | Koboの商品IDとして保持。ISBNとして扱わない                             |
| `itemUrl`       | `BookSource.URL`          | 有効なHTTP(S) URLだけを使用                                            |
| `affiliateUrl`  | `BookSource.AffiliateURL` | 有効なHTTP(S) URLだけを使用。通常URLを置き換えない                     |
| `title`         | `Title`                   | そのまま保持                                                           |
| `titleKana`     | `TitleReading`            | そのまま保持                                                           |
| `subTitle`      | `Subtitle`                | そのまま保持                                                           |
| `seriesName`    | `PublicationSeries[]`     | 1要素として保持。読みは共通Bookへ変換しない                            |
| `author`        | `Authors`                 | `/`で分割し、前後空白を除いて順序どおり保持。空要素は除外              |
| `author`        | `Contributors`            | Authorsと同じ人物を役割なしで保持                                      |
| `authorKana`    | `Contributors[].Reading`  | Kobo固有の検証済み形式だけを著者順に設定。未知形式はRaw responseに残す |
| `publisherName` | `Publishers`              | 1要素として保持                                                        |
| `salesDate`     | `ReleaseDate`             | 発売日として原文の精度を維持                                           |
| `itemCaption`   | `Description`             | そのまま保持                                                           |
| `koboGenreId`   | `Subjects`                | `/`で分割し、Scheme=`rakuten_kobo` とする                              |
| `itemPrice`     | `CurrentPrice`            | JPY / 税込。取得時刻を `ObservedAt` に保持                             |
| API種別         | `Medium`                  | `digital`                                                              |
| 3種の画像URL    | `CoverURL`                | 利用可能な最大サイズのURLを1件保持                                     |

`seriesName` は `PublicationSeries` に保持するが、作品シリーズであることは保証しない。取得元の読みは共通Bookへ変換せず、同じ値を `BookSeries` や `Publishers` へ重複設定したり推測分類したりしない。

`itemNumber` はKobo固有の商品番号であり、`ISBN10`、`ISBN13`、`JAN` へ追加しない。
`LookupBooksByISBN` は提供しない。

`author` は `/` 区切りで人物単位へ分け、各要素の前後空白を除き、空要素を除外する。
この規則は `Authors` と `Contributors[].Name` の両方に適用する。

`authorKana` は、次のいずれかを満たす場合だけ `Contributors[].Reading` に設定する。

- 著者が1名であり、前後空白を除いた `authorKana` が空でなく、`/` と `,` を含まない場合は、その値を設定する
- 著者が複数名であり、空要素を除外する前の `author` の `/` 区間がすべて非空である場合は、`authorKana` の `/` 区間数、各区間の非空性、前後空白を除いた各区間の完全一致を確認する。共通区間を `,` で分けた要素数が著者数と一致し、各要素が非空の場合だけ、著者順に設定する

条件を満たさない `authorKana`、`A/B` に対する `エー/ビー` のような未確認形式、空要素を除外すると件数だけが一致する形式では、すべてのContributorのReadingを空のままにする。部分的な設定、役割、読みの推測は行わない。元の値はRaw responseから確認できる。

`salesDate` は楽天Koboが「発売日」として返す値であるため `ReleaseDate` に保持する。
電子書籍であることだけを根拠に `digital_released` へ読み替えない。日付文字列には年月だけや
「上旬」等を含み得るため、厳密な日付型へ変換しない。

### 6.2 販売情報

`itemPrice` が存在する場合は、取得時点の販売価格として次の `Price` を1件設定する。

- `Type`: `current`
- `Amount`: `itemPrice`。0円も保持する
- `Currency`: `JPY`
- `TaxIncluded`: `true`
- `Source`: `rakutenkobo`
- `ObservedAt`: そのHTTP成功レスポンスを取得した時刻

Affiliate IDを設定した場合に楽天Koboが返す `affiliateUrl` は、通常商品URLとは別に
`BookSource.AffiliateURL` へ保持する。

レビュー件数、レビュー平均、`salesType` 等は共通モデルへ追加せず、Raw responseから参照する。

## 7. Raw response

`SearchBooksWithRawResponse` のRaw responseは、1回の2xx成功HTTPレスポンス本文である。
JSON解析または共通モデル変換に失敗しても、本文を上限内で読み込めていればRaw responseを返す。

成功本文の上限は16 MiBである。上限を超えた場合は `invalid_response` とする。
Application ID、Access Key、Affiliate ID等をライブラリ側からRaw responseへ追加しない。

## 8. HTTP・エラー

共通の `model.Error` / `model.ErrorKind` を使用する。

| 状態                                                 | `ErrorKind`        |
| ---------------------------------------------------- | ------------------ |
| Client設定、検索条件、Limit、Cursorの不正            | `invalid_argument` |
| HTTP 408、429、500〜599                              | `unavailable`      |
| その他の2xx以外                                      | `upstream`         |
| 通信失敗・タイムアウト                               | `unavailable`      |
| JSON解析失敗、レスポンス本文上限超過、ページング矛盾 | `invalid_response` |

`Retry-After` が秒数またはHTTP-dateとして解釈できる場合は `Error.RetryAfter` へ保持する。
エラー本文は64 KiBまで読み捨て、取得元本文を公開エラーへ含めない。

Access KeyはHTTP `accessKey` ヘッダーへ設定する。Application IDと任意のAffiliate IDはqueryへ
設定するが、完全なリクエストURLを公開エラーへ含めない。自動リダイレクトは行わず、認証情報を
意図しない転送先へ送らない。

`context.Context` が `nil`、または初期化されていないClientで検索した場合は通信前に
`invalid_argument` とする。

## 9. ライブラリ内部で行わない処理

`rakutenkobo` は次を行わない。

- 自動リトライ
- キャッシュ
- 内部レートリミッター
- ログ出力
- バックグラウンドgoroutine
- 商品ごとのKoboジャンル検索API呼び出し
- 固定除外語の暗黙適用
- 巻数、単話、分冊、合本、無料版等のタイトルからの推測
- ISBN参照

リクエスト頻度、再試行、保存方針は呼び出し側で制御する。利用条件は
[楽天Koboガイド](guide.md)を参照する。
