package manken_test

import (
	"testing"

	"github.com/eamat-dot/manken"
)

// TestRootPackageAliases は、ルートパッケージだけで共通型と定数を利用できることを検証する
func TestRootPackageAliases(t *testing.T) {
	request := manken.SearchRequest{Title: "作品"}
	result := manken.SearchBooksResult{Books: []manken.Book{{Title: request.Title}}}
	err := &manken.Error{Kind: manken.ErrorKindInvalidArgument}
	if result.Books[0].Title != "作品" || err.Kind != manken.ErrorKindInvalidArgument || manken.SourceMADB == "" {
		t.Fatal("root package aliases are not usable")
	}
}
