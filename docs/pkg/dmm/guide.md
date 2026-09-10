# DMMブックス利用ガイド

`dmm` パッケージは、DMMブックスの一般向け電子コミックをシリーズから探し、そのシリーズに含まれる商品を取得するためのパッケージです。

DMMブックスは他のパッケージの `SearchBooks` とは検索方法が異なり、まずシリーズを検索し、選んだシリーズの商品を取得します。

## 準備するもの

DMM会員、DMMアフィリエイト、Webサービス利用登録で発行されるAPI IDとAffiliate IDが必要です。

認証情報は環境変数やシークレット管理機能で保管し、ソースコードやログへ直接書かないことを推奨します。完全なリクエストURLには認証情報が含まれるため、保存や公開を避けてください。

## 直接利用する

`dmm` パッケージを直接インポートして利用します。

```go
import "github.com/eamat-dot/manken/dmm"
```

シリーズを検索して、そのシリーズの商品を取得する最小例です。

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/dmm"
)

func main() {
	// 認証情報を指定してDMM Clientを初期化する
	client, err := dmm.NewClient(nil,
		dmm.WithAPIID(os.Getenv("DMM_API_ID")),
		dmm.WithAffiliateID(os.Getenv("DMM_AFFILIATE_ID")),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// DMMでシリーズを検索する
	series, err := client.SearchSeries(ctx, dmm.SearchSeriesRequest{
		FreeText: "黄泉のツガイ",
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(series.BookSeries) == 0 {
		return
	}

	// 選んだシリーズに含まれる商品を取得する
	books, err := client.SearchBooksBySeries(ctx, dmm.SearchBooksBySeriesRequest{
		SeriesID: series.BookSeries[0].ID,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(books.Books))
}
```

## シリーズを検索する

`SearchSeries` では、検索語、除外語、商品時期などを指定できます。

```go
// DMMで除外語と商品時期を指定してシリーズを検索する
series, err := client.SearchSeries(ctx, dmm.SearchSeriesRequest{
	FreeText:     "黄泉のツガイ",
	ExcludedText: "特装版",
	DateFrom:     "2024",
	DateTo:       "2024",
})
```

検索結果にはシリーズ候補と、候補を見分けるための商品由来の情報が含まれます。タイトルや著者名などの補助情報は、シリーズ自体の確定情報とは限りません。

## シリーズの商品を取得する

シリーズを選んだら、そのIDを `SearchBooksBySeries` に指定します。

```go
// DMMでシリーズIDを指定して商品を取得する
books, err := client.SearchBooksBySeries(ctx, dmm.SearchBooksBySeriesRequest{
	SeriesID: series.BookSeries[0].ID,
})
```

取得結果には電子書籍の単話、合本、無料版などが含まれる場合があります。

## Raw responseを取得する

変換前の応答も確認したい場合は、次のメソッドを利用できます。

- `SearchSeriesWithRawResponse`
- `SearchBooksBySeriesWithRawResponse`

認証情報を含むURLや、利用条件上公開できない情報を誤って保存・公開しないよう注意してください。

## 利用条件

DMMの商品情報、商品URL、アフィリエイトURL、画像などを表示・保存・再利用する場合は、DMMアフィリエイトとWebサービスの最新条件を確認してください。

- [DMMアフィリエイト（公式）](https://affiliate.dmm.com/)
- [DMM Webサービス（公式）](https://affiliate.dmm.com/api/)

## 詳細仕様

検索条件、シリーズ候補、個別商品、Raw response、エラーの完全な仕様は[manken DMMパッケージ仕様](spec.md)を参照してください。
