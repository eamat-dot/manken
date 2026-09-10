# openBDガイド

`openbd` パッケージは、openBDからISBNで書誌情報を取得するためのパッケージです。タイトルや著者名による書籍検索は行わず、ISBN参照にだけ対応しています。APIキーやアカウント登録は必要ありません。

## 直接利用する

`openbd` パッケージを直接インポートして利用できます。

```go
import "github.com/eamat-dot/manken/openbd"
```

最小限のISBN参照は次のように書けます。

```go
package main

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/openbd"
)

func main() {
	// openBD Clientを初期化する
	client, err := openbd.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// openBDでISBNを参照する
	result, err := client.LookupBooksByISBN(
		context.Background(),
		[]string{"4-08-846636-5"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d results", len(result.Items))
}
```

## ISBNで参照する

`LookupBooksByISBN` は1〜1000件のISBNをまとめて指定できます。

```go
// openBDで複数のISBNをまとめて参照する
result, err := client.LookupBooksByISBN(
	context.Background(),
	[]string{"4-08-846636-5", "9784048689410"},
)
```

ISBN-10とISBN-13に対応し、ASCIIハイフンやUnicode空白を含む表記も受け付けます。問い合わせにはISBN-13を使用します。結果は入力と同じ順序で返り、`RequestedISBN` には指定した文字列がそのまま保持されます。

openBDに該当する書誌がないISBNも結果には残り、その項目の `Books` は空になります。

## Raw responseを取得する

`LookupBooksByISBNWithRawResponse` を使うと、変換済みの結果とopenBDから受信した成功レスポンス本文を取得できます。

## openBDの収録状況について

従来のopenBD APIバージョン1は提供終了が告知されており、現在の同一エンドポイントでは国立国会図書館の書誌情報を従来形式へ簡易変換した代替APIが提供されています。そのため、ISBNによっては未収録だったり、一部の書誌項目や書影が含まれなかったりします。

## 利用条件

openBDの書誌・書影などは、本の紹介・販促目的に限って利用できます。取得データの任意改変は認められておらず、削除要請を受けた場合は対応が必要です。詳細は[openBD API利用規約（公式）](https://openbd.jp/terms/)を確認してください。

## 詳細仕様

入力件数、変換、Raw response、通信、エラーの完全な仕様は[manken openBDパッケージ仕様](spec.md)を参照してください。
