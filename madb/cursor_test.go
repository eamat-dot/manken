package madb

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestCursor_RoundTrip は、カーソルの生成と復元を検証する
func TestCursor_RoundTrip(t *testing.T) {
	after := resourceURI("M123")
	conditions := searchConditions{
		Title: "作品", ISBNs: []string{"4088466365", "9784088466361"},
		Author: "著者", FreeText: "新装版", ExcludedText: "復刻版",
	}
	encoded, err := encodeCursor(after, conditions, 20)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	if strings.Contains(encoded, "=") {
		t.Fatalf("cursor contains padding: %q", encoded)
	}

	payload, err := decodeCursor(encoded, conditions, 20)
	if err != nil {
		t.Fatalf("decodeCursor() error = %v", err)
	}
	if payload.After != after || payload.Version != cursorVersion {
		t.Fatalf("payload = %#v", payload)
	}
}

// TestCursor_RejectsInvalidValues は、不正または検索条件と一致しないカーソルを拒否する
func TestCursor_RejectsInvalidValues(t *testing.T) {
	conditions := searchConditions{
		Title: "作品", ISBNs: []string{"4088466365", "9784088466361"},
		Author: "著者", FreeText: "新装版", ExcludedText: "復刻版",
	}
	valid, err := encodeCursor(resourceURI("M123"), conditions, 20)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	searchHash := mustHashSearchConditions(t, conditions)
	unsupportedVersion := encodeRawCursor(`{"v":1,"after":"` + resourceURI("M123") + `","limit":20,"search_sha256":"` + searchHash + `"}`)
	invalidURI := encodeRawCursor(`{"v":2,"after":"https://example.test/M123","limit":20,"search_sha256":"` + searchHash + `"}`)
	unknownField := encodeRawCursor(`{"v":2,"after":"` + resourceURI("M123") + `","limit":20,"search_sha256":"` + searchHash + `","extra":true}`)
	trailing := base64.RawURLEncoding.EncodeToString([]byte(`{"v":2} {}`))

	tests := []struct {
		name       string
		cursor     string
		conditions searchConditions
		limit      int
	}{
		{name: "invalid base64", cursor: "***", conditions: conditions, limit: 20},
		{name: "invalid JSON", cursor: encodeRawCursor("{"), conditions: conditions, limit: 20},
		{name: "unsupported version", cursor: unsupportedVersion, conditions: conditions, limit: 20},
		{name: "invalid URI", cursor: invalidURI, conditions: conditions, limit: 20},
		{name: "unknown field", cursor: unknownField, conditions: conditions, limit: 20},
		{name: "trailing data", cursor: trailing, conditions: conditions, limit: 20},
		{name: "different title", cursor: valid, conditions: searchConditions{Title: "別作品", ISBNs: conditions.ISBNs, Author: conditions.Author, FreeText: conditions.FreeText}, limit: 20},
		{name: "different ISBN", cursor: valid, conditions: searchConditions{Title: "作品", ISBNs: []string{"9791234567896"}, Author: conditions.Author, FreeText: conditions.FreeText}, limit: 20},
		{name: "different author", cursor: valid, conditions: searchConditions{Title: "作品", ISBNs: conditions.ISBNs, Author: "別著者", FreeText: conditions.FreeText}, limit: 20},
		{name: "different free text", cursor: valid, conditions: searchConditions{Title: "作品", ISBNs: conditions.ISBNs, Author: conditions.Author, FreeText: "増補版"}, limit: 20},
		{name: "different excluded text", cursor: valid, conditions: searchConditions{Title: "作品", ISBNs: conditions.ISBNs, Author: conditions.Author, FreeText: conditions.FreeText, ExcludedText: "愛蔵版"}, limit: 20},
		{name: "different limit", cursor: valid, conditions: conditions, limit: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeCursor(test.cursor, test.conditions, test.limit); err == nil {
				t.Fatal("decodeCursor() error = nil")
			}
		})
	}
}

// TestCursor_DefaultAndExplicitLimitMatch は、Limit 0と既定値20が同じカーソル条件になることを検証する
func TestCursor_DefaultAndExplicitLimitMatch(t *testing.T) {
	cursor, err := encodeCursor(resourceURI("M1"), searchConditions{Title: "作品"}, defaultLimit)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	_, limit, _, err := validateSearchRequest(mankenRequest("作品", 0, cursor))
	if err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
	if limit != defaultLimit {
		t.Fatalf("limit = %d, want %d", limit, defaultLimit)
	}
}

// encodeRawCursor は、JSON文字列をテスト用カーソルへ変換する
func encodeRawCursor(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

// mustHashSearchConditions は、テスト用検索条件のハッシュを生成する
func mustHashSearchConditions(t *testing.T, conditions searchConditions) string {
	t.Helper()
	value, err := hashSearchConditions(conditions)
	if err != nil {
		t.Fatalf("hashSearchConditions() error = %v", err)
	}
	return value
}

// mankenRequest は、カーソル検証用の検索条件を生成する
func mankenRequest(title string, limit int, cursor string) SearchBooksRequest {
	return SearchBooksRequest{Title: title, Limit: limit, Cursor: cursor}
}
