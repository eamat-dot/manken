package openbd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
	customClient, err := NewClient(customHTTPClient, WithEndpoint("http://example.test/get"))
	if err != nil {
		t.Fatalf("NewClient(custom) error = %v", err)
	}
	if customClient.httpClient != customHTTPClient {
		t.Fatal("NewClient replaced the provided HTTP client")
	}
	if customClient.endpoint != "http://example.test/get" {
		t.Fatalf("endpoint = %q", customClient.endpoint)
	}
}

// TestNewClient_InvalidOptions は、不正なOptionとエンドポイントを拒否することを検証する
func TestNewClient_InvalidOptions(t *testing.T) {
	tests := []Option{
		nil,
		WithEndpoint(""),
		WithEndpoint("ftp://example.test/get"),
		WithEndpoint("https://user@example.test/get"),
		WithEndpoint("https://example.test/get?isbn=x"),
		WithEndpoint("https://example.test/get#fragment"),
	}
	for _, option := range tests {
		_, err := NewClient(nil, option)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}
}

// TestClient_LookupBooksByISBN_RequestAndRaw は、ISBNの重複除去、結果展開、raw返却を検証する
func TestClient_LookupBooksByISBN_RequestAndRaw(t *testing.T) {
	body := `[{"onix":{"RecordReference":"9784088466361","ProductIdentifier":{"ProductIDType":"15","IDValue":"9784088466361"},"DescriptiveDetail":{"TitleDetail":{"TitleType":"01","TitleElement":{"TitleElementLevel":"01","TitleText":{"content":"好きって言わせる方法 4","collationkey":"スキッテイワセルホウホウ 4"},"Subtitle":{"content":"恋のレッスン"}}},"Collection":{"TitleDetail":{"TitleElement":{"TitleElementLevel":"02","TitleText":{"content":"マーガレットコミックス"}}}},"Contributor":[{"SequenceNumber":"2","ContributorRole":["A12"],"PersonName":{"content":"作画者"}},{"SequenceNumber":"1","ContributorRole":["A38","A03"],"PersonName":{"content":"原作者"}}]},"PublishingDetail":{"Imprint":{"ImprintName":"発行社"},"Publisher":{"PublisherName":"発売社"},"PublishingDate":[{"PublishingDateRole":"11","Date":"201103"}]}},"hanmoto":{"dateshuppan":"2011-03"},"summary":{"isbn":"978-4-08-846636-1","title":"好きって言わせる方法 4","series":"マーガレットコミックス","publisher":"発行社","pubdate":"201103","cover":"https://example.test/cover.jpg","author":"要約著者"}},null]`
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if got := request.URL.Query().Get("isbn"); got != "9784088466361,9780000000002" {
			t.Errorf("isbn = %q", got)
		}
		if got := request.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(body))
	}))
	defer server.Close()

	result, raw, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(
		context.Background(),
		[]string{"4088466365", "9784088466361", "9780000000002"},
	)
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if string(raw) != body {
		t.Fatalf("raw response changed")
	}
	if len(result.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(result.Items))
	}
	if result.Items[0].RequestedISBN != "4088466365" || len(result.Items[0].Books) != 1 {
		t.Fatalf("Items[0] = %#v", result.Items[0])
	}
	if len(result.Items[1].Books) != 1 || len(result.Items[2].Books) != 0 || result.Items[2].Books == nil {
		t.Fatalf("unexpected expanded results: %#v", result.Items)
	}

	book := result.Items[0].Books[0]
	if book.Title != "好きって言わせる方法 4" {
		t.Fatalf("Title = %q", book.Title)
	}
	if book.TitleReading != "スキッテイワセルホウホウ 4" {
		t.Fatalf("TitleReading = %q", book.TitleReading)
	}
	if book.Volume.Number == nil || *book.Volume.Number != 4 || book.Volume.Label != "4" ||
		len(book.PublicationSeries) != 1 || book.PublicationSeries[0] != "マーガレットコミックス" {
		t.Fatalf("unexpected inferred fields: %#v", book)
	}
	assertStrings(t, book.Authors, []string{"原作者", "作画者"})
	assertStrings(t, book.Publishers, []string{"発行社", "発売社"})
	if len(book.Contributors) != 2 ||
		book.Contributors[0].Name != "原作者" || len(book.Contributors[0].Roles) != 1 ||
		book.Contributors[0].Roles[0] != "脚本" ||
		book.Contributors[1].Name != "作画者" || len(book.Contributors[1].Roles) != 1 ||
		book.Contributors[1].Roles[0] != "作画" {
		t.Fatalf("Contributors = %#v", book.Contributors)
	}
	if book.PublishedDate != "2011-03" {
		t.Fatalf("PublishedDate = %q", book.PublishedDate)
	}
	if len(book.ISBN13) != 1 || book.ISBN13[0] != "9784088466361" || book.CoverURL != "https://example.test/cover.jpg" {
		t.Fatalf("ISBN13 = %#v, CoverURL = %q", book.ISBN13, book.CoverURL)
	}
	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, field := range []string{"parallel_titles", "series"} {
		if strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("JSON contains inferred field %q: %s", field, encoded)
		}
	}
	if book.Sources[0].ID != "9784088466361" || book.Sources[0].URL != "" {
		t.Fatalf("Source = %#v", book.Sources[0])
	}
}

