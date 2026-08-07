package googlebooks

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eamat-dot/manken/api"
)

// TestNewClient_ValidatesOptions は、APIキー、Option、endpointを外部通信前に検証することを確認する
func TestNewClient_ValidatesOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
	}{
		{name: "missing key"},
		{name: "blank key", options: []Option{WithAPIKey(" \t ")}},
		{name: "nil option", options: []Option{WithAPIKey("secret"), nil}},
		{name: "endpoint query", options: []Option{WithAPIKey("secret"), WithEndpoint("https://example.invalid/volumes?key=secret")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient(nil, test.options...)
			assertErrorKind(t, err, ErrorKindInvalidArgument)
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("error exposes API key: %v", err)
			}
		})
	}
}

// TestSearchBooks_BuildsSafeRequestAndCursor は、検索条件、固定パラメーター、カーソルを検証する
func TestSearchBooks_BuildsSafeRequestAndCursor(t *testing.T) {
	var requests []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.URL.Query())
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Query().Get("startIndex") == "0" {
			_, _ = writer.Write([]byte(`{"totalItems":2,"items":[` + sampleVolume("first", "9784088466361") + `]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"totalItems":2,"items":[` + sampleVolume("second", "9784088466361") + `]}`))
	}))
	defer server.Close()

	client, err := NewClient(nil, WithAPIKey("top-secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	request := SearchBooksRequest{Title: `a intitle:evil`, Author: `b"c`, FreeText: "d e", ExcludedText: `f -g`, Limit: 1}
	first, err := client.SearchBooks(context.Background(), request)
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(first.Books) != 1 || first.NextCursor == "" {
		t.Fatalf("first result = %#v", first)
	}
	if got, want := requests[0].Get("q"), `intitle:"a" intitle:"intitle:evil" inauthor:"b\"c" "d" "e" -"f" -"-g"`; got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
	for key, want := range map[string]string{"key": "top-secret", "printType": "books", "projection": "full", "orderBy": "relevance", "maxResults": "1"} {
		if got := requests[0].Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	request.Cursor = first.NextCursor
	second, err := client.SearchBooks(context.Background(), request)
	if err != nil {
		t.Fatalf("second SearchBooks() error = %v", err)
	}
	if len(second.Books) != 1 || second.NextCursor != "" || requests[1].Get("startIndex") != "1" {
		t.Fatalf("second result = %#v, startIndex = %q", second, requests[1].Get("startIndex"))
	}

	request.Limit = 2
	_, err = client.SearchBooks(context.Background(), request)
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooks_RejectsUnsupportedOrInvalidInput は、不正な検索入力を通信前に拒否することを確認する
func TestSearchBooks_RejectsUnsupportedOrInvalidInput(t *testing.T) {
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	for _, request := range []SearchBooksRequest{
		{}, {ExcludedText: "excluded"}, {Title: "title", Limit: 41}, {Title: "title", Cursor: "not a cursor"},
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

// TestSearchBooks_UsesDefaultLimitAndReturnsEmptyResult は、既定Limitと0件結果の形を確認する
func TestSearchBooks_UsesDefaultLimitAndReturnsEmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.Query().Get("maxResults"); got != "20" {
			t.Errorf("maxResults = %q, want 20", got)
		}
		_, _ = writer.Write([]byte(`{"totalItems":0}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.SearchBooks(context.Background(), SearchBooksRequest{FreeText: "keyword"})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if result.Books == nil || len(result.Books) != 0 || result.NextCursor != "" {
		t.Fatalf("result = %#v, want non-nil empty books and no cursor", result)
	}
}

// TestSearchBooksWithRawResponse_PreservesSuccessBody は、解析失敗時も成功本文を無加工で返すことを確認する
func TestSearchBooksWithRawResponse_PreservesSuccessBody(t *testing.T) {
	body := []byte(`{"items":`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(body)
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if !bytes.Equal(raw, body) {
		t.Fatalf("raw = %q, want %q", raw, body)
	}
}

// TestSearchBooksWithRawResponse_ReturnsUnchangedSuccessBody は、正常な変換時も成功本文を変更せず返すことを確認する
func TestSearchBooksWithRawResponse_ReturnsUnchangedSuccessBody(t *testing.T) {
	body := []byte(`{"totalItems":1,"items":[` + sampleVolume("volume-id", "9784088466361") + `]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(body)
	}))
	defer server.Close()

	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchBooksRequest{Title: "title"})
	if err != nil {
		t.Fatalf("SearchBooksWithRawResponse() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("books = %#v, want one book", result.Books)
	}
	if !bytes.Equal(raw, body) {
		t.Fatalf("raw = %q, want %q", raw, body)
	}
}

// TestSearchBooksWithRawResponse_DoesNotReturnOversizedBody は、本文上限超過時にraw本文を返さないことを確認する
func TestSearchBooksWithRawResponse_DoesNotReturnOversizedBody(t *testing.T) {
	body := []byte(strings.Repeat("x", successBodyMax+1))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(body)
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if raw != nil {
		t.Fatalf("raw = %d bytes, want nil", len(raw))
	}
}

// TestSearchBooks_ClassifiesHTTPErrorWithoutKey は、HTTPエラーがAPIキーを含まないことを確認する
func TestSearchBooks_ClassifiesHTTPErrorWithoutKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Retry-After", "10")
		writer.WriteHeader(http.StatusTooManyRequests)
		_, _ = writer.Write([]byte("top-secret"))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("top-secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUnavailable)
	if strings.Contains(err.Error(), "top-secret") {
		t.Fatalf("error exposes API key or response body: %v", err)
	}
	var classified *Error
	if !errors.As(err, &classified) || classified.StatusCode != http.StatusTooManyRequests || classified.RetryAfter.Seconds() != 10 {
		t.Fatalf("error = %#v", classified)
	}
}

// TestSearchBooks_TransportErrorDoesNotExposeKey は、通信エラーのURLからAPIキーが公開されないことを確認する
func TestSearchBooks_TransportErrorDoesNotExposeKey(t *testing.T) {
	cause := errors.New("dial failed")
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, cause
	})}
	client, err := NewClient(httpClient, WithAPIKey("top-secret"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUnavailable)
	if strings.Contains(err.Error(), "top-secret") {
		t.Fatalf("error exposes API key: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(err, cause) = false: %v", err)
	}
}

// TestSearchBooks_ClassifiesHTTPStatuses は、HTTPステータスを共通エラー分類へ対応付けることを確認する
func TestSearchBooks_ClassifiesHTTPStatuses(t *testing.T) {
	tests := []struct {
		name string
		code int
		kind ErrorKind
	}{
		{name: "request timeout", code: http.StatusRequestTimeout, kind: ErrorKindUnavailable},
		{name: "rate limited", code: http.StatusTooManyRequests, kind: ErrorKindUnavailable},
		{name: "server error", code: http.StatusInternalServerError, kind: ErrorKindUnavailable},
		{name: "bad request", code: http.StatusBadRequest, kind: ErrorKindUpstream},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.code)
			}))
			defer server.Close()
			client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}
			_, err = client.SearchBooks(context.Background(), SearchBooksRequest{Title: "title"})
			assertErrorKind(t, err, test.kind)
		})
	}
}

// TestSearchBooks_PreservesContextError は、キャンセル済みコンテキストをエラーチェーンから判定できることを確認する
func TestSearchBooks_PreservesContextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
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

// TestConvertVolume_MapsSupportedFieldsOnly は、安全に対応できるGoogle Books項目だけを変換することを確認する
func TestConvertVolume_MapsSupportedFieldsOnly(t *testing.T) {
	pageCount := 0
	book, err := convertVolume(volume{ID: "volume-id", VolumeInfo: volumeInfo{
		Title: "Title 1", Subtitle: "Subtitle", Authors: []string{"Author", ""}, Publisher: "Publisher", PublishedDate: "2026-08",
		IndustryIdentifiers: []industryIdentifier{{Type: "ISBN_10", Identifier: "4088466365"}, {Type: "ISBN_13", Identifier: "invalid"}},
		Description:         "<p>description</p>", Language: "ja", Categories: []string{"Manga", ""}, PageCount: &pageCount,
		CanonicalVolumeLink: "not a URL", InfoLink: "https://books.google.example/info",
		ImageLinks: imageLinks{Thumbnail: "https://images.example/thumbnail", Large: "https://images.example/large"},
	}})
	if err != nil {
		t.Fatalf("convertVolume() error = %v", err)
	}
	if book.Sources[0].Source != SourceGoogleBooks || book.Sources[0].URL != "https://books.google.example/info" ||
		book.Normalized.Title != "Title 1" || book.Normalized.PageCount != nil {
		t.Fatalf("book = %#v", book)
	}
	if len(book.Normalized.Contributors) != 1 || len(book.Normalized.Contributors[0].Roles) != 0 ||
		len(book.Normalized.Identifiers) != 1 || len(book.Normalized.Images) != 2 || book.Normalized.Volume.Label != "" || book.Normalized.Medium != "" {
		t.Fatalf("normalized = %#v", book.Normalized)
	}
	_, err = convertVolume(volume{})
	if err == nil {
		t.Fatal("convertVolume() error = nil, want missing ID error")
	}
}

// TestPositivePageCount は、0以下を不明値として捨て、正のページ数だけを保持することを確認する
func TestPositivePageCount(t *testing.T) {
	zero := 0
	negative := -1
	positive := 196
	for name, test := range map[string]struct {
		input *int
		want  *int
	}{
		"nil":      {input: nil, want: nil},
		"zero":     {input: &zero, want: nil},
		"negative": {input: &negative, want: nil},
		"positive": {input: &positive, want: &positive},
	} {
		t.Run(name, func(t *testing.T) {
			got := positivePageCount(test.input)
			if test.want == nil {
				if got != nil {
					t.Fatalf("positivePageCount() = %v, want nil", *got)
				}
				return
			}
			if got == nil || *got != *test.want {
				t.Fatalf("positivePageCount() = %v, want %d", got, *test.want)
			}
		})
	}
}

// TestLookupBooksByISBN_UsesSingleRequest は、1件のISBNを1回だけ問い合わせて一致するVolumeだけを返すことを確認する
func TestLookupBooksByISBN_UsesSingleRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		query := request.URL.Query()
		if query.Get("q") != "isbn:9784088466361" || query.Get("startIndex") != "0" || query.Get("maxResults") != "40" {
			t.Fatalf("query = %#v", query)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"totalItems":100,"items":[` +
			sampleVolume("match", "9784088466361") + `,` + sampleVolume("mismatch", "9780000000002") + `]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.LookupBooksByISBN(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if calls != 1 || len(result.Items) != 1 || result.Items[0].RequestedISBN != "4088466365" || len(result.Items[0].Books) != 1 {
		t.Fatalf("calls = %d, result = %#v", calls, result)
	}
	if result.Items[0].Books[0].Sources[0].ID != "match" {
		t.Fatalf("book = %#v", result.Items[0].Books[0])
	}

	_, err = client.LookupBooksByISBN(context.Background(), []string{"4088466365", "9784088466361"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	_, err = client.LookupBooksByISBN(context.Background(), []string{"bad-isbn"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestLookupBooksByISBNWithRawResponse_PreservesBody は、ISBN参照の1回の成功本文を無加工で返すことを確認する
func TestLookupBooksByISBNWithRawResponse_PreservesBody(t *testing.T) {
	body := []byte(`{"totalItems":1,"items":[` + sampleVolume("match", "9784088466361") + `]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(body)
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, raw, err := client.LookupBooksByISBNWithRawResponse(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if len(result.Items) != 1 || len(result.Items[0].Books) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if !bytes.Equal(raw, body) {
		t.Fatalf("raw = %q, want %q", raw, body)
	}
}

// TestLookupBooksByISBNWithRawResponse_ReturnsBodyOnDecodeError は、解析失敗時も読み込み済み成功本文を返すことを確認する
func TestLookupBooksByISBNWithRawResponse_ReturnsBodyOnDecodeError(t *testing.T) {
	body := []byte(`{"items":`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(body)
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, raw, err := client.LookupBooksByISBNWithRawResponse(context.Background(), []string{"4088466365"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if !bytes.Equal(raw, body) {
		t.Fatalf("raw = %q, want %q", raw, body)
	}
}

// TestLookupBooksByISBN_ReturnsNonNilEmptyBooks は、該当しないISBNをエラーにせず空結果として返すことを確認する
func TestLookupBooksByISBN_ReturnsNonNilEmptyBooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"totalItems":0}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.LookupBooksByISBN(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Books == nil || len(result.Items[0].Books) != 0 {
		t.Fatalf("result = %#v, want one item with non-nil empty books", result)
	}
}

// sampleVolume は、テスト用の最小Google Books Volume JSONを返す
func sampleVolume(id string, isbn string) string {
	return `{"id":"` + id + `","volumeInfo":{"title":"` + id + `","industryIdentifiers":[{"type":"ISBN_13","identifier":"` + isbn + `"}]}}`
}

// assertErrorKind は、エラーが指定された分類のErrorか検証する
func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var classified *api.Error
	if !errors.As(err, &classified) {
		t.Fatalf("error = %v, want *api.Error", err)
	}
	if classified.Kind != want {
		t.Fatalf("Error.Kind = %q, want %q", classified.Kind, want)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、関数として指定したHTTP往復処理を実行する
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
