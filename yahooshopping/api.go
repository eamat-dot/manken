// Package yahooshopping は、Yahoo!ショッピングのTower商品から紙書籍情報を検索する機能を提供する
package yahooshopping

import "github.com/eamat-dot/manken/api"

// Source は、書誌情報の取得元を表す
type Source = api.Source

// Book は、取得した1冊の漫画本を表す
type Book = api.Book

// NormalizedBook は、取得元に依存せず利用できる共通書籍情報を表す
type NormalizedBook = api.NormalizedBook

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = api.BookSource

// Identifier は、種類を明示した書誌識別子を表す
type Identifier = api.Identifier

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = api.Contributor

// BookDate は、種類と精度を維持した日付文字列を表す
type BookDate = api.BookDate

// Image は、表紙など書籍に関係する画像を表す
type Image = api.Image

// Price は、種類と出典を明示した価格を表す
type Price = api.Price

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
	// SourceYahooShopping は、Yahoo!ショッピング商品検索APIを表す
	SourceYahooShopping = api.SourceYahooShopping
	// IdentifierTypeISBN13 は、ISBN-13を表す
	IdentifierTypeISBN13 = api.IdentifierTypeISBN13
	// IdentifierTypeJAN は、JANを表す
	IdentifierTypeJAN = api.IdentifierTypeJAN
	// ContributorRoleAuthor は、著者の役割を表す
	ContributorRoleAuthor = api.ContributorRoleAuthor
	// ContributorRoleArtist は、画家などの制作担当を表す
	ContributorRoleArtist = api.ContributorRoleArtist
	// ContributorRoleOriginalCreator は、原作担当を表す
	ContributorRoleOriginalCreator = api.ContributorRoleOriginalCreator
	// BookDateTypeReleased は、発売日を表す
	BookDateTypeReleased = api.BookDateTypeReleased
	// PublicationMediumPrint は、紙書籍を表す
	PublicationMediumPrint = api.PublicationMediumPrint
	// PriceTypeCurrent は、API取得時点の販売価格を表す
	PriceTypeCurrent = api.PriceTypeCurrent
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument = api.ErrorKindInvalidArgument
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream = api.ErrorKindUpstream
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable = api.ErrorKindUnavailable
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse = api.ErrorKindInvalidResponse
)
