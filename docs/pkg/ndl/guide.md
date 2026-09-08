# NDLサーチガイド

`ndl` パッケージは、国立国会図書館サーチの全国書誌から書籍を検索し、書誌情報を取得するためのパッケージです。書籍検索とISBN参照に対応しています。APIキーやアクセストークンは必要ありません。

## 直接利用する

`ndl` パッケージを直接importして利用できます。

```go
import "github.com/eamat-dot/manken/ndl"
```

最小限のタイトル検索は次のように書けます。

```go
package main

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/ndl"
)

func main() {
	// NDLサーチ Clientを初期化する
	client, err := ndl.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// NDLサーチでタイトル検索する
	result, err := client.SearchBooks(
		context.Background(),
		ndl.SearchRequest{Title: "動物のお医者さん"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

## 漫画を検索する

書籍検索は既定で `NDC 726.1` と `NDLC Y84` の分類を使い、漫画に絞り込みます。古い漫画など、分類が付与されていない書誌も探したい場合は、必要に応じて分類による絞り込みを無効にできます。

```go
// NDLCによる漫画の絞り込みを無効にしてClientを初期化する
client, err := ndl.NewClient(nil,
	ndl.WithMangaNDLCFilter(false),
)
```

両方の分類を使わない場合は、次のように指定します。

```go
// NDCとNDLCによる漫画の絞り込みを無効にしてClientを初期化する
client, err := ndl.NewClient(nil,
	ndl.WithMangaNDCFilter(false),
	ndl.WithMangaNDLCFilter(false),
)
```

分類による絞り込みは書籍検索だけに適用され、ISBN参照には影響しません。

## 検索条件を追加する

タイトル、著者名、出版社名、フリーワード、除外語、出版時期で検索できます。NDLサーチ固有の件名や内容記述を使う場合は `SearchBooksWithOptions` を利用します。

```go
// NDLサーチ固有の内容記述を含めて検索する
result, err := client.SearchBooksWithOptions(
	context.Background(),
	ndl.SearchRequest{
		Title:    "動物のお医者さん",
		DateFrom: "2024",
		DateTo:   "2024",
	},
	ndl.SearchOptions{Description: "ハムテル"},
)
```

`DateFrom` と `DateTo` は `YYYY`、`YYYY-MM`、`YYYY-MM-DD` で指定できます。両方を指定する場合は精度をそろえてください。

## ISBNで参照する

`LookupBooksByISBN` はISBNを1件だけ受け付けます。

```go
// NDLサーチでISBNを参照する
result, err := client.LookupBooksByISBN(
	context.Background(),
	[]string{"4-08-846636-5"},
)
```

ISBN-10とISBN-13に対応し、ASCIIハイフンやUnicode空白を含む表記も受け付けます。ISBN参照では、書籍検索で使用する漫画分類による絞り込みは行いません。

古い刊行物では、NDLに書誌が存在していてもISBNが記録されていない場合があります。その場合、ISBN参照では見つかりません。

## Rawレスポンスを取得する

変換前の応答も確認したい場合は、次のメソッドを利用できます。

- `SearchBooksWithRawResponse`
- `SearchBooksWithOptionsAndRawResponse`
- `LookupBooksByISBNWithRawResponse`

## 利用条件

NDLサーチAPIを利用するサイトやアプリケーションでは、NDLサーチAPIを使用していることを表示してください。全国書誌情報を二次利用する場合は、適用されるメタデータの利用条件や表示要件も確認してください。

`ndl.Attributions()` では、API利用表示と全国書誌情報の出典・ライセンス情報を通信なしで取得できます。検索・ISBN参照が成功した場合は、0件でも同じ情報が結果の `Attributions` に含まれるため、JSONへ保存しても出典情報を一緒に残せます。

データのクレジット文には、mankenが共通書誌形式へ変換したことも含まれます。共通書誌データをさらに編集・加工して利用する場合は、利用側で行った加工についても記載してください。

返される情報だけで利用条件への適合が保証されるわけではありません。利用時点の公式条件も確認してください。

同時・継続的な大量アクセスは制限または遮断される場合があります。Clientは待機や自動リトライを行わないため、利用量が大きい場合はNDLサーチの利用案内を確認してください。

- [NDLサーチ APIのご利用について（公式）](https://ndlsearch.ndl.go.jp/help/api)
- [API提供対象データプロバイダ一覧（公式）](https://ndlsearch.ndl.go.jp/help/api/provider)

## 詳細仕様

検索条件、分類による絞り込み、ISBN参照、Rawレスポンス、変換、通信、エラーの完全な仕様は[manken NDLサーチパッケージ仕様](spec.md)を参照してください。
