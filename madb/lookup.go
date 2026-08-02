package madb

import (
	"context"
	"errors"
	"fmt"
	"sort"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

const maxISBNLookupCount = 500

// isbnLookupInput は、元のISBNと問い合わせに使用する対応形式を保持する
type isbnLookupInput struct {
	Requested  string
	Candidates []string
}

// LookupBooksByISBN は、指定されたISBNに対応する漫画本をMADBから参照する
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

// lookupBooksByISBN は、MADBをISBNで参照して変換済み結果と成功レスポンス本文を返す
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

	inputs, candidates, err := validateISBNLookupInput(isbns)
	if err != nil {
		return ISBNLookupResult{}, nil, err
	}
	body, err := client.execute(ctx, operationISBNLookup, buildISBNLookupQuery(candidates))
	if err != nil {
		return ISBNLookupResult{}, body, err
	}
	response, err := decodeSPARQLResponse(body)
	if err != nil {
		return ISBNLookupResult{}, body, newError(
			operationISBNLookup,
			ErrorKindInvalidResponse,
			err,
		)
	}
	result, err := buildISBNLookupResult(response, inputs)
	if err != nil {
		return ISBNLookupResult{}, body, err
	}
	return result, body, nil
}

// validateISBNLookupInput は、ISBN参照入力を検証して問い合わせ候補を生成する
func validateISBNLookupInput(isbns []string) ([]isbnLookupInput, []string, error) {
	if len(isbns) < 1 || len(isbns) > maxISBNLookupCount {
		return nil, nil, newError(
			operationISBNLookup,
			ErrorKindInvalidArgument,
			fmt.Errorf("isbn count must be between 1 and %d", maxISBNLookupCount),
		)
	}

	inputs := make([]isbnLookupInput, 0, len(isbns))
	uniqueCandidates := make(map[string]struct{}, len(isbns)*2)
	for _, requested := range isbns {
		candidates, err := internalisbn.Normalize(requested)
		if err != nil {
			return nil, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
		}
		inputs = append(inputs, isbnLookupInput{
			Requested:  requested,
			Candidates: candidates,
		})
		for _, candidate := range candidates {
			uniqueCandidates[candidate] = struct{}{}
		}
	}

	candidates := make([]string, 0, len(uniqueCandidates))
	for candidate := range uniqueCandidates {
		candidates = append(candidates, candidate)
	}
	sort.Strings(candidates)
	return inputs, candidates, nil
}
