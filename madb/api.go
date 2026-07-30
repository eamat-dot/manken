package madb

import "github.com/eamat-dot/manken/api"

// Source は、書誌情報の取得元を表す
type Source = api.Source

// Book は、検索で取得した1冊の漫画本を表す
type Book = api.Book

// SearchBooksRequest は、漫画本の検索条件を表す
type SearchBooksRequest = api.SearchBooksRequest

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult = api.SearchBooksResult

// ErrorKind は、検索処理で発生したエラーの分類を表す
type ErrorKind = api.ErrorKind

// Error は、検索処理の失敗を分類可能な形で保持する
type Error = api.Error

const (
	// SourceMADB は、メディア芸術データベースを表す
	SourceMADB = api.SourceMADB
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument = api.ErrorKindInvalidArgument
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream = api.ErrorKindUpstream
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable = api.ErrorKindUnavailable
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse = api.ErrorKindInvalidResponse
)
