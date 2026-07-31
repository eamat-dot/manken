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
			"first source URL = %q, cursor = %q; second result = %#v, want one book",
			first.Books[0].Sources[0].URL,
			first.NextCursor,
			second,
		)
	}
	if first.Books[0].Sources[0].URL >= second.Books[0].Sources[0].URL {
		t.Fatalf("pages are not strictly ordered: %q then %q",
			first.Books[0].Sources[0].URL,
			second.Books[0].Sources[0].URL,
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
	assertIntegrationStrings(t, book.Normalized.EditionStatements, []string{"コミック版", "第2版"})
	series := findSeriesByID(book.Normalized.Series, "C323814")
	if series == nil || series.URL != "https://mediaarts-db.artmuseums.go.jp/id/C323814" {
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
	assertIntegrationStrings(t, book.Normalized.Imprints, []string{"藤子不二雄全集"})
	if findSeriesByID(book.Normalized.Series, "C306332") == nil {
		t.Fatalf("Series = %#v, want C306332", book.Normalized.Series)
	}
}

// TestIntegration_SearchBooksAdditionalFields は、タイトル読み、ページ数、大きさ、出版社正規化を実サービスで確認する
func TestIntegration_SearchBooksAdditionalFields(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "動物のお医者さん",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}

	book := findBookByID(result.Books, "M292141")
	if book == nil {
		t.Fatal("M292141 was not found")
	}
	if book.Normalized.TitleKana != "ドウブツノオイシャサン" {
		t.Fatalf("TitleKana = %q", book.Normalized.TitleKana)
	}
	assertIntegrationStrings(t, book.Normalized.Publishers, []string{"白泉社"})
	if book.Normalized.PageCount == nil || *book.Normalized.PageCount != 197 {
		t.Fatalf("PageCount = %#v", book.Normalized.PageCount)
	}
	if book.Normalized.PhysicalSize == nil ||
		book.Normalized.PhysicalSize.HeightMM == nil ||
		*book.Normalized.PhysicalSize.HeightMM != 173 ||
		book.Normalized.PhysicalSize.WidthMM == nil ||
		*book.Normalized.PhysicalSize.WidthMM != 106 {
		t.Fatalf("PhysicalSize = %#v", book.Normalized.PhysicalSize)
	}
	assertIntegrationStrings(t, book.Sources[0].Values.Publishers, []string{
		"白泉社",
		"白泉社　∥　ハクセンシャ",
	})
}

// TestIntegration_SearchBooksContributorRoles は、著者と解説者の役割分離を実サービスで確認する
func TestIntegration_SearchBooksContributorRoles(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "動物のお医者さん",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}

	book := findBookByID(result.Books, "M292132")
	if book == nil {
		t.Fatal("M292132 was not found")
	}
	assertIntegrationStrings(t, book.Normalized.Authors, []string{"佐々木倫子"})
	assertIntegrationContributors(t, book.Normalized.Contributors, []madb.Contributor{
		{Name: "佐々木倫子", Roles: []madb.ContributorRole{madb.ContributorRoleAuthor}},
		{Name: "藤原新也", Roles: []madb.ContributorRole{madb.ContributorRoleCommentator}},
	})
	assertIntegrationStrings(t, book.Sources[0].Values.Authors, []string{
		"[著]佐々木倫子",
		"[解説]藤原新也",
	})
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
		if len(books[index].Sources) != 0 && books[index].Sources[0].ID == id {
			return &books[index]
		}
	}
	return nil
}

// findSeriesByID は、指定したMADB IDのシリーズを検索結果から探す
func findSeriesByID(series []madb.Series, id string) *madb.Series {
	for index := range series {
		if series[index].ID == id {
			return &series[index]
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

// assertIntegrationContributors は、実サービスの寄与者と役割を順序込みで検証する
func assertIntegrationContributors(t *testing.T, got, want []madb.Contributor) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("contributors = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index].Name != want[index].Name || len(got[index].Roles) != len(want[index].Roles) {
			t.Fatalf("contributors = %#v, want %#v", got, want)
		}
		for roleIndex := range want[index].Roles {
			if got[index].Roles[roleIndex] != want[index].Roles[roleIndex] {
				t.Fatalf("contributors = %#v, want %#v", got, want)
			}
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
