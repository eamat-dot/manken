package madb

import (
	"strconv"
	"strings"
	"testing"
)

// TestBuildSearchQuery_Structure は、ページ対象を先に確定するクエリ構造を検証する
func TestBuildSearchQuery_Structure(t *testing.T) {
	query := buildSearchQuery(searchConditions{Title: "作品"}, 20, resourceURI("M123"))
	required := []string{
		`neptune-fts:field schema:name`,
		`neptune-fts:queryType "query_string"`,
		`neptune-fts:sortBy 'Neptune#fts.entity_id'`,
		`neptune-fts:sortOrder 'ASC'`,
		`?resource rdf:type class:MangaBook`,
		`FILTER (STR(?resource) > "` + resourceURI("M123") + `")`,
		`ORDER BY ?resource`,
		`LIMIT 21`,
		`OPTIONAL { ?resource schema:name ?title . FILTER (LANG(?title) = "") }`,
		`OPTIONAL { ?resource schema:name ?titleKana . FILTER (LANG(?titleKana) = "ja-hrkt") }`,
		`OPTIONAL { ?resource schema:numberOfPages ?pageCount . }`,
		`OPTIONAL { ?resource schema:size ?size . }`,
		`OPTIONAL { ?resource schema:version ?version . FILTER (LANG(?version) = "") }`,
		`OPTIONAL { ?resource schema:brand ?brand . FILTER (LANG(?brand) = "") }`,
		`?resource schema:isPartOf ?seriesResource`,
		`?seriesResource rdf:type class:MangaBookSeries`,
		`?seriesResource schema:name ?relatedSeriesName`,
		`FILTER (LANG(?relatedSeriesName) = "")`,
		`?seriesResource schema:identifier ?seriesID`,
	}
	for _, fragment := range required {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query does not contain %q:\n%s", fragment, query)
		}
	}
	for _, fragment := range []string{
		`?seriesResource schema:brand`,
		`?seriesResource schema:version`,
	} {
		if strings.Contains(query, fragment) {
			t.Fatalf("query unexpectedly contains %q:\n%s", fragment, query)
		}
	}
}

// TestBuildFullTextQuery は、Unicode空白で分けた検索語のAND結合を検証する
func TestBuildFullTextQuery(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{name: "single term", title: "作品", want: `"作品"`},
		{name: "ASCII spaces", title: "  うる星   復刻box  ", want: `"うる星" AND "復刻box"`},
		{name: "Unicode spaces", title: "\tうる星\u3000復刻box\n", want: `"うる星" AND "復刻box"`},
		{
			name:  "operators and escapes",
			title: "a\"b\\c AND OR NOT + -",
			want:  `"a\"b\\c" AND "AND" AND "OR" AND "NOT" AND "+" AND "-"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := buildFullTextQuery(test.title); got != test.want {
				t.Fatalf("buildFullTextQuery() = %q, want %q", got, test.want)
			}
		})
	}
}

// TestEscapeFullText_QuoteAndBackslash は、実際の二重引用符とバックスラッシュを引用句用にエスケープすることを検証する
func TestEscapeFullText_QuoteAndBackslash(t *testing.T) {
	input := "a\"b\\c"
	if got, want := escapeFullText(input), `a\"b\\c`; got != want {
		t.Fatalf("escapeFullText() = %q, want %q", got, want)
	}
}

// TestBuildFullTextQueryWithExclusions は、複数の除外語を安全なAND NOT式へ変換する
func TestBuildFullTextQueryWithExclusions(t *testing.T) {
	value := "うる星 a\"b\\c"
	excluded := "復刻box NOT a\"b\\c"
	want := `"うる星" AND "a\"b\\c" AND NOT "復刻box" AND NOT "NOT" AND NOT "a\"b\\c"`
	if got := buildFullTextQueryWithExclusions(value, excluded); got != want {
		t.Fatalf("buildFullTextQueryWithExclusions() = %q, want %q", got, want)
	}
}

// TestBuildExcludedFullTextQuery は、複数の除外語を安全なOR式へ変換する
func TestBuildExcludedFullTextQuery(t *testing.T) {
	value := "復刻box NOT AND a\"b\\c"
	want := `"復刻box" OR "NOT" OR "AND" OR "a\"b\\c"`
	if got := buildExcludedFullTextQuery(value); got != want {
		t.Fatalf("buildExcludedFullTextQuery() = %q, want %q", got, want)
	}
}

