package googlebooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/eamat-dot/manken/model"
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

// TestNewClient_ValidatesEndpointSecurity は、endpointのHTTPS要件と既存のURL制約を通信前に検証する
func TestNewClient_ValidatesEndpointSecurity(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{name: "https", endpoint: "https://example.com/volumes"},
		{name: "localhost HTTP", endpoint: "http://localhost:8080/volumes"},
		{name: "IPv4 loopback HTTP", endpoint: "http://127.0.0.2:8080/volumes"},
		{name: "IPv6 loopback HTTP", endpoint: "http://[::1]:8080/volumes"},
		{name: "non-loopback HTTP", endpoint: "http://example.com/volumes", wantErr: true},
		{name: "user information", endpoint: "https://user@example.com/volumes", wantErr: true},
		{name: "query", endpoint: "https://example.com/volumes?key=secret", wantErr: true},
		{name: "fragment", endpoint: "https://example.com/volumes#part", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(test.endpoint))
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
	request := SearchRequest{Title: `a intitle:evil`, Author: "b\"c\\d", Publisher: "c inpublisher:evil\"d", Query: "d e", Exclude: `f -g`, Limit: 1}
	first, err := client.SearchBooks(context.Background(), request)
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(first.Books) != 1 || first.NextCursor == "" {
		t.Fatalf("first result = %#v", first)
	}
	if got, want := requests[0].Get("q"), `intitle:"a" intitle:"intitle:evil" inauthor:"b\"c\\d" inpublisher:"c" inpublisher:"inpublisher:evil\"d" "d" "e" -"f" -"-g"`; got != want {
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

// TestSearchBooks_PublisherOnlyAndCursorMismatch は、出版社だけの検索と出版社が異なるカーソルの拒否を検証する
func TestSearchBooks_PublisherOnlyAndCursorMismatch(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if got, want := request.URL.Query().Get("q"), `inpublisher:"白泉社"`; got != want {
			t.Errorf("q = %q, want %q", got, want)
		}
		_, _ = writer.Write([]byte(`{"totalItems":2,"items":[` + sampleVolume("first", "9784088466361") + `]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
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

// TestSearchBooks_RejectsUnsupportedOrInvalidInput は、不正な検索入力を通信前に拒否することを確認する
func TestSearchBooks_RejectsUnsupportedOrInvalidInput(t *testing.T) {
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint("http://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	for _, request := range []SearchRequest{
		{}, {Exclude: "excluded"}, {Title: "title", Limit: 41}, {Title: "title", Cursor: "not a cursor"},
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
	result, err := client.SearchBooks(context.Background(), SearchRequest{Query: "keyword"})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if result.Books == nil || len(result.Books) != 0 || result.NextCursor != "" {
		t.Fatalf("result = %#v, want non-nil empty books and no cursor", result)
	}
}

// TestSearchBooks_RejectsNonAdvancingEmptyPage は、残り件数がある空ページを不正な応答として拒否することを確認する
func TestSearchBooks_RejectsNonAdvancingEmptyPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"totalItems":1,"items":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindInvalidResponse)
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
	_, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
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
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
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

// TestSearchBooks_MapsCurrentSalePriceWithObservedAt は、検索結果の販売価格に応答時刻を設定することを確認する
func TestSearchBooks_MapsCurrentSalePriceWithObservedAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"totalItems":1,"items":[` + sampleVolumeWithRetailPrice("volume-id", "9784088466361") + `]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIKey("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	result, err := client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("Books = %#v, want one book", result.Books)
	}
	assertCurrentPriceObservedAt(t, result.Books[0].CurrentPrice)
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
	_, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "title"})
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
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
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
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
	assertErrorKind(t, err, ErrorKindUnavailable)
	if strings.Contains(err.Error(), "top-secret") {
		t.Fatalf("error exposes API key: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(err, cause) = false: %v", err)
	}
}

// TestSearchBooks_DoesNotFollowRedirects は、APIキーを含むリクエストがredirect先へ送られないことを確認する
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
		Timeout:   5 * time.Second,
		Transport: http.DefaultTransport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			redirectCalls++
			return errors.New("caller redirect policy")
		},
	}
	client, err := NewClient(httpClient, WithAPIKey("top-secret"), WithEndpoint(redirect.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.httpClient == httpClient {
		t.Fatal("Client reused the caller HTTP client")
	}
	if httpClient.Timeout != 5*time.Second || httpClient.Transport != http.DefaultTransport || httpClient.CheckRedirect == nil {
		t.Fatalf("caller HTTP client was modified: %#v", httpClient)
	}
	if client.httpClient.Timeout != 5*time.Second || client.httpClient.Transport != http.DefaultTransport {
		t.Fatalf("Client HTTP settings = %#v, want copied timeout and transport", client.httpClient)
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
			_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "title"})
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
	_, err = client.SearchBooks(ctx, SearchRequest{Title: "title"})
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
		IndustryIdentifiers: []industryIdentifier{{Type: "ISBN_10", Identifier: "4088466365"}, {Type: "ISBN_13", Identifier: "9784088466361"}, {Type: "OTHER", Identifier: "unknown"}},
		Description:         "<p>description</p>", Language: "ja", Categories: []string{"Manga", ""}, PageCount: &pageCount,
		CanonicalVolumeLink: "not a URL", InfoLink: "https://books.google.example/info",
		ImageLinks: imageLinks{Thumbnail: "https://images.example/thumbnail", Large: "https://images.example/large"},
	}}, "")
	if err != nil {
		t.Fatalf("convertVolume() error = %v", err)
	}
	if book.Sources[0].Source != SourceGoogleBooks || book.Sources[0].URL != "https://books.google.example/info" ||
		book.Title != "Title 1" || book.PageCount != nil || book.Subtitle != "Subtitle" || book.PublishedDate != "2026-08" ||
		book.CoverURL != "https://images.example/large" {
		t.Fatalf("book = %#v", book)
	}
	if len(book.Contributors) != 1 || len(book.Contributors[0].Roles) != 0 || len(book.ISBN10) != 1 ||
		len(book.ISBN13) != 1 || book.Volume.Number == nil || *book.Volume.Number != 1 || book.Volume.Label != "1" || book.Medium != "" {
		t.Fatalf("book = %#v", book)
	}
	_, err = convertVolume(volume{}, "")
	if err == nil {
		t.Fatal("convertVolume() error = nil, want missing ID error")
	}
}

// TestConvertVolume_DoesNotUseSearchSnippet は、検索結果だけのtextSnippetを書誌Descriptionへ使わないことを確認する
func TestConvertVolume_DoesNotUseSearchSnippet(t *testing.T) {
	response, err := decodeVolumesResponse([]byte(`{
		"totalItems": 1,
		"items": [{
			"id": "volume-id",
			"volumeInfo": {"title": "書誌説明なし"},
			"searchInfo": {"textSnippet": "検索語に依存する断片"}
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	book, err := convertVolume(response.Items[0], "")
	if err != nil {
		t.Fatal(err)
	}
	if book.Description != "" {
		t.Fatalf("Description = %q", book.Description)
	}
}

// TestCoverURL は、Google Books画像候補から利用可能な最大サイズを選ぶことを確認する
func TestCoverURL(t *testing.T) {
	for _, test := range []struct {
		name  string
		links imageLinks
		want  string
	}{
		{name: "prefer extra large", links: imageLinks{Thumbnail: "thumbnail", Large: "large", ExtraLarge: "extra-large"}, want: "extra-large"},
		{name: "fall back to large", links: imageLinks{Thumbnail: "thumbnail", Large: "large"}, want: "large"},
		{name: "fall back to small thumbnail", links: imageLinks{SmallThumbnail: "small-thumbnail"}, want: "small-thumbnail"},
		{name: "omit missing images"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := coverURL(test.links); got != test.want {
				t.Fatalf("coverURL() = %q, want %q", got, test.want)
			}
		})
	}
}

// TestConvertVolume_MapsEbookMedium は、saleInfo.isEbookだけで電子書籍区分を変換することを確認する
func TestConvertVolume_MapsEbookMedium(t *testing.T) {
	tests := []struct {
		name string
		body string
		want PublicationMedium
	}{
		{name: "true", body: `{"totalItems":1,"items":[{"id":"ebook","saleInfo":{"isEbook":true}}]}`, want: PublicationMediumDigital},
		{name: "false", body: `{"totalItems":1,"items":[{"id":"not-ebook","saleInfo":{"isEbook":false}}]}`, want: PublicationMediumUnknown},
		{name: "omitted", body: `{"totalItems":1,"items":[{"id":"unknown"}]}`, want: PublicationMediumUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := decodeVolumesResponse([]byte(test.body))
			if err != nil {
				t.Fatalf("decodeVolumesResponse() error = %v", err)
			}
			book, err := convertVolume(response.Items[0], "")
			if err != nil {
				t.Fatalf("convertVolume() error = %v", err)
			}
			if book.Medium != test.want {
				t.Fatalf("Medium = %q, want %q", book.Medium, test.want)
			}
		})
	}
}

// TestConvertVolume_MapsSalePrices は、条件を満たす日本円の販売価格だけを変換することを確認する
func TestConvertVolume_MapsSalePrices(t *testing.T) {
	listAmount := json.Number("900.0")
	retailAmount := json.Number("8e2")
	zeroAmount := json.Number("0")
	fractionalAmount := json.Number("800.5")
	negativeAmount := json.Number("-1")
	maxInt64Amount := json.Number("9223372036854775807")
	overflowAmount := json.Number("9223372036854775808")
	observedAt := "2026-08-10T01:02:03.456789Z"
	tests := []struct {
		name                  string
		sale                  saleInfo
		wantList, wantCurrent *Price
	}{
		{
			name: "JP JPY integer list and retail",
			sale: saleInfo{
				Country:     "jp",
				ListPrice:   price{Amount: &listAmount, CurrencyCode: "jpy"},
				RetailPrice: price{Amount: &retailAmount, CurrencyCode: "JPY"},
			},
			wantList:    &Price{Amount: 900, Currency: "JPY", Source: SourceGoogleBooks},
			wantCurrent: &Price{Amount: 800, Currency: "JPY", Source: SourceGoogleBooks, ObservedAt: observedAt},
		},
		{
			name:     "explicit zero",
			sale:     saleInfo{Country: "JP", ListPrice: price{Amount: &zeroAmount, CurrencyCode: "JPY"}},
			wantList: &Price{Amount: 0, Currency: "JPY", Source: SourceGoogleBooks},
		},
		{
			name: "fractional amount",
			sale: saleInfo{Country: "JP", ListPrice: price{Amount: &fractionalAmount, CurrencyCode: "JPY"}},
		},
		{
			name: "negative amount",
			sale: saleInfo{Country: "JP", ListPrice: price{Amount: &negativeAmount, CurrencyCode: "JPY"}},
		},
		{
			name:     "int64 maximum",
			sale:     saleInfo{Country: "JP", ListPrice: price{Amount: &maxInt64Amount, CurrencyCode: "JPY"}},
			wantList: &Price{Amount: 9223372036854775807, Currency: "JPY", Source: SourceGoogleBooks},
		},
		{
			name: "amount exceeds int64",
			sale: saleInfo{Country: "JP", ListPrice: price{Amount: &overflowAmount, CurrencyCode: "JPY"}},
		},
		{
			name: "non JP country",
			sale: saleInfo{Country: "US", ListPrice: price{Amount: &listAmount, CurrencyCode: "JPY"}},
		},
		{
			name: "non JPY currency",
			sale: saleInfo{Country: "JP", ListPrice: price{Amount: &listAmount, CurrencyCode: "USD"}},
		},
		{
			name: "missing amount",
			sale: saleInfo{Country: "JP", ListPrice: price{CurrencyCode: "JPY"}},
		},
		{
			name: "missing currency",
			sale: saleInfo{Country: "JP", ListPrice: price{Amount: &listAmount}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			book, err := convertVolume(volume{ID: "volume-id", SaleInfo: test.sale}, observedAt)
			if err != nil {
				t.Fatalf("convertVolume() error = %v", err)
			}
			if !reflect.DeepEqual(book.ListPrice, test.wantList) || !reflect.DeepEqual(book.CurrentPrice, test.wantCurrent) {
				t.Fatalf("ListPrice = %#v, CurrentPrice = %#v; want %#v, %#v", book.ListPrice, book.CurrentPrice, test.wantList, test.wantCurrent)
			}
		})
	}
}

// TestDecodeVolumesResponse_PreservesLargePriceAmount は、2^53を超える価格を丸めずに変換することを確認する
func TestDecodeVolumesResponse_PreservesLargePriceAmount(t *testing.T) {
	response, err := decodeVolumesResponse([]byte(`{"totalItems":1,"items":[{"id":"volume-id","saleInfo":{"country":"JP","listPrice":{"amount":9007199254740993,"currencyCode":"JPY"}}}]}`))
	if err != nil {
		t.Fatalf("decodeVolumesResponse() error = %v", err)
	}
	book, err := convertVolume(response.Items[0], "")
	if err != nil {
		t.Fatalf("convertVolume() error = %v", err)
	}
	want := &Price{Amount: 9007199254740993, Currency: "JPY", Source: SourceGoogleBooks}
	if !reflect.DeepEqual(book.ListPrice, want) || book.CurrentPrice != nil {
		t.Fatalf("ListPrice = %#v, CurrentPrice = %#v; want %#v, nil", book.ListPrice, book.CurrentPrice, want)
	}
}

// TestPriceAmount_RejectsInvalidManualAmounts は、JSONとして現れない不正な金額も変換しないことを確認する
func TestPriceAmount_RejectsInvalidManualAmounts(t *testing.T) {
	for name, amount := range map[string]json.Number{
		"NaN":               "NaN",
		"positive infinity": "+Inf",
		"fraction notation": "2/1",
		"malformed":         "not-a-number",
	} {
		t.Run(name, func(t *testing.T) {
			_, ok := priceAmount(price{Amount: &amount, CurrencyCode: "JPY"})
			if ok {
				t.Fatalf("priceAmount(%q) succeeded, want rejection", amount)
			}
		})
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
			sampleVolumeWithRetailPrice("match", "9784088466361") + `,` + sampleVolume("mismatch", "9780000000002") + `]}`))
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
	assertCurrentPriceObservedAt(t, result.Items[0].Books[0].CurrentPrice)

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

// sampleVolumeWithRetailPrice は、販売価格を持つテスト用Google Books Volume JSONを返す
func sampleVolumeWithRetailPrice(id string, isbn string) string {
	return `{"id":"` + id + `","volumeInfo":{"title":"` + id + `","industryIdentifiers":[{"type":"ISBN_13","identifier":"` + isbn + `"}]},"saleInfo":{"country":"JP","retailPrice":{"amount":800,"currencyCode":"JPY"}}}`
}

// assertCurrentPriceObservedAt は、現在価格とRFC3339Nano形式の応答時刻を検証する
func assertCurrentPriceObservedAt(t *testing.T, currentPrice *Price) {
	t.Helper()
	if currentPrice == nil || currentPrice.Source != SourceGoogleBooks || currentPrice.ObservedAt == "" {
		t.Fatalf("CurrentPrice = %#v, want one current Google Books price with ObservedAt", currentPrice)
	}
	if _, err := time.Parse(time.RFC3339Nano, currentPrice.ObservedAt); err != nil {
		t.Fatalf("ObservedAt = %q, want RFC3339Nano: %v", currentPrice.ObservedAt, err)
	}
}

// assertErrorKind は、エラーが指定された分類のErrorか検証する
func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var classified *model.Error
	if !errors.As(err, &classified) {
		t.Fatalf("error = %v, want *model.Error", err)
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
