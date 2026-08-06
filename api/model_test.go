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
		}},
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	for _, field := range []string{
		"title", "title_kana", "parallel_titles", "subtitle", "volume", "is_final_volume", "page_count", "prices", "values",
	} {
		if strings.Contains(got, `"`+field+`"`) {
			t.Fatalf("JSON contains missing field %q: %s", field, got)
		}
	}
	if !strings.Contains(got, `"normalized":{}`) || !strings.Contains(got, `"sources":[`) {
		t.Fatalf("JSON omits required containers: %s", got)
	}
}

// TestBookJSON_UsesTitleReading は、タイトル読みをtitle_readingとして出力することを検証する
func TestBookJSON_UsesTitleReading(t *testing.T) {
	book := Book{
		Normalized: NormalizedBook{
			Title:        "銀河鉄道の夜",
			TitleReading: "ギンガテツドウノヨル",
			Contributors: []Contributor{{
				Name:    "宮沢賢治",
				Reading: "ミヤザワケンジ",
			}},
		},
		Sources: []BookSource{{
			Source: SourceMADB,
		}},
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	want := `{"normalized":{"title":"銀河鉄道の夜","title_reading":"ギンガテツドウノヨル","contributors":[{"name":"宮沢賢治","reading":"ミヤザワケンジ"}]},"sources":[{"source":"madb"}]}`
	if got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

// TestBookSourceJSON_UsesOnlySourceReference は、BookSourceが取得元の参照情報だけを出力することを検証する
func TestBookSourceJSON_UsesOnlySourceReference(t *testing.T) {
	encoded, err := json.Marshal(BookSource{
		Source: SourceMADB,
		ID:     "M292129",
		URL:    "https://example.test/books/M292129",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got, want := string(encoded), `{"source":"madb","id":"M292129","url":"https://example.test/books/M292129"}`; got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

// TestNormalizedBookJSON_OmitsEmptyTitleReading は、空のタイトル読みと旧JSON名を出力しないことを検証する
func TestNormalizedBookJSON_OmitsEmptyTitleReading(t *testing.T) {
	encoded, err := json.Marshal(NormalizedBook{Title: "銀河鉄道の夜"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(encoded); got != `{"title":"銀河鉄道の夜"}` {
		t.Fatalf("Marshal() = %s", got)
	}
}

// TestContributorJSON_OmitsEmptyReadingAndRoles は、役割なしの寄与者を名前だけで出力することを検証する
func TestContributorJSON_OmitsEmptyReadingAndRoles(t *testing.T) {
	encoded, err := json.Marshal(Contributor{Name: "宮沢賢治"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got := string(encoded); got != `{"name":"宮沢賢治"}` {
		t.Fatalf("Marshal() = %s", got)
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
		Sources: []BookSource{{Source: SourceMADB}},
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
