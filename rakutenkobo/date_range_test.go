package rakutenkobo

import "testing"

// TestValidateSearchRequest_RejectsDateRange は、取得元条件がない日付範囲を拒否することを検証する
func TestValidateSearchRequest_RejectsDateRange(t *testing.T) {
	if _, _, _, _, _, err := validateSearchRequest(SearchRequest{Title: "漫画", DateFrom: "2024"}); err == nil {
		t.Fatal("date range was accepted")
	}
}
