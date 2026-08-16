//go:build integration

package dmm_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/eamat-dot/manken/dmm"
)

// TestIntegrationDMM は、認証済みDMM APIでシリーズ検索とシリーズ内Book取得を確認する
func TestIntegrationDMM(t *testing.T) {
	apiID, affiliateID := os.Getenv("DMM_API_ID"), os.Getenv("DMM_AFFILIATE_ID")
	if apiID == "" || affiliateID == "" {
		t.Skip("DMM_API_ID or DMM_AFFILIATE_ID is not set")
	}
	client, err := dmm.NewClient(nil, dmm.WithAPIID(apiID), dmm.WithAffiliateID(affiliateID))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, raw, err := client.SearchSeriesWithRawResponse(ctx, dmm.SearchSeriesRequest{FreeText: "薬屋のひとりごと", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.BookSeries) < 2 || len(raw) == 0 {
		t.Fatal("DMM series search returned no series or no raw response")
	}
	if bytes.Contains(raw, []byte(apiID)) || bytes.Contains(raw, []byte(affiliateID)) {
		t.Error("raw response leaks credentials")
	}
	expectedSeriesID := os.Getenv("DMM_INTEGRATION_SERIES_ID")
	if expectedSeriesID == "" {
		expectedSeriesID = "761133"
	}
	var series dmm.SeriesSearchItem
	for _, candidate := range result.BookSeries {
		if candidate.ID == expectedSeriesID && len(candidate.Authors) > 0 && len(candidate.Publishers) > 0 && len(candidate.Subjects) > 0 && len(candidate.Images) > 0 && len(candidate.Sources) > 0 {
			series = candidate
			break
		}
	}
	if series.ID == "" {
		t.Fatalf("DMM series search missing enriched 薬屋のひとりごと candidate with ID %q: %#v", expectedSeriesID, result.BookSeries)
	}
	books, err := client.SearchBooksBySeries(ctx, dmm.SearchBooksBySeriesRequest{SeriesID: series.ID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(books.Books) == 0 || books.Books[0].Normalized.BookSeries[0].ID != series.ID || len(books.Books[0].Normalized.Publishers) == 0 {
		t.Fatal("DMM series search did not lead to a matching individual book")
	}
}
