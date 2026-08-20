// Package manken は、登録済みの書誌情報providerへ共通の検索またはISBN参照を委譲する入口を提供する
package manken

import "github.com/eamat-dot/manken/model"

// Source は、書誌情報の取得元を表す
type Source = model.Source

// Book は、取得した1冊の漫画本を表す
type Book = model.Book

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = model.BookSource

// Volume は、整数化できる巻数と正規化済みの巻表示を表す
type Volume = model.Volume

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = model.Contributor

// BookSeries は、書籍が属する作品または個別Bookの系列と取得元内の参照情報を表す
type BookSeries = model.BookSeries

// PublicationMedium は、出版物が紙または電子のどちらかを表す
type PublicationMedium = model.PublicationMedium

// Price は、金額、通貨、税込情報、取得元、観測時刻を表す
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
	// SourceMADB は、メディア芸術データベースを表す
	SourceMADB = model.SourceMADB
	// SourceOpenBD は、openBDを表す
	SourceOpenBD = model.SourceOpenBD
	// SourceGoogleBooks は、Google Booksを表す
	SourceGoogleBooks = model.SourceGoogleBooks
	// SourceRakutenBooks は、楽天ブックス書籍検索APIを表す
	SourceRakutenBooks = model.SourceRakutenBooks
	// SourceRakutenKobo は、楽天Kobo電子書籍検索APIを表す
	SourceRakutenKobo = model.SourceRakutenKobo
	// SourceNDL は、国立国会図書館サーチを表す
	SourceNDL = model.SourceNDL
	// SourceYahooShopping は、Yahoo!ショッピング商品検索APIを表す
	SourceYahooShopping = model.SourceYahooShopping
	// SourceDMM は、DMM.com Webサービスの商品検索APIを表す
	SourceDMM = model.SourceDMM
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
