package madb_test

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/madb"
)

// ExampleClient は、MADBで検索とISBN参照を行う流れを示す
func ExampleClient() {
	// MADB Clientを初期化する
	client, err := madb.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// MADBでタイトル検索する
	books, err := client.SearchBooks(ctx, madb.SearchRequest{Title: "動物のおしゃべり"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))

	// MADBでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(ctx, []string{"4-08-846636-5"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}