// TestBuildSearchQuery_EscapesTwoLayers は、全文検索とSPARQLの二段階エスケープを検証する
func TestBuildSearchQuery_EscapesTwoLayers(t *testing.T) {
	title := "a\"b\\c AND OR NOT + -"
	query := buildSearchQuery(searchConditions{Title: title}, 1, "")
	fullText := `"a\"b\\c" AND "AND" AND "OR" AND "NOT" AND "+" AND "-"`
	if got := buildFullTextQuery(title); got != fullText {
		t.Fatalf("buildFullTextQuery() = %q, want %q", got, fullText)
	}
	want := `neptune-fts:config neptune-fts:query "\"a\\\"b\\\\c\" AND \"AND\" AND \"OR\" AND \"NOT\" AND \"+\" AND \"-\"" .`
	if !strings.Contains(query, want) {
		t.Fatalf("query does not contain escaped value %q:\n%s", want, query)
	}
}

// TestBuildSearchQuery_AuthorConditions は、creator文字列とAgent参照をUNIONして他条件とAND結合する
func TestBuildSearchQuery_AuthorConditions(t *testing.T) {
	author := "KotzDean AND a\"b\\c"
	query := buildSearchQuery(searchConditions{
		Title:  "作品",
		Author: author,
	}, 20, "")

	fullText := escapeSPARQLString(buildFullTextQuery(author))
	required := []string{
		`SELECT DISTINCT ?resource`,
		`neptune-fts:field schema:creator`,
		`neptune-fts:return ?resource`,
		`neptune-fts:field rdfs:label`,
		`neptune-fts:return ?searchAgent`,
		`?resource dcterms:creator ?searchAgent`,
		`UNION`,
		`neptune-fts:config neptune-fts:query "` + fullText + `"`,
		`neptune-fts:field schema:name`,
	}
	for _, fragment := range required {
		if !strings.Contains(query, fragment) {
			t.Fatalf("author query does not contain %q:\n%s", fragment, query)
		}
	}
	if strings.Count(query, `neptune-fts:field schema:creator`) != 1 ||
		strings.Count(query, `neptune-fts:field rdfs:label`) != 1 {
		t.Fatalf("author query does not contain exactly two search paths:\n%s", query)
	}
}

// TestBuildSearchQuery_FreeTextConditions は、フリーワードの対象フィールドと他条件とのAND結合を検証する
func TestBuildSearchQuery_FreeTextConditions(t *testing.T) {
	freeText := "うる星 高橋留美子 新装版 a\"b\\c"
	query := buildSearchQuery(searchConditions{FreeText: freeText}, 20, "")

	wantQuery := `neptune-fts:config neptune-fts:query "` +
		escapeSPARQLString(buildFullTextQuery(freeText)) + `" .`
	if !strings.Contains(query, wantQuery) {
		t.Fatalf("query does not contain escaped free-text value %q:\n%s", wantQuery, query)
	}
	for _, field := range freeTextSearchFields() {
		pattern := `neptune-fts:config neptune-fts:field ` + field
		if strings.Count(query, pattern) != 1 {
			t.Fatalf("free-text field %q count = %d, want 1:\n%s", field, strings.Count(query, pattern), query)
		}
	}

	combined := buildSearchQuery(searchConditions{
		Title:    "うる星",
		Author:   "高橋留美子",
		FreeText: freeText,
	}, 20, "")
	if strings.Count(combined, `SERVICE neptune-fts:search`) != 4 {
		t.Fatalf("full-text service count = %d, want 4:\n%s", strings.Count(combined, `SERVICE neptune-fts:search`), combined)
	}
}

// TestBuildSearchQuery_PublisherCondition は、出版社のRDF値フィルタを他の正条件とAND結合することを検証する
func TestBuildSearchQuery_PublisherCondition(t *testing.T) {
	publisher := "白泉社\u3000a\"b\\\\c"
	query := buildSearchQuery(searchConditions{Publisher: publisher}, 20, "")
	if strings.Contains(query, `neptune-fts:config neptune-fts:field schema:publisher`) ||
		strings.Contains(query, `SERVICE neptune-fts:search`) {
		t.Fatalf("publisher query =\n%s", query)
	}
	terms := strings.Fields(publisher)
	if strings.Count(query, `FILTER EXISTS {`) != len(terms) {
		t.Fatalf("publisher filter count = %d, want %d:\n%s", strings.Count(query, `FILTER EXISTS {`), len(terms), query)
	}
	for index, term := range terms {
		want := `?resource schema:publisher ?publisherValue` + strconv.Itoa(index) + ` .`
		if !strings.Contains(query, want) {
			t.Fatalf("publisher query does not contain %q:\n%s", want, query)
		}
		want = `LCASE("` + escapeSPARQLString(term) + `")`
		if !strings.Contains(query, want) {
			t.Fatalf("publisher query does not contain escaped term %q:\n%s", want, query)
		}
	}
	combined := buildSearchQuery(searchConditions{Title: "作品", Author: "著者", Publisher: "白泉社"}, 20, "")
	if strings.Count(combined, `SERVICE neptune-fts:search`) != 3 {
		t.Fatalf("full-text service count = %d, want 3:\n%s", strings.Count(combined, `SERVICE neptune-fts:search`), combined)
	}
	if strings.Count(combined, `FILTER EXISTS {`) != 1 {
		t.Fatalf("publisher filter count = %d, want 1:\n%s", strings.Count(combined, `FILTER EXISTS {`), combined)
	}
}

