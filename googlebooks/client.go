package googlebooks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEndpoint      = "https://www.googleapis.com/books/v1/volumes"
	successBodyMax       = 16 * 1024 * 1024
	errorBodyMax         = 64 * 1024
	maxRetryAfterSeconds = int64((1<<63 - 1) / time.Second)

	operationNewClient   = "googlebooks.NewClient"
	operationSearchBooks = "googlebooks.SearchBooks"
	operationISBNLookup  = "googlebooks.LookupBooksByISBN"
)

// Client は、Google Booksへの接続設定を保持する
type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

// clientOptions は、Clientの生成時に適用する設定を保持する
type clientOptions struct {
	endpoint string
	apiKey   string
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
	if strings.TrimSpace(config.apiKey) == "" {
		return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("API key must not be empty"))
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{httpClient: httpClient, endpoint: config.endpoint, apiKey: config.apiKey}, nil
}

// WithAPIKey は、Google Books APIキーを設定する
func WithAPIKey(apiKey string) Option {
	return optionFunc(func(options *clientOptions) error {
		if strings.TrimSpace(apiKey) == "" {
			return errors.New("API key must not be empty")
		}
		options.apiKey = apiKey
		return nil
	})
}

// WithEndpoint は、Google Booksへの問い合わせに使用するエンドポイントを設定する
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := validateEndpoint(endpoint); err != nil {
			return err
		}
		options.endpoint = endpoint
		return nil
	})
}

// validateEndpoint は、Google Booksエンドポイントとして使用できる絶対URLか検証する
func validateEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint must be an absolute HTTP URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("endpoint must use http or https and include a host")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("endpoint must not include user information, query, or fragment")
	}
	return nil
}

// execute は、Google Booksリクエストを送信して成功レスポンス本文を読み込む
func (client *Client) execute(ctx context.Context, operation string, query url.Values) ([]byte, error) {
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	query.Set("key", client.apiKey)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, newError(operation, ErrorKindUnavailable, sanitizeTransportError(err))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, copyErr := io.Copy(io.Discard, io.LimitReader(response.Body, errorBodyMax))
		closeErr := response.Body.Close()
		return nil, newHTTPError(operation, response, errors.Join(copyErr, closeErr))
	}
	body, err := readLimitedBody(response.Body, successBodyMax)
	closeErr := response.Body.Close()
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidResponse, err)
	}
	if closeErr != nil {
		return body, newError(operation, ErrorKindInvalidResponse, fmt.Errorf("close response body: %w", closeErr))
	}
	return body, nil
}

// sanitizeTransportError は、APIキーを含む可能性があるリクエストURLを通信エラーから除く
func sanitizeTransportError(err error) error {
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Err != nil {
		return urlError.Err
	}
	return err
}

// readLimitedBody は、上限を超えないレスポンス本文を読み込む
func readLimitedBody(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return body, nil
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
	return &Error{Kind: kind, Operation: operation, StatusCode: response.StatusCode, RetryAfter: parseRetryAfter(response.Header.Get("Retry-After"), time.Now()), Err: cause}
}

// parseRetryAfter は、Retry-Afterを待機時間へ変換する
func parseRetryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 && seconds <= maxRetryAfterSeconds {
		return time.Duration(seconds) * time.Second
	}
	date, err := http.ParseTime(value)
	if err != nil || !date.After(now) {
		return 0
	}
	return date.Sub(now)
}

// newError は、操作と分類を設定したErrorを生成する
func newError(operation string, kind ErrorKind, err error) error {
	return &Error{Kind: kind, Operation: operation, Err: err}
}
