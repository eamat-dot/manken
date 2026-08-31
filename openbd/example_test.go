package openbd_test

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/openbd"
)

// ExampleClient は、openBDでISBN参照を行う流れを示す
func ExampleClient() {
	// openBD Clientを初期化する
	client, err := openbd.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// openBDでISBNから書籍を参照する
	lookup, err := client.LookupBooksByISBN(context.Background(), []string{"4-08-846636-5"})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d ISBN results found", len(lookup.Items))
}
