package madb_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eamat-dot/manken/madb"
)

// TestPublicAPI_SearchAndClassifiedError は、madbだけで検索結果と分類済みエラーを扱えることを検証する
func TestPublicAPI_SearchAndClassifiedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/sparql-results+json")
		_, err := writer.Write([]byte(`{
			"results": {
				"bindings": [{
					"resource": {
						"type": "uri",
						"value": "https://mediaarts-db.artmuseums.go.jp/id/M1"
					},
					"matchedISBN": {"type": "literal", "value": "9784088466361"},
					"id": {"type": "literal", "value": "M1"},
					"title": {"type": "literal", "value": "作品"}
				}]
			}
		}`))
		if err != nil {
			t.Errorf("Write() error = %v", err)
		}
	}))
	defer server.Close()

	client, err := madb.NewClient(server.Client(), madb.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var result madb.SearchBooksResult
	result, err = client.SearchBooks(context.Background(), madb.SearchRequest{
		Title:  "作品",
		Author: "著者",
		Query:  "新装版",
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(result.Books) != 1 ||
		len(result.Books[0].Sources) != 1 ||
		result.Books[0].Sources[0].Source != madb.SourceMADB ||
		len(result.Attributions) != 1 ||
		result.Attributions[0].Scope != madb.AttributionScopeData {
		t.Fatalf("result = %#v, want one MADB book with attribution", result)
	}
	credits := madb.Attributions()
	if len(credits) != 1 || credits[0].Source != madb.SourceMADB || credits[0].Scope != madb.AttributionScopeData {
		t.Fatalf("Attributions() = %#v", credits)
	}
	book := result.Books[0]
	_ = book.Editions
	_ = book.BookSeries
	_ = book.PublicationSeries

	var lookup madb.ISBNLookupResult
	lookup, err = client.LookupBooksByISBN(context.Background(), []string{"9784088466361"})
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if len(lookup.Items) != 1 {
		t.Fatalf("lookup = %#v, want one item", lookup)
	}

	_, err = client.SearchBooks(context.Background(), madb.SearchRequest{})
	var classified *madb.Error
	if !errors.As(err, &classified) {
		t.Fatalf("error = %T, want *madb.Error", err)
	}
	if classified.Kind != madb.ErrorKindInvalidArgument {
		t.Fatalf("Kind = %q, want %q", classified.Kind, madb.ErrorKindInvalidArgument)
	}

	kinds := []madb.ErrorKind{
		madb.ErrorKindInvalidArgument,
		madb.ErrorKindUpstream,
		madb.ErrorKindUnavailable,
		madb.ErrorKindInvalidResponse,
	}
	if len(kinds) != 4 {
		t.Fatalf("error kinds = %d, want 4", len(kinds))
	}

}
