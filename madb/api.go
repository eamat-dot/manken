package madb

import "github.com/eamat-dot/manken/api"

// Source は、書誌情報の取得元を表す
type Source = api.Source

// Book は、取得した1冊の漫画本を表す
type Book = api.Book

// NormalizedBook は、取得元に依存せず利用できる共通書籍情報を表す
type NormalizedBook = api.NormalizedBook

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource = api.BookSource

// Volume は、整数化できる巻数と正規化済みの巻表示を表す
type Volume = api.Volume

// IdentifierType は、書誌識別子の種類を表す
type IdentifierType = api.IdentifierType

// Identifier は、種類を明示した書誌識別子を表す
type Identifier = api.Identifier

// ContributorRole は、制作への寄与者の役割を表す
type ContributorRole = api.ContributorRole

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor = api.Contributor

// Series は、シリーズ名と取得元内の参照情報を表す
type Series = api.Series

// BookDateType は、書誌に関係する日付の種類を表す
type BookDateType = api.BookDateType

// BookDate は、種類と精度を維持した日付文字列を表す
type BookDate = api.BookDate

// PublicationMedium は、出版物が紙または電子のどちらかを表す
type PublicationMedium = api.PublicationMedium

// PhysicalSize は、紙書籍の判型名と寸法をミリメートル単位で表す
type PhysicalSize = api.PhysicalSize

// PriceType は、価格が定価または取得時点価格のどちらかを表す
type PriceType = api.PriceType

// Price は、種類と出典を明示した価格を表す
type Price = api.Price

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
	// SourceMADB は、メディア芸術データベースを表す
	SourceMADB = api.SourceMADB
	// IdentifierTypeISBN10 は、ISBN-10を表す
	IdentifierTypeISBN10 = api.IdentifierTypeISBN10
	// IdentifierTypeISBN13 は、ISBN-13を表す
	IdentifierTypeISBN13 = api.IdentifierTypeISBN13
	// IdentifierTypeJAN は、JANコードを表す
	IdentifierTypeJAN = api.IdentifierTypeJAN
	// ContributorRoleAuthor は、著者を表す
	ContributorRoleAuthor = api.ContributorRoleAuthor
	// ContributorRoleOriginalCreator は、原作者または原案者を表す
	ContributorRoleOriginalCreator = api.ContributorRoleOriginalCreator
	// ContributorRoleWriter は、構成または脚本の執筆者を表す
	ContributorRoleWriter = api.ContributorRoleWriter
	// ContributorRoleArtist は、漫画または作画の担当者を表す
	ContributorRoleArtist = api.ContributorRoleArtist
	// ContributorRoleCharacterCreator は、キャラクター原案者を表す
	ContributorRoleCharacterCreator = api.ContributorRoleCharacterCreator
	// ContributorRoleCharacterDesigner は、キャラクターデザイン担当者を表す
	ContributorRoleCharacterDesigner = api.ContributorRoleCharacterDesigner
	// ContributorRoleEditor は、編集者を表す
	ContributorRoleEditor = api.ContributorRoleEditor
	// ContributorRoleTranslator は、翻訳者を表す
	ContributorRoleTranslator = api.ContributorRoleTranslator
	// ContributorRoleSupervisor は、監修者を表す
	ContributorRoleSupervisor = api.ContributorRoleSupervisor
	// ContributorRoleCommentator は、解説者を表す
	ContributorRoleCommentator = api.ContributorRoleCommentator
	// ContributorRoleDesigner は、装丁またはデザインの担当者を表す
	ContributorRoleDesigner = api.ContributorRoleDesigner
	// BookDateTypePublished は、出版日を表す
	BookDateTypePublished = api.BookDateTypePublished
	// BookDateTypeReleased は、発売日を表す
	BookDateTypeReleased = api.BookDateTypeReleased
	// BookDateTypeDigitalReleased は、電子版の配信開始日を表す
	BookDateTypeDigitalReleased = api.BookDateTypeDigitalReleased
	// PublicationMediumUnknown は、紙または電子を判定できない状態を表す
	PublicationMediumUnknown = api.PublicationMediumUnknown
	// PublicationMediumPrint は、紙書籍を表す
	PublicationMediumPrint = api.PublicationMediumPrint
	// PublicationMediumDigital は、電子書籍を表す
	PublicationMediumDigital = api.PublicationMediumDigital
	// PriceTypeList は、定価を表す
	PriceTypeList = api.PriceTypeList
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
