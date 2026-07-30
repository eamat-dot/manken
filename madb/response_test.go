package madb

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildSearchResult_AggregatesAndConverts は、複数bindingの集約と共通モデル変換を検証する
func TestBuildSearchResult_AggregatesAndConverts(t *testing.T) {
	response := newSPARQLResponse(
		testBinding("M1", map[string]string{
			"id":                "M1",
			"title":             "B",
			"subtitle":          "副題B",
			"seriesName":        "シリーズ",
			"relatedSeriesName": "参照シリーズ",
			"seriesID":          "C1",
			"volumeNumber":      "1",
			"version":           "新装版",
			"creator":           "[著]使わない著者",
			"agentName":         "著者B",
			"publisher":         "出版社B",
			"brand":             "レーベルB",
			"isbn":              "0-8044-2957-X",
			"publishedDate":     "2026-07",
		}),
		testBinding("M1", map[string]string{
			"title":             "A",
			"subtitle":          "副題A",
			"relatedSeriesName": "シリーズ",
			"seriesID":          "C1",
			"agentName":         "著者A",
			"publisher":         "出版社A",
			"version":           "[通常版]",
			"brand":             "レーベルA",
			"isbn":              "978-0-306-40615-7",
		}),
		testBinding("M1", map[string]string{
			"isbn": "0-306-40615-2",
		}),
		testBinding("M1", map[string]string{
			"isbn": "978-3-16-148410-0",
		}),
		testBinding("M1", map[string]string{
			"title":     "A",
			"agentName": "著者A",
			"isbn":      "invalid",
		}),
	)
	response.Results.Bindings[0]["seriesResource"] = sparqlValue{
		Type:  "uri",
		Value: seriesResourceURI("C1"),
	}
	response.Results.Bindings[1]["seriesResource"] = sparqlValue{
		Type:  "uri",
		Value: seriesResourceURI("C1"),
	}

	result, err := buildSearchResult(response, "作品", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("len(Books) = %d, want 1", len(result.Books))
	}

	book := result.Books[0]
	assertStrings(t, book.Titles, []string{"A", "B"})
	assertStrings(t, book.Subtitles, []string{"副題A", "副題B"})
	assertStrings(t, book.SeriesNames, []string{"シリーズ", "参照シリーズ"})
	assertStrings(t, book.EditionStatements, []string{"[通常版]", "新装版"})
	assertStrings(t, book.Authors, []string{"著者A", "著者B"})
	assertStrings(t, book.Publishers, []string{"出版社A", "出版社B"})
	assertStrings(t, book.Imprints, []string{"レーベルA", "レーベルB"})
	assertStrings(t, book.ISBN10s, []string{"0306406152", "080442957X"})
	assertStrings(t, book.ISBN13s, []string{"9780306406157", "9783161484100"})
	if book.SeriesID != "C1" || book.SeriesURL != seriesResourceURI("C1") {
		t.Fatalf("series fields = %#v", book)
	}
	if book.Source != SourceMADB || book.SourceURL != resourceURI("M1") {
		t.Fatalf("source fields = %#v", book)
	}
}

