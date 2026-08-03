package openbd

import (
	"context"
	"errors"
	"fmt"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

const maxISBNLookupCount = 1000

// isbnLookupInput は、元のISBNと問い合わせに使用するISBN-13を保持する
type isbnLookupInput struct {
	requested string
	canonical string
}

// LookupBooksByISBN は、指定されたISBNに対応する書籍をopenBDから参照する
func (client *Client) LookupBooksByISBN(
	ctx context.Context,
	isbns []string,
) (ISBNLookupResult, error) {
	result, _, err := client.lookupBooksByISBN(ctx, isbns)
	return result, err
}

// LookupBooksByISBNWithRawResponse は、ISBN参照結果と受信した成功レスポンス本文を返し、
// 変換失敗時も読み込み済み本文の所有権を呼び出し元へ移す
func (client *Client) LookupBooksByISBNWithRawResponse(
	ctx context.Context,
	isbns []string,
) (ISBNLookupResult, []byte, error) {
	return client.lookupBooksByISBN(ctx, isbns)
}

// lookupBooksByISBN は、openBDをISBNで参照して変換済み結果と成功レスポンス本文を返す
func (client *Client) lookupBooksByISBN(
	ctx context.Context,
	isbns []string,
) (ISBNLookupResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return ISBNLookupResult{}, nil, newError(
			operationISBNLookup,
			ErrorKindInvalidArgument,
			errors.New("client is not initialized"),
		)
	}
	if ctx == nil {
		return ISBNLookupResult{}, nil, newError(
			operationISBNLookup,
			ErrorKindInvalidArgument,
			errors.New("context must not be nil"),
		)
	}

	inputs, queries, err := validateISBNLookupInput(isbns)
	if err != nil {
		return ISBNLookupResult{}, nil, err
	}
	body, err := client.execute(ctx, queries)
	if err != nil {
		return ISBNLookupResult{}, body, err
	}
	response, err := decodeResponse(body)
	if err != nil {
		return ISBNLookupResult{}, body, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
	}
	result, err := buildISBNLookupResult(response, inputs, queries)
	if err != nil {
		return ISBNLookupResult{}, body, err
	}
	return result, body, nil
}

// validateISBNLookupInput は、入力ISBNを検証して重複のないISBN-13を生成する
func validateISBNLookupInput(isbns []string) ([]isbnLookupInput, []string, error) {
	if len(isbns) < 1 || len(isbns) > maxISBNLookupCount {
		return nil, nil, newError(
			operationISBNLookup,
			ErrorKindInvalidArgument,
			fmt.Errorf("isbn count must be between 1 and %d", maxISBNLookupCount),
		)
	}

	inputs := make([]isbnLookupInput, 0, len(isbns))
	queries := make([]string, 0, len(isbns))
	seen := make(map[string]struct{}, len(isbns))
	for _, requested := range isbns {
		canonical, err := canonicalISBN13(requested)
		if err != nil {
			return nil, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
		}
		inputs = append(inputs, isbnLookupInput{requested: requested, canonical: canonical})
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		queries = append(queries, canonical)
	}
	return inputs, queries, nil
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
