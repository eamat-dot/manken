package rakutenkobo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/eamat-dot/manken/model"
)

// TestDecodeKoboResponse_Fixture は、formatVersion=2のItems直下形式をdecodeできることを確認する
func TestDecodeKoboResponse_Fixture(t *testing.T) {
	body, err := os.ReadFile("testdata/format-version-2.json")
	if err != nil {
		t.Fatal(err)
	}
	response, err := decodeKoboResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || response.Items[0].ItemNumber != "test-item-001" || response.Items[0].Title == "" || response.Items[0].Author == "" {
		t.Fatalf("response = %#v", response)
	}
}

// TestNewClient_ValidatesOptions は、認証、漫画区分、Optionを通信前に検証することを確認する
func TestNewClient_ValidatesOptions(t *testing.T) {
	for _, options := range [][]Option{
		nil,
		{WithApplicationID("app-secret")},
		{WithAccessKey("access-secret")},
		{WithApplicationID("app-secret"), WithAccessKey("access\nsecret")},
		{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID(" \t ")},
		{WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithComicGenre("unknown")},
		{WithApplicationID("app-secret"), WithAccessKey("access-secret"), nil},
	} {
		_, err := NewClient(nil, options...)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
}

// TestNewClient_ValidatesEndpointSecurity は、endpointのHTTPS要件とURL制約を通信前に検証する
func TestNewClient_ValidatesEndpointSecurity(t *testing.T) {
	for _, test := range []struct {
		endpoint string
		wantErr  bool
	}{
		{"https://example.com/kobo", false}, {"http://localhost:8080/kobo", false}, {"http://127.0.0.2:8080/kobo", false}, {"http://[::1]:8080/kobo", false},
		{"http://example.com/kobo", true}, {"https://user@example.com/kobo", true}, {"https://example.com/kobo?x=1", true}, {"https://example.com/kobo#part", true},
	} {
		_, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(test.endpoint))
		if test.wantErr {
			assertErrorKind(t, err, ErrorKindInvalidArgument)
		} else if err != nil {
			t.Fatalf("NewClient(%q) error = %v", test.endpoint, err)
		}
	}
}

// TestSearchBooks_UsesComicGenreID は、各漫画区分をKoboのジャンルIDへ設定することを確認する
func TestSearchBooks_UsesComicGenreID(t *testing.T) {
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		queries = append(queries, request.URL.Query())
		_, _ = writer.Write([]byte(sampleResponse(0, 0, 0)))
	}))
	defer server.Close()
	for _, test := range []struct {
		genre ComicGenre
		want  string
	}{{ComicGenreGeneral, rakutenKoboGenreGeneralComic}, {ComicGenreBL, rakutenKoboGenreBLComic}, {ComicGenreTL, rakutenKoboGenreTLComic}} {
		client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithComicGenre(test.genre), WithEndpoint(server.URL))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.SearchBooks(context.Background(), SearchRequest{Title: "title"}); err != nil {
			t.Fatal(err)
		}
		if got := queries[len(queries)-1].Get("koboGenreId"); got != test.want {
			t.Errorf("genre %q = %q, want %q", test.genre, got, test.want)
		}
	}
}

