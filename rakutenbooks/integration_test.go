//go:build integration

package rakutenbooks_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/eamat-dot/manken/rakutenbooks"
)

const integrationRequestInterval = 1100 * time.Millisecond

// TestIntegrationRakutenBooks は、認証情報がある環境だけで代表的な実サービス検索を確認する
func TestIntegrationRakutenBooks(t *testing.T) {
	applicationID := os.Getenv("RAKUTEN_APP_ID")
	accessKey := os.Getenv("RAKUTEN_ACCESS_KEY")
	if applicationID == "" || accessKey == "" {
		t.Skip("RAKUTEN_APP_ID or RAKUTEN_ACCESS_KEY is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := newIntegrationClient(t, applicationID, accessKey, "", rakutenbooks.ComicGenreGeneral, rakutenbooks.BookSizeAll)
	result, raw, err := client.SearchBooksWithRawResponse(ctx, rakutenbooks.SearchBooksRequest{Title: "ふつつかな悪女ではございますが", Limit: 3})
	if err != nil {
		t.Fatalf("general title search error = %v", err)
	}
	if len(result.Books) == 0 || len(raw) == 0 {
		t.Fatalf("general result = %#v, raw bytes = %d", result, len(raw))
	}
	if got := firstAffiliateURL(t, raw); got != "" {
		t.Fatalf("affiliateUrl without Affiliate ID = %q", got)
	}
	if len(result.Books[0].Sources) == 0 || result.Books[0].Sources[0].AffiliateURL != "" {
		t.Fatalf("normalized affiliate URL without Affiliate ID = %#v", result.Books[0].Sources)
	}
	if len(result.Books[0].Normalized.Prices) == 0 {
		t.Fatalf("normalized price is missing: %#v", result.Books[0].Normalized)
	}
	price := result.Books[0].Normalized.Prices[0]
	if price.Type != rakutenbooks.PriceTypeCurrent || price.Currency != "JPY" || price.TaxIncluded == nil || !*price.TaxIncluded || price.Source != rakutenbooks.SourceRakutenBooks {
		t.Fatalf("normalized price = %#v", price)
	}
	if _, err := time.Parse(time.RFC3339Nano, price.ObservedAt); err != nil {
		t.Fatalf("price ObservedAt = %q: %v", price.ObservedAt, err)
	}

	time.Sleep(integrationRequestInterval)
	result, err = client.SearchBooks(ctx, rakutenbooks.SearchBooksRequest{Author: "佐々木倫子", Limit: 3})
	if err != nil || len(result.Books) == 0 {
		t.Fatalf("author search result = %#v, error = %v", result, err)
	}

	time.Sleep(integrationRequestInterval)
	sizeClient := newIntegrationClient(t, applicationID, accessKey, "", rakutenbooks.ComicGenreGeneral, rakutenbooks.BookSizeComic)
	sizeResult, err := sizeClient.SearchBooks(ctx, rakutenbooks.SearchBooksRequest{Title: "ふつつかな悪女ではございますが", Limit: 3})
	if err != nil || len(sizeResult.Books) == 0 {
		t.Fatalf("comic size search result = %#v, error = %v", sizeResult, err)
	}
	for _, book := range sizeResult.Books {
		if book.Normalized.PhysicalSize == nil || book.Normalized.PhysicalSize.Name != "コミック" {
			t.Fatalf("comic size search physical size = %#v", book.Normalized.PhysicalSize)
		}
	}

	time.Sleep(integrationRequestInterval)
	lookup, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{"9784758088732"})
	if err != nil {
		t.Fatalf("ISBN lookup error = %v", err)
	}
	if len(lookup.Items) != 1 || len(lookup.Items[0].Books) == 0 || len(raw) == 0 {
		t.Fatalf("ISBN lookup = %#v, raw bytes = %d", lookup, len(raw))
	}

	genreCases := []struct {
		genre rakutenbooks.ComicGenre
		title string
	}{
		{rakutenbooks.ComicGenreBL, "セブンティーンシロップス"},
		{rakutenbooks.ComicGenreTL, "メロすぎ朔椰"},
	}
	for _, test := range genreCases {
		time.Sleep(integrationRequestInterval)
		genreClient := newIntegrationClient(t, applicationID, accessKey, "", test.genre, rakutenbooks.BookSizeAll)
		genreResult, err := genreClient.SearchBooks(ctx, rakutenbooks.SearchBooksRequest{Title: test.title, Limit: 3})
		if err != nil || len(genreResult.Books) == 0 {
			t.Fatalf("genre %q search result = %#v, error = %v", test.genre, genreResult, err)
		}
	}

	affiliateID := os.Getenv("RAKUTEN_AFFILIATE_ID")
	if affiliateID == "" {
		t.Log("RAKUTEN_AFFILIATE_ID is not set; affiliate URL check skipped")
		return
	}
	time.Sleep(integrationRequestInterval)
	affiliateClient := newIntegrationClient(t, applicationID, accessKey, affiliateID, rakutenbooks.ComicGenreGeneral, rakutenbooks.BookSizeAll)
	affiliateResult, raw, err := affiliateClient.SearchBooksWithRawResponse(ctx, rakutenbooks.SearchBooksRequest{Title: "ふつつかな悪女ではございますが", Limit: 3})
	if err != nil {
		t.Fatalf("affiliate search error = %v", err)
	}
	got := firstAffiliateURL(t, raw)
	if got == "" {
		t.Fatal("affiliateUrl with Affiliate ID is empty")
	}
	if len(affiliateResult.Books) == 0 || len(affiliateResult.Books[0].Sources) == 0 || affiliateResult.Books[0].Sources[0].AffiliateURL != got {
		t.Fatalf("normalized affiliate URL = %#v, raw = %q", affiliateResult.Books, got)
	}
}

// newIntegrationClient は、実サービス確認用の楽天Books Clientを生成する
func newIntegrationClient(t *testing.T, applicationID string, accessKey string, affiliateID string, genre rakutenbooks.ComicGenre, size rakutenbooks.BookSize) *rakutenbooks.Client {
	t.Helper()
	options := []rakutenbooks.Option{
		rakutenbooks.WithApplicationID(applicationID),
		rakutenbooks.WithAccessKey(accessKey),
		rakutenbooks.WithComicGenre(genre),
	}
	if size != rakutenbooks.BookSizeAll {
		options = append(options, rakutenbooks.WithBookSize(size))
	}
	if affiliateID != "" {
		options = append(options, rakutenbooks.WithAffiliateID(affiliateID))
	}
	client, err := rakutenbooks.NewClient(nil, options...)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

// firstAffiliateURL は、Raw responseの先頭商品からaffiliateUrlを取り出す
func firstAffiliateURL(t *testing.T, raw []byte) string {
	t.Helper()
	var response struct {
		Items []struct {
			AffiliateURL string `json:"affiliateUrl"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	if len(response.Items) == 0 {
		t.Fatal("raw response has no items")
	}
	return response.Items[0].AffiliateURL
}
