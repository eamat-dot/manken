// Package dmm は、DMMブックス電子コミックのシリーズ探索と個別商品取得を提供する
package dmm

import "github.com/eamat-dot/manken/model"

// Source は、書誌情報の取得元を表す
type Source = model.Source

// Book は、取得した1冊の電子コミックを表す
type Book = model.Book

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = model.BookSource

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = model.Contributor

// BookSeries は、書籍が属する作品または刊行物のシリーズと取得元内の参照情報を表す
type BookSeries = model.BookSeries

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
type PublicationMedium = model.PublicationMedium

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = model.Subject

// Image は、DMMシリーズ候補の代表商品に含まれる画像を表す
type Image struct {
	URL     string `json:"url"`
	Purpose string `json:"purpose,omitempty"`
	Width   *int   `json:"width,omitempty"`
	Height  *int   `json:"height,omitempty"`
}

// SearchSeriesRequest は、DMMが付与するシリーズを検索する条件を表す
type SearchSeriesRequest struct {
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
	DateFrom     string `json:"date_from"`
	DateTo       string `json:"date_to"`
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
type ErrorKind = model.ErrorKind

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error = model.Error

const (
	// SourceDMM は、DMM.com Webサービス v3 ItemListを表す
	SourceDMM = model.SourceDMM
	// PublicationMediumDigital は、電子書籍を表す
	PublicationMediumDigital = model.PublicationMediumDigital
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument = model.ErrorKindInvalidArgument
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream = model.ErrorKindUpstream
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable = model.ErrorKindUnavailable
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse = model.ErrorKindInvalidResponse
)
