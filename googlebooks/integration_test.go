//go:build integration

package googlebooks_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/eamat-dot/manken/googlebooks"
)

// TestIntegrationGoogleBooks は、APIキーがある環境だけで代表的な実サービス検索を確認する
func TestIntegrationGoogleBooks(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_BOOKS_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_BOOKS_API_KEY is not set")
	}
	client, err := googlebooks.NewClient(nil, googlebooks.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, request := range []googlebooks.SearchBooksRequest{
		{Title: "動物のお医者さん", Limit: 40},
		{Author: "佐々木倫子"},
		{FreeText: "日本 漫画"},
	} {
		result, err := client.SearchBooks(ctx, request)
		if err != nil {
			t.Fatalf("SearchBooks(%#v) error = %v", request, err)
		}
		if request.Limit == 40 && len(result.Books) == 40 && result.NextCursor != "" {
			request.Cursor = result.NextCursor
			if _, err := client.SearchBooks(ctx, request); err != nil {
				t.Fatalf("next page error = %v", err)
			}
		}
	}

	excluded, err := client.SearchBooks(ctx, googlebooks.SearchBooksRequest{Title: "動物のお医者さん", ExcludedText: "動物", Limit: 40})
	if err != nil {
		t.Fatalf("SearchBooks(ExcludedText) error = %v", err)
	}
	if len(excluded.Books) != 0 {
		t.Fatalf("SearchBooks(ExcludedText) returned %d books, want 0", len(excluded.Books))
	}

	result, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{"4088466365"})
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if len(result.Items) != 1 || len(raw) == 0 {
		t.Fatalf("items = %#v, raw bytes = %d", result.Items, len(raw))
	}
}
