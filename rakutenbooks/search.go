package rakutenbooks

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

// SearchBooks は、指定された検索条件に一致する楽天Booksの紙書籍を検索する
func (client *Client) SearchBooks(ctx context.Context, request SearchBooksRequest) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request)
	return result, err
}

// SearchBooksWithRawResponse は、検索結果と受信した成功レスポンス本文を返す
func (client *Client) SearchBooksWithRawResponse(ctx context.Context, request SearchBooksRequest) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request)
}

// searchBooks は、楽天Booksを検索して変換済み結果と成功レスポンス本文を返す
func (client *Client) searchBooks(ctx context.Context, request SearchBooksRequest) (SearchBooksResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	title, author, err := validateSearchRequest(request)
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
	searchKey := buildSearchKey(title, author, client.comicGenre, client.bookSize)
	page, err := decodeCursor(request.Cursor, searchKey, limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	body, err := client.execute(ctx, operationSearchBooks, searchValues(title, author, genreID, client.bookSize, limit, page))
	if err != nil {
		return SearchBooksResult{}, body, err
	}
	observedAt := time.Now().UTC().Format(time.RFC3339Nano)
	response, err := decodeBooksResponse(body)
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	result, err := buildSearchResult(response, page, searchKey, limit, observedAt)
	if err != nil {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
	}
	return result, body, nil
}

// validateSearchRequest は、共通検索条件から楽天Booksが扱えるTitleとAuthorを取り出す
func validateSearchRequest(request SearchBooksRequest) (string, string, error) {
	title := strings.TrimSpace(request.Title)
	author := strings.TrimSpace(request.Author)
	if title == "" && author == "" {
		return "", "", errors.New("at least one of Title or Author must be specified")
	}
	if strings.TrimSpace(request.FreeText) != "" {
		return "", "", errors.New("FreeText is not supported by Rakuten Books")
	}
	if strings.TrimSpace(request.ExcludedText) != "" {
		return "", "", errors.New("ExcludedText is not supported by Rakuten Books")
	}
	return title, author, nil
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
func buildSearchKey(title string, author string, genre ComicGenre, size BookSize) string {
	return title + "\x00" + author + "\x00" + string(genre) + "\x00" + strconv.Itoa(int(size))
}

// searchValues は、楽天ブックス書籍検索APIの検索パラメーターを組み立てる
func searchValues(title string, author string, genreID string, size BookSize, limit int, page int) url.Values {
	values := baseValues()
	if title != "" {
		values.Set("title", title)
	}
	if author != "" {
		values.Set("author", author)
	}
	values.Set("booksGenreId", genreID)
	values.Set("hits", strconv.Itoa(limit))
	values.Set("page", strconv.Itoa(page))
	values.Set("sort", "+releaseDate")
	if size != BookSizeAll {
		values.Set("size", strconv.Itoa(int(size)))
	}
	return values
}

// baseValues は、楽天BooksへのJSONリクエストで共通して使うパラメーターを返す
func baseValues() url.Values {
	values := url.Values{}
	values.Set("format", "json")
	values.Set("formatVersion", "2")
	return values
}

// buildSearchResult は、検索応答を共通の検索結果と次カーソルへ変換する
func buildSearchResult(response booksResponse, page int, searchKey string, limit int, observedAt string) (SearchBooksResult, error) {
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
