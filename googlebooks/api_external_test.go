package googlebooks_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/eamat-dot/manken/googlebooks"
)

func requireLookupBooksByISBNWithRawResponseSignature(
	func(context.Context, []string) (googlebooks.ISBNLookupResult, []byte, error),
) {
}

// TestPublicAPI は、外部パッケージからGoogle Booksの公開APIを利用できることを確認する
func TestPublicAPI(t *testing.T) {
	client, err := googlebooks.NewClient(&http.Client{}, googlebooks.WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() client = nil")
	}
	if googlebooks.SourceGoogleBooks != "googlebooks" {
		t.Fatalf("SourceGoogleBooks = %q", googlebooks.SourceGoogleBooks)
	}
	if googlebooks.PriceTypeList != "list" || googlebooks.PriceTypeCurrent != "current" {
		t.Fatalf("PriceTypeList = %q, PriceTypeCurrent = %q", googlebooks.PriceTypeList, googlebooks.PriceTypeCurrent)
	}
	_ = googlebooks.Price{Type: googlebooks.PriceTypeCurrent, Source: googlebooks.SourceGoogleBooks}
	requireLookupBooksByISBNWithRawResponseSignature(client.LookupBooksByISBNWithRawResponse)
}
