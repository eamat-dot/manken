// Package rakutenbooks は、楽天ブックス書籍検索APIから紙書籍の書誌情報を検索する機能を提供する
package rakutenbooks

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

// Price は、金額と出典を表す
type Price = model.Price

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = model.Subject

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
	// SourceRakutenBooks は、楽天ブックス書籍検索APIを表す
	SourceRakutenBooks = model.SourceRakutenBooks
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

// ComicGenre は、楽天Books検索で対象にする漫画区分を表す
type ComicGenre string

const (
	// ComicGenreGeneral は、一般コミックを表す
	ComicGenreGeneral ComicGenre = "general"
	// ComicGenreBL は、ボーイズラブコミックを表す
	ComicGenreBL ComicGenre = "bl"
	// ComicGenreTL は、ティーンズラブコミックを表す
	ComicGenreTL ComicGenre = "tl"
)

// BookSize は、楽天Books検索で指定できる商品形態の分類を表す
type BookSize int

const (
	// BookSizeAll は、商品形態で絞り込まない既定値を表す
	BookSizeAll BookSize = iota
	// BookSizeTankobon は、単行本を表す
	BookSizeTankobon
	// BookSizeBunko は、文庫を表す
	BookSizeBunko
	// BookSizeShinsho は、新書を表す
	BookSizeShinsho
	// BookSizeZenshuSosho は、全集・双書を表す
	BookSizeZenshuSosho
	// BookSizeJiten は、事・辞典を表す
	BookSizeJiten
	// BookSizeZukan は、図鑑を表す
	BookSizeZukan
	// BookSizeEhon は、絵本を表す
	BookSizeEhon
	// BookSizeCassetteCD は、カセット、CDなどを表す
	BookSizeCassetteCD
	// BookSizeComic は、コミックを表す
	BookSizeComic
	// BookSizeMookOther は、ムックその他を表す
	BookSizeMookOther
)
