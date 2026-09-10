# Google Booksガイド

`googlebooks` パッケージは、Google Booksから書誌情報を検索・参照するためのパッケージです。漫画専用のサービスではないため、書籍検索には小説など漫画以外の書籍も含まれます。

## 準備するもの

Google CloudでBooks APIを有効にし、APIキーを作成してください。APIキーは環境変数やシークレット管理機能で保管し、ソースコードやログへ直接書かないことを推奨します。

`manken` はAPIキーを保存したり、環境変数から自動で読み込んだりしません。

## 直接利用する

`googlebooks` パッケージを直接インポートして利用できます。

```go
import "github.com/eamat-dot/manken/googlebooks"
```

最小限のタイトル検索は次のように書けます。

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/googlebooks"
)

func main() {
	// APIキーを指定してGoogle Books Clientを初期化する
	client, err := googlebooks.NewClient(nil,
		googlebooks.WithAPIKey(os.Getenv("GOOGLE_BOOKS_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Google Booksでタイトル検索する
	result, err := client.SearchBooks(
		context.Background(),
		googlebooks.SearchRequest{Title: "動物のお医者さん"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

## 書籍を検索する

タイトル、著者名、出版社名、フリーワード、除外語を指定できます。

```go
// Google Booksで著者名を指定して検索する
result, err := client.SearchBooks(
	context.Background(),
	googlebooks.SearchRequest{Author: "佐々木倫子"},
)
```

Google Booksでは漫画だけに絞り込む条件を指定できないため、検索結果が漫画だけになることは保証されません。

## ISBNで参照する

`LookupBooksByISBN` はISBNを1件だけ受け付けます。

```go
// Google BooksでISBNを参照する
result, err := client.LookupBooksByISBN(
	context.Background(),
	[]string{"4-08-846636-5"},
)
```

ISBN-10とISBN-13に対応し、ASCIIハイフンやUnicode空白を含む表記も受け付けます。問い合わせにはISBN-13を使用し、`RequestedISBN` には指定した文字列がそのまま保持されます。

## Raw responseを取得する

変換前の応答も確認したい場合は、次のメソッドを利用できます。

- `SearchBooksWithRawResponse`
- `LookupBooksByISBNWithRawResponse`

## 利用条件と表示

Google Booksの結果、画像、プレビュー、販売・閲覧情報をアプリケーションで利用する場合は、次の公式条件を確認してください。

- [Books API Terms of Service（公式）](https://developers.google.com/books/terms)
- [Branding Guidelines（公式）](https://developers.google.com/books/branding)
- [Google APIs Terms of Service（公式）](https://developers.google.com/terms)

Google Booksの情報を画面に表示する場合は、必要なクレジット表示やGoogle Booksへのリンクを設けてください。結果の保存やキャッシュ、表示順の変更を行う場合も、利用方法が公式条件に適合するか確認してください。

`manken` は利用方法が各規約に適合することを保証しません。

## 詳細仕様

検索条件、ページング、ISBN参照、変換項目、Raw response、エラーの完全な仕様は[manken Google Booksパッケージ仕様](spec.md)を参照してください。
