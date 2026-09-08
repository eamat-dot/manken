package manken

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/eamat-dot/manken/model"
	"github.com/eamat-dot/manken/openbd"
	"github.com/eamat-dot/manken/rakutenkobo"
)

// TestClient_DelegatesSearchWithoutChanges は、検索requestと結果を変更せず一度だけ委譲することを検証する
func TestClient_DelegatesSearchWithoutChanges(t *testing.T) {
	request := SearchRequest{Title: "作品", Limit: 3}
	want := SearchBooksResult{
		Books:        []Book{{Title: "結果"}},
		NextCursor:   "next",
		Attributions: []Attribution{{Source: SourceMADB, Scope: AttributionScopeData, Text: "credit"}},
	}
	provider := &fakeSearchProvider{searchResult: want}
	client := testClient(provider, nil)

	got, err := client.SearchBooks(context.Background(), SourceMADB, request)
	if err != nil {
		t.Fatalf("SearchBooks() error = %v", err)
	}
	if provider.searchCalls != 1 || !reflect.DeepEqual(provider.searchRequest, request) {
		t.Fatalf("SearchBooks() delegation = calls %d, request %#v", provider.searchCalls, provider.searchRequest)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SearchBooks() = %#v, want %#v", got, want)
	}
}

// TestClient_DelegatesRawResponsesWithoutChanges は、Raw版の結果と本文を変更せず委譲することを検証する
func TestClient_DelegatesRawResponsesWithoutChanges(t *testing.T) {
	searchRequest := SearchRequest{Query: "作品"}
	searchProvider := &fakeSearchProvider{rawResult: SearchBooksResult{Books: []Book{{Title: "検索"}}}, raw: []byte("search raw")}
	isbnProvider := &fakeISBNProvider{rawResult: ISBNLookupResult{Items: []ISBNLookupItem{{RequestedISBN: "9784000000000"}}}, raw: []byte("isbn raw")}
	client := testClient(searchProvider, isbnProvider)

	searchResult, searchRaw, err := client.SearchBooksWithRawResponse(context.Background(), SourceMADB, searchRequest)
	if err != nil {
		t.Fatalf("SearchBooksWithRawResponse() error = %v", err)
	}
	if searchProvider.rawCalls != 1 || !reflect.DeepEqual(searchProvider.searchRequest, searchRequest) {
		t.Fatalf("SearchBooksWithRawResponse() delegation = calls %d, request %#v", searchProvider.rawCalls, searchProvider.searchRequest)
	}
	if !reflect.DeepEqual(searchResult, searchProvider.rawResult) || !reflect.DeepEqual(searchRaw, searchProvider.raw) {
		t.Fatalf("SearchBooksWithRawResponse() = %#v, %q", searchResult, searchRaw)
	}

	isbns := []string{"9784000000000"}
	lookupResult, lookupRaw, err := client.LookupBooksByISBNWithRawResponse(context.Background(), SourceMADB, isbns)
	if err != nil {
		t.Fatalf("LookupBooksByISBNWithRawResponse() error = %v", err)
	}
	if isbnProvider.rawCalls != 1 || !reflect.DeepEqual(isbnProvider.isbns, isbns) {
		t.Fatalf("LookupBooksByISBNWithRawResponse() delegation = calls %d, ISBNs %#v", isbnProvider.rawCalls, isbnProvider.isbns)
	}
	if !reflect.DeepEqual(lookupResult, isbnProvider.rawResult) || !reflect.DeepEqual(lookupRaw, isbnProvider.raw) {
		t.Fatalf("LookupBooksByISBNWithRawResponse() = %#v, %q", lookupResult, lookupRaw)
	}
}

