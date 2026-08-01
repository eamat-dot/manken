package madb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewClient_DefaultsAndOptions は、既定値と指定された設定を検証する
func TestNewClient_DefaultsAndOptions(t *testing.T) {
	defaultClient, err := NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient(nil) error = %v", err)
	}
	if defaultClient.endpoint != defaultEndpoint {
		t.Fatalf("endpoint = %q, want %q", defaultClient.endpoint, defaultEndpoint)
	}
	if defaultClient.httpClient.Timeout != 60*time.Second {
		t.Fatalf("Timeout = %s, want 60s", defaultClient.httpClient.Timeout)
	}

	customHTTPClient := &http.Client{Timeout: 3 * time.Second}
	customClient, err := NewClient(customHTTPClient, WithEndpoint("http://example.test/sparql"))
	if err != nil {
		t.Fatalf("NewClient(custom) error = %v", err)
	}
	if customClient.httpClient != customHTTPClient {
		t.Fatal("NewClient replaced the provided HTTP client")
	}
	if customHTTPClient.Timeout != 3*time.Second {
		t.Fatalf("provided Timeout changed to %s", customHTTPClient.Timeout)
	}
}

// TestNewClient_InvalidOptions は、不正なOptionとエンドポイントを拒否することを検証する
func TestNewClient_InvalidOptions(t *testing.T) {
	tests := []struct {
		name   string
		option Option
	}{
		{name: "nil option", option: nil},
		{name: "relative URL", option: WithEndpoint("/sparql")},
		{name: "unsupported scheme", option: WithEndpoint("ftp://example.test/sparql")},
		{name: "missing host", option: WithEndpoint("https:///sparql")},
		{name: "user information", option: WithEndpoint("https://user@example.test/sparql")},
		{name: "query", option: WithEndpoint("https://example.test/sparql?q=1")},
		{name: "fragment", option: WithEndpoint("https://example.test/sparql#x")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient(nil, test.option)
			assertErrorKind(t, err, ErrorKindInvalidArgument)

			var classified *Error
			if !errors.As(err, &classified) || classified.Operation != operationNewClient {
				t.Fatalf("error = %#v, want operation %q", err, operationNewClient)
			}
		})
	}
}

