package manken_test

import (
	"context"
	"log"

	"github.com/eamat-dot/manken"
	"github.com/eamat-dot/manken/madb"
)

// ExampleClient は、MADBを登録して書籍検索とISBN参照を行う例を示す
func ExampleClient() {
	// MADB Clientを初期化する
	provider, err := madb.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// MADBをmanken.Clientに登録する
	client, err := manken.NewClient(manken.WithMADBClient(provider))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// MADBでタイトル検索する
	books, err := client.SearchBooks(ctx, manken.SourceMADB, manken.SearchRequest{
		Title: "動物のおしゃべり",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))

	// MADBでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(ctx, manken.SourceMADB, []string{"4-08-846636-5"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}
