package dmm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eamat-dot/manken/internal/daterange"
)

const (
	defaultLimit           = 20
	cursorAPISeries        = "search_series"
	cursorAPIBooksBySeries = "search_books_by_series"
)

// SearchSeries は、DMMが付与するシリーズをフリーワードで検索する
func (client *Client) SearchSeries(ctx context.Context, request SearchSeriesRequest) (SearchSeriesResult, error) {
	result, _, err := client.searchSeries(ctx, request)
	return result, err
}

// SearchSeriesWithRawResponse は、DMMシリーズ検索結果と秘匿済みRaw responseを返す
func (client *Client) SearchSeriesWithRawResponse(ctx context.Context, request SearchSeriesRequest) (SearchSeriesResult, []byte, error) {
	return client.searchSeries(ctx, request)
}

// SearchBooksBySeries は、DMMシリーズに属する個別商品を共通Bookとして返す
func (client *Client) SearchBooksBySeries(ctx context.Context, request SearchBooksBySeriesRequest) (SearchBooksBySeriesResult, error) {
	result, _, err := client.searchBooksBySeries(ctx, request)
	return result, err
}

// SearchBooksBySeriesWithRawResponse は、DMMシリーズの商品と秘匿済みRaw responseを返す
func (client *Client) SearchBooksBySeriesWithRawResponse(ctx context.Context, request SearchBooksBySeriesRequest) (SearchBooksBySeriesResult, []byte, error) {
	return client.searchBooksBySeries(ctx, request)
}

