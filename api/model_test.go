package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBookJSON_OmitsMissingOptionalFields は、欠落した任意項目をJSONから省略することを検証する
func TestBookJSON_OmitsMissingOptionalFields(t *testing.T) {
	book := Book{
		Sources: []BookSource{{
			Source: SourceMADB,
			Values: SourceBookValues{},
		}},
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	for _, field := range []string{
		"title", "subtitle", "volume", "is_final_volume", "page_count", "prices",
	} {
		if strings.Contains(got, `"`+field+`"`) {
			t.Fatalf("JSON contains missing field %q: %s", field, got)
		}
	}
	if !strings.Contains(got, `"normalized":{}`) || !strings.Contains(got, `"sources":[`) {
		t.Fatalf("JSON omits required containers: %s", got)
	}
}

// TestBookJSON_PreservesMeaningfulZeroValues は、0巻、0円、falseの税込区分をJSONへ出力することを検証する
func TestBookJSON_PreservesMeaningfulZeroValues(t *testing.T) {
	zero := 0
	taxExcluded := false
	book := Book{
		Normalized: NormalizedBook{
			Volume:        Volume{Number: &zero, Label: "0"},
			IsFinalVolume: true,
			Prices: []Price{{
				Type:        PriceTypeCurrent,
				Amount:      0,
				Currency:    "JPY",
				TaxIncluded: &taxExcluded,
				Source:      SourceMADB,
				ObservedAt:  "2026-07-31T00:00:00+09:00",
			}},
		},
		Sources: []BookSource{{Source: SourceMADB, Values: SourceBookValues{}}},
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	for _, fragment := range []string{
		`"number":0`, `"is_final_volume":true`, `"amount":0`, `"tax_included":false`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("JSON does not contain %s: %s", fragment, got)
		}
	}
}

// TestSearchBooksRequestJSON_IncludesExcludedText は、除外検索の公開JSON名を検証する
func TestSearchBooksRequestJSON_IncludesExcludedText(t *testing.T) {
	request := SearchBooksRequest{
		Title:        "うる星",
		ExcludedText: "復刻box",
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"excluded_text":"復刻box"`) {
		t.Fatalf("JSON does not contain excluded_text: %s", encoded)
	}
}