// TestClient_DelegatesISBNLookupWithoutChanges は、ISBN入力と結果を変更せず一度だけ委譲することを検証する
func TestClient_DelegatesISBNLookupWithoutChanges(t *testing.T) {
	isbns := []string{"9784000000000", "9784000000001"}
	want := ISBNLookupResult{
		Items:        []ISBNLookupItem{{RequestedISBN: isbns[0]}},
		Attributions: []Attribution{{Source: SourceMADB, Scope: AttributionScopeData, Text: "credit"}},
	}
	provider := &fakeISBNProvider{lookupResult: want}
	client := testClient(nil, provider)

	got, err := client.LookupBooksByISBN(context.Background(), SourceMADB, isbns)
	if err != nil {
		t.Fatalf("LookupBooksByISBN() error = %v", err)
	}
	if provider.lookupCalls != 1 || !reflect.DeepEqual(provider.isbns, isbns) {
		t.Fatalf("LookupBooksByISBN() delegation = calls %d, ISBNs %#v", provider.lookupCalls, provider.isbns)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LookupBooksByISBN() = %#v, want %#v", got, want)
	}
}

// TestClient_PreservesProviderError は、provider由来の分類済みエラーをそのまま返すことを検証する
func TestClient_PreservesProviderError(t *testing.T) {
	cause := errors.New("provider failed")
	want := &model.Error{Kind: model.ErrorKindUpstream, Operation: "provider.search", Err: cause}
	client := testClient(&fakeSearchProvider{searchErr: want}, nil)

	_, err := client.SearchBooks(context.Background(), SourceMADB, SearchRequest{Title: "作品"})
	var got *model.Error
	if !errors.As(err, &got) {
		t.Fatalf("SearchBooks() error = %v, want *model.Error", err)
	}
	if got != want || got.Kind != want.Kind || !errors.Is(got, cause) {
		t.Fatalf("SearchBooks() error = %#v, want original %#v", got, want)
	}
}

// TestNewClient_RejectsMissingNilOptionsAndProviders は、provider未指定、nil Option、nil provider Clientを入力エラーとして返すことを検証する
func TestNewClient_RejectsMissingNilOptionsAndProviders(t *testing.T) {
	if _, err := NewClient(); err == nil {
		t.Fatal("NewClient() error = nil")
	} else {
		assertInvalidArgument(t, err)
	}

	for _, option := range []Option{
		nil,
		WithMADBClient(nil),
		WithOpenBDClient(nil),
		WithGoogleBooksClient(nil),
		WithRakutenBooksClient(nil),
		WithRakutenKoboClient(nil),
		WithNDLClient(nil),
		WithYahooShoppingClient(nil),
	} {
		_, err := NewClient(option)
		assertInvalidArgument(t, err)
	}
}

// TestClient_RejectsInvalidDispatch は、未初期化、未知、未登録、非対応の操作を区別できる入力エラーとして返すことを検証する
func TestClient_RejectsInvalidDispatch(t *testing.T) {
	client := testClientAt(SourceOpenBD, nil, &fakeISBNProvider{})

	for _, test := range []struct {
		name        string
		wantMessage string
		call        func() error
	}{
		{name: "uninitialized", wantMessage: "client is not initialized", call: func() error {
			var c *Client
			_, err := c.SearchBooks(context.Background(), SourceMADB, SearchRequest{})
			return err
		}},
		{name: "unknown", wantMessage: "unknown source", call: func() error {
			_, err := client.SearchBooks(context.Background(), Source("unknown"), SearchRequest{})
			return err
		}},
		{name: "unregistered", wantMessage: "is not registered", call: func() error {
			_, err := client.SearchBooks(context.Background(), SourceMADB, SearchRequest{})
			return err
		}},
		{name: "openbd search", wantMessage: "does not support search", call: func() error {
			_, err := client.SearchBooks(context.Background(), SourceOpenBD, SearchRequest{})
			return err
		}},
		{name: "rakuten kobo ISBN", wantMessage: "does not support ISBN lookup", call: func() error {
			_, err := client.LookupBooksByISBN(context.Background(), SourceRakutenKobo, nil)
			return err
		}},
		{name: "DMM search", wantMessage: "does not support search", call: func() error {
			_, err := client.SearchBooks(context.Background(), SourceDMM, SearchRequest{})
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			assertInvalidArgument(t, err)
			if !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("error = %v, want message containing %q", err, test.wantMessage)
			}
		})
	}
}

