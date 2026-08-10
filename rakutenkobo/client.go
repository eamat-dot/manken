package rakutenkobo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	defaultEndpoint              = "https://openapi.rakuten.co.jp/services/api/Kobo/EbookSearch/20170426"
	successBodyMax               = 16 * 1024 * 1024
	errorBodyMax                 = 64 * 1024
	maxRetryAfterSeconds         = int64((1<<63 - 1) / time.Second)
	operationNewClient           = "rakutenkobo.NewClient"
	operationSearchBooks         = "rakutenkobo.SearchBooks"
	rakutenKoboGenreGeneralComic = "101904"
	rakutenKoboGenreBLComic      = "101940002"
	rakutenKoboGenreTLComic      = "101940011"
)

// Client は、楽天Kobo電子書籍検索APIへの接続設定を保持する
type Client struct {
	httpClient                                      *http.Client
	endpoint, applicationID, accessKey, affiliateID string
	comicGenre                                      ComicGenre
}

// clientOptions は、Clientの生成時に適用する設定を保持する
type clientOptions struct {
	endpoint, applicationID, accessKey, affiliateID string
	comicGenre                                      ComicGenre
}

// Option は、Clientの生成時に適用する設定を表す
type Option interface{ apply(*clientOptions) error }

// optionFunc は、関数をOptionとして適用できるようにする
type optionFunc func(*clientOptions) error

// apply は、関数が表す設定をclientOptionsへ適用する
func (option optionFunc) apply(options *clientOptions) error { return option(options) }

// NewClient は、指定されたHTTPクライアントと設定からClientを生成する
func NewClient(httpClient *http.Client, options ...Option) (*Client, error) {
	config := clientOptions{endpoint: defaultEndpoint, comicGenre: ComicGenreGeneral}
	for _, option := range options {
		if option == nil {
			return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("option must not be nil"))
		}
		if err := option.apply(&config); err != nil {
			return nil, newError(operationNewClient, ErrorKindInvalidArgument, err)
		}
	}
	if strings.TrimSpace(config.applicationID) == "" {
		return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("application ID must not be empty"))
	}
	if strings.TrimSpace(config.accessKey) == "" || strings.ContainsFunc(config.accessKey, unicode.IsControl) {
		return nil, newError(operationNewClient, ErrorKindInvalidArgument, errors.New("access key must not be empty or contain control characters"))
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	internalHTTPClient := *httpClient
	internalHTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{httpClient: &internalHTTPClient, endpoint: config.endpoint, applicationID: config.applicationID, accessKey: config.accessKey, affiliateID: config.affiliateID, comicGenre: config.comicGenre}, nil
}

// WithApplicationID は、楽天ウェブサービスのApplication IDを設定する
func WithApplicationID(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if strings.TrimSpace(value) == "" {
			return errors.New("application ID must not be empty")
		}
		options.applicationID = value
		return nil
	})
}

// WithAccessKey は、楽天ウェブサービスのAccess Keyを設定する
func WithAccessKey(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if strings.TrimSpace(value) == "" || strings.ContainsFunc(value, unicode.IsControl) {
			return errors.New("access key must not be empty or contain control characters")
		}
		options.accessKey = value
		return nil
	})
}

// WithAffiliateID は、楽天アフィリエイトURLの生成に使用するAffiliate IDを設定する
func WithAffiliateID(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if strings.TrimSpace(value) == "" {
			return errors.New("affiliate ID must not be empty")
		}
		options.affiliateID = value
		return nil
	})
}

// WithComicGenre は、楽天Kobo検索で対象にする漫画区分を設定する
func WithComicGenre(value ComicGenre) Option {
	return optionFunc(func(options *clientOptions) error {
		if _, err := comicGenreID(value); err != nil {
			return err
		}
		options.comicGenre = value
		return nil
	})
}

// WithEndpoint は、楽天Koboへの問い合わせに使用するエンドポイントを設定する
func WithEndpoint(value string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := validateEndpoint(value); err != nil {
			return err
		}
		options.endpoint = value
		return nil
	})
}

// comicGenreID は、公開漫画区分を楽天KoboのkoboGenreIdへ変換する
func comicGenreID(value ComicGenre) (string, error) {
	switch value {
	case ComicGenreGeneral:
		return rakutenKoboGenreGeneralComic, nil
	case ComicGenreBL:
		return rakutenKoboGenreBLComic, nil
	case ComicGenreTL:
		return rakutenKoboGenreTLComic, nil
	default:
		return "", fmt.Errorf("unsupported comic genre %q", value)
	}
}

// validateEndpoint は、楽天Koboエンドポイントとして使用できる絶対URLか検証する
func validateEndpoint(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("endpoint must be an absolute HTTP URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("endpoint must use http or https and include a host")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return errors.New("endpoint must use https unless its host is loopback")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("endpoint must not include user information, query, or fragment")
	}
	return nil
}

// isLoopbackHost は、名前解決をせずにhostがlocalhostまたはloopback IPか判定する
func isLoopbackHost(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}

// execute は、楽天Koboリクエストを送信して成功レスポンス本文を読み込む
func (client *Client) execute(ctx context.Context, operation string, query url.Values) ([]byte, error) {
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	query.Set("applicationId", client.applicationID)
	if client.affiliateID != "" {
		query.Set("affiliateId", client.affiliateID)
	}
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, sanitizeTransportError(err))
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("accessKey", client.accessKey)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, newError(operation, ErrorKindUnavailable, sanitizeTransportError(err))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
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

// sanitizeTransportError は、認証情報を含む可能性があるリクエストURLを通信エラーから除く
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
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
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
