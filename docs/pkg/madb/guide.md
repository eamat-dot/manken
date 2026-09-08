# MADBガイド

`madb` パッケージは、メディア芸術データベース（MADB）に登録された漫画単行本を検索し、書誌情報を取得するためのパッケージです。タイトルや著者名などによる書籍検索と、ISBNによる参照に対応しています。APIキーやアカウント登録は必要ありません。

## 直接利用する

`madb` パッケージを直接importして利用できます。

```go
import "github.com/eamat-dot/manken/madb"
```

最小限のタイトル検索は次のように書けます。

```go
package main

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/madb"
)

func main() {
	// MADB Clientを初期化する
	client, err := madb.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// MADBで書籍を検索する
	result, err := client.SearchBooks(
		context.Background(),
		madb.SearchRequest{Title: "動物のおしゃべり"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

## 書籍を検索する

タイトル、著者名、出版社名、フリーワード、除外語、出版時期を指定して検索できます。

```go
// MADBで書籍を検索する
result, err := client.SearchBooks(
	context.Background(),
	madb.SearchRequest{
		Author:    "神仙寺瑛",
		Publisher: "竹書房",
	},
)
```

検索結果には次ページを取得するための `NextCursor` が含まれる場合があります。検索条件やページングの詳しい仕様は[詳細仕様](#詳細仕様)を参照してください。

## ISBNで参照する

`LookupBooksByISBN` は1〜500件のISBNをまとめて指定できます。

```go
// MADBでISBNを参照する
result, err := client.LookupBooksByISBN(
	context.Background(),
	[]string{"4-08-846636-5", "9784048689410"},
)
if err != nil {
	log.Fatal(err)
}
```

ISBN-10とISBN-13に対応し、ASCIIハイフンやUnicode空白を含む表記も受け付けます。結果は入力順で返り、`RequestedISBN` には指定した文字列がそのまま保持されます。

## Rawレスポンスを取得する

変換前の応答も確認したい場合は、次のメソッドを利用できます。

- `SearchBooksWithRawResponse`
- `LookupBooksByISBNWithRawResponse`

どちらも変換済みの結果と、MADBから受信した成功レスポンス本文を返します。

## 利用条件

MADBのデータを利用する場合は出典を記載してください。編集・加工した場合は、その旨も記載してください。詳細は[メディア芸術データベース利用規約（公式）](https://mediaarts-db.artmuseums.go.jp/user_terms)を確認してください。

`madb.Attributions()` では、表示・保存に利用できる出典情報を通信なしで取得できます。検索・ISBN参照が成功した場合は、同じ情報が結果の `Attributions` にも含まれるため、JSONへ保存しても出典情報を一緒に残せます。

クレジット文には、mankenがデータを共通書誌形式へ変換したことも含まれます。共通書誌データをさらに編集・加工して利用する場合は、利用側で行った加工についても記載してください。

返される情報だけで利用規約への適合が保証されるわけではありません。利用時点の公式条件も確認してください。

## 詳細仕様

検索条件、カーソル、ISBN参照、Rawレスポンス、変換、通信、エラーの完全な仕様は[manken MADBパッケージ仕様](spec.md)を参照してください。
