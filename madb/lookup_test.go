package madb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestValidateISBNLookupInput は、入力順の保持と問い合わせ候補の重複除去を検証する
func TestValidateISBNLookupInput(t *testing.T) {
	inputs, candidates, err := validateISBNLookupInput([]string{
		" 978-4-08\u3000846636-1 ",
		"4088466365",
		"0-8044-2957-x",
	})
	if err != nil {
		t.Fatalf("validateISBNLookupInput() error = %v", err)
	}
	if len(inputs) != 3 || inputs[0].Requested != " 978-4-08\u3000846636-1 " {
		t.Fatalf("inputs = %#v", inputs)
	}
	assertStrings(t, candidates, []string{
		"080442957X", "4088466365", "9780804429573", "9784088466361",
	})
}

// TestClient_LookupBooksByISBN_MapsInputOrder は、複数ISBNの結果を元の入力位置へ対応付ける
func TestClient_LookupBooksByISBN_MapsInputOrder(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		receivedQuery = request.Form.Get("query")
		writeSPARQLResponse(t, writer, newSPARQLResponse(
			testBinding("M3", map[string]string{
				"matchedISBN": "9784990524302", "id": "M3", "title": "第三巻", "isbn": "9784990524302",
			}),
			testBinding("M1", map[string]string{
				"matchedISBN": "9784088466361", "id": "M1", "title": "作品", "isbn": "9784088466361",
			}),
			testBinding("M2", map[string]string{
				"matchedISBN": "9784990524302", "id": "M2", "title": "第二巻", "isbn": "9784990524302",
			}),
			testBinding("M1", map[string]string{
				"matchedISBN": "4088466365", "id": "M1", "title": "作品", "isbn": "4088466365",
			}),
		))
	}))
	defer server.Close()

	requested := []string{
		"978-4-08-846636-1",
		"9784990524302",
		"4088466365",
		"9780000000002",
		"9784990524302",
	}
	result, err := newTestClient(t, server.URL).LookupBooksByISBN(context.Background(), requested)
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if len(result.Items) != len(requested) {
		t.Fatalf("len(Items) = %d, want %d", len(result.Items), len(requested))
	}
	for index, value := range requested {
		if result.Items[index].RequestedISBN != value {
			t.Fatalf("Items[%d].RequestedISBN = %q, want %q", index, result.Items[index].RequestedISBN, value)
		}
	}
	for _, index := range []int{0, 2} {
		if len(result.Items[index].Books) != 1 || result.Items[index].Books[0].Sources[0].ID != "M1" {
			t.Fatalf("Items[%d].Books = %#v, want M1", index, result.Items[index].Books)
		}
	}
	for _, index := range []int{1, 4} {
		books := result.Items[index].Books
		if len(books) != 2 || books[0].Sources[0].ID != "M2" || books[1].Sources[0].ID != "M3" {
			t.Fatalf("Items[%d].Books = %#v, want M2 then M3", index, books)
		}
	}
	if result.Items[3].Books == nil || len(result.Items[3].Books) != 0 {
		t.Fatalf("missing Books = %#v, want non-nil empty slice", result.Items[3].Books)
	}
	for _, candidate := range []string{"4088466365", "9784088466361", "9784990524302"} {
		if strings.Count(receivedQuery, `"`+candidate+`"`) != 1 {
			t.Fatalf("candidate %q is not present exactly once:\n%s", candidate, receivedQuery)
		}
	}
}

// TestClient_LookupBooksByISBN_ValidatesBeforeRequest は、不正入力を通信前に拒否する
func TestClient_LookupBooksByISBN_ValidatesBeforeRequest(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requestCount++
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)

	tooMany := make([]string, maxISBNLookupCount+1)
	for index := range tooMany {
		tooMany[index] = "9784088466361"
	}
	for _, input := range [][]string{
		nil,
		{},
		tooMany,
		{"9784088466361", "4088466361"},
	} {
		_, err := client.LookupBooksByISBN(context.Background(), input)
		assertLookupError(t, err, ErrorKindInvalidArgument)
	}
	if requestCount != 0 {
		t.Fatalf("request count = %d, want 0", requestCount)
	}

	var nilClient *Client
	_, err := nilClient.LookupBooksByISBN(context.Background(), []string{"9784088466361"})
	assertLookupError(t, err, ErrorKindInvalidArgument)
	//nolint:staticcheck // nil Contextを通信前に拒否する公開契約を検証する
	_, err = client.LookupBooksByISBN(nil, []string{"9784088466361"})
	assertLookupError(t, err, ErrorKindInvalidArgument)
}

// TestClient_LookupBooksByISBNWithRawResponse は、参照結果と受信本文を1回の通信から返す
func TestClient_LookupBooksByISBNWithRawResponse(t *testing.T) {
	const body = " {\r\n  \"head\": {\"vars\": [\"resource\"]},\r\n" +
		"  \"results\": {\"bindings\": []}\r\n}\n"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/sparql-results+json")
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	result, raw, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(
		context.Background(),
		[]string{"9784088466361"},
	)
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if string(raw) != body || len(result.Items) != 1 || result.Items[0].Books == nil {
		t.Fatalf("result = %#v, raw = %q", result, raw)
	}
}

// TestClient_LookupBooksByISBNWithRawResponse_MissingMatchedISBN は、対応ISBNがない応答をraw付きで拒否する
func TestClient_LookupBooksByISBNWithRawResponse_MissingMatchedISBN(t *testing.T) {
	const body = `{"results":{"bindings":[{"resource":{"type":"uri","value":"https://mediaarts-db.artmuseums.go.jp/id/M1"}}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/sparql-results+json")
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	_, raw, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(
		context.Background(),
		[]string{"9784088466361"},
	)
	assertLookupError(t, err, ErrorKindInvalidResponse)
	if string(raw) != body {
		t.Fatalf("raw = %q, want %q", raw, body)
	}
}

// TestClient_LookupBooksByISBN_ClassifiesHTTPError は、参照時のHTTPエラーOperationを保持する
func TestClient_LookupBooksByISBN_ClassifiesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).LookupBooksByISBN(
		context.Background(),
		[]string{"9784088466361"},
	)
	assertLookupError(t, err, ErrorKindUnavailable)
}

// TestClient_LookupBooksByISBN_PreservesContextError は、参照時のContextエラーとOperationを保持する
func TestClient_LookupBooksByISBN_PreservesContextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	client := newTestClient(t, server.URL)

	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.LookupBooksByISBN(canceledContext, []string{"9784088466361"})
	assertLookupError(t, err, ErrorKindUnavailable)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}

	expiredContext, expire := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer expire()
	_, err = client.LookupBooksByISBN(expiredContext, []string{"9784088466361"})
	assertLookupError(t, err, ErrorKindUnavailable)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}

// assertLookupError は、ISBN参照エラーの分類とOperationを検証する
func assertLookupError(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	assertErrorKind(t, err, want)
	var classified *Error
	if !errors.As(err, &classified) || classified.Operation != operationISBNLookup {
		t.Fatalf("error = %#v, want operation %q", err, operationISBNLookup)
	}
}