// TestProviderOptions_LastOptionWins は、同じproviderの登録Optionを複数指定したとき後から指定したClientを採用することを検証する
func TestProviderOptions_LastOptionWins(t *testing.T) {
	first, err := openbd.NewClient(nil)
	if err != nil {
		t.Fatalf("openbd.NewClient(first) error = %v", err)
	}
	second, err := openbd.NewClient(nil)
	if err != nil {
		t.Fatalf("openbd.NewClient(second) error = %v", err)
	}
	client, err := NewClient(WithOpenBDClient(first), WithOpenBDClient(second))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.isbnProviders[SourceOpenBD] != second {
		t.Fatal("last WithOpenBDClient() was not retained")
	}
}

// TestProviderOptions_RegisterOnlySupportedRoles は、OpenBDと楽天Koboが対応するroleだけを登録することを検証する
func TestProviderOptions_RegisterOnlySupportedRoles(t *testing.T) {
	openBD, err := openbd.NewClient(nil)
	if err != nil {
		t.Fatalf("openbd.NewClient() error = %v", err)
	}
	kobo, err := rakutenkobo.NewClient(nil, rakutenkobo.WithApplicationID("application"), rakutenkobo.WithAccessKey("access"))
	if err != nil {
		t.Fatalf("rakutenkobo.NewClient() error = %v", err)
	}
	client, err := NewClient(WithOpenBDClient(openBD), WithRakutenKoboClient(kobo))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, ok := client.searchProviders[SourceOpenBD]; ok {
		t.Fatal("OpenBD was registered for search")
	}
	if _, ok := client.isbnProviders[SourceRakutenKobo]; ok {
		t.Fatal("Rakuten Kobo was registered for ISBN lookup")
	}
}

// testClient は、指定roleだけを登録したテスト用Clientを返す
func testClient(search searchProvider, isbn isbnProvider) *Client {
	return testClientAt(SourceMADB, search, isbn)
}

// testClientAt は、指定Sourceのroleだけを登録したテスト用Clientを返す
func testClientAt(source Source, search searchProvider, isbn isbnProvider) *Client {
	client := &Client{searchProviders: make(map[Source]searchProvider), isbnProviders: make(map[Source]isbnProvider)}
	if search != nil {
		client.searchProviders[source] = search
	}
	if isbn != nil {
		client.isbnProviders[source] = isbn
	}
	return client
}

// assertInvalidArgument は、エラーがルートの入力エラーとして分類されていることを検証する
func assertInvalidArgument(t *testing.T, err error) {
	t.Helper()
	var got *model.Error
	if !errors.As(err, &got) || got.Kind != model.ErrorKindInvalidArgument {
		t.Fatalf("error = %v, want invalid_argument", err)
	}
}

type fakeSearchProvider struct {
	searchCalls   int
	searchRequest SearchRequest
	searchResult  SearchBooksResult
	searchErr     error
	rawCalls      int
	rawResult     SearchBooksResult
	raw           []byte
	rawErr        error
}

// SearchBooks は、記録済みの検索結果を返す
func (provider *fakeSearchProvider) SearchBooks(_ context.Context, request SearchRequest) (SearchBooksResult, error) {
	provider.searchCalls++
	provider.searchRequest = request
	return provider.searchResult, provider.searchErr
}

// SearchBooksWithRawResponse は、記録済みの検索結果とRaw本文を返す
func (provider *fakeSearchProvider) SearchBooksWithRawResponse(_ context.Context, request SearchRequest) (SearchBooksResult, []byte, error) {
	provider.rawCalls++
	provider.searchRequest = request
	return provider.rawResult, provider.raw, provider.rawErr
}

type fakeISBNProvider struct {
	lookupCalls  int
	rawCalls     int
	isbns        []string
	lookupResult ISBNLookupResult
	lookupErr    error
	rawResult    ISBNLookupResult
	raw          []byte
	rawErr       error
}

// LookupBooksByISBN は、記録済みのISBN参照結果を返す
func (provider *fakeISBNProvider) LookupBooksByISBN(_ context.Context, isbns []string) (ISBNLookupResult, error) {
	provider.lookupCalls++
	provider.isbns = isbns
	return provider.lookupResult, provider.lookupErr
}

// LookupBooksByISBNWithRawResponse は、記録済みのISBN参照結果とRaw本文を返す
func (provider *fakeISBNProvider) LookupBooksByISBNWithRawResponse(_ context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	provider.rawCalls++
	provider.isbns = isbns
	return provider.rawResult, provider.raw, provider.rawErr
}
