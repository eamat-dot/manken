# 楽天Booksガイド

## 1. 利用前に準備するもの

楽天Booksプロバイダは楽天ウェブサービスの「楽天ブックス書籍検索API」を使用する。
利用には次が必要である。

- Application ID
- Access Key

Affiliate IDは検索に必須ではない。楽天アフィリエイトURLが必要な場合だけ設定する。

楽天ウェブサービス:
https://webservice.rakuten.co.jp/

楽天ブックス書籍検索API:
https://webservice.rakuten.co.jp/documentation/books-book-search

## 2. 認証情報の設定

認証情報をソースコードへ直接記述せず、環境変数や利用側の秘密情報管理から読み込む。

このリポジトリのCLIデモでは次の環境変数を使用する。

```text
RAKUTEN_APP_ID
RAKUTEN_ACCESS_KEY
RAKUTEN_AFFILIATE_ID
```

`RAKUTEN_AFFILIATE_ID` は省略できる。

Clientの最小構成:

```go
client, err := rakutenbooks.NewClient(nil,
    rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
    rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
)
if err != nil {
    log.Fatal(err)
}
```

Affiliate IDを利用する場合だけOptionを追加する。

```go
options := []rakutenbooks.Option{
    rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
    rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
}
if affiliateID := os.Getenv("RAKUTEN_AFFILIATE_ID"); affiliateID != "" {
    options = append(options, rakutenbooks.WithAffiliateID(affiliateID))
}
client, err := rakutenbooks.NewClient(nil, options...)
if err != nil {
    log.Fatal(err)
}
```

Affiliate IDを設定しても検索条件は変わらない。楽天Booksが返す `affiliateUrl` は
`BookSource.AffiliateURL` とRaw responseの両方から利用できる。

## 3. タイトル・著者・出版社で検索する

既定では一般コミックを検索する。

```go
result, err := client.SearchBooks(
    context.Background(),
    rakutenbooks.SearchBooksRequest{Title: "動物のお医者さん"},
)
if err != nil {
    log.Fatal(err)
}
```

`Title`、`Author`、`Publisher` のいずれか1つ以上を指定する。複数を同時に指定すると、
すべての条件を満たす書籍に絞り込む。

楽天BooksはBooks Book Search APIに汎用フリーワード検索と除外キーワード検索を持たないため、
`FreeText` と `ExcludedText` は利用できない。

## 4. 一般・BL・TLコミック

検索対象はClient作成時に1区分を選ぶ。

```go
client, err := rakutenbooks.NewClient(nil,
    rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
    rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
    rakutenbooks.WithComicGenre(rakutenbooks.ComicGenreBL),
)
```

利用できる区分:

- `ComicGenreGeneral`: 一般コミック。既定値
- `ComicGenreBL`: BLコミック
- `ComicGenreTL`: TLコミック

1回の検索で3区分を自動的に横断しない。別区分を検索する場合は、その区分を指定したClientを
作成する。

## 5. ISBN参照

ISBN-10またはISBN-13を1件だけ指定する。

```go
result, err := client.LookupBooksByISBN(
    context.Background(),
    []string{"4088466365"},
)
```

ISBN参照では一般・BL・TLの漫画区分と商品形態を使用しない。入力ISBNに一致する楽天Booksの商品を
参照する。

該当商品がない場合はエラーではなく、1件の入力結果の `books` が空になる。

## 6. 商品形態で絞り込む

`WithBookSize` で楽天Books固有の商品形態を絞り込める。たとえば文庫を検索する場合は次のように指定する。

```go
client, err := rakutenbooks.NewClient(nil,
    rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
    rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
    rakutenbooks.WithBookSize(rakutenbooks.BookSizeBunko),
)
```

公式の値は`BookSizeAll`（0、絞り込みなし）、`BookSizeTankobon`（1）、`BookSizeBunko`（2）、
`BookSizeShinsho`（3）、`BookSizeZenshuSosho`（4）、`BookSizeJiten`（5）、`BookSizeZukan`（6）、
`BookSizeEhon`（7）、`BookSizeCassetteCD`（8）、`BookSizeComic`（9）、`BookSizeMookOther`（10）である。

既定値は`BookSizeAll`であり、`size=9`を固定しない。漫画区分と商品形態は別の条件であり、文庫版なども
検索できるよう、必要な場合だけ利用側が商品形態を指定する。ISBN参照ではこの設定を使用しない。

