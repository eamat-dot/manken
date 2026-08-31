package rakutenbooks_test

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/rakutenbooks"
)

// ExampleClient は、楽天Booksで検索とISBN参照を行う流れを示す
func ExampleClient() {
	// Application IDとAccess Keyを指定して楽天Books Clientを初期化する
	client, err := rakutenbooks.NewClient(nil,
		rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
		rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 楽天Booksでタイトル検索する
	books, err := client.SearchBooks(ctx, rakutenbooks.SearchRequest{Title: "動物のお医者さん"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))

	// 楽天BooksでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(ctx, []string{"9784758088732"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}
