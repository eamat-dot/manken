// Package ndl は、国立国会図書館サーチから書誌情報を検索する機能を提供する
package ndl

import "github.com/eamat-dot/manken/model"

// Source は、書誌情報の取得元を表す
type Source = model.Source

// Book は、取得した1冊の漫画本を表す
type Book = model.Book

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = model.BookSource

// Volume は、整数化できる巻数と正規化済みの巻表示を表す
type Volume = model.Volume

// PublicationMedium は、出版物が紙、電子、または未判定のいずれかであることを表す
type PublicationMedium = model.PublicationMedium

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = model.Subject

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = model.Contributor

// Price は、金額、通貨、税込情報、取得元、観測時刻を表す
type Price = model.Price

// SearchRequest は、漫画本の検索条件を表す
type SearchRequest = model.SearchRequest

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult = model.SearchBooksResult

// ISBNLookupResult は、入力ISBNごとの書籍参照結果を表す
type ISBNLookupResult = model.ISBNLookupResult

// ISBNLookupItem は、指定された1つのISBNと対応する書籍を表す
type ISBNLookupItem = model.ISBNLookupItem

// ErrorKind は、書誌情報の取得処理で発生したエラーの分類を表す
type ErrorKind = model.ErrorKind

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error = model.Error

const (
	// SourceNDL は、国立国会図書館サーチを表す
	SourceNDL = model.SourceNDL
	// PublicationMediumUnknown は、紙または電子を判定できない状態を表す
	PublicationMediumUnknown = model.PublicationMediumUnknown
	// PublicationMediumPrint は、紙書籍を表す
	PublicationMediumPrint = model.PublicationMediumPrint
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
