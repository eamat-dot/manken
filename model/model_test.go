package model

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
		"title_reading", "book_series", "publication_series", "editions", "authors", "contributors", "publishers",
		"isbn10", "isbn13", "jan", "published_date", "release_date", "digital_release_date", "description", "languages",
		"subjects", "medium", "size", "list_price", "current_price", "cover_url", "normalized", "edition_statements",
		"identifiers", "dates", "physical_size", "images",
	} {
		if strings.Contains(got, `"`+field+`"`) {
			t.Fatalf("JSON contains missing field %q: %s", field, got)
		}
	}
	if got != `{"sources":[{"source":"madb"}]}` {
		t.Fatalf("Marshal() = %s", got)
	}
}

// TestBookJSON_UsesDirectFields は、最終Bookの直下フィールドをJSONへ出力することを検証する
func TestBookJSON_UsesDirectFields(t *testing.T) {
	pageCount := 0
	book := Book{
		Title:              "銀河鉄道の夜",
		TitleReading:       "ギンガテツドウノヨル",
		Subtitle:           "春と修羅",
		BookSeries:         []BookSeries{{Name: "作品系列"}},
		PublicationSeries:  []string{"叢書"},
		Volume:             Volume{Number: &pageCount, Label: "0"},
		Editions:           []string{"改訂版"},
		IsFinalVolume:      true,
		Authors:            []string{"宮沢賢治"},
		Contributors:       []Contributor{{Name: "宮沢賢治", Roles: []string{"著者"}}},
		Publishers:         []string{"架空社"},
		ISBN10:             []string{"0000000000"},
		ISBN13:             []string{"9780000000002"},
		JAN:                []string{"1920000000000"},
		PublishedDate:      "1934",
		ReleaseDate:        "1934-01-01",
		DigitalReleaseDate: "2026-01-01",
		Description:        "説明",
		Languages:          []string{"ja"},
		Subjects:           []Subject{{Name: "小説"}},
		PageCount:          &pageCount,
		Medium:             PublicationMediumPrint,
		Size:               "四六判",
		ListPrice:          &Price{Amount: 0, Currency: "JPY"},
		CurrentPrice:       &Price{Amount: 0, Currency: "JPY"},
		CoverURL:           "https://example.test/cover.jpg",
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	for _, fragment := range []string{
		`"title":"銀河鉄道の夜"`, `"title_reading":"ギンガテツドウノヨル"`, `"subtitle":"春と修羅"`,
		`"book_series":[{"name":"作品系列"}]`, `"publication_series":["叢書"]`, `"number":0`,
		`"editions":["改訂版"]`, `"is_final_volume":true`, `"authors":["宮沢賢治"]`,
		`"contributors":[{"name":"宮沢賢治","roles":["著者"]}]`, `"publishers":["架空社"]`,
		`"isbn10":["0000000000"]`, `"isbn13":["9780000000002"]`, `"jan":["1920000000000"]`,
		`"published_date":"1934"`, `"release_date":"1934-01-01"`, `"digital_release_date":"2026-01-01"`,
		`"description":"説明"`, `"languages":["ja"]`, `"subjects":[{"name":"小説"}]`, `"page_count":0`,
		`"medium":"print"`, `"size":"四六判"`, `"list_price":{"amount":0,"currency":"JPY","source":""}`,
		`"current_price":{"amount":0,"currency":"JPY","source":""}`, `"cover_url":"https://example.test/cover.jpg"`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("JSON does not contain %s: %s", fragment, got)
		}
	}
}

