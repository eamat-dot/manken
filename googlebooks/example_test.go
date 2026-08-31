package googlebooks_test

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/googlebooks"
)

// ExampleClient は、Google Booksで検索とISBN参照を行う流れを示す
func ExampleClient() {
	// APIキーを指定してGoogle Books Clientを初期化する
	client, err := googlebooks.NewClient(nil, googlebooks.WithAPIKey(os.Getenv("GOOGLE_BOOKS_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Google Booksでタイトル検索する
	books, err := client.SearchBooks(ctx, googlebooks.SearchRequest{Title: "動物のお医者さん"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))

	// Google BooksでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(ctx, []string{"4-08-846636-5"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}
