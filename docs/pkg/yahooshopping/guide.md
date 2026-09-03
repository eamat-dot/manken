# Yahoo!ショッピングガイド

`yahooshopping` パッケージは、Yahoo!ショッピングの商品検索APIを使って、タワーレコード Yahoo!店の紙コミック商品を検索し、商品情報や書誌情報を取得するためのパッケージです。書籍専用の検索APIではないため、タイトルや著者名などは商品検索用のキーワードとして扱われます。

## 準備するもの

Yahoo!デベロッパーネットワークで発行したClient IDが必要です。Client IDは環境変数や秘密情報管理機能で保管し、ソースコードへ直接書かないことを推奨します。

`manken` はClient IDを環境変数から自動では読み込みません。

## 直接利用する

`yahooshopping` パッケージを直接importして利用できます。

```go
import "github.com/eamat-dot/manken/yahooshopping"
```

最小限のタイトル検索は次のように書けます。

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/yahooshopping"
)

func main() {
	// Client IDを指定してYahoo!ショッピング Clientを初期化する
	client, err := yahooshopping.NewClient(nil,
		yahooshopping.WithClientID(os.Getenv("YAHOO_SHOPPING_CLIENT_ID")),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Yahoo!ショッピングでタイトル検索する
	result, err := client.SearchBooks(
		context.Background(),
		yahooshopping.SearchRequest{Title: "動物のお医者さん"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

## 書籍を検索する

タイトル、著者名、出版社名、フリーワードを指定できます。これらの値はYahoo!ショッピングの商品検索用キーワードとしてまとめて送信されるため、書誌データベースの項目検索とは一致の仕方が異なります。

```go
// Yahoo!ショッピングで著者名を指定して検索する
result, err := client.SearchBooks(
	context.Background(),
	yahooshopping.SearchRequest{Author: "佐々木倫子"},
)
```

書籍検索はタワーレコード Yahoo!店のコミック商品に限定されます。他のYahoo!ショッピング出店者の商品は検索しません。

## ISBNで参照する

`LookupBooksByISBN` はISBNを1件だけ受け付けます。

```go
// Yahoo!ショッピングでISBNを参照する
result, err := client.LookupBooksByISBN(
	context.Background(),
	[]string{"4-08-846636-5"},
)
```

ISBN-10とISBN-13に対応し、ASCIIハイフンやUnicode空白を含む表記も受け付けます。問い合わせにはISBN-13を使用し、`RequestedISBN` には指定した文字列がそのまま保持されます。

ISBN参照もタワーレコード Yahoo!店の商品だけを対象にします。

## Rawレスポンスを取得する

変換前の応答も確認したい場合は、次のメソッドを利用できます。

- `SearchBooksWithRawResponse`
- `LookupBooksByISBNWithRawResponse`

## 利用条件

公式の案内に従い、1秒に1回を超えないよう利用側でアクセス頻度を管理してください。クレジット表示も必要です。Clientは待機や自動リトライを行いません。

商品情報、画像、Rawレスポンスを表示・保存・再利用する場合は、利用時点のYahoo!ショッピングの公式条件を確認してください。

- [Yahoo!デベロッパーネットワーク ご利用ガイド（公式）](https://developer.yahoo.co.jp/start/)
- [クレジット表示（公式）](https://developer.yahoo.co.jp/attribution/)

## 詳細仕様

検索条件、タワーレコード Yahoo!店への限定、ISBN参照、Rawレスポンス、変換、通信、エラーの完全な仕様は[manken Yahoo!ショッピングパッケージ仕様](spec.md)を参照してください。