// TestBookJSON_UsesTitleReading は、タイトル読みをtitle_readingとして出力することを検証する
func TestBookJSON_UsesTitleReading(t *testing.T) {
	book := Book{
		Title:        "銀河鉄道の夜",
		TitleReading: "ギンガテツドウノヨル",
		Contributors: []Contributor{{
			Name:    "宮沢賢治",
			Reading: "ミヤザワケンジ",
		}},
		Sources: []BookSource{{
			Source: SourceMADB,
		}},
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	want := `{"title":"銀河鉄道の夜","title_reading":"ギンガテツドウノヨル","contributors":[{"name":"宮沢賢治","reading":"ミヤザワケンジ"}],"sources":[{"source":"madb"}]}`
	if got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

// TestBookSourceJSON_UsesOnlySourceReference は、BookSourceが取得元の参照情報だけを出力することを検証する
func TestBookSourceJSON_UsesOnlySourceReference(t *testing.T) {
	encoded, err := json.Marshal(BookSource{
		Source:       SourceMADB,
		ID:           "M292129",
		URL:          "https://example.test/books/M292129",
		AffiliateURL: "https://example.test/affiliate/M292129",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got, want := string(encoded), `{"source":"madb","id":"M292129","url":"https://example.test/books/M292129","affiliate_url":"https://example.test/affiliate/M292129"}`; got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
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

// TestBookJSON_PreservesMeaningfulZeroValues は、0巻、0円、falseの税込区分をJSONへ出力することを検証する
func TestBookJSON_PreservesMeaningfulZeroValues(t *testing.T) {
	zero := 0
	taxExcluded := false
	book := Book{
		Volume:        Volume{Number: &zero, Label: "0"},
		IsFinalVolume: true,
		CurrentPrice: &Price{
			Amount:      0,
			Currency:    "JPY",
			TaxIncluded: &taxExcluded,
			Source:      SourceMADB,
			ObservedAt:  "2026-07-31T00:00:00+09:00",
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
	if strings.Contains(got, `"type"`) {
		t.Fatalf("JSON contains removed price type: %s", got)
	}
}

// TestSearchRequestJSON_IncludesPublisherAndExclude は、出版社と除外検索の公開JSON名を検証する
func TestSearchRequestJSON_IncludesPublisherAndExclude(t *testing.T) {
	request := SearchRequest{
		Title:     "うる星",
		Publisher: "小学館",
		Query:     "新装版",
		Exclude:   "復刻box",
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"exclude":"復刻box"`) {
		t.Fatalf("JSON does not contain exclude: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"query":"新装版"`) {
		t.Fatalf("JSON does not contain query: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"publisher":"小学館"`) {
		t.Fatalf("JSON does not contain publisher: %s", encoded)
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

// TestAttributionJSON_UsesPublicFieldsAndOmitsMissingLicense は、出典情報のJSON名と任意ライセンス項目の省略を検証する
func TestAttributionJSON_UsesPublicFieldsAndOmitsMissingLicense(t *testing.T) {
	result := SearchBooksResult{
		Books: []Book{},
		Attributions: []Attribution{
			{
				Source:          SourceNDL,
				Scope:           AttributionScopeService,
				Text:            "国立国会図書館サーチAPIを利用",
				URL:             "https://ndlsearch.ndl.go.jp/",
				RequirementsURL: "https://ndlsearch.ndl.go.jp/help/api",
			},
			{
				Source:          SourceNDL,
				Scope:           AttributionScopeData,
				License:         "CC BY 4.0",
				LicenseURL:      "https://creativecommons.org/licenses/by/4.0/",
				RequirementsURL: "https://ndlsearch.ndl.go.jp/help/api/provider",
			},
		},
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	got := string(encoded)
	for _, fragment := range []string{
		`"attributions":[`, `"source":"ndl"`, `"scope":"service"`, `"scope":"data"`,
		`"text":"国立国会図書館サーチAPIを利用"`, `"url":"https://ndlsearch.ndl.go.jp/"`,
		`"license":"CC BY 4.0"`, `"license_url":"https://creativecommons.org/licenses/by/4.0/"`,
		`"requirements_url":"https://ndlsearch.ndl.go.jp/help/api"`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("JSON does not contain %s: %s", fragment, got)
		}
	}
	serviceJSON, err := json.Marshal(result.Attributions[0])
	if err != nil {
		t.Fatalf("Marshal(service) error = %v", err)
	}
	if strings.Contains(string(serviceJSON), `"license"`) || strings.Contains(string(serviceJSON), `"license_url"`) {
		t.Fatalf("service attribution contains missing license fields: %s", serviceJSON)
	}
}
