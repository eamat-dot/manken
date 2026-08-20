// Package openbd は、openBDからISBNで書誌情報を参照する機能を提供する
package openbd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eamat-dot/manken/internal/httpendpoint"
	"github.com/eamat-dot/manken/internal/httpresponse"
)

const (
	defaultEndpoint = "https://api.openbd.jp/v1/get"
	successBodyMax  = 64 * 1024 * 1024
	errorBodyMax    = 64 * 1024

	operationNewClient  = "openbd.NewClient"
	operationISBNLookup = "openbd.LookupBooksByISBN"
)

// Client は、openBDへの接続設定を保持する
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// clientOptions は、Clientの生成時に適用する設定を保持する
type clientOptions struct {
	endpoint string
}

// Option は、Clientの生成時に適用する設定を表す
type Option interface {
	apply(*clientOptions) error
}

// optionFunc は、関数をOptionとして適用できるようにする
type optionFunc func(*clientOptions) error

// apply は、関数が表す設定をclientOptionsへ適用する
func (option optionFunc) apply(options *clientOptions) error {
	return option(options)
}

// NewClient は、指定されたHTTPクライアントと設定からClientを生成する
func NewClient(httpClient *http.Client, options ...Option) (*Client, error) {
	config := clientOptions{endpoint: defaultEndpoint}
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

	return &Client{httpClient: httpClient, endpoint: config.endpoint}, nil
}

// WithEndpoint は、openBDへの問い合わせに使用するエンドポイントを設定する
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := httpendpoint.Validate(endpoint); err != nil {
			return err
		}
		options.endpoint = endpoint
		return nil
	})
}

// execute は、ISBN参照リクエストを送信して成功レスポンス本文を読み込む
func (client *Client) execute(ctx context.Context, isbns []string) ([]byte, error) {
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
	}
	query := endpoint.Query()
	query.Set("isbn", strings.Join(isbns, ","))
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
	}
	httpRequest.Header.Set("Accept", "application/json")

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, newError(operationISBNLookup, ErrorKindUnavailable, err)
	}

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		_, copyErr := io.Copy(io.Discard, io.LimitReader(httpResponse.Body, errorBodyMax))
		closeErr := httpResponse.Body.Close()
		return nil, newHTTPError(httpResponse, errors.Join(copyErr, closeErr))
	}

	body, err := httpresponse.ReadLimitedBody(httpResponse.Body, successBodyMax)
	closeErr := httpResponse.Body.Close()
	if err != nil {
		return nil, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
	}
	if closeErr != nil {
		return body, newError(
			operationISBNLookup,
			ErrorKindInvalidResponse,
			fmt.Errorf("close response body: %w", closeErr),
		)
	}
	return body, nil
}

// newHTTPError は、HTTPステータスとRetry-Afterから分類済みエラーを生成する
func newHTTPError(response *http.Response, cleanupErr error) error {
	kind := ErrorKindUpstream
	if response.StatusCode == http.StatusRequestTimeout ||
		response.StatusCode == http.StatusTooManyRequests ||
		response.StatusCode >= http.StatusInternalServerError {
		kind = ErrorKindUnavailable
	}

	cause := fmt.Errorf("upstream returned HTTP status %d", response.StatusCode)
	if cleanupErr != nil {
		cause = fmt.Errorf("%w: clean up response body: %v", cause, cleanupErr)
	}
	return &Error{
		Kind:       kind,
		Operation:  operationISBNLookup,
		StatusCode: response.StatusCode,
		RetryAfter: httpresponse.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now()),
		Err:        cause,
	}
}

// newError は、操作と分類を設定したErrorを生成する
func newError(operation string, kind ErrorKind, err error) error {
	return &Error{Kind: kind, Operation: operation, Err: err}
}
