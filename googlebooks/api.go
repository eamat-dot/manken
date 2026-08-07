// Package googlebooks は、Google Booksから書誌情報を検索する機能を提供する
package googlebooks

import "github.com/eamat-dot/manken/api"

// Source は、書誌情報の取得元を表す
type Source = api.Source

// Book は、取得した1冊の漫画本を表す
type Book = api.Book

// NormalizedBook は、取得元に依存せず利用できる書誌情報を表す
type NormalizedBook = api.NormalizedBook

// BookSource は、Bookの書誌情報を取得した取得元を表す
type BookSource = api.BookSource

// IdentifierType は、書誌識別子の種類を表す
type IdentifierType = api.IdentifierType

// Identifier は、種類を明示した書誌識別子を表す
type Identifier = api.Identifier

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = api.Contributor

// BookDateType は、書誌に関係する日付の種類を表す
type BookDateType = api.BookDateType

// BookDate は、種類と精度を維持した日付文字列を表す
type BookDate = api.BookDate

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = api.Subject

// Image は、表紙など書籍に関係する画像を表す
type Image = api.Image

// SearchBooksRequest は、漫画本の検索条件を表す
type SearchBooksRequest = api.SearchBooksRequest

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult = api.SearchBooksResult

// ISBNLookupResult は、入力ISBNごとの書籍参照結果を表す
type ISBNLookupResult = api.ISBNLookupResult

// ISBNLookupItem は、指定された1つのISBNと対応する書籍を表す
type ISBNLookupItem = api.ISBNLookupItem

// ErrorKind は、書誌情報の取得処理で発生したエラーの分類を表す
type ErrorKind = api.ErrorKind

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error = api.Error

const (
	// SourceGoogleBooks は、Google Booksを表す
	SourceGoogleBooks = api.SourceGoogleBooks
	// IdentifierTypeISBN10 は、ISBN-10を表す
	IdentifierTypeISBN10 = api.IdentifierTypeISBN10
	// IdentifierTypeISBN13 は、ISBN-13を表す
	IdentifierTypeISBN13 = api.IdentifierTypeISBN13
	// BookDateTypePublished は、出版日を表す
	BookDateTypePublished = api.BookDateTypePublished
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument = api.ErrorKindInvalidArgument
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream = api.ErrorKindUpstream
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable = api.ErrorKindUnavailable
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse = api.ErrorKindInvalidResponse
)
