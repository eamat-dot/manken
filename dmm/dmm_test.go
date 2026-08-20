package dmm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestSearchSeries_SendsKeywordAndSanitizesRawResponse は、シリーズ検索のquery、変換、認証情報の秘匿を検証する
func TestSearchSeries_SendsKeywordAndSanitizesRawResponse(t *testing.T) {
	const apiID = "secret-api"
	const affiliateID = "secret/affiliate"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("site") != "DMM.com" || q.Get("service") != "ebook" || q.Get("floor") != "comic" || q.Get("sort") != "rank" || q.Get("output") != "json" {
			t.Errorf("query = %v", q)
			return
		}
		if q.Get("keyword") != "title text -omit" || q.Get("hits") != "20" || q.Get("offset") != "1" {
			t.Errorf("query = %v", q)
			return
		}
		_, _ = w.Write([]byte(`{"request":{"parameters":{"api_id":"secret-api","affiliate_id":"secret/affiliate"}},"credential_echo":"secret-api","result":{"status":200,"result_count":1,"total_count":1,"first_position":1,"items":[{"content_id":"representative","URL":"https://example.com/item","affiliateURL":"https://example.com/affiliate","title":"Series（コミック）","imageURL":{"list":"https://example.com/list","small":"https://example.com/small","large":"https://example.com/large"},"iteminfo":{"series":[{"id":4009192,"name":"Series"}],"author":[{"id":1,"name":"Author 1"},{"id":2,"name":"Author 2"}],"manufacture":[{"id":99,"name":"Publisher"}],"genre":[{"id":92206,"name":"Genre"}]}}]}}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIID(apiID), WithAffiliateID(affiliateID), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, raw, err := client.SearchSeriesWithRawResponse(context.Background(), SearchSeriesRequest{FreeText: "title text", ExcludedText: "omit"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(apiID)) || bytes.Contains(raw, []byte(affiliateID)) {
		t.Fatal("raw response leaks credentials")
	}
	if got := result.BookSeries; len(got) != 1 || got[0].BookSeries != (BookSeries{Name: "Series", ID: "4009192", Source: SourceDMM}) {
		t.Fatalf("book series = %#v", got)
	}
	series := result.BookSeries[0]
	if series.Title != "Series（コミック）" || len(series.Authors) != 2 || series.Authors[0] != "Author 1" || len(series.Publishers) != 1 || series.Publishers[0] != "Publisher" {
		t.Fatalf("series supplementary information = %#v", series)
	}
	if len(series.Subjects) != 1 || series.Subjects[0] != (Subject{Scheme: "dmm", Code: "92206", Name: "Genre"}) || len(series.Images) != 1 || series.Images[0].URL != "https://example.com/large" || len(series.Sources) != 1 || series.Sources[0].ID != "representative" || series.Sources[0].AffiliateURL != "https://example.com/affiliate" {
		t.Fatalf("series converted fields = %#v", series)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		BookSeries []map[string]any `json:"book_series"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.BookSeries) != 1 || decoded.BookSeries[0]["name"] != "Series" || decoded.BookSeries[0]["id"] != "4009192" || decoded.BookSeries[0]["source"] != "dmm" {
		t.Fatalf("book_series JSON = %s", encoded)
	}
}

// TestSearchSeries_KeepsSeriesWhenSupplementaryInformationIsMissing は、補助情報の欠落で有効なシリーズ候補を除外しないことを検証する
func TestSearchSeries_KeepsSeriesWhenSupplementaryInformationIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"status":200,"result_count":1,"total_count":1,"first_position":1,"items":[{"iteminfo":{"series":[{"id":4009192,"name":"Series"}]}}]}}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIID("api"), WithAffiliateID("affiliate"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.BookSeries) != 1 || result.BookSeries[0].Name != "Series" || result.BookSeries[0].Title != "" || len(result.BookSeries[0].Authors) != 0 || len(result.BookSeries[0].Sources) != 1 {
		t.Fatalf("series = %#v", result.BookSeries)
	}
}

// TestSearchSeriesKeyword_RejectsUnsupportedOrEmptyConditions は、FreeTextだけを正条件として検証する
func TestSearchSeriesKeyword_RejectsUnsupportedOrEmptyConditions(t *testing.T) {
	for _, request := range []SearchSeriesRequest{{}, {ExcludedText: "omit"}, {FreeText: "-omit"}, {FreeText: `"operator"`}, {FreeText: "title|operator"}, {FreeText: "title", ExcludedText: "-omit"}, {FreeText: "title", ExcludedText: `"omit"`}, {FreeText: "title", ExcludedText: "omit|other"}} {
		if _, _, err := searchSeriesKeyword(request); err == nil {
			t.Fatalf("searchSeriesKeyword(%+v) error = nil", request)
		}
	}
}

