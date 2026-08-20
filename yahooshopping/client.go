package yahooshopping

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/eamat-dot/manken/internal/httpendpoint"
	"github.com/eamat-dot/manken/internal/httpresponse"
)

const (
	defaultEndpoint               = "https://shopping.yahooapis.jp/ShoppingWebService/V3/itemSearch"
	successBodyMax                = 16 * 1024 * 1024
	errorBodyMax                  = 64 * 1024
	operationNewClient            = "yahooshopping.NewClient"
	operationSearchBooks          = "yahooshopping.SearchBooks"
	operationISBNLookup           = "yahooshopping.LookupBooksByISBN"
	towerSellerID                 = "tower"
	comicGenreCategoryID    int64 = 10251
	wholeSetGenreCategoryID int64 = 15285
)

// Client は、Yahoo!ショッピング商品検索APIへの接続設定を保持する
type Client struct {
	httpClient         *http.Client
	endpoint, clientID string
}

type clientOptions struct{ endpoint, clientID string }

// Option は、Clientの接続設定を変更する
type Option interface{ apply(*clientOptions) error }

type optionFunc func(*clientOptions) error

// apply は、関数形式の設定をClient設定へ適用する
func (option optionFunc) apply(options *clientOptions) error { return option(options) }

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
	if strings.TrimSpace(config.clientID) == "" || strings.ContainsFunc(config.clientID, unicode.IsControl) {
		return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("client ID must not be empty or contain control characters"))
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	internalClient := *httpClient
	// Client IDを含むqueryをredirect先へ転送しないため、呼び出し側のredirect設定は引き継がない
	internalClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{httpClient: &internalClient, endpoint: config.endpoint, clientID: config.clientID}, nil
}

// WithClientID は、Yahoo!デベロッパーネットワークのClient IDを設定する
func WithClientID(clientID string) Option {
	return optionFunc(func(options *clientOptions) error {
		if strings.TrimSpace(clientID) == "" || strings.ContainsFunc(clientID, unicode.IsControl) {
			return errors.New("client ID must not be empty or contain control characters")
		}
		options.clientID = clientID
		return nil
	})
}

// WithEndpoint は、Yahoo!ショッピングへの問い合わせに使用するエンドポイントを設定する
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := httpendpoint.ValidateHTTPSOrLoopback(endpoint); err != nil {
			return err
		}
		options.endpoint = endpoint
		return nil
	})
}

// execute は、認証情報とクエリを付けたGET要求を実行する
func (client *Client) execute(ctx context.Context, operation string, query url.Values) ([]byte, error) {
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	query.Set("appid", client.clientID)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, sanitizeTransportError(err))
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, newError(operation, ErrorKindUnavailable, sanitizeTransportError(err))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
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

// sanitizeTransportError は、URLに含まれる認証情報をエラーから取り除く
func sanitizeTransportError(err error) error {
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Err != nil {
		return urlError.Err
	}
	return err
}

// newHTTPError は、取得元HTTP応答を共通エラーへ変換する
func newHTTPError(operation string, response *http.Response, cleanupErr error) error {
	kind := ErrorKindUpstream
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		kind = ErrorKindUnavailable
	}
	cause := fmt.Errorf("upstream returned HTTP status %d", response.StatusCode)
	if cleanupErr != nil {
		cause = fmt.Errorf("%w: clean up response body: %v", cause, cleanupErr)
	}
	return &Error{Kind: kind, Operation: operation, StatusCode: response.StatusCode, RetryAfter: httpresponse.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now()), Err: cause}
}

// newError は、共通エラーに操作名と分類を設定する
func newError(operation string, kind ErrorKind, err error) error {
	return &Error{Kind: kind, Operation: operation, Err: err}
}
