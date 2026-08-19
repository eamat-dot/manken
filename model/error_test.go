package model

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestError_ErrorAndUnwrap は、分類済みエラーの表示と原因の取り出しを検証する
func TestError_ErrorAndUnwrap(t *testing.T) {
	cause := context.DeadlineExceeded
	err := &Error{
		Kind:      ErrorKindUnavailable,
		Operation: "test.Operation",
		Err:       cause,
	}

	if !strings.Contains(err.Error(), "test.Operation: unavailable") {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("errors.Is did not find context.DeadlineExceeded")
	}

	var classified *Error
	if !errors.As(err, &classified) {
		t.Fatal("errors.As did not find *Error")
	}
}