// TestSearchBooksBySeries_SendsArticleAndConvertsIndividualBooks は、シリーズ指定検索のqueryと個別Book変換を検証する
func TestSearchBooksBySeries_SendsArticleAndConvertsIndividualBooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("article") != "series" || q.Get("article_id") != "4009192" || q.Get("sort") != "rank" {
			t.Errorf("query = %v", q)
			return
		}
		_, _ = w.Write([]byte(`{"result":{"status":200,"result_count":2,"total_count":2,"first_position":1,"items":[{"content_id":"cid-7","URL":"https://example.com/item","affiliateURL":"https://example.com/affiliate","title":"Series 7","imageURL":{"list":"https://example.com/list","small":"https://example.com/small","large":"https://example.com/large"},"number":"7","iteminfo":{"author":[{"id":1,"name":"Author"}],"series":[{"id":4009192,"name":"Series"}],"manufacture":[{"id":77,"name":"Publisher"},{"id":88,"name":"  "}],"genre":[{"id":92206,"name":"  Genre  "}]}},{"content_id":"cid-6","title":"Series 6","iteminfo":{"series":[{"id":4009192,"name":"Series"}]}}]}}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIID("api"), WithAffiliateID("affiliate"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.SearchBooksBySeries(context.Background(), SearchBooksBySeriesRequest{SeriesID: "4009192"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Books) != 2 {
		t.Fatalf("books = %#v", result.Books)
	}
	if result.Books[0].Title != "Series 7" || result.Books[1].Title != "Series 6" {
		t.Fatalf("book order = %#v", result.Books)
	}
	book := result.Books[0]
	if book.Sources[0].ID != "cid-7" || book.Sources[0].URL == book.Sources[0].AffiliateURL || book.Title != "Series 7" || book.Medium != PublicationMediumDigital {
		t.Fatal("book conversion did not preserve individual product fields")
	}
	if book.Volume.Number == nil || *book.Volume.Number != 7 || book.Volume.Label != "7" {
		t.Fatalf("volume = %#v", book.Volume)
	}
	if got := book.BookSeries; len(got) != 1 || got[0] != (BookSeries{Name: "Series", ID: "4009192", Source: SourceDMM}) {
		t.Fatalf("book series = %#v", got)
	}
	if len(book.PublicationSeries) != 0 {
		t.Fatalf("publication series = %#v", book.PublicationSeries)
	}
	if got := book.Publishers; len(got) != 1 || got[0] != "Publisher" {
		t.Fatalf("publishers = %#v", got)
	}
	if got := book.Subjects; len(got) != 1 || got[0] != (Subject{Scheme: "dmm", Code: "92206", Name: "Genre"}) {
		t.Fatalf("subjects = %#v", got)
	}
	if book.CoverURL != "https://example.com/large" {
		t.Fatalf("CoverURL = %q", book.CoverURL)
	}
}

// TestConvertItem_ExtractsParenthesizedVolume は、DMMシリーズ内商品の括弧付き巻表示を補うことを確認する
func TestConvertItem_ExtractsParenthesizedVolume(t *testing.T) {
	book := convertItem(item{Title: "シリーズ作品（7）"}, BookSeries{Name: "シリーズ", ID: "1", Source: SourceDMM})
	if book.Title != "シリーズ作品（7）" || book.Volume.Number == nil || *book.Volume.Number != 7 || book.Volume.Label != "7" {
		t.Fatalf("book = %#v", book)
	}
}

// TestSearchBooksBySeries_RejectsInvalidSeriesResponses は、空ID、欠落、不一致のシリーズを拒否する
func TestSearchBooksBySeries_RejectsInvalidSeriesResponses(t *testing.T) {
	client, err := NewClient(nil, WithAPIID("api"), WithAffiliateID("affiliate"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.SearchBooksBySeries(context.Background(), SearchBooksBySeriesRequest{}); !hasErrorKind(err, ErrorKindInvalidArgument) {
		t.Fatalf("empty series ID error = %v", err)
	}
	for _, itemInfo := range []string{`{}`, `{"series":[{"id":2,"name":"Other"}]}`, `{"series":[{"id":1,"name":"One"},{"id":1,"name":"One"}]}`} {
		t.Run(itemInfo, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"result":{"status":200,"result_count":1,"total_count":1,"first_position":1,"items":[{"content_id":"cid","iteminfo":` + itemInfo + `}]}}`))
			}))
			defer server.Close()
			c, err := NewClient(nil, WithAPIID("api"), WithAffiliateID("affiliate"), WithEndpoint(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.SearchBooksBySeries(context.Background(), SearchBooksBySeriesRequest{SeriesID: "1"}); !hasErrorKind(err, ErrorKindInvalidResponse) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

// TestLimitAndCursor は、Limit境界とAPI種別を含むCursor拘束を検証する
func TestLimitAndCursor(t *testing.T) {
	for _, value := range []int{0, 1, 100} {
		if _, err := effectiveLimit(value); err != nil {
			t.Fatalf("effectiveLimit(%d): %v", value, err)
		}
	}
	for _, value := range []int{-1, 101} {
		if _, err := effectiveLimit(value); err == nil {
			t.Fatalf("effectiveLimit(%d) error = nil", value)
		}
	}
	cursor, err := encodeCursor(101, cursorAPISeries, "query", 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		api, key string
		limit    int
	}{{cursorAPIBooksBySeries, "query", 20}, {cursorAPISeries, "other", 20}, {cursorAPISeries, "query", 10}} {
		if _, err := decodeCursor(cursor, test.api, test.key, test.limit); err == nil {
			t.Fatalf("cursor accepted by %+v", test)
		}
	}
	if _, err := encodeCursor(maxOffset+1, cursorAPISeries, "query", 20); err == nil {
		t.Fatal("offset above maximum accepted")
	}
}

// TestTrailingJSONIsRejected は、Cursor、通常レスポンス、Raw responseの末尾JSON値を拒否することを検証する
func TestTrailingJSONIsRejected(t *testing.T) {
	payload, err := encodeCursor(2, cursorAPISeries, "query", 20)
	if err != nil {
		t.Fatal(err)
	}
	cursorBody, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatal(err)
	}
	trailingCursor := base64.RawURLEncoding.EncodeToString(append(cursorBody, []byte(`{}`)...))
	if _, err := decodeCursor(trailingCursor, cursorAPISeries, "query", 20); err == nil {
		t.Fatal("trailing JSON cursor accepted")
	}
	if _, err := decodeResponse([]byte(`{"result":{"status":200}}{}`)); err == nil {
		t.Fatal("trailing JSON response accepted")
	}
	if _, err := sanitizeRaw([]byte(`{"result":{"status":200}}{}`), "api", "affiliate"); err == nil {
		t.Fatal("trailing JSON raw response accepted")
	}
}

// TestClientRejectsBadConfigurationAndRedirect は、不正設定とリダイレクトを拒否する
func TestClientRejectsBadConfigurationAndRedirect(t *testing.T) {
	for _, options := range [][]Option{{WithAPIID("a")}, {WithAffiliateID("a")}, {WithAPIID("a"), WithAffiliateID("b"), WithEndpoint("https://example.com/?x=1")}} {
		if _, err := NewClient(nil, options...); err == nil {
			t.Fatal("NewClient error = nil")
		}
	}
	client, err := NewClient(nil, WithAPIID("a"), WithAffiliateID("b"))
	if err != nil {
		t.Fatal(err)
	}
	var nilContext context.Context
	if _, err := client.SearchSeries(nilContext, SearchSeriesRequest{FreeText: "title"}); err == nil {
		t.Fatal("nil context accepted")
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://other.example/")
		w.WriteHeader(http.StatusFound)
	}))
	defer redirect.Close()
	client, err = NewClient(nil, WithAPIID("a"), WithAffiliateID("b"), WithEndpoint(redirect.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"})
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound || apiErr.Operation != operationSearchSeries {
		t.Fatalf("error = %v", err)
	}
}

// TestSearchSeries_RedactsSecretsFromTransportErrors は、RoundTripperの通信エラーから認証情報を秘匿する
func TestSearchSeries_RedactsSecretsFromTransportErrors(t *testing.T) {
	const apiID = "api id"
	const affiliateID = "affiliate/id"
	client, err := NewClient(&http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("request failed: %s", request.URL.String())
	})}, WithAPIID(apiID), WithAffiliateID(affiliateID), WithEndpoint("https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"})
	if !hasErrorKind(err, ErrorKindUnavailable) {
		t.Fatalf("error kind = %v", err)
	}
	for _, secret := range []string{apiID, affiliateID, url.QueryEscape(apiID), url.QueryEscape(affiliateID)} {
		if strings.Contains(err.Error(), secret) || strings.Contains(errors.Unwrap(err).Error(), secret) {
			t.Fatalf("transport error leaks secret %q: %v", secret, err)
		}
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("transport error = %v", err)
	}
}

