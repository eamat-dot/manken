package madb

import (
	"strings"
	"testing"
)

// TestBuildSearchQuery_Structure は、ページ対象を先に確定するクエリ構造を検証する
func TestBuildSearchQuery_Structure(t *testing.T) {
	query := buildSearchQuery("作品", 20, resourceURI("M123"))
	required := []string{
		`neptune-fts:field schema:name`,
		`neptune-fts:queryType "simple_query_string"`,
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

// TestBuildSearchQuery_EscapesTwoLayers は、全文検索とSPARQLの二段階エスケープを検証する
func TestBuildSearchQuery_EscapesTwoLayers(t *testing.T) {
	title := " a\"b\\c\n+\t- | "
	query := buildSearchQuery(title, 1, "")
	fullText := `"` + escapeFullText(title) + `"`
	want := `neptune-fts:config neptune-fts:query "` + escapeSPARQLString(fullText) + `" .`
	if !strings.Contains(query, want) {
		t.Fatalf("query does not contain escaped value %q:\n%s", want, query)
	}
	if strings.Contains(query, "\n+\t- |") {
		t.Fatalf("query contains unescaped control characters:\n%s", query)
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
