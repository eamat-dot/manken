package yahooshopping

import "testing"

// TestSearchQuery_RejectsDateRange は、取得元条件がない日付範囲を拒否することを検証する
func TestSearchQuery_RejectsDateRange(t *testing.T) {
	if _, err := searchQuery(SearchRequest{Title: "漫画", DateFrom: "2024"}); err == nil {
		t.Fatal("date range was accepted")
	}
}
