package rakutenkobo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLimit = 20
	maxLimit     = 30
	maxPage      = 100
)

// SearchBooks は、指定された検索条件に一致する楽天Koboの電子書籍を検索する
func (client *Client) SearchBooks(ctx context.Context, request SearchRequest) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request)
	return result, err
}

// SearchBooksWithRawResponse は、検索結果と受信した成功レスポンス本文を返す
func (client *Client) SearchBooksWithRawResponse(ctx context.Context, request SearchRequest) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request)
}

// searchBooks は、楽天Koboを検索して変換済み結果と成功レスポンス本文を返す
func (client *Client) searchBooks(ctx context.Context, request SearchRequest) (SearchBooksResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	title, author, publisher, freeText, excludedText, err := validateSearchRequest(request)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	limit, err := effectiveLimit(request.Limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	genreID, err := comicGenreID(client.comicGenre)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	searchKey := buildSearchKey(title, author, publisher, freeText, excludedText, string(client.comicGenre))
	page, err := decodeCursor(request.Cursor, searchKey, limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	body, err := client.execute(ctx, operationSearchBooks, searchValues(title, author, publisher, freeText, excludedText, genreID, limit, page))
	if err != nil {
		return SearchBooksResult{}, body, err
	}
	response, err := decodeKoboResponse(body)
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	result, err := buildSearchResult(response, page, searchKey, limit, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	return result, body, nil
}

// validateSearchRequest は、共通検索条件から楽天Koboが扱える検索条件を取り出す
func validateSearchRequest(request SearchRequest) (string, string, string, string, string, error) {
	title, author, publisher, freeText, excludedText := strings.TrimSpace(request.Title), strings.TrimSpace(request.Author), strings.TrimSpace(request.Publisher), strings.TrimSpace(request.Query), strings.TrimSpace(request.Exclude)
	if title == "" && author == "" && publisher == "" && freeText == "" {
		return "", "", "", "", "", errors.New("at least one of Title, Author, Publisher, or Query must be specified")
	}
	if strings.TrimSpace(request.DateFrom) != "" || strings.TrimSpace(request.DateTo) != "" {
		return "", "", "", "", "", errors.New("date range search is not supported by Rakuten Kobo")
	}
	return title, author, publisher, freeText, excludedText, nil
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

// buildSearchKey は、カーソルを検索条件と漫画区分へ関連付ける文字列を生成する
func buildSearchKey(values ...string) string {
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
	}
	return builder.String()
}

// searchValues は、楽天Kobo電子書籍検索APIの検索パラメーターを組み立てる
func searchValues(title, author, publisher, freeText, excludedText, genreID string, limit, page int) url.Values {
	values := url.Values{"format": {"json"}, "formatVersion": {"2"}, "sort": {"+releaseDate"}, "language": {"JA"}, "field": {"0"}, "orFlag": {"0"}, "koboGenreId": {genreID}, "hits": {strconv.Itoa(limit)}, "page": {strconv.Itoa(page)}}
	if title != "" {
		values.Set("title", title)
	}
	if author != "" {
		values.Set("author", author)
	}
	if publisher != "" {
		values.Set("publisherName", publisher)
	}
	if freeText != "" {
		values.Set("keyword", freeText)
	} else if excludedText != "" {
		for _, fallback := range []string{title, author, publisher} {
			if fallback != "" {
				values.Set("keyword", fallback)
				break
			}
		}
	}
	if excludedText != "" {
		values.Set("NGKeyword", excludedText)
	}
	return values
}

// buildSearchResult は、検索応答を共通の検索結果と次カーソルへ変換する
func buildSearchResult(response koboResponse, page int, searchKey string, limit int, observedAt string) (SearchBooksResult, error) {
	if response.PageCount > 0 && page > response.PageCount {
		return SearchBooksResult{}, errors.New("response pageCount is smaller than requested page")
	}
	if len(response.Items) > 0 && response.Page != page {
		return SearchBooksResult{}, errors.New("response page does not match requested page")
	}
	books := make([]Book, 0, len(response.Items))
	for _, item := range response.Items {
		books = append(books, convertItem(item, observedAt))
	}
	result := SearchBooksResult{Books: books}
	if len(response.Items) == 0 || response.PageCount == 0 || page >= response.PageCount || page >= maxPage {
		return result, nil
	}
	cursor, err := encodeCursor(page+1, searchKey, limit)
	if err != nil {
		return SearchBooksResult{}, err
	}
	result.NextCursor = cursor
	return result, nil
}
