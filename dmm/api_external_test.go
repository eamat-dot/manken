package dmm_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/eamat-dot/manken/dmm"
)

// TestExternalAPI は、外部パッケージからDMMの公開APIを利用できることをコンパイル時に検証する
func TestExternalAPI(t *testing.T) {
	client, err := dmm.NewClient(&http.Client{}, dmm.WithAPIID("api"), dmm.WithAffiliateID("affiliate"))
	if err != nil {
		t.Fatal(err)
	}
	requireClient(client)
	requireOption(dmm.WithEndpoint("https://example.com"))
	requireSource(dmm.SourceDMM)
	requirePublicationMedium(dmm.PublicationMediumDigital)
	requireBookSeries(dmm.BookSeries{})
	requireSeriesSearchItem(dmm.SeriesSearchItem{})
	requireSeriesRequest(dmm.SearchSeriesRequest{})
	requireSeriesResult(dmm.SearchSeriesResult{})
	requireBooksBySeriesRequest(dmm.SearchBooksBySeriesRequest{})
	requireBooksBySeriesResult(dmm.SearchBooksBySeriesResult{})
	requireSearchSeriesWithRawResponse(client.SearchSeriesWithRawResponse)
	requireSearchBooksBySeriesWithRawResponse(client.SearchBooksBySeriesWithRawResponse)
}

// requireClient は、外部利用時のClient型をコンパイル時に確認する
func requireClient(*dmm.Client) {}

// requireOption は、外部利用時のOption型をコンパイル時に確認する
func requireOption(dmm.Option) {}

// requireSource は、外部利用時のSource型をコンパイル時に確認する
func requireSource(dmm.Source) {}

// requirePublicationMedium は、外部利用時のPublicationMedium型をコンパイル時に確認する
func requirePublicationMedium(dmm.PublicationMedium) {}

// requireBookSeries は、外部利用時のBookSeries型をコンパイル時に確認する
func requireBookSeries(dmm.BookSeries) {}

// requireSeriesSearchItem は、外部利用時のSeriesSearchItem型をコンパイル時に確認する
func requireSeriesSearchItem(dmm.SeriesSearchItem) {}

// requireSeriesRequest は、外部利用時のSearchSeriesRequest型をコンパイル時に確認する
func requireSeriesRequest(dmm.SearchSeriesRequest) {}

// requireSeriesResult は、外部利用時のSearchSeriesResult型をコンパイル時に確認する
func requireSeriesResult(dmm.SearchSeriesResult) {}

// requireBooksBySeriesRequest は、外部利用時のSearchBooksBySeriesRequest型をコンパイル時に確認する
func requireBooksBySeriesRequest(dmm.SearchBooksBySeriesRequest) {}

// requireBooksBySeriesResult は、外部利用時のSearchBooksBySeriesResult型をコンパイル時に確認する
func requireBooksBySeriesResult(dmm.SearchBooksBySeriesResult) {}

// requireSearchSeriesWithRawResponse は、シリーズRaw検索メソッドの公開シグネチャをコンパイル時に確認する
func requireSearchSeriesWithRawResponse(func(context.Context, dmm.SearchSeriesRequest) (dmm.SearchSeriesResult, []byte, error)) {
}

// requireSearchBooksBySeriesWithRawResponse は、シリーズ内BookのRaw検索メソッドの公開シグネチャをコンパイル時に確認する
func requireSearchBooksBySeriesWithRawResponse(func(context.Context, dmm.SearchBooksBySeriesRequest) (dmm.SearchBooksBySeriesResult, []byte, error)) {
}