// TestBuildSearchQuery_FullTextServicesUseEntityIDSort は、すべてのFTS SERVICEがentity ID昇順を指定することを検証する
func TestBuildSearchQuery_FullTextServicesUseEntityIDSort(t *testing.T) {
	tests := []struct {
		name         string
		conditions   searchConditions
		wantServices int
	}{
		{
			name:         "title",
			conditions:   searchConditions{Title: "動物のお医者さん"},
			wantServices: 1,
		},
		{
			name:         "author creator and agent",
			conditions:   searchConditions{Author: "佐々木倫子"},
			wantServices: 2,
		},
		{
			name:         "free text",
			conditions:   searchConditions{FreeText: "動物のお医者さん"},
			wantServices: 1,
		},
		{
			name:         "excluded text without free text",
			conditions:   searchConditions{ExcludedText: "愛蔵版"},
			wantServices: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := buildSearchQuery(test.conditions, 20, "")
			if got := strings.Count(query, `SERVICE neptune-fts:search`); got != test.wantServices {
				t.Fatalf("full-text service count = %d, want %d:\n%s", got, test.wantServices, query)
			}
			if got := strings.Count(query, `neptune-fts:sortBy 'Neptune#fts.entity_id' .`); got != test.wantServices {
				t.Fatalf("sortBy count = %d, want %d:\n%s", got, test.wantServices, query)
			}
			if got := strings.Count(query, `neptune-fts:sortOrder 'ASC' .`); got != test.wantServices {
				t.Fatalf("sortOrder count = %d, want %d:\n%s", got, test.wantServices, query)
			}
		})
	}
}

// TestBuildSearchQuery_DoesNotSetFullTextBatchSize は、Publisherの有無にかかわらずFTSのbatchSizeとmaxResultsを設定しないことを検証する
func TestBuildSearchQuery_DoesNotSetFullTextBatchSize(t *testing.T) {
	tests := []struct {
		name       string
		conditions searchConditions
	}{
		{
			name:       "publisher and title",
			conditions: searchConditions{Publisher: "小学館", Title: "動物のお医者さん"},
		},
		{
			name:       "publisher and author",
			conditions: searchConditions{Publisher: "小学館", Author: "佐々木倫子"},
		},
		{
			name:       "publisher and free text",
			conditions: searchConditions{Publisher: "小学館", FreeText: "動物のお医者さん"},
		},
		{
			name:       "publisher and excluded text",
			conditions: searchConditions{Publisher: "小学館", ExcludedText: "愛蔵版"},
		},
		{
			name:       "publisher only",
			conditions: searchConditions{Publisher: "小学館"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := buildSearchQuery(test.conditions, 20, "")
			if strings.Contains(query, `neptune-fts:config neptune-fts:batchSize`) {
				t.Fatalf("query unexpectedly sets batchSize:\n%s", query)
			}
			if strings.Contains(query, `neptune-fts:config neptune-fts:maxResults`) {
				t.Fatalf("query unexpectedly sets maxResults:\n%s", query)
			}
		})
	}
}

// TestBuildSearchQuery_FreeTextDoesNotUseImplicitFields は、対象省略やワイルドカードへ戻さないことを検証する
func TestBuildSearchQuery_FreeTextDoesNotUseImplicitFields(t *testing.T) {
	query := buildSearchQuery(searchConditions{FreeText: "作品"}, 20, "")
	fields := freeTextSearchFields()
	if strings.Contains(query, `neptune-fts:field "*"`) ||
		strings.Count(query, `neptune-fts:field`) != len(fields) {
		t.Fatalf("free-text query does not explicitly contain all fields:\n%s", query)
	}
}

