// Package rakutenkobo は、楽天Kobo電子書籍検索APIから電子書籍の商品情報を検索する機能を提供する
package rakutenkobo

import "github.com/eamat-dot/manken/model"

// Source は、書誌情報の取得元を表す
type Source = model.Source

// Book は、取得した1冊の電子書籍を表す
type Book = model.Book

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = model.BookSource

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = model.Contributor

// PublicationMedium は、出版物の媒体を表し、未設定または不明な値も取り得る
type PublicationMedium = model.PublicationMedium

// Price は、金額と出典を表す
type Price = model.Price

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject = model.Subject

// SearchRequest は、漫画本の検索条件を表す
type SearchRequest = model.SearchRequest

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult = model.SearchBooksResult

// ErrorKind は、書誌情報の取得処理で発生したエラーの分類を表す
type ErrorKind = model.ErrorKind

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error = model.Error

const (
	// SourceRakutenKobo は、楽天Kobo電子書籍検索APIを表す
	SourceRakutenKobo = model.SourceRakutenKobo
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
