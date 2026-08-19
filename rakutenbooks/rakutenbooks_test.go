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

	"github.com/eamat-dot/manken/model"
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
		{name: "access key contains control character", options: []Option{WithApplicationID("app-secret"), WithAccessKey("access\nsecret")}},
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

// TestNewClient_ValidatesEndpointSecurity は、endpointのHTTPS要件と既存のURL制約を通信前に検証する
func TestNewClient_ValidatesEndpointSecurity(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{name: "https", endpoint: "https://example.com/books"},
		{name: "localhost HTTP", endpoint: "http://localhost:8080/books"},
		{name: "IPv4 loopback HTTP", endpoint: "http://127.0.0.2:8080/books"},
		{name: "IPv6 loopback HTTP", endpoint: "http://[::1]:8080/books"},
		{name: "non-loopback HTTP", endpoint: "http://example.com/books", wantErr: true},
		{name: "user information", endpoint: "https://user@example.com/books", wantErr: true},
		{name: "query", endpoint: "https://example.com/books?applicationId=secret", wantErr: true},
		{name: "fragment", endpoint: "https://example.com/books#part", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(
				nil,
				WithApplicationID("app"),
				WithAccessKey("access"),
				WithEndpoint(test.endpoint),
			)
			if test.wantErr {
				assertErrorKind(t, err, ErrorKindInvalidArgument)
				return
			}
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			if client.endpoint != test.endpoint {
				t.Fatalf("endpoint = %q, want %q", client.endpoint, test.endpoint)
			}
		})
	}
}