// TestBuildSearchResult_CreatorFallbackAndMissingValues は、creatorフォールバックと欠落値を検証する
func TestBuildSearchResult_CreatorFallbackAndMissingValues(t *testing.T) {
	response := newSPARQLResponse(
		testBinding("M1", map[string]string{"creator": "[原作][構成]  山田太郎 "}),
		testBinding("M1", map[string]string{"creator": "[著]佐藤花子"}),
		testBinding("M1", map[string]string{"creator": "[役割]"}),
	)

	result, err := buildSearchResult(response, "作品", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	book := result.Books[0]
	assertStrings(t, book.Authors, []string{"佐藤花子", "山田太郎"})
	for name, values := range map[string][]string{
		"Titles":            book.Titles,
		"Subtitles":         book.Subtitles,
		"SeriesNames":       book.SeriesNames,
		"EditionStatements": book.EditionStatements,
		"Publishers":        book.Publishers,
		"Imprints":          book.Imprints,
		"ISBN10s":           book.ISBN10s,
		"ISBN13s":           book.ISBN13s,
	} {
		if values == nil || len(values) != 0 {
			t.Fatalf("%s = %#v, want non-nil empty slice", name, values)
		}
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, field := range []string{"edition_statements", "imprints"} {
		if strings.Contains(string(encoded), `"`+field+`":null`) {
			t.Fatalf("JSON contains null %s: %s", field, encoded)
		}
	}
}

// TestBuildSearchResult_RejectsInvalidBindings は、不正なresource、型、競合単一値を拒否する
func TestBuildSearchResult_RejectsInvalidBindings(t *testing.T) {
	tests := []struct {
		name     string
		bindings []map[string]sparqlValue
	}{
		{name: "missing resource", bindings: []map[string]sparqlValue{{"title": {Type: "literal", Value: "作品"}}}},
		{name: "invalid resource type", bindings: []map[string]sparqlValue{{
			"resource": {Type: "literal", Value: resourceURI("M1")},
		}}},
		{name: "invalid known type", bindings: []map[string]sparqlValue{{
			"resource": {Type: "uri", Value: resourceURI("M1")},
			"title":    {Type: "uri", Value: "https://example.test"},
		}}},
		{name: "conflicting scalar", bindings: []map[string]sparqlValue{
			testBinding("M1", map[string]string{"volumeNumber": "1"}),
			testBinding("M1", map[string]string{"volumeNumber": "2"}),
		}},
		{name: "invalid series resource type", bindings: []map[string]sparqlValue{{
			"resource":       {Type: "uri", Value: resourceURI("M1")},
			"seriesResource": {Type: "literal", Value: seriesResourceURI("C1")},
		}}},
		{name: "invalid series resource URI", bindings: []map[string]sparqlValue{{
			"resource":       {Type: "uri", Value: resourceURI("M1")},
			"seriesResource": {Type: "uri", Value: resourceURI("M2")},
		}}},
		{name: "conflicting series references", bindings: []map[string]sparqlValue{
			seriesBinding("M1", "C1", nil),
			seriesBinding("M1", "C2", nil),
		}},
		{name: "conflicting series IDs", bindings: []map[string]sparqlValue{
			seriesBinding("M1", "C1", map[string]string{"seriesID": "C1"}),
			seriesBinding("M1", "C1", map[string]string{"seriesID": "C2"}),
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := newSPARQLResponse(test.bindings...)
			_, err := buildSearchResult(response, "作品", 20)
			assertErrorKind(t, err, ErrorKindInvalidResponse)
		})
	}
}

// seriesBinding は、参照先シリーズを含むテスト用bindingを生成する
func seriesBinding(bookID, seriesID string, fields map[string]string) map[string]sparqlValue {
	binding := testBinding(bookID, fields)
	binding["seriesResource"] = sparqlValue{
		Type:  "uri",
		Value: seriesResourceURI(seriesID),
	}
	return binding
}

// seriesResourceURI は、シリーズIDからテスト用リソースURIを生成する
func seriesResourceURI(id string) string {
	return "https://mediaarts-db.artmuseums.go.jp/id/" + id
}

// TestBuildSearchResult_IgnoresUnknownVariables は、未知変数を無視することを検証する
func TestBuildSearchResult_IgnoresUnknownVariables(t *testing.T) {
	binding := testBinding("M1", map[string]string{"title": "作品"})
	binding["futureField"] = sparqlValue{Type: "uri", Value: "https://example.test/value"}
	result, err := buildSearchResult(newSPARQLResponse(binding), "作品", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("len(Books) = %d, want 1", len(result.Books))
	}
}

// TestISBNValidation は、ISBN-10とISBN-13のチェックディジットを検証する
func TestISBNValidation(t *testing.T) {
	tests := []struct {
		value  string
		isbn10 bool
		isbn13 bool
	}{
		{value: "080442957X", isbn10: true},
		{value: "080442957x"},
		{value: "0804429570"},
		{value: "9780306406157", isbn13: true},
		{value: "9780306406150"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if got := isValidISBN10(test.value); got != test.isbn10 {
				t.Fatalf("isValidISBN10() = %t, want %t", got, test.isbn10)
			}
			if got := isValidISBN13(test.value); got != test.isbn13 {
				t.Fatalf("isValidISBN13() = %t, want %t", got, test.isbn13)
			}
		})
	}
}

// assertStrings は、文字列スライスの内容と順序を検証する
func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("values = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("values = %#v, want %#v", got, want)
		}
	}
}
