// Package httpresponse は、取得元に依存しないHTTPレスポンス処理を提供する
package httpresponse

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const maxRetryAfterSeconds = int64((1<<63 - 1) / time.Second)

// ReadLimitedBody は、上限を超える本文を拒否して読み込む
func ReadLimitedBody(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 0 {
		return nil, fmt.Errorf("response body limit must be non-negative: %d", limit)
	}

	body, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	extra, err := io.ReadAll(io.LimitReader(reader, 1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if len(extra) != 0 {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return body, nil
}

// ParseRetryAfter は、Retry-After値を待機時間へ変換する
func ParseRetryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 && seconds <= maxRetryAfterSeconds {
		return time.Duration(seconds) * time.Second
	}
	date, err := http.ParseTime(value)
	if err != nil || !date.After(now) {
		return 0
	}
	return date.Sub(now)
}