## 7. 次ページの取得

`Limit` は1〜30件を指定できる。0は既定値の20件である。

```go
request := rakutenbooks.SearchBooksRequest{
    Title: "動物のお医者さん",
    Limit: 5,
}
result, err := client.SearchBooks(ctx, request)
```

`result.NextCursor` が空でなければ、同じ検索条件・Limit・漫画区分・商品形態で次回の `Cursor` へ渡す。

```go
request.Cursor = result.NextCursor
next, err := client.SearchBooks(ctx, request)
```

楽天Books側のページ番号は公開APIから隠蔽している。Cursorを別の検索条件、Limit、漫画区分、商品形態で
再利用すると入力エラーになる。

## 8. Raw response

楽天Books固有の販売情報を確認する場合はRaw response用メソッドを使う。

```go
result, raw, err := client.SearchBooksWithRawResponse(ctx, request)
```

ISBN参照にも `LookupBooksByISBNWithRawResponse` がある。

変換済み結果では、楽天Booksの販売情報のうち次を共通モデルから利用できる。

- `itemPrice`: `Normalized.Prices` の `current` 価格。JPY、税込、取得時刻付き
- `itemUrl`: `BookSource.URL` の通常商品URL
- `affiliateUrl`: Affiliate ID指定時の `BookSource.AffiliateURL`

Raw responseには、共通モデルへ変換していない在庫・販売状態、レビュー、試し読みURLなども
含まれ得る。通常商品URLとアフィリエイトURLは別フィールドとして保持する。

Rawを取得できることは、取得した情報を無期限に保存・再配布できることを意味しない。
楽天ウェブサービスの現行利用条件を確認する。

## 9. CLIデモ

リポジトリのルートで、必要な環境変数を設定して実行する。

一般コミックをタイトル検索する。

```text
go run ./examples/rakutenbooks -title "動物のお医者さん" -limit 5
```

BLコミックを検索する。

```text
go run ./examples/rakutenbooks -genre bl -title "セブンティーンシロップス"
```

TLコミックを検索する。

```text
go run ./examples/rakutenbooks -genre tl -title "メロすぎ朔椰"
```

文庫を検索する。

```text
go run ./examples/rakutenbooks -size 2 -title "動物のお医者さん"
```

ISBNを参照する。

```text
go run ./examples/rakutenbooks 9784758088732
```

Raw responseも保存する。

```text
go run ./examples/rakutenbooks -raw-output rakutenbooks-raw.json -title "動物のお医者さん"
```

全オプションは [CLIデモ](../../../examples/README.md) を参照する。

## 10. リクエスト頻度

楽天ウェブサービスの公式ヘルプでは、1つのApplication IDにつき1秒に1回以下の
リクエストとするよう案内されている。

`rakutenbooks.Client` は待機、直列化、自動リトライを行わない。複数goroutineや複数Clientから
同じApplication IDを使用する場合も、利用側で全体のリクエスト頻度を管理する。

公式ヘルプ:
https://webservice.faq.rakuten.net/hc/ja

## 11. 表示・保存上の注意

楽天ウェブサービスの利用条件はAPIで取得できることとは別に確認する必要がある。
2026-08-09時点の公式一次資料として、次を確認する。

- [楽天ウェブサービス 利用規約](https://webservice.rakuten.co.jp/guide/rule)
- [楽天ブックス書籍検索API](https://webservice.rakuten.co.jp/documentation/books-book-search)

公開されている利用規約とAPIドキュメントでは、具体的な保存期限を確認できない。楽天由来の
価格・販売可能情報を含む商品情報、およびその他のAPI取得情報を保存、表示、再利用する前に、
利用時点の最新の規約とガイドを確認する。

ライブラリはこれらの条件を自動的に履行しない。表示、保存、広告利用を行うアプリケーションは、
利用時点の公式ガイドと規約を確認する。

クレジット表示:
https://webservice.rakuten.co.jp/guide/credit

利用規約:
https://webservice.rakuten.co.jp/guide/rule

調査時点の詳細と判断根拠は
[楽天ブックス書籍検索APIの現行仕様とmankenでの利用範囲調査](../../research/030-rakuten-books-api.md)
を参照する。

## 12. 詳細仕様

検索条件、漫画区分、Cursor、ISBN参照、変換項目、HTTP・エラーの完全な動作は
[楽天Booksパッケージ仕様](spec.md)を参照する。
