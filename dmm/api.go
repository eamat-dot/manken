// Package dmm は、DMMブックス電子コミックのシリーズ探索と個別商品取得を提供する
package dmm

import "github.com/eamat-dot/manken/api"

// Source は、書誌情報の取得元を表す
type Source = api.Source

// Book は、取得した1冊の電子コミックを表す
type Book = api.Book

// NormalizedBook は、取得元に依存せず利用できる共通書籍情報を表す
type NormalizedBook = api.NormalizedBook

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = api.BookSource

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = api.Contributor

// BookSeries は、書籍が属する作品または刊行物のシリーズと取得元内の参照情報を表す
type BookSeries = api.BookSeries

// SeriesSearchItem は、DMMシリーズの候補とkeyword応答の代表商品由来の補助情報を表す
type SeriesSearchItem struct {
	BookSeries
	Title      string       `json:"title,omitempty"`
	Authors    []string     `json:"authors,omitempty"`
	Publishers []string     `json:"publishers,omitempty"`
	Subjects   []Subject    `json:"subjects,omitempty"`
	Images     []Image      `json:"images,omitempty"`
	Sources    []BookSource `json:"sources,omitempty"`
}

// PublicationMedium は、出版物の媒体を表し、未設定または不明な値も取り得る
type PublicationMedium = api.PublicationMedium

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = api.Subject

// Image は、表紙など書籍に関係する画像を表す
type Image = api.Image

// SearchSeriesRequest は、DMMが付与するシリーズを検索する条件を表す
type SearchSeriesRequest struct {
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}

// SearchSeriesResult は、DMMが付与するシリーズの検索結果と続きの取得に使うカーソルを表す
type SearchSeriesResult struct {
	BookSeries []SeriesSearchItem `json:"book_series"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// SearchBooksBySeriesRequest は、指定されたDMMシリーズに属する個別商品を取得する条件を表す
type SearchBooksBySeriesRequest struct {
	SeriesID string `json:"series_id"`
	Limit    int    `json:"limit"`
	Cursor   string `json:"cursor"`
}

// SearchBooksBySeriesResult は、DMMシリーズに属する個別Bookと続きの取得に使うカーソルを表す
type SearchBooksBySeriesResult struct {
	Books      []Book `json:"books"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ErrorKind は、書誌情報の取得処理で発生したエラーの分類を表す
type ErrorKind = api.ErrorKind

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error = api.Error

const (
	// SourceDMM は、DMM.com Webサービス v3 ItemListを表す
	SourceDMM = api.SourceDMM
	// PublicationMediumDigital は、電子書籍を表す
	PublicationMediumDigital = api.PublicationMediumDigital
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument = api.ErrorKindInvalidArgument
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream = api.ErrorKindUpstream
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable = api.ErrorKindUnavailable
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse = api.ErrorKindInvalidResponse
)
