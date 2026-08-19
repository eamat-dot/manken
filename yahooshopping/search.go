package yahooshopping

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
	imageSize    = 600
)

// SearchBooks は、Towerの紙書籍候補を商品キーワードで検索する
func (client *Client) SearchBooks(ctx context.Context, request SearchRequest) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request)
	return result, err
}

// SearchBooksWithRawResponse は、検索結果と受信した成功レスポンス本文を返す
func (client *Client) SearchBooksWithRawResponse(ctx context.Context, request SearchRequest) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request)
}

// searchBooks は、検索条件を検証してYahoo!ショッピングへ問い合わせる
func (client *Client) searchBooks(ctx context.Context, request SearchRequest) (SearchBooksResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	query, err := searchQuery(request)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	limit, err := effectiveLimit(request.Limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	key := query
	start, err := decodeCursor(request.Cursor, key, limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	values := url.Values{"query": {query}, "seller_id": {towerSellerID}, "genre_category_id": {strconv.FormatInt(comicGenreCategoryID, 10)}, "image_size": {strconv.Itoa(imageSize)}, "results": {strconv.Itoa(limit)}, "start": {strconv.Itoa(start)}}
	body, err := client.execute(ctx, operationSearchBooks, values)
	if err != nil {
		return SearchBooksResult{}, body, err
	}
	response, err := decodeResponse(body)
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	return buildSearchResult(response, key, limit, time.Now().UTC().Format(time.RFC3339Nano)), body, nil
}

// searchQuery は、共通検索条件をYahoo!ショッピングの商品キーワードへ変換する
func searchQuery(request SearchRequest) (string, error) {
	if strings.TrimSpace(request.Exclude) != "" {
		return "", errors.New("exclude is not supported by Yahoo Shopping")
	}
	if strings.TrimSpace(request.DateFrom) != "" || strings.TrimSpace(request.DateTo) != "" {
		return "", errors.New("date range search is not supported by Yahoo Shopping")
	}
	fields := []string{strings.TrimSpace(request.Title), strings.TrimSpace(request.Author), strings.TrimSpace(request.Publisher), strings.TrimSpace(request.Query)}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != "" {
			parts = append(parts, field)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("at least one of Title, Author, Publisher, or Query must be specified")
	}
	return strings.Join(parts, " "), nil
}

// effectiveLimit は、既定値を補いYahoo!ショッピングの取得件数範囲を検証する
func effectiveLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 1 || limit > 100 {
		return 0, fmt.Errorf("limit must be between 1 and 100")
	}
	return limit, nil
}

// buildSearchResult は、取得元の応答から検索結果と次ページカーソルを作成する
func buildSearchResult(response searchResponse, key string, limit int, observedAt string) SearchBooksResult {
	result := SearchBooksResult{Books: make([]Book, 0, len(response.Hits))}
	for _, value := range response.Hits {
		if !isTowerComic(value) {
			continue
		}
		result.Books = append(result.Books, convertItem(value, observedAt))
	}
	next := response.FirstResultsPosition + response.TotalResultsReturned
	if response.TotalResultsReturned > 0 && requestStartIsValid(next, limit) && next <= response.TotalResultsAvailable {
		if cursor, err := encodeCursor(next, key, limit); err == nil {
			result.NextCursor = cursor
		}
	}
	return result
}

// isTowerComic は、Towerの漫画単巻候補かを判定する
func isTowerComic(value item) bool {
	if value.Seller.SellerID != towerSellerID {
		return false
	}
	if value.GenreCategory.ID == wholeSetGenreCategoryID || isWholeSetName(value.Name) {
		return false
	}
	return value.GenreCategory.ID == comicGenreCategoryID || hasGenre(value.ParentGenreCategories, comicGenreCategoryID)
}

// isWholeSetName は、商品名が全巻セットを示すかを判定する
func isWholeSetName(name string) bool {
	return strings.Contains(name, "全巻セット") || wholeSetPattern.MatchString(name)
}

// hasGenre は、ジャンル一覧に指定IDが含まれるかを判定する
func hasGenre(values []genre, id int64) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
