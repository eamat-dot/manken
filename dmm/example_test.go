package dmm_test

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/dmm"
)

// ExampleClient は、DMMでシリーズを検索して個別商品を取得する流れを示す
func ExampleClient() {
	// API IDとAffiliate IDを指定してDMM Clientを初期化する
	client, err := dmm.NewClient(nil,
		dmm.WithAPIID(os.Getenv("DMM_API_ID")),
		dmm.WithAffiliateID(os.Getenv("DMM_AFFILIATE_ID")),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// DMMでシリーズを検索する
	series, err := client.SearchSeries(ctx, dmm.SearchSeriesRequest{FreeText: "黄泉のツガイ"})
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
