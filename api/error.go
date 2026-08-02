package api

import (
	"fmt"
	"time"
)

// ErrorKind は、書誌情報の取得処理で発生したエラーの分類を表す
type ErrorKind string

const (
	// ErrorKindInvalidArgument は、呼び出し側が修正できる入力エラーを表す
	ErrorKindInvalidArgument ErrorKind = "invalid_argument"
	// ErrorKindUpstream は、取得元が返した恒久的または未分類のエラーを表す
	ErrorKindUpstream ErrorKind = "upstream"
	// ErrorKindUnavailable は、取得元または通信が一時的に利用できない状態を表す
	ErrorKindUnavailable ErrorKind = "unavailable"
	// ErrorKindInvalidResponse は、取得元の成功応答を解釈できない状態を表す
	ErrorKindInvalidResponse ErrorKind = "invalid_response"
)

// Error は、書誌情報の取得処理の失敗を分類可能な形で保持する
type Error struct {
	Kind       ErrorKind
	Operation  string
	StatusCode int
	RetryAfter time.Duration
	Err        error
}

// Error は、分類と操作を含む安全なエラーメッセージを返す
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Operation, e.Kind, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Kind)
}

// Unwrap は、原因となったエラーを返す
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
