package googlebooks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

const maxISBNLookupCount = 1

// isbnLookupInput は、元のISBNと問い合わせに使用するISBN-13を保持する
type isbnLookupInput struct {
	requested string
	canonical string
}

// LookupBooksByISBN は、指定された1件のISBNに対応する書籍をGoogle Booksから参照する
func (client *Client) LookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, error) {
	result, _, err := client.lookupBooksByISBN(ctx, isbns)
	return result, err
}

// LookupBooksByISBNWithRawResponse は、ISBN参照結果と受信した成功レスポンス本文を返す
func (client *Client) LookupBooksByISBNWithRawResponse(
	ctx context.Context,
	isbns []string,
) (ISBNLookupResult, []byte, error) {
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

// lookupISBN は、1件のISBNをGoogle Booksへ問い合わせて一致するVolumeだけを返す
func (client *Client) lookupISBN(ctx context.Context, isbn string) ([]Book, []byte, error) {
	values := url.Values{}
	values.Set("q", "isbn:"+isbn)
	values.Set("printType", "books")
	values.Set("projection", "full")
	values.Set("maxResults", strconv.Itoa(maxLimit))
	values.Set("startIndex", "0")

	body, err := client.execute(ctx, operationISBNLookup, values)
	if err != nil {
		return nil, body, err
	}
	response, err := decodeVolumesResponse(body)
	if err != nil {
		return nil, body, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
	}
	if response.TotalItems < len(response.Items) {
		return nil, body, newError(operationISBNLookup, ErrorKindInvalidResponse, errors.New("response totalItems is smaller than returned items"))
	}

	books := make([]Book, 0, len(response.Items))
	for _, item := range response.Items {
		if !volumeHasISBN(item, isbn) {
			continue
		}
		book, err := convertVolume(item)
		if err != nil {
			return nil, body, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
		}
		books = append(books, book)
	}
	return books, body, nil
}

// volumeHasISBN は、Volumeの検証済みISBNが問い合わせISBNと一致するか判定する
func volumeHasISBN(value volume, expected string) bool {
	for _, identifier := range value.VolumeInfo.IndustryIdentifiers {
		if identifier.Type != "ISBN_10" && identifier.Type != "ISBN_13" {
			continue
		}
		canonical, err := canonicalISBN13(identifier.Identifier)
		if err == nil && canonical == expected {
			return true
		}
	}
	return false
}