// TestClient_LookupBooksByISBN_InvalidInput は、不正な件数、ISBN、Clientを通信前に拒否する
func TestClient_LookupBooksByISBN_InvalidInput(t *testing.T) {
	client := &Client{httpClient: &http.Client{}, endpoint: defaultEndpoint}
	tests := [][]string{
		nil,
		{},
		{"invalid"},
		make([]string, maxISBNLookupCount+1),
	}
	for _, input := range tests {
		_, err := client.LookupBooksByISBN(context.Background(), input)
		assertErrorKind(t, err, ErrorKindInvalidArgument)
	}

	var nilClient *Client
	_, err := nilClient.LookupBooksByISBN(context.Background(), []string{"9784088466361"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	var nilContext context.Context
	_, err = client.LookupBooksByISBN(nilContext, []string{"9784088466361"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestValidateISBNLookupInput_MaximumAndNormalization は、最大件数とISBN表記の正規化を検証する
func TestValidateISBNLookupInput_MaximumAndNormalization(t *testing.T) {
	isbns := make([]string, maxISBNLookupCount)
	for index := range isbns {
		isbns[index] = "978 4-08-846636-1"
	}
	inputs, queries, err := validateISBNLookupInput(isbns)
	if err != nil {
		t.Fatalf("validateISBNLookupInput() error = %v", err)
	}
	if len(inputs) != maxISBNLookupCount || len(queries) != 1 || queries[0] != "9784088466361" {
		t.Fatalf("inputs = %d, queries = %#v", len(inputs), queries)
	}

	_, queries, err = validateISBNLookupInput([]string{"0-8044-2957-x"})
	if err != nil || len(queries) != 1 || queries[0] != "9780804429573" {
		t.Fatalf("lowercase x normalization: queries=%#v error=%v", queries, err)
	}
}

// TestClient_LookupBooksByISBN_InvalidResponses は、不正JSON、件数、ISBN対応を拒否する
func TestClient_LookupBooksByISBN_InvalidResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `[`},
		{name: "trailing JSON", body: `[] {}`},
		{name: "not array", body: `{}`},
		{name: "wrong count", body: `[]`},
		{name: "missing ISBN", body: `[{"onix":{},"summary":{}}]`},
		{name: "mismatched ISBN", body: `[{"onix":{"RecordReference":"9784048689410"}}]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				_, _ = response.Write([]byte(test.body))
			}))
			defer server.Close()

			_, raw, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(
				context.Background(),
				[]string{"9784088466361"},
			)
			assertErrorKind(t, err, ErrorKindInvalidResponse)
			if string(raw) != test.body {
				t.Fatalf("raw = %q, want %q", raw, test.body)
			}
		})
	}
}

// TestClient_LookupBooksByISBN_AllMissing は、全件nullを空のBooksとして返す
func TestClient_LookupBooksByISBN_AllMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`[null]`))
	}))
	defer server.Close()

	result, err := newTestClient(t, server.URL).LookupBooksByISBN(
		context.Background(),
		[]string{"9780000000002"},
	)
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Books == nil || len(result.Items[0].Books) != 0 {
		t.Fatalf("Items = %#v", result.Items)
	}
}

// TestClient_LookupBooksByISBN_HTTPError は、HTTPステータスを共通エラーへ分類する
func TestClient_LookupBooksByISBN_HTTPError(t *testing.T) {
	tests := []struct {
		status int
		kind   ErrorKind
	}{
		{status: http.StatusBadRequest, kind: ErrorKindUpstream},
		{status: http.StatusRequestTimeout, kind: ErrorKindUnavailable},
		{status: http.StatusTooManyRequests, kind: ErrorKindUnavailable},
		{status: http.StatusInternalServerError, kind: ErrorKindUnavailable},
	}
	for _, test := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(test.status)
			_, _ = response.Write([]byte("sensitive body"))
		}))
		_, raw, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(
			context.Background(),
			[]string{"9784088466361"},
		)
		server.Close()
		assertErrorKind(t, err, test.kind)
		if raw != nil || strings.Contains(err.Error(), "sensitive body") {
			t.Fatalf("raw or error exposed upstream body: raw=%q error=%v", raw, err)
		}
	}
}

// TestClient_LookupBooksByISBN_CanceledContext は、Contextのキャンセルをエラーチェーンへ保持する
func TestClient_LookupBooksByISBN_CanceledContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "canceled", ctx: canceledContext(), want: context.Canceled},
		{name: "deadline", ctx: expiredContext(), want: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newTestClient(t, "https://example.test/get").LookupBooksByISBN(
				test.ctx,
				[]string{"9784088466361"},
			)
			assertErrorKind(t, err, ErrorKindUnavailable)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

// canceledContext は、キャンセル済みContextを生成する
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// expiredContext は、期限切れContextを生成する
func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	return ctx
}

// newTestClient は、テスト用エンドポイントを使用するClientを生成する
func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	client, err := NewClient(nil, WithEndpoint(endpoint))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

// assertErrorKind は、エラーが指定した分類を持つことを検証する
func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var classified *Error
	if !errors.As(err, &classified) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if classified.Kind != want {
		t.Fatalf("Kind = %q, want %q", classified.Kind, want)
	}
}

// assertStrings は、文字列スライスの内容と順序を検証する
func assertStrings(t *testing.T, got, want []string) {
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
