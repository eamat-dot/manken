package ndl

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/eamat-dot/manken/internal/daterange"
)

const (
	defaultLimit       = 20
	maxLimit           = 500
	defaultSortCQLTerm = "sortBy=issued_date/sort.ascending"
)

// buildSearchQuery は、共通検索条件をNDLサーチ用CQLへ変換する
func buildSearchQuery(request SearchRequest, mangaNDCFilter, mangaNDLCFilter bool) (string, error) {
	return buildSearchQueryWithOptions(request, SearchOptions{}, mangaNDCFilter, mangaNDLCFilter)
}

// buildSearchQueryWithOptions は、共通検索条件とNDL固有検索条件をNDLサーチ用CQLへ変換する
func buildSearchQueryWithOptions(request SearchRequest, options SearchOptions, mangaNDCFilter, mangaNDLCFilter bool) (string, error) {
	if request.Exclude != "" {
		return "", errors.New("exclude is not supported")
	}
	terms := fixedCQLTerms()
	userConditionCount := 0
	for _, pair := range []struct{ index, value string }{{"title", request.Title}, {"creator", request.Author}, {"publisher", request.Publisher}, {"anywhere", request.Query}} {
		if strings.TrimSpace(pair.value) != "" {
			terms = append(terms, pair.index+" = "+quoteCQL(pair.value))
			userConditionCount++
		}
	}
	dateRange, err := daterange.Parse(request.DateFrom, request.DateTo)
	if err != nil {
		return "", err
	}
	for _, pair := range []struct{ index, value string }{{"from", dateRange.From}, {"until", dateRange.To}, {"subject", options.Subject}, {"description", options.Description}} {
		if strings.TrimSpace(pair.value) != "" {
			terms = append(terms, pair.index+" = "+quoteCQL(pair.value))
			userConditionCount++
		}
	}
	if userConditionCount == 0 {
		return "", errors.New("at least one common or NDL-specific search condition must be specified")
	}
	if mangaNDCFilter {
		terms = append(terms, `ndc = "726.1"`)
	}
	if mangaNDLCFilter {
		terms = append(terms, `ndlc = "Y84"`)
	}
	return strings.Join(append(terms, defaultSortCQLTerm), " AND "), nil
}

// fixedCQLTerms は、すべてのNDLサーチ要求へ付与する固定検索条件を返す
func fixedCQLTerms() []string {
	return []string{`dpid = "iss-ndl-opac-national"`, `mediatype = "books"`}
}

// quoteCQL は、値をCQL文字列として引用し内部の特殊文字だけをエスケープする
func quoteCQL(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

// effectiveLimit は、検索件数の既定値と有効範囲を適用する
func effectiveLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 1 || limit > maxLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxLimit)
	}
	return limit, nil
}

// searchValues は、CQLとページ情報からSRU固定パラメータを生成する
func searchValues(query string, limit, startRecord int) url.Values {
	values := url.Values{}
	values.Set("operation", "searchRetrieve")
	values.Set("version", "1.2")
	values.Set("recordSchema", "dcndl_v3")
	values.Set("recordPacking", "xml")
	values.Set("onlyBib", "true")
	values.Set("query", query)
	values.Set("maximumRecords", strconv.Itoa(limit))
	values.Set("startRecord", strconv.Itoa(startRecord))
	return values
}
