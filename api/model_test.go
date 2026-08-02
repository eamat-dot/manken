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
		"title", "parallel_titles", "subtitle", "volume", "is_final_volume", "page_count", "prices",
	} {
		if strings.Contains(got, `"`+field+`"`) {
			t.Fatalf("JSON contains missing field %q: %s", field, got)
		}
	}
	if !strings.Contains(got, `"normalized":{}`) || !strings.Contains(got, `"sources":[`) {
		t.Fatalf("JSON omits required containers: %s", got)
	}
}

// TestBookJSON_PreservesTitleContracts は、タイトル項目と完全な元タイトルを同時に保持することを検証する
func TestBookJSON_PreservesTitleContracts(t *testing.T) {
	book := Book{
		Normalized: NormalizedBook{
			Title:          "銀河鉄道の夜",
			ParallelTitles: []string{"Night on the Galactic Railroad", "Nokto de la Galaksia Fervojo"},
			Subtitle:       "初期形",
		},
		Sources: []BookSource{{
			Source: SourceMADB,
			Values: SourceBookValues{
				Titles: []string{"銀河鉄道の夜 = Night on the Galactic Railroad = Nokto de la Galaksia Fervojo"},
			},
		}},
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	want := `{"normalized":{"title":"銀河鉄道の夜","parallel_titles":["Night on the Galactic Railroad","Nokto de la Galaksia Fervojo"],"subtitle":"初期形"},"sources":[{"source":"madb","values":{"titles":["銀河鉄道の夜 = Night on the Galactic Railroad = Nokto de la Galaksia Fervojo"]}}]}`
	if got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

// TestBookJSON_OmitsEmptyParallelTitles は、空の並列タイトルをJSONから省略することを検証する
func TestBookJSON_OmitsEmptyParallelTitles(t *testing.T) {
	tests := []struct {
		name           string
		parallelTitles []string
	}{
		{name: "nil", parallelTitles: nil},
		{name: "empty", parallelTitles: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(NormalizedBook{ParallelTitles: tt.parallelTitles})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if got := string(encoded); got != `{}` {
				t.Fatalf("Marshal() = %s, want {}", got)
			}
		})
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

// TestISBNLookupResultJSON_PreservesEmptySlices は、ISBN参照の空結果をJSON配列として出力する
func TestISBNLookupResultJSON_PreservesEmptySlices(t *testing.T) {
	result := ISBNLookupResult{
		Items: []ISBNLookupItem{{
			RequestedISBN: "9780000000002",
			Books:         []Book{},
		}},
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(encoded); got != `{"items":[{"requested_isbn":"9780000000002","books":[]}]}` {
		t.Fatalf("Marshal() = %s", got)
	}

	empty, err := json.Marshal(ISBNLookupResult{Items: []ISBNLookupItem{}})
	if err != nil {
		t.Fatalf("Marshal(empty) error = %v", err)
	}
	if got := string(empty); got != `{"items":[]}` {
		t.Fatalf("Marshal(empty) = %s", got)
	}
}
