package manken

import (
	"context"
	"errors"
	"fmt"

	"github.com/eamat-dot/manken/googlebooks"
	"github.com/eamat-dot/manken/madb"
	"github.com/eamat-dot/manken/model"
	"github.com/eamat-dot/manken/ndl"
	"github.com/eamat-dot/manken/openbd"
	"github.com/eamat-dot/manken/rakutenbooks"
	"github.com/eamat-dot/manken/rakutenkobo"
	"github.com/eamat-dot/manken/yahooshopping"
)

const (
	operationNewClient = "manken.NewClient"
	operationSearch    = "manken.SearchBooks"
	operationLookup    = "manken.LookupBooksByISBN"
)

type searchProvider interface {
	SearchBooks(context.Context, SearchRequest) (SearchBooksResult, error)
	SearchBooksWithRawResponse(context.Context, SearchRequest) (SearchBooksResult, []byte, error)
}

type isbnProvider interface {
	LookupBooksByISBN(context.Context, []string) (ISBNLookupResult, error)
	LookupBooksByISBNWithRawResponse(context.Context, []string) (ISBNLookupResult, []byte, error)
}

// Client は、登録済みproviderへ共通操作を明示Sourceで委譲する
type Client struct {
	searchProviders map[Source]searchProvider
	isbnProviders   map[Source]isbnProvider
}

// Option は、Clientへprovider Clientを登録する設定を表す
type Option interface {
	apply(*Client) error
}

type optionFunc func(*Client) error

// apply は、関数が表すprovider登録をClientへ適用する
func (option optionFunc) apply(client *Client) error {
	return option(client)
}

// NewClient は、指定したprovider Clientを登録したClientを生成する
func NewClient(options ...Option) (*Client, error) {
	if len(options) == 0 {
		return nil, invalidArgument(operationNewClient, errors.New("at least one provider client must be configured"))
	}

	client := &Client{
		searchProviders: make(map[Source]searchProvider),
		isbnProviders:   make(map[Source]isbnProvider),
	}
	for _, option := range options {
		if option == nil {
			return nil, invalidArgument(operationNewClient, errors.New("option must not be nil"))
		}
		if err := option.apply(client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

// WithMADBClient は、MADB Clientを検索とISBN参照に登録する
func WithMADBClient(provider *madb.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceMADB)
		}
		client.searchProviders[SourceMADB] = provider
		client.isbnProviders[SourceMADB] = provider
		return nil
	})
}

// WithOpenBDClient は、openBD ClientをISBN参照に登録する
func WithOpenBDClient(provider *openbd.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceOpenBD)
		}
		client.isbnProviders[SourceOpenBD] = provider
		return nil
	})
}

// WithGoogleBooksClient は、Google Books Clientを検索とISBN参照に登録する
func WithGoogleBooksClient(provider *googlebooks.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceGoogleBooks)
		}
		client.searchProviders[SourceGoogleBooks] = provider
		client.isbnProviders[SourceGoogleBooks] = provider
		return nil
	})
}

// WithRakutenBooksClient は、楽天Books Clientを検索とISBN参照に登録する
func WithRakutenBooksClient(provider *rakutenbooks.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceRakutenBooks)
		}
		client.searchProviders[SourceRakutenBooks] = provider
		client.isbnProviders[SourceRakutenBooks] = provider
		return nil
	})
}

// WithRakutenKoboClient は、楽天Kobo Clientを検索に登録する
func WithRakutenKoboClient(provider *rakutenkobo.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceRakutenKobo)
		}
		client.searchProviders[SourceRakutenKobo] = provider
		return nil
	})
}

// WithNDLClient は、NDL Clientを検索とISBN参照に登録する
func WithNDLClient(provider *ndl.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceNDL)
		}
		client.searchProviders[SourceNDL] = provider
		client.isbnProviders[SourceNDL] = provider
		return nil
	})
}

// WithYahooShoppingClient は、Yahoo!ショッピング Clientを検索とISBN参照に登録する
func WithYahooShoppingClient(provider *yahooshopping.Client) Option {
	return optionFunc(func(client *Client) error {
		if provider == nil {
			return nilProviderError(SourceYahooShopping)
		}
		client.searchProviders[SourceYahooShopping] = provider
		client.isbnProviders[SourceYahooShopping] = provider
		return nil
	})
}

// SearchBooks は、明示されたSourceの登録済みproviderへ検索をそのまま委譲する
func (client *Client) SearchBooks(ctx context.Context, source Source, request SearchRequest) (SearchBooksResult, error) {
	provider, err := client.searchProvider(source)
	if err != nil {
		return SearchBooksResult{}, err
	}
	return provider.SearchBooks(ctx, request)
}