// TestSearchBooks_UsesComicGenreID は、各漫画区分をBooksBook SearchのbooksGenreIdへ1回だけ設定する
func TestSearchBooks_UsesComicGenreID(t *testing.T) {
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		queries = append(queries, request.URL.Query())
		if request.URL.Path != "/" {
			t.Errorf("path = %q, want /", request.URL.Path)
		}
		_, _ = writer.Write([]byte(sampleResponse(0, 0, 0)))
	}))
	defer server.Close()

	tests := []struct {
		genre ComicGenre
		want  string
	}{
		{genre: ComicGenreGeneral, want: rakutenBooksGenreGeneralComic},
		{genre: ComicGenreBL, want: rakutenBooksGenreBLComic},
		{genre: ComicGenreTL, want: rakutenBooksGenreTLComic},
	}
	for _, test := range tests {
		client, err := NewClient(
			nil,
			WithApplicationID("app"),
			WithAccessKey("access"),
			WithComicGenre(test.genre),
			WithEndpoint(server.URL),
		)
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if _, err := client.SearchBooks(context.Background(), SearchRequest{Title: "title"}); err != nil {
			t.Fatalf("SearchBooks(%q) error = %v", test.genre, err)
		}
	}
	if len(queries) != len(tests) {
		t.Fatalf("requests = %d, want %d", len(queries), len(tests))
	}
	for index, test := range tests {
		if got := queries[index].Get("booksGenreId"); got != test.want {
			t.Errorf("request %d booksGenreId = %q, want %q", index, got, test.want)
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
	request := SearchRequest{Title: "  動物のお医者さん  ", Author: "佐々木倫子", Publisher: " 白泉社 ", Limit: 1}
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
		"publisherName": "白泉社",
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

// TestSearchBooks_PublisherOnlyAndCursorMismatch は、出版社だけの検索と出版社が異なるカーソルの通信前拒否を検証する
func TestSearchBooks_PublisherOnlyAndCursorMismatch(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if got, want := request.URL.Query().Get("publisherName"), "白泉社"; got != want {
			t.Errorf("publisherName = %q, want %q", got, want)
		}
		_, _ = writer.Write([]byte(sampleResponse(2, 1, 2, sampleItem("first", "9784088466361"))))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	first, err := client.SearchBooks(context.Background(), SearchRequest{Publisher: "白泉社", Limit: 1})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("SearchBooks() = %#v, %v", first, err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Publisher: "集英社", Limit: 1, Cursor: first.NextCursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

// TestBuildSearchKey_PreservesMainFormatWithoutPublisher は、Publisher未指定時にmainの検索キー形式を維持することを確認する
func TestBuildSearchKey_PreservesMainFormatWithoutPublisher(t *testing.T) {
	title := "title"
	author := "author"
	genre := ComicGenreBL
	size := BookSizeComic
	mainKey := "5:title6:author2:bl1:9"

	if got := buildSearchKey(title, author, "", genre, size); got != mainKey {
		t.Fatalf("buildSearchKey() = %q, want main format %q", got, mainKey)
	}
	if got := buildSearchKey(title, author, "publisher", genre, size); got != mainKey+"9:publisher" {
		t.Fatalf("buildSearchKey() = %q, want length-prefixed publisher", got)
	}
}

// TestBuildSearchKey_DistinguishesEmbeddedNull は、NULを含む異なる検索条件を別のカーソルへ関連付けることを確認する
func TestBuildSearchKey_DistinguishesEmbeddedNull(t *testing.T) {
	firstKey := buildSearchKey("a\x00b", "", "", ComicGenreGeneral, BookSizeAll)
	secondKey := buildSearchKey("a", "b\x00", "", ComicGenreGeneral, BookSizeAll)
	if firstKey == secondKey {
		t.Fatal("buildSearchKey() produced the same key for different title and author")
	}
	cursor, err := encodeCursor(2, firstKey, defaultLimit)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	if _, err := decodeCursor(cursor, secondKey, defaultLimit); err == nil {
		t.Fatal("decodeCursor() error = nil, want search-condition mismatch")
	}

	publisherKey := buildSearchKey("title", "author", "publisher\x00suffix", ComicGenreGeneral, BookSizeAll)
	shiftedKey := buildSearchKey("title", "author\x00publisher", "suffix", ComicGenreGeneral, BookSizeAll)
	if publisherKey == shiftedKey {
		t.Fatal("buildSearchKey() produced the same key for different publisher fields")
	}
}

// TestSearchBooks_RejectsUnsupportedOrInvalidInput は、非対応条件と不正入力を通信前に拒否することを確認する
func TestSearchBooks_RejectsUnsupportedOrInvalidInput(t *testing.T) {
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	for _, request := range []SearchRequest{
		{},
		{Query: "keyword"},
		{Title: "title", Query: "keyword"},
		{Title: "title", Exclude: "excluded"},
		{Title: "title", Limit: 31},
		{Title: "title", Cursor: "not-a-cursor"},
	} {
		_, err := client.SearchBooks(context.Background(), request)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
	_, err = (*Client)(nil).SearchBooks(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	var nilContext context.Context
	_, err = client.SearchBooks(nilContext, SearchRequest{Title: "title"})
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
	result, err := client.SearchBooks(context.Background(), SearchRequest{Title: "none"})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if hits != "20" || result.Books == nil || len(result.Books) != 0 || result.NextCursor != "" {
		t.Fatalf("hits = %q, result = %#v", hits, result)
	}

	cursor, err := encodeCursor(2, buildSearchKey("title", "", "", ComicGenreGeneral, BookSizeAll), 20)
	if err != nil {
		t.Fatalf("encodeCursor() error = %v", err)
	}
	tlClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithComicGenre(ComicGenreTL), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient(TL) error = %v", err)
	}
	_, err = tlClient.SearchBooks(context.Background(), SearchRequest{Title: "title", Cursor: cursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "different", Cursor: cursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	sizeClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithBookSize(BookSizeComic), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient(size) error = %v", err)
	}
	_, err = sizeClient.SearchBooks(context.Background(), SearchRequest{Title: "title", Cursor: cursor})
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
	if _, err := allClient.SearchBooks(context.Background(), SearchRequest{Title: "all"}); err != nil {
		t.Fatalf("SearchBooks(All) error = %v", err)
	}
	comicClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithBookSize(BookSizeComic), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient(Comic) error = %v", err)
	}
	first, err := comicClient.SearchBooks(context.Background(), SearchRequest{Title: "comic"})
	if err != nil {
		t.Fatalf("SearchBooks(Comic) error = %v", err)
	}
	if queries[0].Has("size") || queries[1].Get("size") != "9" {
		t.Fatalf("queries = %#v", queries)
	}
	if _, err := comicClient.SearchBooks(context.Background(), SearchRequest{Title: "comic", Cursor: first.NextCursor}); err != nil {
		t.Fatalf("SearchBooks(same size cursor) error = %v", err)
	}
	if len(queries) != 3 || queries[2].Get("page") != "2" {
		t.Fatalf("cursor query = %#v", queries)
	}
}

// TestSearchValues_BookSize は、商品形態の値に応じた検索queryを組み立てることを確認する
func TestSearchValues_BookSize(t *testing.T) {
	for size := BookSizeAll; size <= BookSizeMookOther; size++ {
		values := searchValues("title", "author", "", "001001", size, 20, 1)
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
		SeriesNameKana: "シリーズヨミ",
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
	if book.Title != "作品 1" || book.TitleReading != "サクヒンイチ" || book.Subtitle != "副題" {
		t.Fatalf("titles = %#v", book)
	}
	if book.Volume.Number == nil || *book.Volume.Number != 1 || book.Volume.Label != "1" {
		t.Fatalf("volume = %#v", book.Volume)
	}
	if len(book.Authors) != 2 || book.Authors[0] != "原作者" || book.Authors[1] != "作画者" {
		t.Fatalf("authors = %#v", book.Authors)
	}
	if len(book.Contributors) != 2 || book.Contributors[0].Name != "原作者" || book.Contributors[0].Reading != "ゲンサクシャ" || book.Contributors[1].Name != "作画者" || book.Contributors[1].Reading != "サクガシャ" || len(book.Contributors[0].Roles) != 0 || len(book.Contributors[1].Roles) != 0 {
		t.Fatalf("contributors = %#v", book.Contributors)
	}
	if len(book.PublicationSeries) != 1 || book.PublicationSeries[0] != "シリーズ" || len(book.Publishers) != 1 {
		t.Fatalf("publication series/publishers = %#v / %#v", book.PublicationSeries, book.Publishers)
	}
	if len(book.ISBN10) != 0 || len(book.ISBN13) != 1 || book.ISBN13[0] != "9784088466361" {
		t.Fatalf("ISBN = %#v / %#v", book.ISBN10, book.ISBN13)
	}
	if book.ReleaseDate != "2026年8月上旬" {
		t.Fatalf("release date = %q", book.ReleaseDate)
	}
	if len(book.Subjects) != 2 || book.Subjects[0].Scheme != "rakuten_books" {
		t.Fatalf("subjects = %#v", book.Subjects)
	}
	if book.Medium != PublicationMediumPrint || book.Size != "コミック" {
		t.Fatalf("medium/size = %q / %q", book.Medium, book.Size)
	}
	if book.CoverURL != "https://example.invalid/l.jpg" || book.CurrentPrice == nil {
		t.Fatalf("cover/current price = %q / %#v", book.CoverURL, book.CurrentPrice)
	}
	gotPrice := book.CurrentPrice
	if gotPrice.Amount != 999 || gotPrice.Currency != "JPY" || gotPrice.TaxIncluded == nil || !*gotPrice.TaxIncluded || gotPrice.Source != SourceRakutenBooks || gotPrice.ObservedAt != observedAt {
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
	if book.Title != "title" || book.Authors != nil || book.PublicationSeries != nil || book.ISBN10 != nil || book.ISBN13 != nil || book.ReleaseDate != "" || book.Size != "" || book.CurrentPrice != nil || book.CoverURL != "" {
		t.Fatalf("book = %#v", book)
	}
	if book.Medium != PublicationMediumPrint || len(book.Sources) != 1 || book.Sources[0].Source != SourceRakutenBooks || book.Sources[0].AffiliateURL != "" {
		t.Fatalf("medium/sources = %q / %#v", book.Medium, book.Sources)
	}
}

// TestConvertItem_OmitsSeriesNameKana は、楽天Booksの系列名だけを出版系列として保持する
func TestConvertItem_OmitsSeriesNameKana(t *testing.T) {
	book := convertItem(booksItem{Title: "title", SeriesName: "叢書", SeriesNameKana: "ソウショ"}, "")
	if !slicesEqual(book.PublicationSeries, []string{"叢書"}) {
		t.Fatalf("PublicationSeries = %#v", book.PublicationSeries)
	}
}

// TestConvertItem_PreservesZeroPrice は、itemPriceの0円を欠落と区別して取得時点価格として保持することを確認する
func TestConvertItem_PreservesZeroPrice(t *testing.T) {
	zero := int64(0)
	book := convertItem(booksItem{Title: "free", ItemPrice: &zero}, "2026-08-08T02:00:00Z")
	if book.CurrentPrice == nil || book.CurrentPrice.Amount != 0 {
		t.Fatalf("price = %#v", book.CurrentPrice)
	}
}

// TestConvertItem_SelectsLargestAvailableCover は、楽天Booksの表紙候補から最大サイズを優先して選ぶ
func TestConvertItem_SelectsLargestAvailableCover(t *testing.T) {
	for _, test := range []struct {
		name string
		item booksItem
		want string
	}{
		{name: "large", item: booksItem{LargeImageURL: "large", MediumImageURL: "medium", SmallImageURL: "small"}, want: "large"},
		{name: "medium fallback", item: booksItem{MediumImageURL: "medium", SmallImageURL: "small"}, want: "medium"},
		{name: "small fallback", item: booksItem{SmallImageURL: "small"}, want: "small"},
		{name: "no image", item: booksItem{}, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := convertItem(test.item, "").CoverURL; got != test.want {
				t.Fatalf("CoverURL = %q, want %q", got, test.want)
			}
		})
	}
}

// TestISBN_SplitsValidatedValues は、楽天Booksの検証済みISBNを種類ごとに保持する
func TestISBN_SplitsValidatedValues(t *testing.T) {
	for _, test := range []struct {
		isbn       string
		wantISBN10 []string
		wantISBN13 []string
	}{
		{isbn: "0306406152", wantISBN10: []string{"0306406152"}},
		{isbn: "9784088466361", wantISBN13: []string{"9784088466361"}},
		{isbn: "9784088466362"},
	} {
		gotISBN10, gotISBN13 := isbn(test.isbn)
		if !slicesEqual(gotISBN10, test.wantISBN10) || !slicesEqual(gotISBN13, test.wantISBN13) {
			t.Fatalf("isbn(%q) = %#v / %#v, want %#v / %#v", test.isbn, gotISBN10, gotISBN13, test.wantISBN10, test.wantISBN13)
		}
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
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
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
	_, raw, err = client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
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
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
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
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
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

	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
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
		_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
		server.Close()
		assertErrorKind(t, err, test.kind)
	}
}

// TestBuildSearchResult_RejectsPageMismatch は、楽天Booksの応答ページが要求ページと異なる場合を拒否することを確認する
func TestBuildSearchResult_RejectsPageMismatch(t *testing.T) {
	response := booksResponse{Count: 2, Page: 1, PageCount: 2, Hits: 1, Items: []booksItem{{Title: "unexpected"}}}
	_, err := buildSearchResult(response, 2, buildSearchKey("title", "", "", ComicGenreGeneral, BookSizeAll), 1, "")
	if err == nil {
		t.Fatal("buildSearchResult() error = nil")
	}
}

// TestPaginationStopsAtPage100 は、楽天Booksの最大100ページで次カーソルを生成しないことを確認する
func TestPaginationStopsAtPage100(t *testing.T) {
	response := booksResponse{Count: 3000, Page: 100, PageCount: 100, Hits: 30, Items: []booksItem{{Title: "last"}}}
	result, err := buildSearchResult(response, 100, buildSearchKey("title", "", "", ComicGenreGeneral, BookSizeAll), 30, "")
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
	_, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
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
	if _, err := client.SearchBooks(context.Background(), SearchRequest{Title: "title"}); err != nil {
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
	_, err = client.SearchBooks(ctx, SearchRequest{Title: "title"})
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
	var classified *model.Error
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
