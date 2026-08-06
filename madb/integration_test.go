//go:build integration

package madb_test

import (
	"context"
	"fmt"
	"strings"
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

// TestIntegration_SearchBooksMultipleTerms は、複数語のAND検索とページングを実サービスで確認する
func TestIntegration_SearchBooksMultipleTerms(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	first, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title: "うる星 復刻box",
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("first SearchBooks() error = %v", err)
	}
	if len(first.Books) != 2 || first.NextCursor == "" {
		t.Fatalf("first result = %#v, want two books and cursor", first)
	}

	second, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title:  "うる星 復刻box",
		Limit:  2,
		Cursor: first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second SearchBooks() error = %v", err)
	}
	if len(second.Books) != 2 || second.NextCursor != "" {
		t.Fatalf("second result = %#v, want final two books", second)
	}

	books := append(first.Books, second.Books...)
	seen := make(map[string]struct{}, len(books))
	lastURL := ""
	for _, book := range books {
		title := strings.ToLower(book.Normalized.Title)
		if !strings.Contains(title, "うる星") || !strings.Contains(title, "復刻box") {
			t.Fatalf("title = %q, want both search terms", book.Normalized.Title)
		}
		url := book.Sources[0].URL
		if _, exists := seen[url]; exists {
			t.Fatalf("duplicate source URL = %q", url)
		}
		if lastURL >= url {
			t.Fatalf("source URLs are not strictly ordered: %q then %q", lastURL, url)
		}
		seen[url] = struct{}{}
		lastURL = url
	}
}

// TestIntegration_SearchBooksExcludedText は、各正条件と除外語の組み合わせを実サービスで確認する
func TestIntegration_SearchBooksExcludedText(t *testing.T) {
	client := newIntegrationClient(t)
	tests := []struct {
		name   string
		input  madb.SearchBooksRequest
		bookID string
	}{
		{
			name: "title",
			input: madb.SearchBooksRequest{
				Title:        "うる星",
				ExcludedText: "復刻box",
				Limit:        20,
			},
		},
		{
			name: "author",
			input: madb.SearchBooksRequest{
				Author:       "佐々木倫子",
				ExcludedText: "復刻box",
				Limit:        20,
			},
			bookID: "M292132",
		},
		{
			name: "free text",
			input: madb.SearchBooksRequest{
				FreeText:     "うる星",
				ExcludedText: "復刻box",
				Limit:        20,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			startedAt := time.Now()
			result, err := client.SearchBooks(ctx, test.input)
			elapsed := time.Since(startedAt)
			if err != nil {
				t.Fatalf("SearchBooks() error = %v", err)
			}
			if len(result.Books) == 0 {
				t.Fatal("SearchBooks() returned no books")
			}
			if test.bookID != "" && findBookByID(result.Books, test.bookID) == nil {
				t.Fatalf("%s was not found: %#v", test.bookID, result)
			}
			for _, book := range result.Books {
				if strings.Contains(strings.ToLower(book.Normalized.Title), "復刻box") {
					t.Fatalf("excluded title was returned: %q", book.Normalized.Title)
				}
			}
			t.Logf("ExcludedText with %s elapsed time: %s", test.name, elapsed)
		})
	}
}

// TestIntegration_SearchBooksExcludedTextPagination は、除外条件をページ間で維持する
func TestIntegration_SearchBooksExcludedTextPagination(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	request := madb.SearchBooksRequest{
		FreeText:     "うる星",
		ExcludedText: "復刻box",
		Limit:        2,
	}
	first, err := client.SearchBooks(ctx, request)
	if err != nil {
		t.Fatalf("first SearchBooks() error = %v", err)
	}
	if len(first.Books) != 2 || first.NextCursor == "" {
		t.Fatalf("first result = %#v, want two books and cursor", first)
	}

	request.Cursor = first.NextCursor
	second, err := client.SearchBooks(ctx, request)
	if err != nil {
		t.Fatalf("second SearchBooks() error = %v", err)
	}
	if len(second.Books) == 0 {
		t.Fatal("second SearchBooks() returned no books")
	}

	books := append(first.Books, second.Books...)
	seen := make(map[string]struct{}, len(books))
	lastURL := ""
	for _, book := range books {
		if strings.Contains(strings.ToLower(book.Normalized.Title), "復刻box") {
			t.Fatalf("excluded title was returned: %q", book.Normalized.Title)
		}
		url := book.Sources[0].URL
		if _, exists := seen[url]; exists {
			t.Fatalf("duplicate source URL = %q", url)
		}
		if lastURL >= url {
			t.Fatalf("source URLs are not strictly ordered: %q then %q", lastURL, url)
		}
		seen[url] = struct{}{}
		lastURL = url
	}
}

