package dmm

import (
	"testing"

	"github.com/eamat-dot/manken/internal/daterange"
)

// TestKeywordQuery_DateRange は、DMMの包含日時境界へ日付範囲を変換することを検証する
func TestKeywordQuery_DateRange(t *testing.T) {
	dateRange, err := daterange.Parse("2024-02", "2024-02")
	if err != nil {
		t.Fatal(err)
	}
	values := keywordQuery("title", dateRange, 20, 1)
	if values.Get("gte_date") != "2024-02-01T00:00:00" || values.Get("lte_date") != "2024-02-29T23:59:59" {
		t.Fatalf("date query = %v", values)
	}
}

// TestSeriesSearchKey_DateRange は、異なる日付範囲でCursor検索キーが変わることを検証する
func TestSeriesSearchKey_DateRange(t *testing.T) {
	first, err := daterange.Parse("2024", "2024")
	if err != nil {
		t.Fatal(err)
	}
	second, err := daterange.Parse("2025", "2025")
	if err != nil {
		t.Fatal(err)
	}
	if seriesSearchKey("title", first) == seriesSearchKey("title", second) {
		t.Fatal("different date ranges share a cursor key")
	}
}
