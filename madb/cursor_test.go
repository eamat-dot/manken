package madb

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestCursor_RoundTrip は、カーソルの生成と復元を検証する
func TestCursor_RoundTrip(t *testing.T) {
	after := resourceURI("M123")
	encoded, err := encodeCursor(after, "作品", 20)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	if strings.Contains(encoded, "=") {
		t.Fatalf("cursor contains padding: %q", encoded)
	}

	payload, err := decodeCursor(encoded, "作品", 20)
	if err != nil {
		t.Fatalf("decodeCursor() error = %v", err)
	}
	if payload.After != after || payload.Version != cursorVersion {
		t.Fatalf("payload = %#v", payload)
	}
}

// TestCursor_RejectsInvalidValues は、不正または検索条件と一致しないカーソルを拒否する
func TestCursor_RejectsInvalidValues(t *testing.T) {
	valid, err := encodeCursor(resourceURI("M123"), "作品", 20)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	unsupportedVersion := encodeRawCursor(`{"v":2,"after":"` + resourceURI("M123") + `","limit":20,"title_sha256":"` + hashTitle("作品") + `"}`)
	invalidURI := encodeRawCursor(`{"v":1,"after":"https://example.test/M123","limit":20,"title_sha256":"` + hashTitle("作品") + `"}`)
	unknownField := encodeRawCursor(`{"v":1,"after":"` + resourceURI("M123") + `","limit":20,"title_sha256":"` + hashTitle("作品") + `","extra":true}`)
	trailing := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1} {}`))

	tests := []struct {
		name   string
		cursor string
		title  string
		limit  int
	}{
		{name: "invalid base64", cursor: "***", title: "作品", limit: 20},
		{name: "invalid JSON", cursor: encodeRawCursor("{"), title: "作品", limit: 20},
		{name: "unsupported version", cursor: unsupportedVersion, title: "作品", limit: 20},
		{name: "invalid URI", cursor: invalidURI, title: "作品", limit: 20},
		{name: "unknown field", cursor: unknownField, title: "作品", limit: 20},
		{name: "trailing data", cursor: trailing, title: "作品", limit: 20},
		{name: "different title", cursor: valid, title: "別作品", limit: 20},
		{name: "different limit", cursor: valid, title: "作品", limit: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeCursor(test.cursor, test.title, test.limit); err == nil {
				t.Fatal("decodeCursor() error = nil")
			}
		})
	}
}

// TestCursor_DefaultAndExplicitLimitMatch は、Limit 0と既定値20が同じカーソル条件になることを検証する
func TestCursor_DefaultAndExplicitLimitMatch(t *testing.T) {
	cursor, err := encodeCursor(resourceURI("M1"), "作品", defaultLimit)
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

// mankenRequest は、カーソル検証用の検索条件を生成する
func mankenRequest(title string, limit int, cursor string) SearchBooksRequest {
	return SearchBooksRequest{Title: title, Limit: limit, Cursor: cursor}
}
