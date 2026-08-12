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
	result, raw, err := client.SearchBooksWithOptionsAndRawResponse(ctx, ndl.SearchBooksRequest{Title: "動物のお医者さん", Author: "佐々木倫子", Limit: 1}, ndl.SearchOptions{Description: "ハムテル"})
	if err != nil || len(result.Books) == 0 || len(raw) == 0 {
		t.Fatalf("SearchBooksWithOptionsAndRawResponse() result = %#v, raw = %d, error = %v", result, len(raw), err)
	}
	lookup, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{"9784098627417"})
	if err != nil || len(lookup.Items) != 1 || len(lookup.Items[0].Books) == 0 || len(raw) == 0 {
		t.Fatalf("LookupBooksByISBNWithRawResponse() result = %#v, raw = %d, error = %v", lookup, len(raw), err)
	}
	book := lookup.Items[0].Books[0]
	if len(book.Sources) == 0 || book.Sources[0].Source != ndl.SourceNDL || book.Sources[0].ID != "033811835" || book.Normalized.Title != "動物のお医者さん" || !containsString(book.Normalized.Authors, "佐々木倫子") || !hasContributorRole(book.Normalized.Contributors, "佐々木倫子", ndl.ContributorRoleAuthor) || !containsString(book.Normalized.Publishers, "小学館") || book.Normalized.Volume.Label != "11" || !containsIdentifier(book.Normalized.Identifiers, "9784098627417") || book.Normalized.Medium != ndl.PublicationMediumPrint || !hasSubject(book.Normalized.Subjects, "NDLC", "Y84", "") || !hasSubject(book.Normalized.Subjects, "NDC", "726.1", "") || !hasSubject(book.Normalized.Subjects, "NDLGFT", "001347325", "漫画") {
		t.Fatalf("LookupBooksByISBNWithRawResponse() book = %#v", book)
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

// containsIdentifier は、識別子配列に指定値が含まれるか判定する
func containsIdentifier(values []ndl.Identifier, want string) bool {
	for _, value := range values {
		if value.Value == want {
			return true
		}
	}
	return false
}

// hasContributorRole は、指定した表示名の寄与者に役割が含まれるか判定する
func hasContributorRole(values []ndl.Contributor, name string, want ndl.ContributorRole) bool {
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