// TestBuildSearchQuery_ExcludedTextWithFreeText は、フリーワード式へ除外語をAND NOTで追加する
func TestBuildSearchQuery_ExcludedTextWithFreeText(t *testing.T) {
	conditions := searchConditions{
		FreeText:     "うる星 a\"b\\c",
		ExcludedText: `復刻box NOT`,
	}
	query := buildSearchQuery(conditions, 20, "")
	wantQuery := `neptune-fts:config neptune-fts:query "` +
		escapeSPARQLString(buildFullTextQueryWithExclusions(
			conditions.FreeText,
			conditions.ExcludedText,
		)) + `" .`
	if !strings.Contains(query, wantQuery) {
		t.Fatalf("query does not contain escaped exclusion expression %q:\n%s", wantQuery, query)
	}
	if strings.Contains(query, "MINUS") {
		t.Fatalf("free-text exclusion unexpectedly uses MINUS:\n%s", query)
	}
	for _, field := range freeTextSearchFields() {
		if strings.Count(query, `neptune-fts:field `+field) != 1 {
			t.Fatalf("field %q is not used exactly once:\n%s", field, query)
		}
	}
}

// TestBuildSearchQuery_ExcludedTextWithoutFreeText は、各正条件へ12項目のMINUS除外を結合する
func TestBuildSearchQuery_ExcludedTextWithoutFreeText(t *testing.T) {
	tests := []struct {
		name       string
		conditions searchConditions
	}{
		{name: "title", conditions: searchConditions{Title: "うる星"}},
		{name: "author", conditions: searchConditions{Author: "高橋留美子"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.conditions.ExcludedText = `復刻box NOT a"b\c`
			query := buildSearchQuery(test.conditions, 20, "")
			wantQuery := `neptune-fts:config neptune-fts:query "` +
				escapeSPARQLString(buildExcludedFullTextQuery(test.conditions.ExcludedText)) + `" .`
			if strings.Count(query, "MINUS {") != 1 || !strings.Contains(query, wantQuery) {
				t.Fatalf("query does not contain one MINUS exclusion %q:\n%s", wantQuery, query)
			}
			for _, field := range freeTextSearchFields() {
				pattern := `neptune-fts:field ` + field
				wantCount := 1
				if test.name == "title" && field == "schema:name" {
					wantCount = 2
				}
				if test.name == "author" && field == "schema:creator" {
					wantCount = 2
				}
				if strings.Count(query, pattern) != wantCount {
					t.Fatalf("field %q count = %d, want %d:\n%s", field, strings.Count(query, pattern), wantCount, query)
				}
			}
			minusStart := strings.Index(query, "MINUS {")
			if minusStart < 0 {
				t.Fatalf("MINUS exclusion unexpectedly targets Agent labels:\n%s", query)
			}
			conditionsEnd := strings.Index(query[minusStart:], "?resource rdf:type class:MangaBook")
			if conditionsEnd < 0 ||
				strings.Contains(query[minusStart:minusStart+conditionsEnd], "rdfs:label") {
				t.Fatalf("MINUS exclusion unexpectedly targets Agent labels:\n%s", query)
			}
		})
	}
}

// TestBuildISBNLookupQuery は、複数ISBN候補の対応付けと書誌取得構造を検証する
func TestBuildISBNLookupQuery(t *testing.T) {
	query := buildISBNLookupQuery([]string{"080442957X", "4088466365", "9784088466361"})
	for _, fragment := range []string{
		`VALUES ?matchedISBN { "080442957X" "4088466365" "9784088466361" }`,
		`SELECT DISTINCT ?resource ?matchedISBN`,
		`?resource schema:isbn ?matchedISBN`,
		`?resource rdf:type class:MangaBook`,
		`OPTIONAL { ?resource schema:isbn ?isbn . }`,
		`ORDER BY ?resource ?matchedISBN`,
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("lookup query does not contain %q:\n%s", fragment, query)
		}
	}
	if strings.Contains(query, "LIMIT") || strings.Contains(query, "neptune-fts:search") {
		t.Fatalf("lookup query unexpectedly contains paging or full-text search:\n%s", query)
	}
}

// TestEscapeSPARQLString_ControlCharacters は、SPARQLで特別な制御文字をエスケープする
func TestEscapeSPARQLString_ControlCharacters(t *testing.T) {
	input := "\"\\\t\n\r\b\f" + string(rune(0x01))
	want := `\"\\\t\n\r\b\f\u0001`
	if got := escapeSPARQLString(input); got != want {
		t.Fatalf("escapeSPARQLString() = %q, want %q", got, want)
	}
}
