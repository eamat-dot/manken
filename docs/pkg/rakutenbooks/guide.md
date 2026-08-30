# 楽天Booksガイド

`rakutenbooks` パッケージは、楽天ブックスから紙書籍を検索し、商品情報や書誌情報を取得するためのパッケージです。書籍検索とISBN参照に対応しています。

## 準備するもの

楽天ウェブサービスで発行したApplication IDとAccess Keyが必要です。Affiliate IDは検索に必須ではなく、楽天アフィリエイトのURLが必要な場合だけ設定します。

認証情報は環境変数や秘密情報管理機能で保管し、ソースコードへ直接書かないことを推奨します。`manken` は環境変数から自動では読み込みません。

## 直接利用する

`rakutenbooks` パッケージを直接importして利用できます。

```go
import "github.com/eamat-dot/manken/rakutenbooks"
```

最小限のタイトル検索は次のように書けます。

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/rakutenbooks"
)

func main() {
	// 認証情報を指定して楽天Books Clientを初期化する
	client, err := rakutenbooks.NewClient(nil,
		rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
		rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 楽天Booksでタイトル検索する
	result, err := client.SearchBooks(
		context.Background(),
		rakutenbooks.SearchRequest{Title: "動物のお医者さん"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

Affiliate IDを使う場合は、Client作成時に `WithAffiliateID` を追加してください。

```go
// アフィリエイトURLを使う場合はAffiliate IDを設定する
rakutenbooks.WithAffiliateID(os.Getenv("RAKUTEN_AFFILIATE_ID"))
```

## 書籍を検索する

既定では一般コミックを検索します。タイトル、著者名、出版社名を指定でき、複数指定した場合はすべての条件を満たす書籍へ絞り込みます。

```go
// 楽天Booksで著者名を指定して検索する
result, err := client.SearchBooks(
	context.Background(),
	rakutenbooks.SearchRequest{Author: "佐々木倫子"},
)
```

BLまたはTLを検索する場合は、Client作成時に `WithComicGenre` で `ComicGenreBL` または `ComicGenreTL` を指定してください。1回の検索で複数の漫画区分をまとめて検索することはありません。

`WithBookSize` を使うと、単行本、文庫、コミックなどの商品形態でも絞り込めます。指定できる値は[manken 楽天Booksパッケージ仕様](spec.md)を参照してください。

楽天Booksでは汎用のフリーワード検索と除外語検索を利用できないため、`Query` と `Exclude` には対応していません。

## ISBNで参照する

`LookupBooksByISBN` はISBNを1件だけ受け付けます。

```go
// 楽天BooksでISBNを参照する
result, err := client.LookupBooksByISBN(
	context.Background(),
	[]string{"9784758088732"},
)
```

ISBN-10とISBN-13に対応し、ASCIIハイフンやUnicode空白を含む表記も受け付けます。ISBN参照では、書籍検索で使用する漫画区分や商品形態による絞り込みは行いません。

該当する商品がない場合はエラーではなく、その項目の `Books` が空になります。

## Rawレスポンスを取得する

変換前の応答も確認したい場合は、次のメソッドを利用できます。

- `SearchBooksWithRawResponse`
- `LookupBooksByISBNWithRawResponse`

Affiliate IDを指定した場合、楽天Booksが返すアフィリエイトURLは `BookSource.AffiliateURL` から取得できます。

## CLIデモ

```text
go run ./examples/rakutenbooks -title "動物のお医者さん" -limit 5
go run ./examples/rakutenbooks -genre bl -title "セブンティーンシロップス"
go run ./examples/rakutenbooks 9784758088732
```

全オプションとRawレスポンスの保存方法は[CLIデモ](../../../examples/README.md)を参照してください。

## 利用条件

楽天ウェブサービスは1つのApplication IDにつき1秒に1回以下のリクエストを案内しています。`rakutenbooks.Client` は待機や自動リトライを行わないため、利用側でアクセス頻度を管理してください。

商品情報、画像、価格、アフィリエイトURLなどを表示・保存・再利用する場合は、利用時点の最新条件を確認してください。

- [楽天ウェブサービス 利用規約（公式）](https://webservice.rakuten.co.jp/guide/rule)
- [クレジット表示（公式）](https://webservice.rakuten.co.jp/guide/credit)
- [楽天ブックス書籍検索API（公式）](https://webservice.rakuten.co.jp/documentation/books-book-search)

## 詳細仕様

検索条件、漫画区分、商品形態、カーソル、ISBN参照、変換、Rawレスポンス、通信、エラーの完全な仕様は[manken 楽天Booksパッケージ仕様](spec.md)を参照してください。