// TestClient_SearchBooks_RequestAndResult は、HTTPリクエストと正常な検索結果を検証する
func TestClient_SearchBooks_RequestAndResult(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.Header.Get("Accept") != "application/sparql-results+json" {
			t.Errorf("Accept = %q", request.Header.Get("Accept"))
		}
		if request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		receivedQuery = request.Form.Get("query")

		response := newSPARQLResponse(
			testBinding("M3", map[string]string{"title": "第三巻"}),
			testBinding("M1", map[string]string{
				"id":            "M1",
				"title":         "作品",
				"subtitle":      "副題",
				"seriesName":    "シリーズ",
				"volumeNumber":  "1",
				"agentName":     "著者",
				"publisher":     "出版社",
				"isbn":          "978-0-306-40615-7",
				"publishedDate": "2026-07",
			}),
			testBinding("M2", map[string]string{"title": "第二巻"}),
		)
		writeSPARQLResponse(t, writer, response)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	result, err := client.SearchBooks(context.Background(), SearchBooksRequest{
		Title:        " 作品 ",
		ISBN:         "978-4-08-846636-1",
		Author:       " 著者 ",
		FreeText:     " 新装版　B6判 ",
		ExcludedText: " 復刻版　愛蔵版 ",
		Limit:        2,
	})
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if !strings.Contains(receivedQuery, `neptune-fts:queryType "query_string"`) {
		t.Fatalf("query does not use query_string:\n%s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, `VALUES ?searchISBN { "4088466365" "9784088466361" }`) {
		t.Fatalf("query does not contain ISBN candidates:\n%s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "LIMIT 3") {
		t.Fatalf("query does not use Limit+1:\n%s", receivedQuery)
	}
	if len(result.Books) != 2 {
		t.Fatalf("len(Books) = %d, want 2", len(result.Books))
	}
	if len(result.Books[0].Sources) != 1 ||
		result.Books[0].Sources[0].ID != "M1" ||
		result.Books[0].Sources[0].Source != SourceMADB {
		t.Fatalf("first book = %#v", result.Books[0])
	}
	if result.Books[0].Sources[0].URL != resourceURI("M1") {
		t.Fatalf("source URL = %q", result.Books[0].Sources[0].URL)
	}
	if result.NextCursor == "" {
		t.Fatal("NextCursor is empty")
	}

	cursor, err := decodeCursor(result.NextCursor, searchConditions{
		Title:        "作品",
		ISBNs:        []string{"4088466365", "9784088466361"},
		Author:       "著者",
		FreeText:     "新装版 B6判",
		ExcludedText: "復刻版 愛蔵版",
	}, 2)
	if err != nil {
		t.Fatalf("decodeCursor() error = %v", err)
	}
	if cursor.After != resourceURI("M2") {
		t.Fatalf("cursor.After = %q", cursor.After)
	}
}

// TestValidateSearchRequest_ExcludedText は、除外語を正規化して正条件を必須にする
func TestValidateSearchRequest_ExcludedText(t *testing.T) {
	conditions, _, _, err := validateSearchRequest(SearchBooksRequest{
		Title:        " うる星 ",
		ExcludedText: "  復刻box　愛蔵版\n",
	})
	if err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
	if conditions.ExcludedText != "復刻box 愛蔵版" {
		t.Fatalf("ExcludedText = %q", conditions.ExcludedText)
	}

	for _, request := range []SearchBooksRequest{
		{ExcludedText: "復刻box"},
		{ExcludedText: "\u3000\t"},
	} {
		_, _, _, err := validateSearchRequest(request)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
}

// TestValidateSearchRequest_FreeTextOnly は、フリーワードだけの条件とUnicode空白を正規化する
func TestValidateSearchRequest_FreeTextOnly(t *testing.T) {
	conditions, limit, _, err := validateSearchRequest(SearchBooksRequest{
		FreeText: "  うる星　新装版\n",
	})
	if err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
	if conditions.FreeText != "うる星 新装版" {
		t.Fatalf("FreeText = %q", conditions.FreeText)
	}
	if conditions.Title != "" || len(conditions.ISBNs) != 0 || conditions.Author != "" {
		t.Fatalf("conditions = %#v, want free text only", conditions)
	}
	if limit != defaultLimit {
		t.Fatalf("limit = %d, want %d", limit, defaultLimit)
	}
}

// TestValidateSearchRequest_AuthorOnly は、著者名だけの条件とUnicode空白を正規化する
func TestValidateSearchRequest_AuthorOnly(t *testing.T) {
	conditions, limit, _, err := validateSearchRequest(SearchBooksRequest{
		Author: "  佐々木　倫子\n",
	})
	if err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
	if conditions.Author != "佐々木 倫子" {
		t.Fatalf("Author = %q", conditions.Author)
	}
	if conditions.Title != "" || len(conditions.ISBNs) != 0 {
		t.Fatalf("conditions = %#v, want author only", conditions)
	}
	if limit != defaultLimit {
		t.Fatalf("limit = %d, want %d", limit, defaultLimit)
	}
}

// TestClient_SearchBooksWithRawResponse_ResultAndRaw は、1回の通信から検索結果と受信本文を返すことを検証する
func TestClient_SearchBooksWithRawResponse_ResultAndRaw(t *testing.T) {
	const body = " {\r\n  \"head\": {\"vars\": [\"resource\"]},\r\n" +
		"  \"results\": {\"bindings\": []}\r\n}\n"
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requestCount++
		_, _ = io.WriteString(writer, body)
	}))
	defer server.Close()

	result, rawResponse, err := newTestClient(t, server.URL).SearchBooksWithRawResponse(
		context.Background(),
		SearchBooksRequest{Title: "作品"},
	)
	if err != nil {
		t.Fatalf("SearchBooksWithRawResponse() error = %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
	if string(rawResponse) != body {
		t.Fatalf("raw response = %q, want %q", rawResponse, body)
	}
	if result.Books == nil || len(result.Books) != 0 {
		t.Fatalf("Books = %#v, want non-nil empty slice", result.Books)
	}
}

// TestClient_SearchBooksWithRawResponse_ReturnsRawOnDecodeError は、不正JSONでも受信本文を返すことを検証する
func TestClient_SearchBooksWithRawResponse_ReturnsRawOnDecodeError(t *testing.T) {
	const body = `{"results":{"bindings":[`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, body)
	}))
	defer server.Close()

	_, rawResponse, err := newTestClient(t, server.URL).SearchBooksWithRawResponse(
		context.Background(),
		SearchBooksRequest{Title: "作品"},
	)
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if string(rawResponse) != body {
		t.Fatalf("raw response = %q, want %q", rawResponse, body)
	}
}

