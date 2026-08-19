//go:build integration

package rakutenkobo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/eamat-dot/manken/rakutenkobo"
)

// TestIntegrationRakutenKobo は、認証情報がある環境だけで代表的な実サービス検索を確認する
func TestIntegrationRakutenKobo(t *testing.T) {
	applicationID, accessKey := os.Getenv("RAKUTEN_APP_ID"), os.Getenv("RAKUTEN_ACCESS_KEY")
	if applicationID == "" || accessKey == "" {
		t.Skip("RAKUTEN_APP_ID or RAKUTEN_ACCESS_KEY is not set")
	}
	client, err := rakutenkobo.NewClient(nil, rakutenkobo.WithApplicationID(applicationID), rakutenkobo.WithAccessKey(accessKey))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, raw, err := client.SearchBooksWithRawResponse(ctx, rakutenkobo.SearchRequest{Title: "ふつつかな悪女ではございますが", Exclude: "分冊", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Books) == 0 || len(raw) == 0 || result.Books[0].Medium != rakutenkobo.PublicationMediumDigital {
		t.Fatalf("result = %#v, raw bytes = %d", result, len(raw))
	}
}
