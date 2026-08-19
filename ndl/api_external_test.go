package ndl_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/eamat-dot/manken/ndl"
)

// requireRawSearchSignature は、SearchBooksWithRawResponseの公開シグネチャをコンパイル時に確認する
func requireRawSearchSignature(
	func(context.Context, ndl.SearchRequest) (ndl.SearchBooksResult, []byte, error),
) {
}

// requireRawLookupSignature は、LookupBooksByISBNWithRawResponseの公開シグネチャをコンパイル時に確認する
func requireRawLookupSignature(
	func(context.Context, []string) (ndl.ISBNLookupResult, []byte, error),
) {
}

// TestPublicAPI は、外部パッケージからNDLサーチの直下Bookフィールドを利用できることを確認する
func TestPublicAPI(t *testing.T) {
	client, err := ndl.NewClient(&http.Client{})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() client = nil")
	}
	if ndl.SourceNDL != "ndl" {
		t.Fatalf("SourceNDL = %q", ndl.SourceNDL)
	}
	_ = ndl.Book{
		Title:     "Title",
		ISBN13:    []string{"9784098627417"},
		ListPrice: &ndl.Price{Amount: 700, Currency: "JPY", Source: ndl.SourceNDL},
		Contributors: []ndl.Contributor{{
			Name:  "著者",
			Roles: []string{"著者"},
		}},
	}
	requireRawSearchSignature(client.SearchBooksWithRawResponse)
	requireRawLookupSignature(client.LookupBooksByISBNWithRawResponse)
}
