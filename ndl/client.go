package ndl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/eamat-dot/manken/internal/httpendpoint"
	"github.com/eamat-dot/manken/internal/httpresponse"
)

const (
	defaultEndpoint = "https://ndlsearch.ndl.go.jp/api/sru"
	successBodyMax  = 16 * 1024 * 1024
	errorBodyMax    = 64 * 1024

	operationNewClient   = "ndl.NewClient"
	operationSearchBooks = "ndl.SearchBooks"
	operationISBNLookup  = "ndl.LookupBooksByISBN"
)

// Client は、国立国会図書館サーチへの接続設定を保持する
type Client struct {
	httpClient      *http.Client
	endpoint        string
	mangaNDCFilter  bool
	mangaNDLCFilter bool
}

// clientOptions は、Clientの生成時に適用する設定を保持する
type clientOptions struct {
	endpoint        string
	mangaNDCFilter  bool
	mangaNDLCFilter bool
}

// Option は、Clientの生成時に適用する設定を表す
type Option interface{ apply(*clientOptions) error }

// optionFunc は、関数をOptionとして適用できるようにする
type optionFunc func(*clientOptions) error

// apply は、関数が表す設定をclientOptionsへ適用する
func (option optionFunc) apply(options *clientOptions) error { return option(options) }

// NewClient は、指定されたHTTPクライアントと設定から認証不要のClientを生成する
func NewClient(httpClient *http.Client, options ...Option) (*Client, error) {
	config := clientOptions{endpoint: defaultEndpoint, mangaNDCFilter: true, mangaNDLCFilter: true}
	for _, option := range options {
		if option == nil {
			return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("option must not be nil"))
		}
		if err := option.apply(&config); err != nil {
			return nil, newError(operationNewClient, ErrorKindInvalidArgument, err)
		}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	internal := *httpClient
	// 検索条件を含むqueryを意図しないredirect先へ転送しないため、呼び出し側のredirect設定は引き継がない
	internal.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{
		httpClient:      &internal,
		endpoint:        config.endpoint,
		mangaNDCFilter:  config.mangaNDCFilter,
		mangaNDLCFilter: config.mangaNDLCFilter,
	}, nil
}

// WithEndpoint は、問い合わせに使用するSRUエンドポイントを設定する
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := httpendpoint.ValidateHTTPSOrLoopback(endpoint); err != nil {
			return err
		}
		options.endpoint = endpoint
		return nil
	})
}

// WithMangaNDCFilter は、通常検索でNDC 726.1の漫画分類フィルタを有効または無効にする
func WithMangaNDCFilter(enabled bool) Option {
	return optionFunc(func(options *clientOptions) error {
		options.mangaNDCFilter = enabled
		return nil
	})
}

// WithMangaNDLCFilter は、通常検索でNDLC Y84の漫画分類フィルタを有効または無効にする
func WithMangaNDLCFilter(enabled bool) Option {
	return optionFunc(func(options *clientOptions) error {
		options.mangaNDLCFilter = enabled
		return nil
	})
}

// execute は、SRUリクエストを送信して成功レスポンス本文を読み込む
func (client *Client) execute(ctx context.Context, operation string, values url.Values) ([]byte, error) {
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	endpoint.RawQuery = values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	request.Header.Set("Accept", "application/xml")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, newError(operation, ErrorKindUnavailable, sanitizeTransportError(err))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, copyErr := io.Copy(io.Discard, io.LimitReader(response.Body, errorBodyMax))
		closeErr := response.Body.Close()
		return nil, newHTTPError(operation, response, errors.Join(copyErr, closeErr))
	}
	body, err := httpresponse.ReadLimitedBody(response.Body, successBodyMax)
	closeErr := response.Body.Close()
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidResponse, err)
	}
	if closeErr != nil {
		return body, newError(operation, ErrorKindInvalidResponse, fmt.Errorf("close response body: %w", closeErr))
	}
	return body, nil
}

// sanitizeTransportError は、検索語を含む可能性があるリクエストURLを通信エラーから除く
func sanitizeTransportError(err error) error {
	for {
		var urlError *url.Error
		if !errors.As(err, &urlError) || urlError.Err == nil {
			return err
		}
		if urlError.Err == err {
			return err
		}
		err = urlError.Err
	}
}

// newHTTPError は、HTTPステータスとRetry-Afterから分類済みエラーを生成する
func newHTTPError(operation string, response *http.Response, cleanupErr error) error {
	kind := ErrorKindUpstream
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		kind = ErrorKindUnavailable
	}
	cause := fmt.Errorf("upstream returned HTTP status %d", response.StatusCode)
	if cleanupErr != nil {
		cause = fmt.Errorf("%w: clean up response body: %v", cause, cleanupErr)
	}
	return &Error{Kind: kind, Operation: operation, StatusCode: response.StatusCode, RetryAfter: httpresponse.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now()), Err: cause}
}

// newError は、操作と分類を設定したErrorを生成する
func newError(operation string, kind ErrorKind, err error) error {
	return &Error{Kind: kind, Operation: operation, Err: err}
}