// TestClient_SearchBooksWithRawResponse_ReturnsRawOnConversionError は、binding変換失敗でも受信本文を返すことを検証する
func TestClient_SearchBooksWithRawResponse_ReturnsRawOnConversionError(t *testing.T) {
	const body = `{"results":{"bindings":[{"title":{"type":"literal","value":"作品"}}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, body)
	}))
	defer server.Close()

	_, rawResponse, err := newTestClient(t, server.URL).SearchBooksWithRawResponse(
		context.Background(),
		SearchBooksRequest{Title: "作品"},
	)
	assertErrorKind(t, err, ErrorKindInvalidResponse)
	if string(rawResponse) != body {
		t.Fatalf("raw response = %q, want %q", rawResponse, body)
	}
}

// TestClient_SearchBooksWithRawResponse_DoesNotReturnUnavailableBody は、上限超過とHTTPエラー本文を返さないことを検証する
func TestClient_SearchBooksWithRawResponse_DoesNotReturnUnavailableBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		kind ErrorKind
	}{
		{
			name: "success body limit",
			body: strings.Repeat("x", successBodyMax+1),
			code: http.StatusOK,
			kind: ErrorKindInvalidResponse,
		},
		{
			name: "HTTP error body",
			body: "sensitive upstream response",
			code: http.StatusBadGateway,
			kind: ErrorKindUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.code)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			_, rawResponse, err := newTestClient(t, server.URL).SearchBooksWithRawResponse(
				context.Background(),
				SearchBooksRequest{Title: "作品"},
			)
			assertErrorKind(t, err, test.kind)
			if rawResponse != nil {
				t.Fatalf("raw response = %q, want nil", rawResponse)
			}
		})
	}
}

// TestClient_SearchBooks_EmptyResult は、該当なしを空スライスとして返すことを検証する
func TestClient_SearchBooks_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeSPARQLResponse(t, writer, newSPARQLResponse())
	}))
	defer server.Close()

	result, err := newTestClient(t, server.URL).SearchBooks(
		context.Background(),
		SearchBooksRequest{ISBN: "9784088466361"},
	)
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if result.Books == nil || len(result.Books) != 0 {
		t.Fatalf("Books = %#v, want non-nil empty slice", result.Books)
	}
	if result.NextCursor != "" {
		t.Fatalf("NextCursor = %q, want empty", result.NextCursor)
	}
}

// TestClient_SearchBooks_ValidatesRequest は、不正な検索条件を通信前に拒否することを検証する
func TestClient_SearchBooks_ValidatesRequest(t *testing.T) {
	client, err := NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	tests := []SearchBooksRequest{
		{},
		{Title: "\u3000\t"},
		{Author: "\u3000\t"},
		{FreeText: "\u3000\t"},
		{ISBN: "4088466361"},
		{ISBN: "9784088466362"},
		{ISBN: "4901234567894"},
		{ISBN: "9784778031404 (set)"},
		{Title: "作品", Limit: -1},
		{Title: "作品", Limit: 101},
		{Title: "作品", Cursor: "not-base64"},
	}

	for _, request := range tests {
		_, err := client.SearchBooks(context.Background(), request)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}

	var nilClient *Client
	_, err = nilClient.SearchBooks(context.Background(), SearchBooksRequest{Title: "作品"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestValidateSearchRequest_ISBNOnly は、ISBNだけの検索条件を正規化することを検証する
func TestValidateSearchRequest_ISBNOnly(t *testing.T) {
	conditions, limit, _, err := validateSearchRequest(SearchBooksRequest{
		ISBN: " 978-4-08\u3000846636-1 ",
	})
	if err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
	if conditions.Title != "" {
		t.Fatalf("Title = %q, want empty", conditions.Title)
	}
	assertStrings(t, conditions.ISBNs, []string{"4088466365", "9784088466361"})
	if limit != defaultLimit {
		t.Fatalf("limit = %d, want %d", limit, defaultLimit)
	}
}

// TestValidateSearchRequest_NormalizesTitleConditions は、等価なUnicode空白を同じタイトル条件へ整形する
func TestValidateSearchRequest_NormalizesTitleConditions(t *testing.T) {
	conditions, _, _, err := validateSearchRequest(SearchBooksRequest{Title: "  うる星\u3000復刻box\n"})
	if err != nil {
		t.Fatalf("validateSearchRequest() error = %v", err)
	}
	if conditions.Title != "うる星 復刻box" {
		t.Fatalf("Title = %q", conditions.Title)
	}
}

// TestValidateSearchRequest_LimitBoundaries は、Limitの既定値、最小値、最大値を検証する
func TestValidateSearchRequest_LimitBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		input     int
		wantLimit int
	}{
		{name: "default", input: 0, wantLimit: defaultLimit},
		{name: "minimum", input: 1, wantLimit: 1},
		{name: "maximum", input: maxLimit, wantLimit: maxLimit},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, limit, _, err := validateSearchRequest(SearchBooksRequest{
				Title: "作品",
				Limit: test.input,
			})
			if err != nil {
				t.Fatalf("validateSearchRequest() error = %v", err)
			}
			if limit != test.wantLimit {
				t.Fatalf("limit = %d, want %d", limit, test.wantLimit)
			}
		})
	}
}

// TestClient_SearchBooks_UsesCursor は、次ページのクエリへ末尾URIを反映することを検証する
func TestClient_SearchBooks_UsesCursor(t *testing.T) {
	var queries []string
	var mutex sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		mutex.Lock()
		queries = append(queries, request.Form.Get("query"))
		call := len(queries)
		mutex.Unlock()

		if call == 1 {
			writeSPARQLResponse(t, writer, newSPARQLResponse(
				testBinding("M1", map[string]string{"title": "作品"}),
				testBinding("M2", map[string]string{"title": "作品"}),
			))
			return
		}
		writeSPARQLResponse(t, writer, newSPARQLResponse(
			testBinding("M2", map[string]string{"title": "作品"}),
		))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	first, err := client.SearchBooks(context.Background(), SearchBooksRequest{
		Title:        "作品",
		ISBN:         "9784088466361",
		ExcludedText: "復刻版",
		Limit:        1,
	})
	if err != nil {
		t.Fatalf("first SearchBooks() error = %v", err)
	}
	second, err := client.SearchBooks(context.Background(), SearchBooksRequest{
		Title:        "作品",
		ISBN:         "4088466365",
		ExcludedText: " 復刻版 ",
		Limit:        1,
		Cursor:       first.NextCursor,
	})
	if err != nil {
		t.Fatalf("second SearchBooks() error = %v", err)
	}
	if len(second.Books) != 1 || second.Books[0].Sources[0].URL != resourceURI("M2") {
		t.Fatalf("second result = %#v", second)
	}
	if !strings.Contains(queries[1], `FILTER (STR(?resource) > "`+resourceURI("M1")+`")`) {
		t.Fatalf("second query does not contain cursor filter:\n%s", queries[1])
	}
}

// TestClient_SearchBooks_ClassifiesHTTPError は、HTTPステータスとRetry-Afterを分類する
func TestClient_SearchBooks_ClassifiesHTTPError(t *testing.T) {
	tests := []struct {
		status     int
		retryAfter string
		kind       ErrorKind
		wantRetry  time.Duration
	}{
		{status: http.StatusBadRequest, kind: ErrorKindUpstream},
		{status: http.StatusRequestTimeout, kind: ErrorKindUnavailable},
		{
			status:     http.StatusTooManyRequests,
			retryAfter: "12",
			kind:       ErrorKindUnavailable,
			wantRetry:  12 * time.Second,
		},
		{status: http.StatusInternalServerError, kind: ErrorKindUnavailable},
	}

	for _, test := range tests {
		t.Run(strconv.Itoa(test.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Retry-After", test.retryAfter)
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, "sensitive upstream response")
			}))
			defer server.Close()

			_, err := newTestClient(t, server.URL).SearchBooks(
				context.Background(),
				SearchBooksRequest{Title: "作品"},
			)
			assertErrorKind(t, err, test.kind)

			var classified *Error
			if !errors.As(err, &classified) {
				t.Fatalf("error = %T, want *Error", err)
			}
			if classified.StatusCode != test.status || classified.RetryAfter != test.wantRetry {
				t.Fatalf("classified error = %#v", classified)
			}
			if strings.Contains(err.Error(), "sensitive") {
				t.Fatalf("error exposes response body: %v", err)
			}
		})
	}
}

// TestParseRetryAfter は、秒数形式、HTTP-date形式、不正値の解析を検証する
func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC)
	value := now.Add(30 * time.Second).Format(http.TimeFormat)
	if got := parseRetryAfter(value, now); got != 30*time.Second {
		t.Fatalf("parseRetryAfter() = %s, want 30s", got)
	}
	maxSeconds := strconv.FormatInt(maxRetryAfterSeconds, 10)
	maxDuration := time.Duration(maxRetryAfterSeconds) * time.Second
	if got := parseRetryAfter(maxSeconds, now); got != maxDuration {
		t.Fatalf("parseRetryAfter(max seconds) = %s, want %s", got, maxDuration)
	}
	overflowSeconds := strconv.FormatInt(maxRetryAfterSeconds+1, 10)
	if got := parseRetryAfter(overflowSeconds, now); got != 0 {
		t.Fatalf("parseRetryAfter(overflow seconds) = %s, want 0", got)
	}
	if got := parseRetryAfter("invalid", now); got != 0 {
		t.Fatalf("parseRetryAfter(invalid) = %s, want 0", got)
	}
	if got := parseRetryAfter("-1", now); got != 0 {
		t.Fatalf("parseRetryAfter(-1) = %s, want 0", got)
	}
}

// TestClient_SearchBooks_InvalidResponse は、不正JSONと本文上限超過を分類する
func TestClient_SearchBooks_InvalidResponse(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: "{"},
		{name: "missing results", body: `{}`},
		{name: "missing bindings", body: `{"results":{}}`},
		{name: "trailing JSON", body: `{"results":{"bindings":[]}} {}`},
		{name: "body limit", body: strings.Repeat("x", successBodyMax+1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			_, err := newTestClient(t, server.URL).SearchBooks(
				context.Background(),
				SearchBooksRequest{Title: "作品"},
			)
			assertErrorKind(t, err, ErrorKindInvalidResponse)
		})
	}
}

// TestClient_SearchBooks_PreservesContextError は、キャンセルと期限超過をerrors.Is可能に保つ
func TestClient_SearchBooks_PreservesContextError(t *testing.T) {
	tests := []struct {
		name  string
		cause error
	}{
		{name: "canceled", cause: context.Canceled},
		{name: "deadline", cause: context.DeadlineExceeded},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return nil, test.cause
			})}
			client, err := NewClient(httpClient)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			_, err = client.SearchBooks(
				context.Background(),
				SearchBooksRequest{Title: "作品"},
			)
			assertErrorKind(t, err, ErrorKindUnavailable)
			if !errors.Is(err, test.cause) {
				t.Fatalf("errors.Is(%v) = false", test.cause)
			}
		})
	}
}

// roundTripFunc は、関数をhttp.RoundTripperとして使用できるようにする
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、設定された関数でHTTPリクエストを処理する
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// newTestClient は、テストサーバーを使用するClientを生成する
func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	client, err := NewClient(nil, WithEndpoint(endpoint))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

// newSPARQLResponse は、指定bindingを含むテスト用レスポンスを生成する
func newSPARQLResponse(bindings ...map[string]sparqlValue) sparqlResponse {
	if bindings == nil {
		bindings = make([]map[string]sparqlValue, 0)
	}
	return sparqlResponse{
		Results: &sparqlResults{Bindings: bindings},
	}
}

// testBinding は、MADB IDと文字列項目からテスト用bindingを生成する
func testBinding(id string, fields map[string]string) map[string]sparqlValue {
	binding := map[string]sparqlValue{
		"resource": {Type: "uri", Value: resourceURI(id)},
	}
	for name, value := range fields {
		binding[name] = sparqlValue{Type: "literal", Value: value}
	}
	return binding
}

// resourceURI は、MADB IDからテスト用リソースURIを生成する
func resourceURI(id string) string {
	return "https://mediaarts-db.artmuseums.go.jp/id/" + id
}

// writeSPARQLResponse は、テスト用SPARQLレスポンスをJSONで書き込む
func writeSPARQLResponse(t *testing.T, writer http.ResponseWriter, response sparqlResponse) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/sparql-results+json")
	if err := json.NewEncoder(writer).Encode(response); err != nil {
		t.Errorf("Encode() error = %v", err)
	}
}

// assertErrorKind は、エラーが指定された分類のErrorか検証する
func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want kind %q", want)
	}
	var classified *Error
	if !errors.As(err, &classified) {
		t.Fatalf("error = %T, want *Error", err)
	}
	if classified.Kind != want {
		t.Fatalf("Kind = %q, want %q: %v", classified.Kind, want, err)
	}
}

// TestValidateEndpoint_AcceptsHTTPAndHTTPS は、有効なHTTP URLを受け付けることを検証する
func TestValidateEndpoint_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, endpoint := range []string{"http://example.test", "https://example.test/sparql"} {
		if err := validateEndpoint(endpoint); err != nil {
			t.Fatalf("validateEndpoint(%q) error = %v", endpoint, err)
		}
	}
}

// TestFormEncoding_RoundTrip は、SPARQLクエリがフォーム値として欠落なく復元できることを検証する
func TestFormEncoding_RoundTrip(t *testing.T) {
	query := buildSearchQuery(searchConditions{Title: `a+b & c`}, 20, "")
	encoded := url.Values{"query": []string{query}}.Encode()
	decoded, err := url.ParseQuery(encoded)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}
	if decoded.Get("query") != query {
		t.Fatal("form encoding changed the query")
	}
}
