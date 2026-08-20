package yahooshopping

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

// LookupBooksByISBN は、指定された1件のISBNに対応するTower商品を参照する
func (client *Client) LookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, error) {
	result, _, err := client.lookupBooksByISBN(ctx, isbns)
	return result, err
}

// LookupBooksByISBNWithRawResponse は、ISBN参照結果と受信した成功レスポンス本文を返す
func (client *Client) LookupBooksByISBNWithRawResponse(ctx context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	return client.lookupBooksByISBN(ctx, isbns)
}

// lookupBooksByISBN は、ISBNを検証してYahoo!ショッピングへ問い合わせる
func (client *Client) lookupBooksByISBN(ctx context.Context, isbns []string) (ISBNLookupResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	if len(isbns) != 1 {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, fmt.Errorf("isbn count must be exactly 1"))
	}
	canonical, err := internalisbn.Canonical13(isbns[0])
	if err != nil {
		return ISBNLookupResult{}, nil, newError(operationISBNLookup, ErrorKindInvalidArgument, err)
	}
	values := url.Values{"jan_code": {canonical}, "seller_id": {towerSellerID}, "image_size": {strconv.Itoa(imageSize)}, "results": {strconv.Itoa(100)}}
	body, err := client.execute(ctx, operationISBNLookup, values)
	if err != nil {
		return ISBNLookupResult{}, body, err
	}
	response, err := decodeResponse(body)
	if err != nil {
		return ISBNLookupResult{}, body, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
	}
	books := make([]Book, 0, len(response.Hits))
	observedAt := time.Now().UTC().Format(time.RFC3339Nano)
	for _, value := range response.Hits {
		if value.Seller.SellerID == towerSellerID && canonicalItemISBN(value.JanCode) == canonical {
			books = append(books, convertItem(value, observedAt))
		}
	}
	return ISBNLookupResult{Items: []ISBNLookupItem{{RequestedISBN: isbns[0], Books: books}}}, body, nil
}

// canonicalItemISBN は、有効なISBN-13だけを返す
func canonicalItemISBN(value string) string {
	if internalisbn.IsValidISBN13(value) {
		return value
	}
	return ""
}
