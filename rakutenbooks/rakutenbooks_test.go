package rakutenbooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eamat-dot/manken/api"
)

// TestNewClient_ValidatesOptions は、必須認証、任意認証、漫画区分、endpointを通信前に検証することを確認する
func TestNewClient_ValidatesOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
	}{
		{name: "missing all"},
		{name: "missing access key", options: []Option{WithApplicationID("app-secret")}},
		{name: "missing application ID", options: []Option{WithAccessKey("access-secret")}},
		{name: "blank affiliate", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID(" \t ")}},
		{name: "invalid genre", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithComicGenre("unknown")}},
		{name: "invalid book size", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithBookSize(BookSize(-1))}},
		{name: "book size too large", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithBookSize(BookSize(11))}},
		{name: "nil option", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access-secret"), nil}},
		{name: "endpoint query", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithEndpoint("https://example.invalid/books?applicationId=secret")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient(nil, test.options...)
			assertErrorKind(t, err, ErrorKindInvalidArgument)
			for _, secret := range []string{"app-secret", "access-secret"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("error exposes secret %q: %v", secret, err)
				}
			}
		})
	}

	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.affiliateID != "" || client.comicGenre != ComicGenreGeneral || client.bookSize != BookSizeAll {
		t.Fatalf("defaults = affiliate %q, genre %q, book size %d", client.affiliateID, client.comicGenre, client.bookSize)
	}
	for size := BookSizeAll; size <= BookSizeMookOther; size++ {
		if _, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithBookSize(size)); err != nil {
			t.Fatalf("NewClient(WithBookSize(%d)) error = %v", size, err)
		}
	}
}

