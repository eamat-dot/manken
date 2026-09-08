package ndl

import (
	"context"
	"errors"
	"strings"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

// LookupBooksByISBN は、1件のISBNに対応する国立国会図書館サーチの書誌を参照する
func (client *Client) LookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, error) {
	result, _, err := client.lookupBooksByISBN(ctx, isbns)
	return result, err
}

// LookupBooksByISBNWithRawResponse は、ISBN参照結果と受信した成功レスポンスXMLを返す
func (client *Client) LookupBooksByISBNWithRawResponse(ctx context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	return client.lookupBooksByISBN(ctx, isbns)
}

// lookupBooksByISBN は、ISBN参照を実行して成功本文を併せて返す
func (client *Client) lookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	if len(isbns) != 1 {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("isbn count must be exactly 1"))
	}
	candidates, err := internalisbn.Normalize(isbns[0])
	if err != nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
	}
	canonical := candidates[0]
	query := strings.Join(append(fixedCQLTerms(), "isbn = "+quoteCQL(canonical)), " AND ")
	body, err := client.execute(ctx, operationISBNLookup, searchValues(query, 1, 1))
	if err != nil {
		return ISBNLookupResult{}, body, err
	}
	response, err := decodeResponse(body)
	if err != nil {
		return ISBNLookupResult{}, body, responseError(operationISBNLookup, err)
	}
	books := make([]Book, 0)
	if !response.notFound {
		for _, record := range response.records {
			book, err := convertRecord(record)
			if err != nil {
				return ISBNLookupResult{}, body, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
			}
			if hasISBN(book, candidates) {
				books = append(books, book)
			}
		}
	}
	return ISBNLookupResult{
		Items:        []ISBNLookupItem{{RequestedISBN: isbns[0], Books: books}},
		Attributions: Attributions(),
	}, body, nil
}

// hasISBN は、取得元ISBNを正規化候補と比較して同じISBNか判定する
func hasISBN(book Book, candidates []string) bool {
	for _, identifier := range book.ISBN10 {
		if hasNormalizedISBN(identifier, candidates) {
			return true
		}
	}
	for _, identifier := range book.ISBN13 {
		if hasNormalizedISBN(identifier, candidates) {
			return true
		}
	}
	return false
}

// hasNormalizedISBN は、取得元ISBNを一度正規化して候補群と比較する
func hasNormalizedISBN(identifier string, candidates []string) bool {
	normalized, err := internalisbn.Normalize(identifier)
	if err != nil {
		return false
	}
	for _, candidate := range candidates {
		if containsISBN(normalized, candidate) {
			return true
		}
	}
	return false
}

// containsISBN は、ISBN候補群に指定値が含まれるか判定する
func containsISBN(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
