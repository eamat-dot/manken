package googlebooks

import "testing"

// TestBuildSearchQuery_RejectsDateRange は、取得元条件がない日付範囲を拒否することを検証する
func TestBuildSearchQuery_RejectsDateRange(t *testing.T) {
	if _, err := buildSearchQuery(SearchRequest{Title: "漫画", DateFrom: "2024"}); err == nil {
		t.Fatal("date range was accepted")
	}
}