// TestSearchBooks_BuildsRequestAndCursor は、認証、検索条件、漫画区分、固定パラメーター、カーソルを検証する
func TestSearchBooks_BuildsRequestAndCursor(t *testing.T) {
	var requests []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Clone(request.Context()))
		page := request.URL.Query().Get("page")
		if page == "1" {
			_, _ = writer.Write([]byte(sampleResponse(2, 1, 2, sampleItem("first", "9784088466361"))))
			return
		}
		_, _ = writer.Write([]byte(sampleResponse(2, 2, 2, sampleItem("second", "9784088466361"))))
	}))
	defer server.Close()

	client, err := NewClient(nil,
		WithApplicationID("app-secret"),
		WithAccessKey("access-secret"),
		WithAffiliateID("affiliate-secret"),
		WithComicGenre(ComicGenreBL),
		WithBookSize(BookSizeComic),
		WithEndpoint(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	request := SearchBooksRequest{Title: "  動物のお医者さん  ", Author: "佐々木倫子", Limit: 1}
	first, err := client.SearchBooks(context.Background(), request)
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(first.Books) != 1 || first.NextCursor == "" {
		t.Fatalf("first result = %#v", first)
	}
	query := requests[0].URL.Query()
	for key, want := range map[string]string{
		"applicationId": "app-secret",
		"affiliateId":   "affiliate-secret",
		"title":         "動物のお医者さん",
		"author":        "佐々木倫子",
		"booksGenreId":  "001021002",
		"hits":          "1",
		"page":          "1",
		"format":        "json",
		"formatVersion": "2",
		"sort":          "+releaseDate",
		"size":          "9",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if got := requests[0].Header.Get("accessKey"); got != "access-secret" {
		t.Fatalf("accessKey header = %q", got)
	}

	request.Cursor = first.NextCursor
	second, err := client.SearchBooks(context.Background(), request)
	if err != nil {
		t.Fatalf("second SearchBooks() error = %v", err)
	}
	if len(second.Books) != 1 || second.NextCursor != "" || requests[1].URL.Query().Get("page") != "2" {
		t.Fatalf("second result = %#v", second)
	}

	request.Limit = 2
	_, err = client.SearchBooks(context.Background(), request)
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooks_RejectsUnsupportedOrInvalidInput は、非対応条件と不正入力を通信前に拒否することを確認する
func TestSearchBooks_RejectsUnsupportedOrInvalidInput(t *testing.T) {
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	for _, request := range []SearchBooksRequest{
		{},
		{FreeText: "keyword"},
		{Title: "title", FreeText: "keyword"},
		{Title: "title", ExcludedText: "excluded"},
		{Title: "title", Limit: 31},
		{Title: "title", Cursor: "not-a-cursor"},
	} {
		_, err := client.SearchBooks(context.Background(), request)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
	_, err = (*Client)(nil).SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	var nilContext context.Context
	_, err = client.SearchBooks(nilContext, SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooks_DefaultLimitEmptyAndGenreCursor は、既定Limit、0件、漫画区分に拘束されたカーソルを確認する
func TestSearchBooks_DefaultLimitEmptyAndGenreCursor(t *testing.T) {
	var hits string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits = request.URL.Query().Get("hits")
		_, _ = writer.Write([]byte(`{"count":0,"page":0,"pageCount":0,"hits":0,"Items":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.SearchBooks(context.Background(), SearchBooksRequest{Title: "none"})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if hits != "20" || result.Books == nil || len(result.Books) != 0 || result.NextCursor != "" {
		t.Fatalf("hits = %q, result = %#v", hits, result)
	}

	cursor, err := encodeCursor(2, buildSearchKey("title", "", ComicGenreGeneral, BookSizeAll), 20)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	tlClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithComicGenre(ComicGenreTL), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient(TL) error = %v", err)
	}
	_, err = tlClient.SearchBooks(context.Background(), SearchBooksRequest{Title: "title", Cursor: cursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "different", Cursor: cursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	sizeClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithBookSize(BookSizeComic), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient(size) error = %v", err)
	}
	_, err = sizeClient.SearchBooks(context.Background(), SearchBooksRequest{Title: "title", Cursor: cursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooks_BookSizeIsOptionalAndCursorBound は、商品形態の検索条件とカーソル拘束を確認する
func TestSearchBooks_BookSizeIsOptionalAndCursorBound(t *testing.T) {
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		queries = append(queries, request.URL.Query())
		page, _ := strconv.Atoi(request.URL.Query().Get("page"))
		_, _ = writer.Write([]byte(sampleResponse(2, page, 2, sampleItem(strconv.Itoa(page), "9784088466361"))))
	}))
	defer server.Close()
	allClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithBookSize(BookSizeAll), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient(All) error = %v", err)
	}
	if _, err := allClient.SearchBooks(context.Background(), SearchBooksRequest{Title: "all"}); err != nil {
		t.Fatalf("SearchBooks(All) error = %v", err)
	}
	comicClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithBookSize(BookSizeComic), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient(Comic) error = %v", err)
	}
	first, err := comicClient.SearchBooks(context.Background(), SearchBooksRequest{Title: "comic"})
	if err != nil {
		t.Fatalf("SearchBooks(Comic) error = %v", err)
	}
	if queries[0].Has("size") || queries[1].Get("size") != "9" {
		t.Fatalf("queries = %#v", queries)
	}
	if _, err := comicClient.SearchBooks(context.Background(), SearchBooksRequest{Title: "comic", Cursor: first.NextCursor}); err != nil {
		t.Fatalf("SearchBooks(same size cursor) error = %v", err)
	}
	if len(queries) != 3 || queries[2].Get("page") != "2" {
		t.Fatalf("cursor query = %#v", queries)
	}
}

// TestSearchValues_BookSize は、商品形態の値に応じた検索queryを組み立てることを確認する
func TestSearchValues_BookSize(t *testing.T) {
	for size := BookSizeAll; size <= BookSizeMookOther; size++ {
		values := searchValues("title", "author", "001001", size, 20, 1)
		if got := values.Get("sort"); got != "+releaseDate" {
			t.Fatalf("sort = %q, want +releaseDate", got)
		}
		if size == BookSizeAll {
			if values.Has("size") {
				t.Fatalf("size = %q, want absent", values.Get("size"))
			}
			continue
		}
		if got, want := values.Get("size"), strconv.Itoa(int(size)); got != want {
			t.Fatalf("size for %d = %q, want %q", size, got, want)
		}
	}
}

// TestConvertItem_MapsSupportedFieldsOnly は、楽天Books項目を推測せず共通モデルへ変換することを確認する
func TestConvertItem_MapsSupportedFieldsOnly(t *testing.T) {
	price := int64(999)
	const observedAt = "2026-08-08T01:23:45Z"
	book := convertItem(booksItem{
		Title:          "作品 1",
		TitleKana:      "サクヒンイチ",
		SubTitle:       "副題",
		SeriesName:     "シリーズ",
		Author:         "原作者/作画者",
		AuthorKana:     "ゲンサクシャ/サクガシャ",
		PublisherName:  "出版社",
		ISBN:           "9784088466361",
		SalesDate:      "2026年8月上旬",
		ItemCaption:    "説明",
		BooksGenreID:   "001021002016/001001002015",
		Size:           "コミック",
		ItemURL:        "https://books.rakuten.co.jp/rb/123/",
		AffiliateURL:   "https://hb.afl.rakuten.co.jp/secret",
		ItemPrice:      &price,
		SmallImageURL:  "https://example.invalid/s.jpg",
		MediumImageURL: "https://example.invalid/m.jpg",
		LargeImageURL:  "https://example.invalid/l.jpg",
	}, observedAt)
	if book.Normalized.Title != "作品 1" || book.Normalized.TitleReading != "サクヒンイチ" || book.Normalized.Subtitle != "副題" {
		t.Fatalf("titles = %#v", book.Normalized)
	}
	if len(book.Normalized.Authors) != 2 || book.Normalized.Authors[0] != "原作者" || book.Normalized.Authors[1] != "作画者" {
		t.Fatalf("authors = %#v", book.Normalized.Authors)
	}
	if len(book.Normalized.Contributors) != 2 || book.Normalized.Contributors[0].Name != "原作者" || book.Normalized.Contributors[0].Reading != "ゲンサクシャ" || book.Normalized.Contributors[1].Name != "作画者" || book.Normalized.Contributors[1].Reading != "サクガシャ" || len(book.Normalized.Contributors[0].Roles) != 0 || len(book.Normalized.Contributors[1].Roles) != 0 {
		t.Fatalf("contributors = %#v", book.Normalized.Contributors)
	}
	if len(book.Normalized.Series) != 1 || book.Normalized.Series[0].Name != "シリーズ" || len(book.Normalized.Publishers) != 1 {
		t.Fatalf("series/publishers = %#v / %#v", book.Normalized.Series, book.Normalized.Publishers)
	}
	if len(book.Normalized.Identifiers) != 1 || book.Normalized.Identifiers[0].Type != IdentifierTypeISBN13 {
		t.Fatalf("identifiers = %#v", book.Normalized.Identifiers)
	}
	if len(book.Normalized.Dates) != 1 || book.Normalized.Dates[0].Type != BookDateTypeReleased || book.Normalized.Dates[0].Value != "2026年8月上旬" {
		t.Fatalf("dates = %#v", book.Normalized.Dates)
	}
	if len(book.Normalized.Subjects) != 2 || book.Normalized.Subjects[0].Scheme != "rakuten_books" {
		t.Fatalf("subjects = %#v", book.Normalized.Subjects)
	}
	if book.Normalized.Medium != PublicationMediumPrint || book.Normalized.PhysicalSize == nil || book.Normalized.PhysicalSize.Name != "コミック" {
		t.Fatalf("medium/size = %q / %#v", book.Normalized.Medium, book.Normalized.PhysicalSize)
	}
	if len(book.Normalized.Images) != 3 || len(book.Normalized.Prices) != 1 {
		t.Fatalf("images/prices = %#v / %#v", book.Normalized.Images, book.Normalized.Prices)
	}
	gotPrice := book.Normalized.Prices[0]
	if gotPrice.Type != PriceTypeCurrent || gotPrice.Amount != 999 || gotPrice.Currency != "JPY" || gotPrice.TaxIncluded == nil || !*gotPrice.TaxIncluded || gotPrice.Source != SourceRakutenBooks || gotPrice.ObservedAt != observedAt {
		t.Fatalf("price = %#v", gotPrice)
	}
	if len(book.Sources) != 1 || book.Sources[0].Source != SourceRakutenBooks || book.Sources[0].ID != "" || book.Sources[0].URL != "https://books.rakuten.co.jp/rb/123/" || book.Sources[0].AffiliateURL != "https://hb.afl.rakuten.co.jp/secret" {
		t.Fatalf("sources = %#v", book.Sources)
	}
	if strings.Contains(book.Sources[0].URL, "afl.rakuten") {
		t.Fatalf("source URL uses affiliate URL: %q", book.Sources[0].URL)
	}
}

// TestContributors_SplitsAuthorsWithoutNormalizingNames は、著者を人物単位へ分割し内部表記を維持することを確認する
func TestContributors_SplitsAuthorsWithoutNormalizingNames(t *testing.T) {
	authors, values := contributors("  采　和輝 / 苗字, 名前 / / 単著  ", "サイ　カズキ/ミョウジ, ナマエ")
	wantAuthors := []string{"采　和輝", "苗字, 名前", "単著"}
	if !slicesEqual(authors, wantAuthors) {
		t.Fatalf("authors = %#v, want %#v", authors, wantAuthors)
	}
	if len(values) != 3 || values[0].Reading != "" || values[1].Reading != "" || values[2].Reading != "" {
		t.Fatalf("contributors with unmatched readings = %#v", values)
	}

	authors, values = contributors(" 単著 ", " タンチョ ")
	if !slicesEqual(authors, []string{"単著"}) || len(values) != 1 || values[0].Reading != "タンチョ" {
		t.Fatalf("single author = %#v / %#v", authors, values)
	}
	authors, values = contributors("", "")
	if authors != nil || values != nil {
		t.Fatalf("empty authors = %#v / %#v", authors, values)
	}
}

// slicesEqual は、文字列スライスの要素と順序が等しいか判定する
func slicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// TestConvertItem_MissingOptionalFields は、楽天Booksの任意項目欠落を正常な空値として扱うことを確認する
func TestConvertItem_MissingOptionalFields(t *testing.T) {
	book := convertItem(booksItem{Title: "title"}, "")
	if book.Normalized.Title != "title" || book.Normalized.Authors != nil || book.Normalized.Series != nil || book.Normalized.Identifiers != nil || book.Normalized.Dates != nil || book.Normalized.PhysicalSize != nil || book.Normalized.Prices != nil {
		t.Fatalf("book = %#v", book)
	}
	if book.Normalized.Medium != PublicationMediumPrint || len(book.Sources) != 1 || book.Sources[0].Source != SourceRakutenBooks || book.Sources[0].AffiliateURL != "" {
		t.Fatalf("medium/sources = %q / %#v", book.Normalized.Medium, book.Sources)
	}
}

// TestConvertItem_PreservesZeroPrice は、itemPriceの0円を欠落と区別して取得時点価格として保持することを確認する
func TestConvertItem_PreservesZeroPrice(t *testing.T) {
	zero := int64(0)
	book := convertItem(booksItem{Title: "free", ItemPrice: &zero}, "2026-08-08T02:00:00Z")
	if len(book.Normalized.Prices) != 1 || book.Normalized.Prices[0].Amount != 0 || book.Normalized.Prices[0].Type != PriceTypeCurrent {
		t.Fatalf("prices = %#v", book.Normalized.Prices)
	}
}

// TestRawResponseAndInvalidResponse は、成功本文を変更せず返し解析失敗時にも保持することを確認する
func TestRawResponseAndInvalidResponse(t *testing.T) {
	valid := []byte(sampleResponse(1, 1, 1, sampleItem("one", "9784088466361")))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(valid)
	}))
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchBooksRequest{Title: "title"})
	server.Close()
	if err != nil || len(result.Books) != 1 || !bytes.Equal(raw, valid) {
		t.Fatalf("result/raw/error = %#v / %q / %v", result, raw, err)
	}

	invalid := []byte(`{"Items":`)
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(invalid)
	}))
	defer server.Close()
	client, err = NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, raw, err = client.SearchBooksWithRawResponse(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if !bytes.Equal(raw, invalid) {
		t.Fatalf("raw = %q, want %q", raw, invalid)
	}
}

// TestLookupBooksByISBN_ValidatesAndMatches は、ISBNを1件だけ問い合わせて一致商品だけを返すことを確認する
func TestLookupBooksByISBN_ValidatesAndMatches(t *testing.T) {
	var query url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query = request.URL.Query()
		body := fmt.Sprintf(`{"count":2,"page":1,"pageCount":1,"hits":30,"Items":[%s,%s]}`,
			sampleItem("match", "9784088466361"), sampleItem("other", "9784091400017"))
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithComicGenre(ComicGenreTL), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, raw, err := client.LookupBooksByISBNWithRawResponse(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].RequestedISBN != "4088466365" || len(result.Items[0].Books) != 1 || len(raw) == 0 {
		t.Fatalf("result/raw = %#v / %d bytes", result, len(raw))
	}
	if query.Get("isbn") != "9784088466361" || query.Get("booksGenreId") != "" || query.Get("size") != "" || query.Get("hits") != "30" || query.Get("page") != "1" {
		t.Fatalf("query = %#v", query)
	}
	for _, isbns := range [][]string{nil, {}, {"bad"}, {"4088466365", "9784088466361"}} {
		_, err := client.LookupBooksByISBN(context.Background(), isbns)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
}

// TestLookupBooksByISBN_ReturnsNonNilEmptyBooks は、ISBNが見つからない場合も1件の入力結果とnon-nilの空Booksを返すことを確認する
func TestLookupBooksByISBN_ReturnsNonNilEmptyBooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"count":0,"page":0,"pageCount":0,"hits":0,"Items":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.LookupBooksByISBN(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Books == nil || len(result.Items[0].Books) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

// TestHTTPAndTransportErrorsDoNotExposeSecrets は、HTTP・通信エラーを分類し認証値を公開しないことを確認する
func TestHTTPAndTransportErrorsDoNotExposeSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Retry-After", "10")
		writer.WriteHeader(http.StatusTooManyRequests)
		_, _ = writer.Write([]byte("app-secret access-secret affiliate-secret"))
	}))
	client, err := NewClient(nil, WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID("affiliate-secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
	server.Close()
	assertErrorKind(t, err, ErrorKindUnavailable)
	for _, secret := range []string{"app-secret", "access-secret", "affiliate-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error exposes %q: %v", secret, err)
		}
	}
	var classified *Error
	if !errors.As(err, &classified) || classified.StatusCode != http.StatusTooManyRequests || classified.RetryAfter.Seconds() != 10 {
		t.Fatalf("classified error = %#v", classified)
	}

	cause := errors.New("dial failed")
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, cause
	})}
	client, err = NewClient(httpClient, WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID("affiliate-secret"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUnavailable)
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(err, cause) = false: %v", err)
	}
	for _, secret := range []string{"app-secret", "access-secret", "affiliate-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("transport error exposes %q: %v", secret, err)
		}
	}
}

// TestSearchBooks_DoesNotFollowRedirects は、Access Keyを含むリクエストがredirect先へ送られないことを確認する
func TestSearchBooks_DoesNotFollowRedirects(t *testing.T) {
	targetRequests := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetRequests++
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", target.URL)
		writer.WriteHeader(http.StatusFound)
	}))
	defer redirect.Close()

	redirectCalls := 0
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			redirectCalls++
			return errors.New("caller redirect policy")
		},
	}
	client, err := NewClient(httpClient,
		WithApplicationID("app-secret"),
		WithAccessKey("access-secret"),
		WithEndpoint(redirect.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.httpClient == httpClient {
		t.Fatal("Client reused the caller HTTP client")
	}
	if httpClient.Timeout != 5*time.Second || httpClient.CheckRedirect == nil {
		t.Fatalf("caller HTTP client was modified: %#v", httpClient)
	}

	_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUpstream)
	var classified *Error
	if !errors.As(err, &classified) || classified.StatusCode != http.StatusFound {
		t.Fatalf("classified error = %#v", classified)
	}
	if targetRequests != 0 {
		t.Fatalf("redirect target requests = %d, want 0", targetRequests)
	}
	if redirectCalls != 0 {
		t.Fatalf("caller CheckRedirect calls = %d, want 0", redirectCalls)
	}
}

// TestHTTPStatusClassification は、代表HTTPステータスを共通エラー分類へ対応付けることを確認する
func TestHTTPStatusClassification(t *testing.T) {
	for _, test := range []struct {
		code int
		kind ErrorKind
	}{
		{http.StatusBadRequest, ErrorKindUpstream},
		{http.StatusRequestTimeout, ErrorKindUnavailable},
		{http.StatusTooManyRequests, ErrorKindUnavailable},
		{http.StatusInternalServerError, ErrorKindUnavailable},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(test.code)
		}))
		client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
		if err != nil {
			server.Close()
			t.Fatalf("NewClient() error = %v", err)
		}
		_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
		server.Close()
		assertErrorKind(t, err, test.kind)
	}
}

// TestBuildSearchResult_RejectsPageMismatch は、楽天Booksの応答ページが要求ページと異なる場合を拒否することを確認する
func TestBuildSearchResult_RejectsPageMismatch(t *testing.T) {
	response := booksResponse{Count: 2, Page: 1, PageCount: 2, Hits: 1, Items: []booksItem{{Title: "unexpected"}}}
	_, err := buildSearchResult(response, 2, buildSearchKey("title", "", ComicGenreGeneral, BookSizeAll), 1, "")
	if err == nil {
		t.Fatal("buildSearchResult() error = nil")
	}
}

// TestPaginationStopsAtPage100 は、楽天Booksの最大100ページで次カーソルを生成しないことを確認する
func TestPaginationStopsAtPage100(t *testing.T) {
	response := booksResponse{Count: 3000, Page: 100, PageCount: 100, Hits: 30, Items: []booksItem{{Title: "last"}}}
	result, err := buildSearchResult(response, 100, buildSearchKey("title", "", ComicGenreGeneral, BookSizeAll), 30, "")
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 || result.NextCursor != "" {
		t.Fatalf("result = %#v", result)
	}
}

// TestSearchBooks_RejectsOversizedBody は、成功本文上限を超えた応答をRawなしのinvalid_responseとすることを確認する
func TestSearchBooks_RejectsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", successBodyMax+1)))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if raw != nil {
		t.Fatalf("raw = %d bytes, want nil", len(raw))
	}
}

// TestSearchBooks_OmitsAffiliateIDWhenUnset は、Affiliate ID未設定時にリクエストへaffiliateIdを付けないことを確認する
func TestSearchBooks_OmitsAffiliateIDWhenUnset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.Query().Get("affiliateId"); got != "" {
			t.Errorf("affiliateId = %q, want empty", got)
		}
		_, _ = writer.Write([]byte(sampleResponse(0, 0, 0)))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"}); err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
}

// TestContextErrorIsPreserved は、キャンセル済みコンテキストをエラーチェーンから判定できることを確認する
func TestContextErrorIsPreserved(t *testing.T) {
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.SearchBooks(ctx, SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUnavailable)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("errors.Is(err, context.Canceled) = false: %v", err)
	}
}

// assertErrorKind は、エラーが期待する共通分類を持つことを確認する
func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want kind %q", want)
	}
	var classified *api.Error
	if !errors.As(err, &classified) || classified.Kind != want {
		t.Fatalf("error = %v, want kind %q", err, want)
	}
}

// sampleResponse は、単体テスト用の楽天BooksレスポンスJSONを生成する
func sampleResponse(count int, page int, pageCount int, items ...string) string {
	return fmt.Sprintf(`{"count":%d,"page":%d,"pageCount":%d,"hits":%d,"Items":[%s]}`,
		count, page, pageCount, len(items), strings.Join(items, ","))
}

// sampleItem は、単体テスト用の楽天Books商品JSONを生成する
func sampleItem(title string, isbn string) string {
	return fmt.Sprintf(`{"title":%q,"titleKana":"ヨミ","subTitle":"副題","seriesName":"シリーズ","author":"著者A/著者B","authorKana":"チョシャエー/チョシャビー","publisherName":"出版社","isbn":%q,"salesDate":"2026年8月8日","itemCaption":"説明","booksGenreId":"001001003/001001012","size":"コミック","itemUrl":"https://books.rakuten.co.jp/rb/123/","affiliateUrl":"https://hb.afl.rakuten.co.jp/ignored","smallImageUrl":"https://example.invalid/s.jpg","mediumImageUrl":"https://example.invalid/m.jpg","largeImageUrl":"https://example.invalid/l.jpg","itemPrice":999,"availability":"1"}`,
		title, isbn)
}

// roundTripFunc は、関数をhttp.RoundTripperとして使用できるようにする
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、設定された関数でHTTPリクエストを処理する
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