// SearchBooksWithRawResponse は、明示されたSourceの検索と成功レスポンス本文をそのまま返す
func (client *Client) SearchBooksWithRawResponse(ctx context.Context, source Source, request SearchRequest) (SearchBooksResult, []byte, error) {
	provider, err := client.searchProvider(source)
	if err != nil {
		return SearchBooksResult{}, nil, err
	}
	return provider.SearchBooksWithRawResponse(ctx, request)
}

// LookupBooksByISBN は、明示されたSourceの登録済みproviderへISBN参照をそのまま委譲する
func (client *Client) LookupBooksByISBN(ctx context.Context, source Source, isbns []string) (ISBNLookupResult, error) {
	provider, err := client.isbnProvider(source)
	if err != nil {
		return ISBNLookupResult{}, err
	}
	return provider.LookupBooksByISBN(ctx, isbns)
}

// LookupBooksByISBNWithRawResponse は、明示されたSourceのISBN参照と成功レスポンス本文をそのまま返す
func (client *Client) LookupBooksByISBNWithRawResponse(ctx context.Context, source Source, isbns []string) (ISBNLookupResult, []byte, error) {
	provider, err := client.isbnProvider(source)
	if err != nil {
		return ISBNLookupResult{}, nil, err
	}
	return provider.LookupBooksByISBNWithRawResponse(ctx, isbns)
}

// nilProviderError は、nilのprovider Clientが指定された入力エラーを返す
func nilProviderError(source Source) error {
	return invalidArgument(operationNewClient, fmt.Errorf("%s client must not be nil", source))
}

// searchProvider は、Sourceに対応する検索providerを返す
func (client *Client) searchProvider(source Source) (searchProvider, error) {
	if client == nil || client.searchProviders == nil || client.isbnProviders == nil {
		return nil, invalidArgument(operationSearch, errors.New("client is not initialized"))
	}
	if !knownSource(source) {
		return nil, invalidArgument(operationSearch, fmt.Errorf("unknown source %q", source))
	}
	if !supportsSearch(source) {
		return nil, invalidArgument(operationSearch, fmt.Errorf("source %q does not support search", source))
	}
	provider, ok := client.searchProviders[source]
	if !ok {
		return nil, invalidArgument(operationSearch, fmt.Errorf("source %q is not registered", source))
	}
	return provider, nil
}

// isbnProvider は、Sourceに対応するISBN参照providerを返す
func (client *Client) isbnProvider(source Source) (isbnProvider, error) {
	if client == nil || client.searchProviders == nil || client.isbnProviders == nil {
		return nil, invalidArgument(operationLookup, errors.New("client is not initialized"))
	}
	if !knownSource(source) {
		return nil, invalidArgument(operationLookup, fmt.Errorf("unknown source %q", source))
	}
	if !supportsISBNLookup(source) {
		return nil, invalidArgument(operationLookup, fmt.Errorf("source %q does not support ISBN lookup", source))
	}
	provider, ok := client.isbnProviders[source]
	if !ok {
		return nil, invalidArgument(operationLookup, fmt.Errorf("source %q is not registered", source))
	}
	return provider, nil
}

// knownSource は、共通モデルに定義された取得元か判定する
func knownSource(source Source) bool {
	switch source {
	case SourceMADB, SourceOpenBD, SourceGoogleBooks, SourceRakutenBooks, SourceRakutenKobo, SourceNDL, SourceYahooShopping, SourceDMM:
		return true
	default:
		return false
	}
}

// supportsSearch は、ルートファサードが共通検索へ委譲できる取得元か判定する
func supportsSearch(source Source) bool {
	switch source {
	case SourceMADB, SourceGoogleBooks, SourceRakutenBooks, SourceRakutenKobo, SourceNDL, SourceYahooShopping:
		return true
	default:
		return false
	}
}

// supportsISBNLookup は、ルートファサードがISBN参照へ委譲できる取得元か判定する
func supportsISBNLookup(source Source) bool {
	switch source {
	case SourceMADB, SourceOpenBD, SourceGoogleBooks, SourceRakutenBooks, SourceNDL, SourceYahooShopping:
		return true
	default:
		return false
	}
}

// invalidArgument は、ルート自身が検出した入力または設定エラーを生成する
func invalidArgument(operation string, err error) error {
	return &model.Error{Kind: model.ErrorKindInvalidArgument, Operation: operation, Err: err}
}
