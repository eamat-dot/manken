package rakutenbooks

import "testing"

// TestValidateSearchRequest_RejectsDateRange は、DateFromまたはDateToだけの指定を拒否することを検証する
func TestValidateSearchRequest_RejectsDateRange(t *testing.T) {
	for _, request := range []SearchRequest{{Title: "漫画", DateFrom: "2024"}, {Title: "漫画", DateTo: "2024"}} {
		if _, _, _, err := validateSearchRequest(request); err == nil {
			t.Fatalf("request = %#v was accepted", request)
		}
	}
}