// TestSearchSeries_PreservesSecretFreeTransportError は、秘密値を含まない通信エラーの同一性を維持する
func TestSearchSeries_PreservesSecretFreeTransportError(t *testing.T) {
	want := errors.New("transport failed")
	client, err := NewClient(&http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, want
	})}, WithAPIID("api"), WithAffiliateID("affiliate"), WithEndpoint("https://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"})
	if !errors.Is(err, want) {
		t.Fatalf("error does not preserve transport cause: %v", err)
	}
}

// TestHTTPAndInvalidResponse は、HTTPと本文の異常を共通エラー種別へ対応付ける
func TestHTTPAndInvalidResponse(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		kind   ErrorKind
	}{{429, "", ErrorKindUnavailable}, {400, "", ErrorKindUpstream}, {200, "not json", ErrorKindInvalidResponse}, {200, `{"result":{"status":400}}`, ErrorKindUpstream}} {
		t.Run(test.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := NewClient(nil, WithAPIID("a"), WithAffiliateID("b"), WithEndpoint(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "x"})
			if !hasErrorKind(err, test.kind) {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

// TestValidURL は、HTTP(S)以外と相対URLを除外する
func TestValidURL(t *testing.T) {
	for _, value := range []string{"", "ftp://example.com", "//example.com"} {
		if validURL(value) != "" {
			t.Fatalf("validURL(%q) accepted", value)
		}
	}
	if validURL((&url.URL{Scheme: "https", Host: "example.com"}).String()) == "" {
		t.Fatal("https URL rejected")
	}
}

// TestConvertItem_SetsCoverURL は、画像項目が欠落しても利用可能な最大画像をCoverURLへ設定することを検証する
func TestConvertItem_SetsCoverURL(t *testing.T) {
	item := item{ImageURL: imageURLs{Small: "https://example.com/small"}}
	book := convertItem(item, BookSeries{Name: "Series", ID: "1", Source: SourceDMM})
	if book.CoverURL != "https://example.com/small" {
		t.Fatalf("CoverURL = %q", book.CoverURL)
	}
}

// TestImagesFromItem は、DMM画像URLを優先順で最大1件だけ変換することを検証する
func TestImagesFromItem(t *testing.T) {
	for _, test := range []struct {
		name     string
		imageURL imageURLs
		wantURL  string
	}{
		{
			name:     "prefer large",
			imageURL: imageURLs{List: "https://example.com/list", Small: "https://example.com/small", Large: "https://example.com/large"},
			wantURL:  "https://example.com/large",
		},
		{
			name:     "fall back to list when large is invalid",
			imageURL: imageURLs{List: "https://example.com/list", Large: "ftp://example.com/large"},
			wantURL:  "https://example.com/list",
		},
		{
			name:     "fall back to small when larger images are invalid",
			imageURL: imageURLs{Small: "https://example.com/small", List: "//example.com/list", Large: ""},
			wantURL:  "https://example.com/small",
		},
		{
			name:     "omit unavailable images",
			imageURL: imageURLs{List: "relative", Small: "ftp://example.com/small", Large: "//example.com/large"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := imagesFromItem(item{ImageURL: test.imageURL})
			if test.wantURL == "" {
				if len(got) != 0 {
					t.Fatalf("images = %#v", got)
				}
				return
			}
			if len(got) != 1 || got[0].URL != test.wantURL || got[0].Purpose != "" {
				t.Fatalf("images = %#v", got)
			}
		})
	}
}

// TestSanitizeRaw_PreservesJSONNumbersAndPlainStrings は、秘匿時に数値と非URL文字列を変質させないことを検証する
func TestSanitizeRaw_PreservesJSONNumbersAndPlainStrings(t *testing.T) {
	const apiID = "api-secret"
	const affiliateID = "affiliate-secret"
	raw, err := sanitizeRaw([]byte(`{"large":9007199254740993,"plain":"A?b=c d","link":"https://example.com/?affiliate_id=affiliate-secret&x=1","secret":"api-secret"}`), apiID, affiliateID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"large":9007199254740993`)) || !bytes.Contains(raw, []byte(`"plain":"A?b=c d"`)) {
		t.Fatalf("raw response changed: %s", raw)
	}
	if bytes.Contains(raw, []byte(apiID)) || bytes.Contains(raw, []byte(affiliateID)) {
		t.Fatalf("raw response leaks credentials: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"link":"https://example.com/?affiliate_id=%5BREDACTED%5D&x=1"`)) || bytes.Contains(raw, []byte(`\u0026`)) {
		t.Fatalf("raw response HTML-escapes URL: %s", raw)
	}
	if bytes.HasSuffix(raw, []byte("\n")) {
		t.Fatalf("raw response ends with a newline: %q", raw)
	}
}

// TestSanitizeRawString_DoesNotRedactEmptyQueryValue は、空の認証情報でURL queryの空値を秘匿しないことを検証する
func TestSanitizeRawString_DoesNotRedactEmptyQueryValue(t *testing.T) {
	const value = "https://example.com/?empty=&x=1"
	if got := sanitizeRawString(value, []string{"secret", ""}); got != value {
		t.Fatalf("sanitizeRawString() = %q, want %q", got, value)
	}
}

// TestExecute_HandlesResponseBodyCleanupAndLimits は、本文cleanup失敗と本文上限をinvalid_responseへ分類することを検証する
func TestExecute_HandlesResponseBodyCleanupAndLimits(t *testing.T) {
	closeFailure := errors.New("close failed")
	for _, test := range []struct {
		name       string
		statusCode int
		body       string
		closeErr   error
		wantKind   ErrorKind
	}{
		{name: "HTTP error cleanup", statusCode: http.StatusBadRequest, body: "error", closeErr: closeFailure, wantKind: ErrorKindUpstream},
		{name: "success close failure", statusCode: http.StatusOK, body: `{"result":{"status":200}}`, closeErr: closeFailure, wantKind: ErrorKindInvalidResponse},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(&http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.statusCode, Header: make(http.Header), Body: errReadCloser{Reader: strings.NewReader(test.body), err: test.closeErr}}, nil
			})}, WithAPIID("api"), WithAffiliateID("affiliate"), WithEndpoint("https://example.com"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"})
			if !hasErrorKind(err, test.wantKind) || !strings.Contains(err.Error(), closeFailure.Error()) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

// TestSearchSeries_RedactsSecretsFromResponseBodyErrors は、本文のcleanupエラーから認証情報を秘匿する
func TestSearchSeries_RedactsSecretsFromResponseBodyErrors(t *testing.T) {
	const apiID = "api id"
	const affiliateID = "affiliate/id"
	for _, test := range []struct {
		name       string
		statusCode int
		wantKind   ErrorKind
	}{
		{name: "success close error", statusCode: http.StatusOK, wantKind: ErrorKindInvalidResponse},
		{name: "HTTP error cleanup", statusCode: http.StatusBadRequest, wantKind: ErrorKindUpstream},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(&http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				closeErr := fmt.Errorf("close failed: %s", request.URL.String())
				return &http.Response{
					StatusCode: test.statusCode,
					Header:     make(http.Header),
					Body:       errReadCloser{Reader: strings.NewReader(`{"result":{"status":200}}`), err: closeErr},
				}, nil
			})}, WithAPIID(apiID), WithAffiliateID(affiliateID), WithEndpoint("https://example.com"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"})
			if !hasErrorKind(err, test.wantKind) {
				t.Fatalf("error kind = %v", err)
			}
			for _, secret := range []string{apiID, affiliateID, url.QueryEscape(apiID), url.QueryEscape(affiliateID)} {
				if strings.Contains(err.Error(), secret) || strings.Contains(errors.Unwrap(err).Error(), secret) {
					t.Fatalf("response body error leaks secret %q: %v", secret, err)
				}
			}
			if !strings.Contains(err.Error(), "close failed: ") || !strings.Contains(err.Error(), "[REDACTED]") {
				t.Fatalf("response body error = %v", err)
			}
		})
	}
}

// TestSearchSeries_RejectsInconsistentPagination は、Itemsを含む応答のfirst_position不整合を拒否する
func TestSearchSeries_RejectsInconsistentPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"status":200,"result_count":1,"total_count":2,"first_position":0,"items":[{"content_id":"cid","iteminfo":{"series":[{"id":1,"name":"Series"}]}}]}}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithAPIID("api"), WithAffiliateID("affiliate"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.SearchSeries(context.Background(), SearchSeriesRequest{FreeText: "title"}); !hasErrorKind(err, ErrorKindInvalidResponse) {
		t.Fatalf("error = %v", err)
	}
}

// hasErrorKind は、エラーが期待する公開エラー種別を持つか判定する
func hasErrorKind(err error, kind ErrorKind) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Kind == kind
}

// roundTripperFunc は、関数をHTTP RoundTripperとして使用できるようにする
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、HTTP要求を関数へ渡す
func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// errReadCloser は、指定されたCloseエラーを返す読み取り専用本文を表す
type errReadCloser struct {
	io.Reader
	err error
}

// Close は、設定されたCloseエラーを返す
func (reader errReadCloser) Close() error {
	return reader.err
}
