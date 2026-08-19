//go:build integration

package ndl_test

import (
	"context"
	"testing"
	"time"

	"github.com/eamat-dot/manken/ndl"
)

// TestIntegrationNDL は、実サービスの代表検索とISBN参照を最小限確認する
func TestIntegrationNDL(t *testing.T) {
	client, err := ndl.NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, raw, err := client.SearchBooksWithOptionsAndRawResponse(ctx, ndl.SearchRequest{Title: "動物のお医者さん", Author: "佐々木倫子", Limit: 1}, ndl.SearchOptions{Description: "ハムテル"})
	if err != nil || len(result.Books) == 0 || len(raw) == 0 {
		t.Fatalf("SearchBooksWithOptionsAndRawResponse() result = %#v, raw = %d, error = %v", result, len(raw), err)
	}
	lookup, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{"9784098627417"})
	if err != nil || len(lookup.Items) != 1 || len(lookup.Items[0].Books) == 0 || len(raw) == 0 {
		t.Fatalf("LookupBooksByISBNWithRawResponse() result = %#v, raw = %d, error = %v", lookup, len(raw), err)
	}
	book := lookup.Items[0].Books[0]
	if len(book.Sources) == 0 || book.Sources[0].Source != ndl.SourceNDL || book.Sources[0].ID != "033811835" {
		t.Errorf("Sources = %#v", book.Sources)
	}
	if book.Title != "動物のお医者さん" {
		t.Errorf("Title = %q", book.Title)
	}
	if !containsString(book.Authors, "佐々木倫子") {
		t.Errorf("Authors = %#v", book.Authors)
	}
	if !hasContributorRole(book.Contributors, "佐々木倫子", "著者") {
		t.Errorf("Contributors = %#v", book.Contributors)
	}
	if !containsString(book.Publishers, "小学館") {
		t.Errorf("Publishers = %#v", book.Publishers)
	}
	if book.Volume.Label != "11" {
		t.Errorf("Volume = %#v", book.Volume)
	}
	if !containsString(book.ISBN13, "9784098627417") {
		t.Errorf("ISBN13 = %#v", book.ISBN13)
	}
	if book.Medium != ndl.PublicationMediumPrint {
		t.Errorf("Medium = %q", book.Medium)
	}
	if !hasSubject(book.Subjects, "NDLC", "Y84", "") || !hasSubject(book.Subjects, "NDC", "726.1", "") || !hasSubject(book.Subjects, "NDLGFT", "001347325", "漫画") {
		t.Errorf("Subjects = %#v", book.Subjects)
	}
}

// containsString は、文字列配列に指定値が含まれるか判定する
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// hasContributorRole は、指定した表示名の寄与者に役割が含まれるか判定する
func hasContributorRole(values []ndl.Contributor, name, want string) bool {
	for _, value := range values {
		if value.Name != name {
			continue
		}
		for _, role := range value.Roles {
			if role == want {
				return true
			}
		}
	}
	return false
}

// hasSubject は、主題配列に指定値が含まれるか判定する
func hasSubject(values []ndl.Subject, scheme, code, name string) bool {
	for _, value := range values {
		if value.Scheme == scheme && value.Code == code && value.Name == name {
			return true
		}
	}
	return false
}
