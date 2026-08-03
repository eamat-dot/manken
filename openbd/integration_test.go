//go:build integration

package openbd_test

import (
	"context"
	"testing"
	"time"

	"github.com/eamat-dot/manken/openbd"
)

// TestIntegrationLookupBooksByISBN は、実サービスで該当ありと該当なしの位置対応を確認する
func TestIntegrationLookupBooksByISBN(t *testing.T) {
	client, err := openbd.NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{
		"4088466365",
		"9780000000002",
		"9784088466361",
	})
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if len(raw) == 0 || len(result.Items) != 3 {
		t.Fatalf("raw length = %d, items = %d", len(raw), len(result.Items))
	}
	if len(result.Items[0].Books) != 1 || len(result.Items[1].Books) != 0 ||
		len(result.Items[2].Books) != 1 {
		t.Fatalf("unexpected result counts: %d, %d, %d",
			len(result.Items[0].Books), len(result.Items[1].Books), len(result.Items[2].Books))
	}
	if result.Items[0].Books[0].Sources[0].ID != "9784088466361" {
		t.Fatalf("source ID = %q", result.Items[0].Books[0].Sources[0].ID)
	}
}