// TestSearchBooks_MapsRequest は、検索条件と固定パラメーターをKobo APIへ対応付ける
func TestSearchBooks_MapsRequest(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		if r.Header.Get("accessKey") != "access-secret" {
			t.Error("missing access key")
		}
		_, _ = w.Write([]byte(`{"count":0,"page":1,"pageCount":0,"hits":1,"Items":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID("affiliate"), WithComicGenre(ComicGenreBL), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: " title ", Author: " author ", Publisher: " publisher ", Query: " keyword ", Exclude: " exclude ", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"title": "title", "author": "author", "publisherName": "publisher", "keyword": "keyword", "NGKeyword": "exclude", "koboGenreId": rakutenKoboGenreBLComic, "format": "json", "formatVersion": "2", "sort": "+releaseDate", "language": "JA", "field": "0", "orFlag": "0", "hits": "1", "page": "1", "applicationId": "app-secret", "affiliateId": "affiliate"} {
		if value := got.Get(key); value != want {
			t.Errorf("%s = %q, want %q", key, value, want)
		}
	}
}

// TestSearchBooks_ExcludeOnly は、除外語だけの検索を通信前に拒否する
func TestSearchBooks_ExcludeOnly(t *testing.T) {
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Exclude: "exclude"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooks_ExcludeRequiresKeyword は、除外語を含む検索で上流のkeyword要件を満たすことを確認する
func TestSearchBooks_ExcludeRequiresKeyword(t *testing.T) {
	var queries []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		queries = append(queries, query)
		if query.Get("NGKeyword") != "exclude" {
			t.Errorf("NGKeyword = %q", query.Get("NGKeyword"))
		}
		_, _ = writer.Write([]byte(sampleResponse(0, 0, 0)))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		request SearchRequest
		want    map[string]string
	}{
		{name: "free text wins", request: SearchRequest{Title: "title", Author: "author", Publisher: "publisher", Query: "free", Exclude: "exclude"}, want: map[string]string{"title": "title", "author": "author", "publisherName": "publisher", "keyword": "free"}},
		{name: "title beats author and publisher", request: SearchRequest{Title: "title", Author: "author", Publisher: "publisher", Exclude: "exclude"}, want: map[string]string{"title": "title", "author": "author", "publisherName": "publisher", "keyword": "title"}},
		{name: "author beats publisher", request: SearchRequest{Author: "author", Publisher: "publisher", Exclude: "exclude"}, want: map[string]string{"author": "author", "publisherName": "publisher", "keyword": "author"}},
		{name: "publisher fallback", request: SearchRequest{Publisher: "publisher", Exclude: "exclude"}, want: map[string]string{"publisherName": "publisher", "keyword": "publisher"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := len(queries)
			if _, err := client.SearchBooks(context.Background(), test.request); err != nil {
				t.Fatal(err)
			}
			if len(queries) != before+1 {
				t.Fatalf("requests = %d, want %d", len(queries), before+1)
			}
			for key, want := range test.want {
				if got := queries[before].Get(key); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
	for _, request := range []SearchRequest{{Author: "author"}, {Publisher: "publisher"}} {
		values := searchValues(request.Title, request.Author, request.Publisher, request.Query, request.Exclude, rakutenKoboGenreGeneralComic, 1, 1)
		if got := values.Get("keyword"); got != "" {
			t.Errorf("keyword = %q, want empty", got)
		}
	}
}

// TestConvertItem_PreservesKoboRules は、Kobo固有IDと販売情報をISBNへ推測せず変換する
func TestConvertItem_PreservesKoboRules(t *testing.T) {
	price := int64(880)
	book := convertItem(koboItem{Title: "薬屋のひとりごと〜猫猫の後宮謎解き手帳〜（１）", SeriesName: "電子叢書", Author: " A / / B ", AuthorKana: "a,b/a,b", ItemNumber: "item-1", SalesDate: "2026年8月10日", ItemPrice: &price, ItemURL: "https://example.com/item", AffiliateURL: "https://example.com/affiliate", KoboGenreID: "101/102", SmallImageURL: "https://example.com/s.jpg"}, "2026-08-10T00:00:00Z")
	if book.Title != "薬屋のひとりごと〜猫猫の後宮謎解き手帳〜（１）" || book.Medium != PublicationMediumDigital || len(book.ISBN10) != 0 || len(book.ISBN13) != 0 {
		t.Fatalf("book = %#v", book)
	}
	if book.Volume.Number == nil || *book.Volume.Number != 1 || book.Volume.Label != "1" {
		t.Fatalf("volume = %#v", book.Volume)
	}
	if len(book.PublicationSeries) != 1 || book.PublicationSeries[0] != "電子叢書" {
		t.Fatalf("PublicationSeries = %#v", book.PublicationSeries)
	}
	if len(book.Sources) != 1 || book.Sources[0].ID != "item-1" || book.Sources[0].AffiliateURL == "" {
		t.Fatalf("sources = %#v", book.Sources)
	}
	if got := book.Authors; len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatalf("authors = %#v", got)
	}
	for _, contributor := range book.Contributors {
		if contributor.Reading != "" {
			t.Fatalf("contributor reading = %#v", contributor)
		}
	}
	if book.CurrentPrice == nil || book.CurrentPrice.Source != SourceRakutenKobo || book.CurrentPrice.Amount != 880 || book.CurrentPrice.Currency != "JPY" || book.CurrentPrice.TaxIncluded == nil || !*book.CurrentPrice.TaxIncluded {
		t.Fatalf("current price = %#v", book.CurrentPrice)
	}
	if book.ReleaseDate != "2026年8月10日" || len(book.Subjects) != 2 || book.CoverURL != "https://example.com/s.jpg" || book.Sources[0].URL != "https://example.com/item" || book.Sources[0].AffiliateURL != "https://example.com/affiliate" {
		t.Fatalf("converted book = %#v", book)
	}
}

// TestConvertItem_MapsSafeAuthorReadings は、安全に検証できる著者読みを変換することを確認する
func TestConvertItem_MapsSafeAuthorReadings(t *testing.T) {
	book := convertItem(koboItem{Author: "著者A/著者B/著者C", AuthorKana: "チョシャエー,チョシャビー,チョシャシー/チョシャエー,チョシャビー,チョシャシー/チョシャエー,チョシャビー,チョシャシー"}, "")
	wantNames := []string{"著者A", "著者B", "著者C"}
	wantReadings := []string{"チョシャエー", "チョシャビー", "チョシャシー"}
	if len(book.Authors) != len(wantNames) || len(book.Contributors) != len(wantNames) {
		t.Fatalf("authors/contributors = %#v / %#v", book.Authors, book.Contributors)
	}
	for index := range wantNames {
		if book.Authors[index] != wantNames[index] || book.Contributors[index].Name != wantNames[index] || book.Contributors[index].Reading != wantReadings[index] {
			t.Fatalf("author/contributor at %d = %q / %#v", index, book.Authors[index], book.Contributors[index])
		}
	}
}

// TestContributors_MapsOnlyValidatedKoboAuthorKana は、検証済みのKobo形式だけへ読みを設定することを確認する
func TestContributors_MapsOnlyValidatedKoboAuthorKana(t *testing.T) {
	for _, test := range []struct {
		name         string
		author       string
		authorKana   string
		wantAuthors  []string
		wantReadings []string
	}{
		{name: "single author", author: " 佐々木倫子 ", authorKana: " ササキノリコ ", wantAuthors: []string{"佐々木倫子"}, wantReadings: []string{"ササキノリコ"}},
		{name: "three author repeated list", author: "著者A/著者B/著者C", authorKana: "チョシャエー,チョシャビー,チョシャシー/チョシャエー,チョシャビー,チョシャシー/チョシャエー,チョシャビー,チョシャシー", wantAuthors: []string{"著者A", "著者B", "著者C"}, wantReadings: []string{"チョシャエー", "チョシャビー", "チョシャシー"}},
		{name: "kana slash count mismatch", author: "A/B", authorKana: "エー,ビー/エー,ビー/エー,ビー", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "kana slash segments differ", author: "A/B", authorKana: "エー,ビー/エー,シー", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "kana comma count mismatch", author: "A/B", authorKana: "エー,ビー,シー/エー,ビー,シー", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "empty author segment", author: "A//B", authorKana: "エー,ビー/エー,ビー/エー,ビー", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "empty kana slash segment", author: "A/B", authorKana: "エー,ビー//エー,ビー", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "empty kana comma component", author: "A/B", authorKana: "エー,/エー,", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "unobserved direct slash readings", author: "A/B", authorKana: "エー/ビー", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
		{name: "blank author kana", author: "A/B", authorKana: " ", wantAuthors: []string{"A", "B"}, wantReadings: []string{"", ""}},
	} {
		t.Run(test.name, func(t *testing.T) {
			authors, values := contributors(test.author, test.authorKana)
			if len(authors) != len(test.wantAuthors) || len(values) != len(test.wantAuthors) {
				t.Fatalf("authors/contributors = %#v / %#v", authors, values)
			}
			for index := range test.wantAuthors {
				if authors[index] != test.wantAuthors[index] || values[index].Name != test.wantAuthors[index] || values[index].Reading != test.wantReadings[index] {
					t.Fatalf("author/contributor at %d = %q / %#v", index, authors[index], values[index])
				}
			}
		})
	}
}

// TestConvertItem_PreservesZeroPrice は、0円の価格を欠落と区別して保持することを確認する
func TestConvertItem_PreservesZeroPrice(t *testing.T) {
	zero := int64(0)
	book := convertItem(koboItem{Title: "free", ItemPrice: &zero}, "")
	if book.CurrentPrice == nil || book.CurrentPrice.Amount != 0 {
		t.Fatalf("price = %#v", book.CurrentPrice)
	}
}

// TestConvertItem_SelectsLargestAvailableCover は、楽天Koboの表紙候補から最大サイズを優先して選ぶ
func TestConvertItem_SelectsLargestAvailableCover(t *testing.T) {
	for _, test := range []struct {
		name string
		item koboItem
		want string
	}{
		{name: "large", item: koboItem{LargeImageURL: "large", MediumImageURL: "medium", SmallImageURL: "small"}, want: "large"},
		{name: "medium fallback", item: koboItem{MediumImageURL: "medium", SmallImageURL: "small"}, want: "medium"},
		{name: "small fallback", item: koboItem{SmallImageURL: "small"}, want: "small"},
		{name: "no image", item: koboItem{}, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := convertItem(test.item, "").CoverURL; got != test.want {
				t.Fatalf("CoverURL = %q, want %q", got, test.want)
			}
		})
	}
}

// TestCursor_BindsSearchConditions は、カーソルを検索条件と件数へ関連付ける
func TestCursor_BindsSearchConditions(t *testing.T) {
	cursor, err := encodeCursor(2, buildSearchKey("title", "", "", "", "", string(ComicGenreGeneral)), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCursor(cursor, buildSearchKey("other", "", "", "", "", string(ComicGenreGeneral)), 1); err == nil {
		t.Fatal("cursor mismatch was accepted")
	}
}

// TestSearchBooks_PaginatesAndBindsCursor は、次ページと検索条件に拘束されたカーソルを確認する
func TestSearchBooks_PaginatesAndBindsCursor(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		page := request.URL.Query().Get("page")
		pages = append(pages, page)
		_, _ = writer.Write([]byte(sampleResponse(2, mustAtoi(t, page), 2, sampleItem(page))))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.SearchBooks(context.Background(), SearchRequest{Title: "title", Limit: 1})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first = %#v, %v", first, err)
	}
	second, err := client.SearchBooks(context.Background(), SearchRequest{Title: "title", Limit: 1, Cursor: first.NextCursor})
	if err != nil || second.NextCursor != "" || len(pages) != 2 || pages[1] != "2" {
		t.Fatalf("second/pages = %#v / %#v / %v", second, pages, err)
	}
	for _, request := range []SearchRequest{{Title: "other", Limit: 1, Cursor: first.NextCursor}, {Title: "title", Limit: 2, Cursor: first.NextCursor}} {
		_, err := client.SearchBooks(context.Background(), request)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
	tlClient, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithComicGenre(ComicGenreTL), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tlClient.SearchBooks(context.Background(), SearchRequest{Title: "title", Limit: 1, Cursor: first.NextCursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooks_DefaultLimitAndInvalidInputs は、既定Limit、0件、通信前入力検証を確認する
func TestSearchBooks_DefaultLimitAndInvalidInputs(t *testing.T) {
	var hits string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits = request.URL.Query().Get("hits")
		_, _ = writer.Write([]byte(sampleResponse(0, 0, 0)))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.SearchBooks(context.Background(), SearchRequest{Title: "none"})
	if err != nil || hits != "20" || result.Books == nil || len(result.Books) != 0 {
		t.Fatalf("result/hits = %#v / %q / %v", result, hits, err)
	}
	for _, request := range []SearchRequest{{}, {Title: "title", Limit: 31}} {
		_, err := client.SearchBooks(context.Background(), request)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
	_, err = (*Client)(nil).SearchBooks(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	var nilContext context.Context
	_, err = client.SearchBooks(nilContext, SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestNewClient_SanitizesSecrets は、初期化エラーが認証情報を露出しないことを確認する
func TestNewClient_SanitizesSecrets(t *testing.T) {
	_, err := NewClient(nil, WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithEndpoint("https://example.invalid/?applicationId=secret"))
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	if strings.Contains(err.Error(), "app-secret") || strings.Contains(err.Error(), "access-secret") {
		t.Fatalf("error exposes secret: %v", err)
	}
}

// TestSearchBooks_StopsRedirects は、リダイレクト先へ認証情報を送らないことを確認する
func TestSearchBooks_StopsRedirects(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirected = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, target.URL, http.StatusFound)
	}))
	defer source.Close()
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(source.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUpstream)
	if redirected {
		t.Fatal("redirect target was requested")
	}
}

// TestSearchBooks_RawResponseAndHTTPFailures は、Raw response保持、本文上限、HTTP分類を確認する
func TestSearchBooks_RawResponseAndHTTPFailures(t *testing.T) {
	valid := []byte(sampleResponse(1, 1, 1, sampleItem("one")))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write(valid) }))
	client, err := NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
	if err != nil || len(result.Books) != 1 || !bytes.Equal(raw, valid) {
		t.Fatalf("result/raw/error = %#v / %q / %v", result, raw, err)
	}
	server.Close()

	invalid := []byte(`{"Items":`)
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write(invalid) }))
	defer server.Close()
	client, err = NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, raw, err = client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if !bytes.Equal(raw, invalid) {
		t.Fatalf("raw = %q, want %q", raw, invalid)
	}

	server.Close()
	invalidResponse := []byte(sampleResponse(1, 0, 2, sampleItem("wrong-page")))
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(invalidResponse)
	}))
	client, err = NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, raw, err = client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if !bytes.Equal(raw, invalidResponse) {
		t.Fatalf("invalid-response raw = %q, want %q", raw, invalidResponse)
	}

	server.Close()
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("x", successBodyMax+1)))
	}))
	defer server.Close()
	client, err = NewClient(nil, WithApplicationID("app"), WithAccessKey("access"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, raw, err = client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if raw != nil {
		t.Fatalf("oversized raw = %d bytes, want nil", len(raw))
	}
}

// TestHTTPAndTransportErrorsDoNotExposeSecrets は、HTTP分類、Retry-After、通信エラーの秘匿を確認する
func TestHTTPAndTransportErrorsDoNotExposeSecrets(t *testing.T) {
	for _, test := range []struct {
		code int
		kind ErrorKind
	}{{http.StatusTooManyRequests, ErrorKindUnavailable}, {http.StatusInternalServerError, ErrorKindUnavailable}} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Retry-After", "10")
			writer.WriteHeader(test.code)
			_, _ = writer.Write([]byte("app-secret access-secret affiliate-secret"))
		}))
		client, err := NewClient(nil, WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID("affiliate-secret"), WithEndpoint(server.URL))
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
		server.Close()
		assertErrorKind(t, err, test.kind)
		var classified *Error
		if !errors.As(err, &classified) || classified.StatusCode != test.code || classified.RetryAfter.Seconds() != 10 {
			t.Fatalf("classified error = %#v", classified)
		}
		for _, secret := range []string{"app-secret", "access-secret", "affiliate-secret"} {
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error exposes %q: %v", secret, err)
			}
		}
	}
	cause := errors.New("dial failed")
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, cause })}
	client, err := NewClient(httpClient, WithApplicationID("app-secret"), WithAccessKey("access-secret"), WithAffiliateID("affiliate-secret"))
	if err != nil {
		t.Fatal(err)
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

// sampleResponse は、単体テスト用の楽天KoboレスポンスJSONを生成する
func sampleResponse(count int, page int, pageCount int, items ...string) string {
	return fmt.Sprintf(`{"count":%d,"page":%d,"pageCount":%d,"hits":%d,"Items":[%s]}`, count, page, pageCount, len(items), strings.Join(items, ","))
}

// sampleItem は、単体テスト用の楽天Kobo商品JSONを生成する
func sampleItem(title string) string {
	return fmt.Sprintf(`{"title":%q,"author":"著者A/著者B","itemNumber":%q}`, title, "item-"+title)
}

// mustAtoi は、テスト用のページ文字列を整数へ変換する
func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	result, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v", value, err)
	}
	return result
}

// roundTripFunc は、関数をhttp.RoundTripperとして使用できるようにする
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、設定された関数でHTTPリクエストを処理する
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// assertErrorKind は、エラーが期待する共通分類を持つことを確認する
func assertErrorKind(t *testing.T, err error, want model.ErrorKind) {
	t.Helper()
	var actual *model.Error
	if !errors.As(err, &actual) || actual.Kind != want {
		t.Fatalf("error = %v, want kind %q", err, want)
	}
}
