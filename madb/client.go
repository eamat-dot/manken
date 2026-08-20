// Package madb は、メディア芸術データベースから漫画本を検索する機能を提供する
package madb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eamat-dot/manken/internal/daterange"
	"github.com/eamat-dot/manken/internal/httpendpoint"
	"github.com/eamat-dot/manken/internal/httpresponse"
)

const (
	defaultEndpoint = "https://mediaarts-db.artmuseums.go.jp/sparql"
	defaultLimit    = 20
	maxLimit        = 100
	successBodyMax  = 4 * 1024 * 1024
	errorBodyMax    = 64 * 1024
	yearPrecision   = 4
	monthPrecision  = 7

	operationNewClient   = "madb.NewClient"
	operationSearchBooks = "madb.SearchBooks"
	operationISBNLookup  = "madb.LookupBooksByISBN"
)

// Client は、メディア芸術データベースへの接続設定を保持する
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

	return &Client{
		httpClient: httpClient,
		endpoint:   config.endpoint,
	}, nil
}

// WithEndpoint は、MADBへの問い合わせに使用するSPARQLエンドポイントを設定する
func WithEndpoint(endpoint string) Option {
	return optionFunc(func(options *clientOptions) error {
		if err := httpendpoint.Validate(endpoint); err != nil {
			return err
		}
		options.endpoint = endpoint
		return nil
	})
}

// SearchBooks は、指定条件に一致する漫画本をMADBから検索する
func (client *Client) SearchBooks(
	ctx context.Context,
	request SearchRequest,
) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request)
	return result, err
}

// SearchBooksWithRawResponse は、検索結果と受信した成功レスポンス本文を返し、
// 変換失敗時も読み込み済み本文の所有権を呼び出し元へ移す
func (client *Client) SearchBooksWithRawResponse(
	ctx context.Context,
	request SearchRequest,
) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request)
}

// searchBooks は、MADBを検索して変換済み結果と成功レスポンス本文を返す
func (client *Client) searchBooks(
	ctx context.Context,
	request SearchRequest,
) (SearchBooksResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksResult{}, nil, newError(
			operationSearchBooks,
			ErrorKindInvalidArgument,
			errors.New("client is not initialized"),
		)
	}
	if ctx == nil {
		return SearchBooksResult{}, nil, newError(
			operationSearchBooks,
			ErrorKindInvalidArgument,
			errors.New("context must not be nil"),
		)
	}

	conditions, limit, cursor, err := validateSearchRequest(request)
	if err != nil {
		return SearchBooksResult{}, nil, err
	}

	query := buildSearchQuery(conditions, limit, cursor.After)
	body, err := client.execute(ctx, operationSearchBooks, query)
	if err != nil {
		return SearchBooksResult{}, body, err
	}

	response, err := decodeSPARQLResponse(body)
	if err != nil {
		return SearchBooksResult{}, body, newError(
			operationSearchBooks,
			ErrorKindInvalidResponse,
			err,
		)
	}

	result, err := buildSearchResult(response, conditions, limit)
	if err != nil {
		return SearchBooksResult{}, body, err
	}
	return result, body, nil
}

// execute は、SPARQLクエリを送信して成功レスポンス本文を読み込む
func (client *Client) execute(ctx context.Context, operation string, query string) ([]byte, error) {
	form := url.Values{"query": []string{query}}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidArgument, err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.Header.Set("Accept", "application/sparql-results+json")

	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, newError(operation, ErrorKindUnavailable, err)
	}

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		_, copyErr := io.Copy(io.Discard, io.LimitReader(httpResponse.Body, errorBodyMax))
		closeErr := httpResponse.Body.Close()
		return nil, newHTTPError(operation, httpResponse, errors.Join(copyErr, closeErr))
	}

	body, err := httpresponse.ReadLimitedBody(httpResponse.Body, successBodyMax)
	closeErr := httpResponse.Body.Close()
	if err != nil {
		return nil, newError(operation, ErrorKindInvalidResponse, err)
	}
	if closeErr != nil {
		return body, newError(
			operation,
			ErrorKindInvalidResponse,
			fmt.Errorf("close response body: %w", closeErr),
		)
	}

	return body, nil
}

// validateSearchRequest は、検索条件を検証して実際に使用する値へ変換する
func validateSearchRequest(
	request SearchRequest,
) (searchConditions, int, cursorPayload, error) {
	conditions := searchConditions{
		Title:     strings.Join(strings.Fields(request.Title), " "),
		Author:    strings.Join(strings.Fields(request.Author), " "),
		Publisher: strings.Join(strings.Fields(request.Publisher), " "),
		Query:     strings.Join(strings.Fields(request.Query), " "),
		Exclude:   strings.Join(strings.Fields(request.Exclude), " "),
	}
	dateRange, err := daterange.Parse(request.DateFrom, request.DateTo)
	if err != nil {
		return searchConditions{}, 0, cursorPayload{}, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	conditions.DateFrom = dateRange.From
	conditions.DatePrecision = dateRange.Precision
	if !dateRange.ToExclusive().IsZero() {
		conditions.DateTo = formatDateExclusive(dateRange)
	}
	if conditions.Title == "" && conditions.Author == "" && conditions.Publisher == "" && conditions.Query == "" && conditions.DateFrom == "" && conditions.DateTo == "" {
		return searchConditions{}, 0, cursorPayload{}, newError(
			operationSearchBooks,
			ErrorKindInvalidArgument,
			errors.New("at least one search condition must be specified"),
		)
	}

	limit := request.Limit
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return searchConditions{}, 0, cursorPayload{}, newError(
			operationSearchBooks,
			ErrorKindInvalidArgument,
			fmt.Errorf("limit must be between 1 and %d", maxLimit),
		)
	}

	cursor, err := decodeCursor(request.Cursor, conditions, limit)
	if err != nil {
		return searchConditions{}, 0, cursorPayload{}, newError(
			operationSearchBooks,
			ErrorKindInvalidArgument,
			err,
		)
	}
	return conditions, limit, cursor, nil
}

// formatDateExclusive は、MADBの文字列比較に使う指定精度の半開区間終端を返す
func formatDateExclusive(dateRange daterange.Range) string {
	switch dateRange.Precision {
	case yearPrecision:
		return dateRange.ToExclusive().Format("2006")
	case monthPrecision:
		return dateRange.ToExclusive().Format("2006-01")
	default:
		return dateRange.ToExclusive().Format("2006-01-02")
	}
}

// newHTTPError は、HTTPステータスとRetry-Afterから分類済みエラーを生成する
func newHTTPError(operation string, response *http.Response, cleanupErr error) error {
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
		Operation:  operation,
		StatusCode: response.StatusCode,
		RetryAfter: httpresponse.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now()),
		Err:        cause,
	}
}

// newError は、操作と分類を設定したErrorを生成する
func newError(operation string, kind ErrorKind, err error) error {
	return &Error{
		Kind:      kind,
		Operation: operation,
		Err:       err,
	}
}
