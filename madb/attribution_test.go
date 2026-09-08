package madb

import (
	"reflect"
	"testing"
)

// TestAttributionsReturnsStableIndependentValues は、MADBの出典情報と返却sliceの独立性を検証する
func TestAttributionsReturnsStableIndependentValues(t *testing.T) {
	want := []Attribution{{
		Source:          SourceMADB,
		Scope:           AttributionScopeData,
		Text:            `独立行政法人国立美術館国立アートリサーチセンター「メディア芸術データベース」のデータをもとに、mankenで共通書誌形式へ変換して作成`,
		URL:             "https://mediaarts-db.artmuseums.go.jp/",
		RequirementsURL: "https://mediaarts-db.artmuseums.go.jp/user_terms",
	}}

	got := Attributions()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Attributions() = %#v, want %#v", got, want)
	}
	got[0].Text = "changed"
	if next := Attributions(); !reflect.DeepEqual(next, want) {
		t.Fatalf("Attributions() after mutation = %#v, want %#v", next, want)
	}
}

// TestBuildResultsIncludeAttributions は、MADBの成功した検索・ISBN参照結果へ出典情報を同梱することを検証する
func TestBuildResultsIncludeAttributions(t *testing.T) {
	response := sparqlResponse{Results: &sparqlResults{Bindings: []map[string]sparqlValue{}}}

	searchResult, err := buildSearchResult(response, searchConditions{Title: "作品"}, 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if !reflect.DeepEqual(searchResult.Attributions, Attributions()) {
		t.Fatalf("search attributions = %#v", searchResult.Attributions)
	}

	lookupResult, err := buildISBNLookupResult(response, nil)
	if err != nil {
		t.Fatalf("buildISBNLookupResult() error = %v", err)
	}
	if !reflect.DeepEqual(lookupResult.Attributions, Attributions()) {
		t.Fatalf("lookup attributions = %#v", lookupResult.Attributions)
	}
}
