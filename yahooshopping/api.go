// Package yahooshopping は、Yahoo!ショッピングのTower商品から紙書籍情報を検索する機能を提供する
package yahooshopping

import "github.com/eamat-dot/manken/model"

// Source は、書誌情報の取得元を表す
type Source = model.Source

// Book は、取得した1冊の漫画本を表す
type Book = model.Book

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = model.BookSource

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = model.Contributor

// PublicationMedium は、出版物が紙または電子のどちらかを表す
type PublicationMedium = model.PublicationMedium

// Price は、金額、通貨、税込情報、取得元、確認時刻を表す
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
	// SourceYahooShopping は、Yahoo!ショッピング商品検索APIを表す
	SourceYahooShopping = model.SourceYahooShopping
	// PublicationMediumPrint は、紙書籍を表す
	PublicationMediumPrint = model.PublicationMediumPrint
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument = model.ErrorKindInvalidArgument
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream = model.ErrorKindUpstream
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable = model.ErrorKindUnavailable
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse = model.ErrorKindInvalidResponse
)
