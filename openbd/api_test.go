package openbd_test

import (
	"errors"
	"testing"

	"github.com/eamat-dot/manken/openbd"
)

// TestPublicAPI は、openbdだけのimportで共通結果とエラーを参照できることを検証する
func TestPublicAPI(t *testing.T) {
	book := openbd.Book{
		Normalized: openbd.NormalizedBook{
			Identifiers: []openbd.Identifier{{
				Type:  openbd.IdentifierTypeISBN13,
				Value: "9784088466361",
			}},
		},
		Sources: []openbd.BookSource{{Source: openbd.SourceOpenBD}},
	}
	result := openbd.ISBNLookupResult{Items: []openbd.ISBNLookupItem{{Books: []openbd.Book{book}}}}
	if result.Items[0].Books[0].Sources[0].Source != openbd.SourceOpenBD {
		t.Fatalf("source = %q", result.Items[0].Books[0].Sources[0].Source)
	}

	err := &openbd.Error{Kind: openbd.ErrorKindInvalidArgument}
	var classified *openbd.Error
	if !errors.As(err, &classified) {
		t.Fatal("errors.As() = false")
	}
}
