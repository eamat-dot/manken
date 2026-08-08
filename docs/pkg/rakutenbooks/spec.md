# 楽天Booksパッケージ仕様

## 1. 概要

`rakutenbooks` パッケージは、楽天ブックス書籍検索APIを使って紙書籍を検索・ISBN参照し、
取得結果を `api.Book` へ変換する。

import path:

```text
github.com/eamat-dot/manken/rakutenbooks
```

楽天Books固有の販売情報をすべて共通モデルへ変換することは目的としない。価格、在庫、
レビュー、試し読みURL、アフィリエイトURLなどはRaw responseから参照できる。

## 2. Client

### 2.1 NewClient

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)
```

`httpClient` が `nil` の場合は、60秒のTimeoutを持つ `http.Client` を使用する。
呼び出し側から渡された `http.Client` とTransportは変更しない。

Client作成にはApplication IDとAccess Keyが必要である。Affiliate IDは任意である。

### 2.2 Options

```go
WithApplicationID(applicationID string)
WithAccessKey(accessKey string)
WithAffiliateID(affiliateID string)
WithComicGenre(genre ComicGenre)
WithEndpoint(endpoint string)
```

| Option | 必須 | 内容 |
| --- | --- | --- |
| `WithApplicationID` | 必須 | 楽天ウェブサービスのApplication ID |
| `WithAccessKey` | 必須 | 楽天ウェブサービスのAccess Key |
| `WithAffiliateID` | 任意 | アフィリエイトURL生成に使用するAffiliate ID |
| `WithComicGenre` | 任意 | 検索対象の漫画区分。省略時は一般コミック |
| `WithEndpoint` | 任意 | 通常は使用しない。HTTP(S)の絶対URLだけを受け付ける |

空文字列または空白だけの認証値を明示設定した場合は `invalid_argument` となる。
Affiliate IDを設定しなくても検索・ISBN参照を利用できる。

`WithEndpoint` はuser information、query、fragmentを含むURLを受け付けない。

## 3. 漫画区分

`ComicGenre` は検索1回で楽天Booksへ指定する漫画区分を表す。

| 定数 | 値 | `booksGenreId` |
| --- | --- | --- |
| `ComicGenreGeneral` | `general` | `001001` |
| `ComicGenreBL` | `bl` | `001021002` |
| `ComicGenreTL` | `tl` | `001029002` |

Clientの既定値は `ComicGenreGeneral` である。
1回の `SearchBooks` で複数区分を横断しない。BLまたはTLを検索する場合は、対応する
`WithComicGenre` を指定したClientを使用する。

ISBN参照ではClientの漫画区分を使用しない。

## 4. SearchBooks

```go
func (client *Client) SearchBooks(
    ctx context.Context,
    request SearchBooksRequest,
) (SearchBooksResult, error)
```

`SearchBooksWithRawResponse` は同じ検索結果に加えて、受信した2xx成功レスポンス本文を返す。

```go
func (client *Client) SearchBooksWithRawResponse(
    ctx context.Context,
    request SearchBooksRequest,
) (SearchBooksResult, []byte, error)
```

### 4.1 対応する共通検索条件

| `SearchBooksRequest` | 楽天Books | 動作 |
| --- | --- | --- |
| `Title` | `title` | 対応 |
| `Author` | `author` | 対応 |
| `FreeText` | 直接対応なし | 非空なら `invalid_argument` |
| `ExcludedText` | 直接対応なし | 非空なら `invalid_argument` |
| `Limit` | `hits` | 対応 |
| `Cursor` | `page` | 不透明Cursor経由で対応 |

`Title` または `Author` の少なくとも一方が必要である。前後の空白は除いて送信する。
両方を指定した場合は両方を楽天Booksへ渡す。

`FreeText` と `ExcludedText` は取得後フィルターで擬似対応しない。楽天Books上の件数と
ページングの意味が変わるため、非空入力を通信前に拒否する。

### 4.2 固定パラメータ

検索では次を設定する。

```text
format=json
formatVersion=2
sort=standard
booksGenreId=<Clientの漫画区分>
```

楽天Booksが返した順序を維持し、ライブラリ内で独自に並べ替えない。

## 5. Limitとページング

`Limit == 0` は20件として扱う。有効範囲は1〜30件である。

楽天Booksの1始まり `page` は公開せず、`SearchBooksResult.NextCursor` に不透明なCursorとして
保持する。Cursorは次の情報へ関連付けられる。

- Cursor形式のバージョン
- 次のページ番号
- 実効Limit
- Title
- Author
- Clientの漫画区分

Cursorを異なる検索条件、Limit、漫画区分で再利用した場合は `invalid_argument` となる。
Cursor内へApplication ID、Access Key、Affiliate IDは保存しない。

次のいずれかでは `NextCursor` を返さない。

- 結果が0件
- `pageCount == 0`
- 現在ページが `pageCount` 以上
- 現在ページが100

楽天Booksの上限に合わせ、100ページを超える取得は行わない。

## 6. ISBN参照

```go
func (client *Client) LookupBooksByISBN(
    ctx context.Context,
    isbns []string,
) (ISBNLookupResult, error)
```

```go
func (client *Client) LookupBooksByISBNWithRawResponse(
    ctx context.Context,
    isbns []string,
) (ISBNLookupResult, []byte, error)
```

入力は正確に1件だけ受け付ける。ISBN-10とISBN-13を検証し、楽天Booksへの問い合わせでは
ISBN-13へ統一する。

1回のISBN参照につきHTTPリクエストは1回で、次を使用する。

```text
isbn=<ISBN-13>
hits=30
page=1
format=json
formatVersion=2
```

`booksGenreId` は指定しない。返却商品のISBNを再検証し、要求ISBNと一致する商品だけを
`Books` に含める。

`ISBNLookupResult.Items` は1要素で、`RequestedISBN` は利用者が入力した文字列を保持する。
該当商品がない場合もエラーにせず、`Books` はnon-nilの空スライスになる。

## 7. 共通モデルへの変換

### 7.1 変換する項目

| 楽天Books | 共通モデル | 規則 |
| --- | --- | --- |
| `itemUrl` | `BookSource.URL` | 有効なHTTP(S) URLだけを使用 |
| `title` | `Normalized.Title` | そのまま保持 |
| `titleKana` | `Normalized.TitleReading` | そのまま保持 |
| `subTitle` | `Normalized.Subtitle` | そのまま保持 |
| `seriesName` | `Normalized.Series[].Name` | 1要素として保持 |
| `author` | `Normalized.Authors` | 取得元の文字列を1要素として保持 |
| `author` / `authorKana` | `Normalized.Contributors` | 1要素のName / Reading。役割は設定しない |
| `publisherName` | `Normalized.Publishers` | 1要素として保持 |
| `isbn` | `Normalized.Identifiers` | 検証できるISBN-10 / ISBN-13だけを保持 |
| `salesDate` | `Normalized.Dates` | `released` として原文の精度を維持 |
| `itemCaption` | `Normalized.Description` | そのまま保持 |
| `booksGenreId` | `Normalized.Subjects` | `/` で分割し、Scheme=`rakuten_books` とする |
| `size` | `Normalized.PhysicalSize.Name` | 判型名として保持 |
| API種別 | `Normalized.Medium` | `print` |
| 3種の画像URL | `Normalized.Images` | small / medium / largeの順に保持 |

`author` に `/` が含まれても人物へ分割しない。楽天Booksの公式仕様から一般的な人物区切り規則を
保証できないためである。Contributorの役割も推測しない。

`BookSource.ID` は設定しない。楽天Books固有の安定した商品IDをレスポンスの専用項目から
確認できないため、商品URLのpath等から独自IDを生成しない。

### 7.2 初期共通化しない項目

次はRaw responseへ残し、初期実装では共通モデルへ変換しない。

- `affiliateUrl`
- `itemPrice`
- `availability`
- `postageFlag`
- `limitedFlag`
- `reviewCount`
- `reviewAverage`
- `chirayomiUrl`
- `contents` / `contentsKana`

特に `affiliateUrl` を `BookSource.URL` へ設定しない。`BookSource.URL` は通常の `itemUrl` を使う。

## 8. Raw response

Raw response用メソッドは、1回の2xx成功HTTPレスポンス本文を変更せず `[]byte` で返す。

- 正常に変換できた場合もRawを返す
- JSON解析または共通モデル変換に失敗した場合も、読み込み済みの2xx本文を返す
- 通信失敗、2xx以外、本文読み込み失敗、本文上限超過ではRawを返さない
- 成功本文の上限は16 MiB
- エラー本文は最大64 KiBまで読み捨て、本文内容を公開エラーへ含めない

Affiliate IDを設定した場合、楽天Booksが返す `affiliateUrl` はRaw responseに含まれ得る。
ライブラリはRaw responseを永続保存しない。

## 9. 認証情報とHTTP

リクエストでは認証情報を次のように送る。

| 情報 | 送信方法 |
| --- | --- |
| Application ID | query `applicationId` |
| Access Key | HTTP header `accessKey` |
| Affiliate ID | 設定時だけquery `affiliateId` |

公開エラーには完全なリクエストURLを含めない。通信エラーが `url.Error` を含む場合は、
認証情報を含み得るURLを除き、原因エラーだけを公開エラーへ保持する。

自動リトライ、内部レート制御、キャッシュ、ログ出力は行わない。

## 10. エラー

楽天Booksパッケージは共通の `api.Error` / `ErrorKind` を使用する。

| 状態 | ErrorKind |
| --- | --- |
| Client設定不正、検索条件不正、Limit不正、Cursor不正、ISBN不正 | `invalid_argument` |
| HTTP 408 / 429 / 500〜599、通信失敗 | `unavailable` |
| その他の2xx以外 | `upstream` |
| 成功本文のJSON不正、本文上限超過、ページング矛盾 | `invalid_response` |

HTTPエラーでは `StatusCode` を保持する。`Retry-After` が秒数または有効なHTTP日時なら
`RetryAfter` へ変換する。

contextのcancel / deadlineは原因エラーを保持し、`errors.Is` で判定できる。

## 11. 並行利用とリクエスト制限

Clientは検索ごとの可変状態を保持しないため、共有可能である。ただし楽天ウェブサービスの
リクエスト制限を満たすための待機・直列化は行わない。

利用者はApplication ID単位のリクエスト頻度を管理する必要がある。利用条件と表示・保存上の
注意は [楽天Booksガイド](guide.md) を参照する。
