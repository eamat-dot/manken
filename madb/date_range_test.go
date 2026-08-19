package madb

import (
	"strings"
	"testing"
)

// TestBuildSearchQuery_DateRange は、日付条件を候補選択サブクエリ内でAND結合することを検証する
func TestBuildSearchQuery_DateRange(t *testing.T) {
	query := buildSearchQuery(searchConditions{DateFrom: "2024-01", DateTo: "2024-04", DatePrecision: 7}, 20, "")
	for _, want := range []string{
		"?resource schema:datePublished ?searchPublishedDate .",
		"FILTER (STRLEN(STR(?searchPublishedDate)) >= 7)",
		`FILTER (STR(?searchPublishedDate) >= "2024-01")`,
		`FILTER (STR(?searchPublishedDate) < "2024-04")`,
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query does not contain %q:\n%s", want, query)
		}
	}
}

// TestValidateSearchRequest_DateRange は、日付だけの検索とCursor拘束を検証する
func TestValidateSearchRequest_DateRange(t *testing.T) {
	conditions, limit, _, err := validateSearchRequest(SearchRequest{DateFrom: "2024-01", DateTo: "2024-03"})
	if err != nil || limit != defaultLimit || conditions.DateFrom != "2024-01" || conditions.DateTo != "2024-04" || conditions.DatePrecision != 7 {
		t.Fatalf("conditions = %#v, limit = %d, err = %v", conditions, limit, err)
	}
	cursor, err := encodeCursor("https://mediaarts-db.artmuseums.go.jp/id/M1", conditions, limit)
	if err != nil {
		t.Fatal(err)
	}
	conditions.DateTo = "2024-05"
	if _, err := decodeCursor(cursor, conditions, limit); err == nil {
		t.Fatal("cursor for a different date range was accepted")
	}
}
