//go:build integration

package madb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eamat-dot/manken/madb"
)

// TestIntegration_SearchBooksJapaneseTitle は、日本語タイトルを実サービスで検索する
func TestIntegration_SearchBooksJapaneseTitle(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "動物のおしゃべり",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(result.Books) == 0 {
		t.Fatal("SearchBooks() returned no books")
	}
}

// TestIntegration_SearchBooksEmpty は、存在しないタイトルを空結果として取得する
func TestIntegration_SearchBooksEmpty(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	title := fmt.Sprintf("manken-no-result-%d", time.Now().UnixNano())
	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{Title: title})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(result.Books) != 0 || result.NextCursor != "" {
		t.Fatalf("result = %#v, want empty", result)
	}
}

// TestIntegration_SearchBooksPagination は、実サービスの複数ページが重複しないことを確認する
func TestIntegration_SearchBooksPagination(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	first, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "動物のおしゃべり",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("first SearchBooks() error = %v", err)
	}
	if len(first.Books) != 1 || first.NextCursor == "" {
		t.Fatalf("first result = %#v, want one book and cursor", first)
	}

	second, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title:  "動物のおしゃべり",
		Limit:  1,
		Cursor: first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second SearchBooks() error = %v", err)
	}
	if len(second.Books) != 1 {
		t.Fatalf(
			"first SourceURL = %q, cursor = %q; second result = %#v, want one book",
			first.Books[0].SourceURL,
			first.NextCursor,
			second,
		)
	}
	if first.Books[0].SourceURL >= second.Books[0].SourceURL {
		t.Fatalf("pages are not strictly ordered: %q then %q",
			first.Books[0].SourceURL,
			second.Books[0].SourceURL,
		)
	}
}

// TestIntegration_SearchBooksEditionAndSeries は、版表示とシリーズ参照を実サービスで確認する
func TestIntegration_SearchBooksEditionAndSeries(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "プロジェクトX挑戦者たち",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}

	book := findBookByID(result.Books, "M335551")
	if book == nil {
		t.Fatal("M335551 was not found")
	}
	assertIntegrationStrings(t, book.EditionStatements, []string{"コミック版", "第2版"})
	if book.SeriesID != "C323814" ||
		book.SeriesURL != "https://mediaarts-db.artmuseums.go.jp/id/C323814" {
		t.Fatalf("series fields = %#v", book)
	}
}

// TestIntegration_SearchBooksDoesNotMixSeriesBrand は、シリーズのレーベルを単行本へ混入しないことを確認する
func TestIntegration_SearchBooksDoesNotMixSeriesBrand(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "スリーＺメン",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}

	book := findBookByID(result.Books, "M1079781")
	if book == nil {
		t.Fatal("M1079781 was not found")
	}
	assertIntegrationStrings(t, book.Imprints, []string{"藤子不二雄全集"})
	if book.SeriesID != "C306332" {
		t.Fatalf("SeriesID = %q, want C306332", book.SeriesID)
	}
}

// TestIntegration_SearchBooksResponseSize は、100冊取得時の応答が本文上限内か確認する
func TestIntegration_SearchBooksResponseSize(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, rawResponse, err := client.SearchBooksWithRawResponse(ctx, madb.SearchBooksRequest{
		Title: "動物のおしゃべり",
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("SearchBooksWithRawResponse() error = %v", err)
	}
	if len(result.Books) == 0 {
		t.Fatal("SearchBooksWithRawResponse() returned no books")
	}
	if len(rawResponse) > 4<<20 {
		t.Fatalf("raw response size = %d, want at most 4 MiB", len(rawResponse))
	}
}

// findBookByID は、指定したMADB IDの本を検索結果から探す
func findBookByID(books []madb.Book, id string) *madb.Book {
	for index := range books {
		if books[index].ID == id {
			return &books[index]
		}
	}
	return nil
}

// assertIntegrationStrings は、実サービスの文字列スライスを順序込みで検証する
func assertIntegrationStrings(t *testing.T, got, want []string) {
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

// newIntegrationClient は、実サービスを使用するClientを生成する
func newIntegrationClient(t *testing.T) *madb.Client {
	t.Helper()
	client, err := madb.NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}
