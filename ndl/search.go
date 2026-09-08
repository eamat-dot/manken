package ndl

import (
	"context"
	"errors"
)

// SearchOptions は、NDLサーチ固有の検索条件を表す
type SearchOptions struct {
	// Subject は、件名に含める検索語を指定する
	Subject string
	// Description は、内容記述に含める検索語を指定する
	Description string
}

// SearchBooks は、指定条件に一致する国立国会図書館サーチの書誌を検索する
func (client *Client) SearchBooks(ctx context.Context, request SearchRequest) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request, SearchOptions{})
	return result, err
}

// SearchBooksWithRawResponse は、検索結果と受信した成功レスポンスXMLを返す
func (client *Client) SearchBooksWithRawResponse(ctx context.Context, request SearchRequest) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request, SearchOptions{})
}

// SearchBooksWithOptions は、共通検索条件とNDL固有検索条件で書誌を検索する
func (client *Client) SearchBooksWithOptions(ctx context.Context, request SearchRequest, options SearchOptions) (SearchBooksResult, error) {
	result, _, err := client.searchBooks(ctx, request, options)
	return result, err
}

// SearchBooksWithOptionsAndRawResponse は、検索結果と受信した成功レスポンスXMLを返す
func (client *Client) SearchBooksWithOptionsAndRawResponse(ctx context.Context, request SearchRequest, options SearchOptions) (SearchBooksResult, []byte, error) {
	return client.searchBooks(ctx, request, options)
}

// searchBooks は、検索を実行して成功本文を併せて返す
func (client *Client) searchBooks(ctx context.Context, request SearchRequest, options SearchOptions) (SearchBooksResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	query, err := buildSearchQueryWithOptions(request, options, client.mangaNDCFilter, client.mangaNDLCFilter)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	limit, err := effectiveLimit(request.Limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	start, err := decodeCursor(request.Cursor, query, limit)
	if err != nil {
		return SearchBooksResult{}, nil, newError(operationSearchBooks, ErrorKindInvalidArgument, err)
	}
	body, err := client.execute(ctx, operationSearchBooks, searchValues(query, limit, start))
	if err != nil {
		return SearchBooksResult{}, body, err
	}
	response, err := decodeResponse(body)
	if err != nil {
		return SearchBooksResult{}, body, responseError(operationSearchBooks, err)
	}
	if response.notFound {
		return SearchBooksResult{Books: []Book{}, Attributions: Attributions()}, body, nil
	}
	if response.next > 0 && response.next <= start {
		return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, errors.New("response pagination did not advance"))
	}
	books := make([]Book, 0, len(response.records))
	for _, record := range response.records {
		book, err := convertRecord(record)
		if err != nil {
			return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
		}
		books = append(books, book)
	}
	result := SearchBooksResult{Books: books, Attributions: Attributions()}
	if response.next > 0 {
		if response.next < 501 {
			cursor, err := encodeCursor(response.next, query, limit)
			if err != nil {
				return SearchBooksResult{}, body, newError(operationSearchBooks, ErrorKindInvalidResponse, err)
			}
			result.NextCursor = cursor
		}
	}
	return result, body, nil
}

// responseError は、SRU Diagnosticsと壊れた成功XMLを既存のエラー分類へ変換する
func responseError(operation string, err error) error {
	var diagnostic diagnosticError
	if errors.As(err, &diagnostic) {
		return newError(operation, ErrorKindUpstream, err)
	}
	return newError(operation, ErrorKindInvalidResponse, err)
}