// searchSeries は、入力を検証してDMMシリーズ検索結果と秘匿済みRaw responseを返す
func (client *Client) searchSeries(ctx context.Context, request SearchSeriesRequest) (SearchSeriesResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchSeriesResult{}, nil, newError(operationSearchSeries, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchSeriesResult{}, nil, newError(operationSearchSeries, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	keyword, dateRange, err := searchSeriesKeyword(request)
	if err != nil {
		return SearchSeriesResult{}, nil, newError(operationSearchSeries, ErrorKindInvalidArgument, err)
	}
	limit, err := effectiveLimit(request.Limit)
	if err != nil {
		return SearchSeriesResult{}, nil, newError(operationSearchSeries, ErrorKindInvalidArgument, err)
	}
	key := seriesSearchKey(keyword, dateRange)
	offset, err := decodeCursor(request.Cursor, cursorAPISeries, key, limit)
	if err != nil {
		return SearchSeriesResult{}, nil, newError(operationSearchSeries, ErrorKindInvalidArgument, err)
	}
	body, err := client.execute(ctx, operationSearchSeries, keywordQuery(keyword, dateRange, limit, offset))
	if err != nil {
		return SearchSeriesResult{}, nil, err
	}
	response, err := decodeResponse(body)
	raw, rawErr := sanitizeRaw(body, client.apiID, client.affiliateID)
	if rawErr != nil {
		return SearchSeriesResult{}, nil, newError(operationSearchSeries, ErrorKindInvalidResponse, rawErr)
	}
	if err != nil {
		return SearchSeriesResult{}, raw, newError(operationSearchSeries, ErrorKindInvalidResponse, err)
	}
	if response.Result.Status != http.StatusOK {
		return SearchSeriesResult{}, raw, newError(operationSearchSeries, ErrorKindUpstream, fmt.Errorf("dmm result status %d", response.Result.Status))
	}
	result, err := buildSeriesResult(response.Result, offset, key, limit)
	if err != nil {
		return SearchSeriesResult{}, raw, newError(operationSearchSeries, ErrorKindInvalidResponse, err)
	}
	return result, raw, nil
}

// searchBooksBySeries は、入力を検証してDMMシリーズ内Bookと秘匿済みRaw responseを返す
func (client *Client) searchBooksBySeries(ctx context.Context, request SearchBooksBySeriesRequest) (SearchBooksBySeriesResult, []byte, error) {
	if client == nil || client.httpClient == nil {
		return SearchBooksBySeriesResult{}, nil, newError(operationSearchBooksBySeries, ErrorKindInvalidArgument, errors.New("client is not initialized"))
	}
	if ctx == nil {
		return SearchBooksBySeriesResult{}, nil, newError(operationSearchBooksBySeries, ErrorKindInvalidArgument, errors.New("context must not be nil"))
	}
	seriesID := strings.TrimSpace(request.SeriesID)
	if seriesID == "" {
		return SearchBooksBySeriesResult{}, nil, newError(operationSearchBooksBySeries, ErrorKindInvalidArgument, errors.New("series ID must not be empty"))
	}
	limit, err := effectiveLimit(request.Limit)
	if err != nil {
		return SearchBooksBySeriesResult{}, nil, newError(operationSearchBooksBySeries, ErrorKindInvalidArgument, err)
	}
	offset, err := decodeCursor(request.Cursor, cursorAPIBooksBySeries, seriesID, limit)
	if err != nil {
		return SearchBooksBySeriesResult{}, nil, newError(operationSearchBooksBySeries, ErrorKindInvalidArgument, err)
	}
	body, err := client.execute(ctx, operationSearchBooksBySeries, seriesQuery(seriesID, limit, offset))
	if err != nil {
		return SearchBooksBySeriesResult{}, nil, err
	}
	response, err := decodeResponse(body)
	raw, rawErr := sanitizeRaw(body, client.apiID, client.affiliateID)
	if rawErr != nil {
		return SearchBooksBySeriesResult{}, nil, newError(operationSearchBooksBySeries, ErrorKindInvalidResponse, rawErr)
	}
	if err != nil {
		return SearchBooksBySeriesResult{}, raw, newError(operationSearchBooksBySeries, ErrorKindInvalidResponse, err)
	}
	if response.Result.Status != http.StatusOK {
		return SearchBooksBySeriesResult{}, raw, newError(operationSearchBooksBySeries, ErrorKindUpstream, fmt.Errorf("dmm result status %d", response.Result.Status))
	}
	result, err := buildBooksBySeriesResult(response.Result, offset, seriesID, limit)
	if err != nil {
		return SearchBooksBySeriesResult{}, raw, newError(operationSearchBooksBySeries, ErrorKindInvalidResponse, err)
	}
	return result, raw, nil
}

// keywordQuery は、DMMのkeyword検索用queryを構成する
func keywordQuery(keyword string, dateRange daterange.Range, limit, offset int) url.Values {
	values := url.Values{"sort": {"rank"}, "keyword": {keyword}}
	if !dateRange.FromStart().IsZero() {
		values.Set("gte_date", dateRange.FromStart().Format("2006-01-02T15:04:05"))
	}
	if !dateRange.ToEnd().IsZero() {
		values.Set("lte_date", dateRange.ToEnd().Format("2006-01-02T15:04:05"))
	}
	return baseQuery(limit, offset, values)
}

// seriesQuery は、DMMシリーズ指定の商品検索用queryを構成する
func seriesQuery(seriesID string, limit, offset int) url.Values {
	return baseQuery(limit, offset, url.Values{"article": {"series"}, "article_id": {seriesID}, "sort": {"rank"}})
}

// baseQuery は、DMMブックスItemListへ共通で送るqueryを構成する
func baseQuery(limit, offset int, values url.Values) url.Values {
	values.Set("site", "DMM.com")
	values.Set("service", "ebook")
	values.Set("floor", "comic")
	values.Set("output", "json")
	values.Set("hits", strconv.Itoa(limit))
	values.Set("offset", strconv.Itoa(offset))
	return values
}

// searchSeriesKeyword は、シリーズ検索条件をDMMのkeyword値へ変換する
func searchSeriesKeyword(request SearchSeriesRequest) (string, daterange.Range, error) {
	freeText := strings.Join(strings.Fields(request.FreeText), " ")
	if freeText == "" {
		return "", daterange.Range{}, errors.New("free text must be specified")
	}
	if err := validateKeyword(freeText); err != nil {
		return "", daterange.Range{}, err
	}
	terms := []string{freeText}
	for _, value := range strings.Fields(request.ExcludedText) {
		if err := validateKeyword(value); err != nil {
			return "", daterange.Range{}, err
		}
		terms = append(terms, "-"+value)
	}
	dateRange, err := daterange.Parse(request.DateFrom, request.DateTo)
	if err != nil {
		return "", daterange.Range{}, err
	}
	return strings.Join(terms, " "), dateRange, nil
}

// seriesSearchKey は、keywordと時期条件をCursor照合用の検索キーへ変換する
func seriesSearchKey(keyword string, dateRange daterange.Range) string {
	return keyword + "\x00" + dateRange.From + "\x00" + dateRange.To
}

// validateKeyword は、リテラルエスケープ方法が未確認のDMM演算子構文を拒否する
func validateKeyword(value string) error {
	if strings.Contains(value, `"`) {
		return errors.New("search conditions must not contain double quotes")
	}
	if strings.Contains(value, "|") {
		return errors.New("search conditions must not contain OR operators")
	}
	for _, term := range strings.Fields(value) {
		if strings.HasPrefix(term, "-") {
			return errors.New("search terms must not begin with exclusion operators")
		}
	}
	return nil
}

// effectiveLimit は、要求されたLimitをDMMで使用する範囲へ検証する
func effectiveLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 1 || limit > 100 {
		return 0, errors.New("limit must be between 1 and 100")
	}
	return limit, nil
}

// buildSeriesResult は、DMM keyword応答をシリーズ検索結果と次Cursorへ変換する
func buildSeriesResult(response result, offset int, key string, limit int) (SearchSeriesResult, error) {
	if err := validateFirstPosition(response, offset); err != nil {
		return SearchSeriesResult{}, err
	}
	bookSeries := make([]SeriesSearchItem, 0, len(response.Items))
	for _, item := range response.Items {
		series, err := seriesSearchItemFromItem(item)
		if err != nil {
			return SearchSeriesResult{}, err
		}
		bookSeries = append(bookSeries, series)
	}
	nextCursor, err := resultCursor(response, cursorAPISeries, key, limit)
	if err != nil {
		return SearchSeriesResult{}, err
	}
	return SearchSeriesResult{BookSeries: bookSeries, NextCursor: nextCursor}, nil
}

// buildBooksBySeriesResult は、DMMシリーズ指定応答を個別Bookと次Cursorへ変換する
func buildBooksBySeriesResult(response result, offset int, seriesID string, limit int) (SearchBooksBySeriesResult, error) {
	if err := validateFirstPosition(response, offset); err != nil {
		return SearchBooksBySeriesResult{}, err
	}
	books := make([]Book, 0, len(response.Items))
	for _, item := range response.Items {
		series, err := bookSeriesFromItem(item)
		if err != nil {
			return SearchBooksBySeriesResult{}, err
		}
		if series.ID != seriesID {
			return SearchBooksBySeriesResult{}, errors.New("response series ID does not match requested series ID")
		}
		books = append(books, convertItem(item, series))
	}
	nextCursor, err := resultCursor(response, cursorAPIBooksBySeries, seriesID, limit)
	if err != nil {
		return SearchBooksBySeriesResult{}, err
	}
	return SearchBooksBySeriesResult{Books: books, NextCursor: nextCursor}, nil
}

// validateFirstPosition は、Itemsを含む応答の開始位置を検証する
func validateFirstPosition(response result, offset int) error {
	if len(response.Items) > 0 && response.FirstPosition != offset {
		return errors.New("response first_position does not match requested offset")
	}
	return nil
}

// resultCursor は、次ページが存在する場合だけ検索用Cursorを返す
func resultCursor(response result, api, key string, limit int) (string, error) {
	next := response.FirstPosition + response.ResultCount
	if response.ResultCount == 0 || next > maxOffset || next > response.TotalCount {
		return "", nil
	}
	return encodeCursor(next, api, key, limit)
}
