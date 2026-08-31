package ndl_test

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/ndl"
)

// ExampleClient は、NDLサーチで検索とISBN参照を行う流れを示す
func ExampleClient() {
	// NDLサーチ Clientを初期化する
	client, err := ndl.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// NDLサーチでタイトル検索する
	books, err := client.SearchBooks(ctx, ndl.SearchRequest{Title: "動物のお医者さん"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))

	// NDLサーチでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(ctx, []string{"4-08-846636-5"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}

// ExampleClient_SearchBooksWithOptions は、NDLサーチ固有の条件を追加して検索する方法を示す
func ExampleClient_SearchBooksWithOptions() {
	// NDLサーチ Clientを初期化する
	client, err := ndl.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// NDLサーチ固有の条件を追加して書籍を検索する
	books, err := client.SearchBooksWithOptions(
		context.Background(),
		ndl.SearchRequest{Title: "動物のお医者さん"},
		ndl.SearchOptions{Description: "ハムテル"},
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d books found", len(books.Books))
}
