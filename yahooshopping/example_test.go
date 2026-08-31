package yahooshopping_test

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/yahooshopping"
)

// ExampleClient は、Yahoo!ショッピングで検索とISBN参照を行う流れを示す
func ExampleClient() {
	// Client IDを指定してYahoo!ショッピング Clientを初期化する
	client, err := yahooshopping.NewClient(nil,
		yahooshopping.WithClientID(os.Getenv("YAHOO_SHOPPING_CLIENT_ID")),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Yahoo!ショッピングでタイトル検索する
	books, err := client.SearchBooks(ctx, yahooshopping.SearchRequest{Title: "動物のお医者さん"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))

	// Yahoo!ショッピングでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(ctx, []string{"4-08-846636-5"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}
