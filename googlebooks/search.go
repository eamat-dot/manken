package googlebooks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	defaultLimit = 20
	maxLimit     = 40
)

// SearchBooks は、指定された検索条件に一致するGoogle Booksの書籍を検索する
func (client *Client) SearchBooks(ctx context.Context, request SearchBooksRequest) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request)
	return result, err
}

// SearchBooksWithRawResponse は、検索結果と受信した成功レスポンス本文を返す
func (client *Client) SearchBooksWithRawResponse(ctx context.Context, request SearchBooksRequest) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request)
}

// searchBooks は、Google Booksを検索して変換済み結果と成功レスポンス本文を返す
func (client *Client) searchBooks(ctx context.Context, request SearchBooksRequest) (SearchBooksResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	queryText, err := buildSearchQuery(request)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	limit, err := effectiveLimit(request.Limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	startIndex, err := decodeCursor(request.Cursor, queryText, limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	body, err := client.execute(ctx, operationSearchBooks, searchValues(queryText, limit, startIndex, true))
	if err != nil {
		return SearchBooksResult{}, body, err
	}
	response, err := decodeVolumesResponse(body)
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	observedAt := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := buildSearchResult(response, startIndex, queryText, limit, observedAt)
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	return result, body, nil
}

// buildSearchQuery は、共通検索条件をGoogle Booksの安全な検索式へ変換する
func buildSearchQuery(request SearchBooksRequest) (string, error) {
	parts := make([]string, 0)
	parts = append(parts, queryTerms("intitle:", request.Title)...)
	parts = append(parts, queryTerms("inauthor:", request.Author)...)
	parts = append(parts, queryTerms("inpublisher:", request.Publisher)...)
	parts = append(parts, queryTerms("", request.FreeText)...)
	if len(parts) == 0 {
		return "", errors.New("at least one of Title, Author, Publisher, or FreeText must be specified")
	}
	parts = append(parts, queryTerms("-", request.ExcludedText)...)
	return strings.Join(parts, " "), nil
}

// queryTerms は、利用者入力を演算子として解釈されない引用済み検索語へ変換する
func queryTerms(prefix string, value string) []string {
	words := strings.FieldsFunc(value, unicode.IsSpace)
	terms := make([]string, 0, len(words))
	for _, word := range words {
		escaped := strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(word)
		terms = append(terms, prefix+`"`+escaped+`"`)
	}
	return terms
}

// effectiveLimit は、検索件数を既定値または有効範囲の値へ変換する
func effectiveLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 1 || limit > maxLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxLimit)
	}
	return limit, nil
}

// searchValues は、Google Books Volumes listの検索パラメーターを組み立てる
func searchValues(queryText string, limit int, startIndex int, includeOrder bool) url.Values {
	values := url.Values{}
	values.Set("q", queryText)
	values.Set("printType", "books")
	values.Set("projection", "full")
	values.Set("maxResults", strconv.Itoa(limit))
	values.Set("startIndex", strconv.Itoa(startIndex))
	if includeOrder {
		values.Set("orderBy", "relevance")
	}
	return values
}

// buildSearchResult は、検索応答を共通の検索結果と次カーソルへ変換する
func buildSearchResult(response volumesResponse, startIndex int, queryText string, limit int, observedAt string) (SearchBooksResult, error) {
	if response.TotalItems < startIndex+len(response.Items) {
		return SearchBooksResult{}, errors.New("response totalItems is smaller than returned items")
	}
	books := make([]Book, 0, len(response.Items))
	for _, item := range response.Items {
		book, err := convertVolume(item, observedAt)
		if err != nil {
			return SearchBooksResult{}, err
		}
		books = append(books, book)
	}
	result := SearchBooksResult{Books: books}
	if len(response.Items) == 0 {
		if response.TotalItems > startIndex {
			return SearchBooksResult{}, errors.New("response pagination did not advance")
		}
		return result, nil
	}
	if response.TotalItems == startIndex+len(response.Items) {
		return result, nil
	}
	if response.TotalItems > startIndex+len(response.Items) {
		cursor, err := encodeCursor(startIndex+len(response.Items), queryText, limit)
		if err != nil {
			return SearchBooksResult{}, err
		}
		result.NextCursor = cursor
	}
	return result, nil
}
