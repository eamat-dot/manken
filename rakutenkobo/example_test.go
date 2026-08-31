package rakutenkobo_test

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/rakutenkobo"
)

// ExampleClient は、楽天Koboで電子コミックを検索する流れを示す
func ExampleClient() {
	// Application IDとAccess Keyを指定して楽天Kobo Clientを初期化する
	client, err := rakutenkobo.NewClient(nil,
		rakutenkobo.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
		rakutenkobo.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 楽天Koboでタイトル検索する
	books, err := client.SearchBooks(
		context.Background(),
		rakutenkobo.SearchRequest{Title: "ふつつかな悪女ではございますが"},
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))
}
