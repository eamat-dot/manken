package rakutenbooks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

const maxISBNLookupCount = 1

// isbnLookupInput は、元のISBNと問い合わせに使用するISBN-13を保持する
type isbnLookupInput struct {
	requested string
	canonical string
}

// LookupBooksByISBN は、指定された1件のISBNに対応する書籍を楽天Booksから参照する
func (client *Client) LookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, error) {
	result, _, err := client.lookupBooksByISBN(ctx, isbns)
	return result, err
}

// LookupBooksByISBNWithRawResponse は、ISBN参照結果と受信した成功レスポンス本文を返す
func (client *Client) LookupBooksByISBNWithRawResponse(ctx context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	return client.lookupBooksByISBN(ctx, isbns)
}

// lookupBooksByISBN は、1件のISBNを1回のHTTPリクエストで参照する
func (client *Client) lookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	input, err := validateISBNLookupInput(isbns)
	if err != nil {
		return ISBNLookupResult{}, nil, err
	}
	books, raw, err := client.lookupISBN(ctx, input.canonical)
	if err != nil {
		return ISBNLookupResult{}, raw, err
	}
	return ISBNLookupResult{Items: []ISBNLookupItem{{RequestedISBN: input.requested, Books: books}}}, raw, nil
}

// validateISBNLookupInput は、1件の入力ISBNを検証して問い合わせ用ISBN-13を生成する
func validateISBNLookupInput(isbns []string) (isbnLookupInput, error) {
	if len(isbns) != maxISBNLookupCount {
		return isbnLookupInput{}, newError(operationISBNLookup, ErrorKindInvalidArgument, fmt.Errorf("isbn count must be exactly %d", maxISBNLookupCount))
	}
	canonical, err := canonicalISBN13(isbns[0])
	if err != nil {
		return isbnLookupInput{}, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
	}
	return isbnLookupInput{requested: isbns[0], canonical: canonical}, nil
}

// canonicalISBN13 は、有効なISBNを問い合わせ用のISBN-13へ変換する
func canonicalISBN13(value string) (string, error) {
	candidates, err := internalisbn.Normalize(value)
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if internalisbn.IsValidISBN13(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("isbn cannot be represented as ISBN-13")
}

// lookupISBN は、1件のISBNを楽天Booksへ問い合わせて一致する商品だけを返す
func (client *Client) lookupISBN(ctx context.Context, isbn string) ([]Book, []byte, error) {
	values := baseValues()
	values.Set("isbn", isbn)
	values.Set("hits", strconv.Itoa(maxLimit))
	values.Set("page", "1")

	body, err := client.execute(ctx, operationISBNLookup, values)
	if err != nil {
		return nil, body, err
	}
	observedAt := time.Now().UTC().Format(time.RFC3339Nano)
	response, err := decodeBooksResponse(body)
	if err != nil {
		return nil, body, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
	}
	books := make([]Book, 0, len(response.Items))
	for _, item := range response.Items {
		if !itemHasISBN(item, isbn) {
			continue
		}
		books = append(books, convertItem(item, observedAt))
	}
	return books, body, nil
}

// itemHasISBN は、楽天Books商品の検証済みISBNが問い合わせISBNと一致するか判定する
func itemHasISBN(value booksItem, expected string) bool {
	canonical, err := canonicalISBN13(value.ISBN)
	return err == nil && canonical == expected
}
