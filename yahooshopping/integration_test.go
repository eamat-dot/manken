package yahooshopping

import (
	"context"
	"os"
	"testing"
)

// TestIntegrationTowerSearchAndISBNLookup は、認証情報がある環境でTowerの商品検索とISBN参照を確認する
func TestIntegrationTowerSearchAndISBNLookup(t *testing.T) {
	clientID := os.Getenv("YAHOO_SHOPPING_CLIENT_ID")
	if clientID == "" {
		t.Skip("YAHOO_SHOPPING_CLIENT_ID is not set")
	}
	client, err := NewClient(nil, WithClientID(clientID))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.SearchBooks(context.Background(), SearchBooksRequest{Title: "The Savior's Pride", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !hasISBN(result.Books, "9784758088336") {
		t.Fatalf("search result did not include ISBN 9784758088336: %+v", result.Books)
	}
	lookup, err := client.LookupBooksByISBN(context.Background(), []string{"9784758088336"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Items) != 1 || !hasISBN(lookup.Items[0].Books, "9784758088336") {
		t.Fatalf("lookup result did not include ISBN 9784758088336: %+v", lookup)
	}
}

// hasISBN は、書籍一覧に指定ISBN-13が含まれるかを判定する
func hasISBN(books []Book, isbn string) bool {
	for _, book := range books {
		for _, identifier := range book.Normalized.Identifiers {
			if identifier.Type == IdentifierTypeISBN13 && identifier.Value == isbn {
				return true
			}
		}
	}
	return false
}
