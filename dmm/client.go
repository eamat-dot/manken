package dmm

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
	defaultEndpoint                    = "https://api.dmm.com/affiliate/v3/ItemList"
	successBodyMax               int64 = 16 * 1024 * 1024
	errorBodyMax                 int64 = 64 * 1024
	operationNewClient                 = "dmm.NewClient"
	operationSearchSeries              = "dmm.SearchSeries"
	operationSearchBooksBySeries       = "dmm.SearchBooksBySeries"
)

// Client は、DMMブックスItemListへの接続設定を保持する
type Client struct {
	httpClient                   *http.Client
	endpoint, apiID, affiliateID string
}

// clientOptions は、Clientの生成時に適用する設定を保持する
type clientOptions struct{ endpoint, apiID, affiliateID string }

// Option は、Clientの生成時に適用する設定を表す
type Option interface{ apply(*clientOptions) error }

// optionFunc は、関数をOptionとして適用できるようにする
type optionFunc func(*clientOptions) error

// apply は、関数が表す設定をclientOptionsへ適用する
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
	if invalidSecret(config.apiID) {
		return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("api ID must not be empty or contain control characters"))
	}
	if invalidSecret(config.affiliateID) {
		return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("affiliate ID must not be empty or contain control characters"))
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	internalClient := *httpClient
	// API IDとAffiliate IDを含むqueryをredirect先へ転送しないため、呼び出し側のredirect設定は引き継がない
	internalClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{httpClient: &internalClient, endpoint: config.endpoint, apiID: config.apiID, affiliateID: config.affiliateID}, nil
}

// invalidSecret は、認証情報として空白または制御文字を含む値か判定する
func invalidSecret(value string) bool {
	return strings.TrimSpace(value) == "" || strings.ContainsFunc(value, unicode.IsControl)
}

// WithAPIID は、DMM WebサービスのAPI IDを設定する
func WithAPIID(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if invalidSecret(value) {
			return errors.New("api ID must not be empty or contain control characters")
		}
		options.apiID = value
		return nil
	})
}

// WithAffiliateID は、DMMアフィリエイトIDを設定する
func WithAffiliateID(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if invalidSecret(value) {
			return errors.New("affiliate ID must not be empty or contain control characters")
		}
		options.affiliateID = value
		return nil
	})
}

// WithEndpoint は、DMMへの問い合わせに使用するエンドポイントを設定する
func WithEndpoint(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := httpendpoint.ValidateHTTPSOrLoopback(value); err != nil {
			return err
		}
		options.endpoint = value
		return nil
	})
}

// execute は、DMMリクエストを送信して成功レスポンス本文を読み込む
func (client *Client) execute(ctx context.Context, operation string, query url.Values) ([]byte, error) {
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	query.Set("api_id", client.apiID)
	query.Set("affiliate_id", client.affiliateID)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, sanitizeTransportError(err, client.apiID, client.affiliateID))
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, newError(operation, ErrorKindUnavailable, sanitizeTransportError(err, client.apiID, client.affiliateID))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, copyErr := io.Copy(io.Discard, io.LimitReader(response.Body, errorBodyMax))
		closeErr := response.Body.Close()
		return nil, newHTTPError(operation, response, sanitizeCredentialError(errors.Join(copyErr, closeErr), client.apiID, client.affiliateID))
	}
	body, err := httpresponse.ReadLimitedBody(response.Body, successBodyMax)
	closeErr := response.Body.Close()
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidResponse, sanitizeCredentialError(err, client.apiID, client.affiliateID))
	}
	if closeErr != nil {
		return body, newError(operation, ErrorKindInvalidResponse, fmt.Errorf("close response body: %w", sanitizeCredentialError(closeErr, client.apiID, client.affiliateID)))
	}
	return body, nil
}

// newHTTPError は、HTTPステータスとRetry-Afterから分類済みエラーを生成する
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

// sanitizeTransportError は、通信エラーの外側に含まれるリクエストURLを除く
func sanitizeTransportError(err error, secrets ...string) error {
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Err != nil {
		err = urlError.Err
	}
	return sanitizeCredentialError(err, secrets...)
}

// sanitizeCredentialError は、エラー文字列に含まれる認証情報を秘匿する
func sanitizeCredentialError(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	sanitized := err.Error()
	for _, secret := range secrets {
		sanitized = redactSecretTokens(sanitized, secret)
		sanitized = redactSecretTokens(sanitized, url.QueryEscape(secret))
	}
	if sanitized != err.Error() {
		return errors.New(sanitized)
	}
	return err
}

// newError は、操作と分類を設定したErrorを生成する
func newError(operation string, kind ErrorKind, err error) error {
	return &Error{Kind: kind, Operation: operation, Err: err}
}
