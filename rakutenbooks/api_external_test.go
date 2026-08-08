package rakutenbooks_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/eamat-dot/manken/rakutenbooks"
)

// requireRawSearchSignature は、SearchBooksWithRawResponseの公開シグネチャをコンパイル時に確認する
func requireRawSearchSignature(
	func(context.Context, rakutenbooks.SearchBooksRequest) (rakutenbooks.SearchBooksResult, []byte, error),
) {
}

// requireRawLookupSignature は、LookupBooksByISBNWithRawResponseの公開シグネチャをコンパイル時に確認する
func requireRawLookupSignature(
	func(context.Context, []string) (rakutenbooks.ISBNLookupResult, []byte, error),
) {
}

// TestPublicAPI は、外部パッケージから楽天Booksの公開APIを利用できることを確認する
func TestPublicAPI(t *testing.T) {
	client, err := rakutenbooks.NewClient(
		&http.Client{},
		rakutenbooks.WithApplicationID("application-id"),
		rakutenbooks.WithAccessKey("access-key"),
		rakutenbooks.WithComicGenre(rakutenbooks.ComicGenreBL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() client = nil")
	}
	if rakutenbooks.SourceRakutenBooks != "rakutenbooks" {
		t.Fatalf("SourceRakutenBooks = %q", rakutenbooks.SourceRakutenBooks)
	}
	requireRawSearchSignature(client.SearchBooksWithRawResponse)
	requireRawLookupSignature(client.LookupBooksByISBNWithRawResponse)
}