// TestIntegration_LookupBooksByISBN は、複数ISBNと同一ISBNの複数リソースを参照する
func TestIntegration_LookupBooksByISBN(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, rawResponse, err := client.LookupBooksByISBNWithRawResponse(ctx, []string{
		"4088466365",
		"9784088466361",
		"9784990524302",
		"9780000000002",
	})
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if len(result.Items) != 4 {
		t.Fatalf("len(Items) = %d, want 4", len(result.Items))
	}
	for _, index := range []int{0, 1} {
		if findBookByID(result.Items[index].Books, "M190399") == nil {
			t.Fatalf("M190399 was not found in item %d: %#v", index, result.Items[index])
		}
	}
	wantIDs := []string{"M409358", "M409359", "M409360"}
	if len(result.Items[2].Books) != len(wantIDs) {
		t.Fatalf("duplicate ISBN books = %#v, want %v", result.Items[2].Books, wantIDs)
	}
	for index, id := range wantIDs {
		if result.Items[2].Books[index].Sources[0].ID != id {
			t.Fatalf("duplicate ISBN books = %#v, want %v", result.Items[2].Books, wantIDs)
		}
	}
	if result.Items[3].Books == nil || len(result.Items[3].Books) != 0 {
		t.Fatalf("missing item Books = %#v, want non-nil empty", result.Items[3].Books)
	}
	if len(rawResponse) == 0 || len(rawResponse) > 4<<20 {
		t.Fatalf("raw response size = %d, want 1..4 MiB", len(rawResponse))
	}
}

// TestIntegration_SearchBooksAuthor は、creator文字列とAgent参照の著者を実サービスで検索する
func TestIntegration_SearchBooksAuthor(t *testing.T) {
	client := newIntegrationClient(t)
	tests := []struct {
		name   string
		author string
		bookID string
	}{
		{name: "creator string", author: "佐々木倫子", bookID: "M292132"},
		{name: "agent reference", author: "KotzDean", bookID: "M830542"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
				Author: test.author,
				Limit:  100,
			})
			if err != nil {
				t.Fatalf("SearchBooks() error = %v", err)
			}
			if findBookByID(result.Books, test.bookID) == nil {
				t.Fatalf("%s was not found: %#v", test.bookID, result)
			}
		})
	}
}

// TestIntegration_SearchBooksTitleAndAuthor は、タイトルと著者名をAND条件で検索する
func TestIntegration_SearchBooksTitleAndAuthor(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title:  "動物のお医者さん",
		Author: "佐々木倫子",
		Limit:  20,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if findBookByID(result.Books, "M292132") == nil {
		t.Fatalf("M292132 was not found: %#v", result)
	}
}

// TestIntegration_SearchBooksFreeText は、単行本の複数フィールドを横断するAND検索と応答時間を確認する
func TestIntegration_SearchBooksFreeText(t *testing.T) {
	client := newIntegrationClient(t)
	tests := []struct {
		name     string
		freeText string
	}{
		{name: "one term", freeText: "うる星"},
		{name: "two terms in one field", freeText: "うる星 復刻box"},
		{name: "two terms across fields", freeText: "うる星 高橋留美子"},
		{name: "three terms", freeText: "うる星 高橋留美子 新装版"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			startedAt := time.Now()
			result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
				FreeText: test.freeText,
				Limit:    20,
			})
			elapsed := time.Since(startedAt)
			if err != nil {
				t.Fatalf("SearchBooks() error = %v", err)
			}
			if len(result.Books) == 0 {
				t.Fatal("SearchBooks() returned no books")
			}
			t.Logf("FreeText %q elapsed time: %s", test.freeText, elapsed)
		})
	}
}

// TestIntegration_SearchBooksFreeTextFields は、主要な対象フィールドの代表値を検索する
func TestIntegration_SearchBooksFreeTextFields(t *testing.T) {
	client := newIntegrationClient(t)
	tests := []struct {
		name     string
		freeText string
	}{
		{name: "title", freeText: "動物のお医者さん"},
		{name: "subtitle", freeText: "幼馴染の大公閣下の溺愛が止まらないのです"},
		{name: "series name", freeText: "ねこぱんち文庫"},
		{name: "creator", freeText: "佐々木倫子"},
		{name: "publisher", freeText: "白泉社"},
		{name: "brand", freeText: "花とゆめCOMICS"},
		{name: "version", freeText: "新装版"},
		{name: "size", freeText: "18cm"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, rawResponse, err := client.SearchBooksWithRawResponse(ctx, madb.SearchBooksRequest{
				FreeText: test.freeText,
				Limit:    100,
			})
			if err != nil {
				t.Fatalf("SearchBooks() error = %v", err)
			}
			if len(result.Books) == 0 {
				t.Fatalf("FreeText %q returned no books", test.freeText)
			}
			if !strings.Contains(string(rawResponse), test.freeText) {
				t.Fatalf("Raw response does not contain FreeText %q", test.freeText)
			}
		})
	}
}

// TestIntegration_SearchBooksTitleAuthorAndFreeText は、専用条件とフリーワードのAND検索を確認する
func TestIntegration_SearchBooksTitleAuthorAndFreeText(t *testing.T) {
	client := newIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.SearchBooks(ctx, madb.SearchBooksRequest{
		Title:    "うる星",
		Author:   "高橋留美子",
		FreeText: "新装版",
		Limit:    20,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(result.Books) == 0 {
		t.Fatal("SearchBooks() returned no books")
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
	if book.Normalized.TitleReading != "ドウブツノオイシャサン" {
		t.Fatalf("TitleReading = %q", book.Normalized.TitleReading)
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
