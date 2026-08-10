// Package rakutenkobo は、楽天Kobo電子書籍検索APIから電子書籍の商品情報を検索する機能を提供する
package rakutenkobo

import "github.com/eamat-dot/manken/api"

// Source は、書誌情報の取得元を表す
type Source = api.Source

// Book は、取得した1冊の電子書籍を表す
type Book = api.Book

// NormalizedBook は、取得元に依存せず利用できる共通書籍情報を表す
type NormalizedBook = api.NormalizedBook

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = api.BookSource

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = api.Contributor

// Series は、シリーズ名と取得元内の参照情報を表す
type Series = api.Series

// BookDate は、種類と精度を維持した日付文字列を表す
type BookDate = api.BookDate

// BookDateType は、書誌に関係する日付の種類を表す
type BookDateType = api.BookDateType

// PublicationMedium は、出版物の媒体を表し、未設定または不明な値も取り得る
type PublicationMedium = api.PublicationMedium

// Price は、種類と出典を明示した価格を表す
type Price = api.Price

// PriceType は、価格の種類を表す
type PriceType = api.PriceType

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = api.Subject

// Image は、表紙など書籍に関係する画像を表す
type Image = api.Image

// SearchBooksRequest は、漫画本の検索条件を表す
type SearchBooksRequest = api.SearchBooksRequest

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult = api.SearchBooksResult

// ErrorKind は、書誌情報の取得処理で発生したエラーの分類を表す
type ErrorKind = api.ErrorKind

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error = api.Error

const (
	// SourceRakutenKobo は、楽天Kobo電子書籍検索APIを表す
	SourceRakutenKobo = api.SourceRakutenKobo
	// BookDateTypeReleased は、発売日を表す
	BookDateTypeReleased = api.BookDateTypeReleased
	// PublicationMediumDigital は、電子書籍を表す
	PublicationMediumDigital = api.PublicationMediumDigital
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

// ComicGenre は、楽天Kobo検索で対象にする漫画区分を表す
type ComicGenre string

const (
	// ComicGenreGeneral は、一般コミックを表す
	ComicGenreGeneral ComicGenre = "general"
	// ComicGenreBL は、ボーイズラブコミックを表す
	ComicGenreBL ComicGenre = "bl"
	// ComicGenreTL は、ティーンズラブコミックを表す
	ComicGenreTL ComicGenre = "tl"
)
