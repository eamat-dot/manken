package httpresponse

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type errorReader struct{}

// Read は、常に読み取りエラーを返す
func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

// TestReadLimitedBody は、本文の上限と読み取りエラーを検証する
func TestReadLimitedBody(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		limit       int64
		want        string
		wantErrText string
	}{
		{name: "below limit", input: "12", limit: 3, want: "12"},
		{name: "at limit", input: "123", limit: 3, want: "123"},
		{name: "exceeds limit", input: "1234", limit: 3, wantErrText: "response body exceeds 3 bytes"},
		{name: "maximum limit", input: "123", limit: math.MaxInt64, want: "123"},
		{name: "negative limit", input: "123", limit: -1, wantErrText: "response body limit must be non-negative"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := ReadLimitedBody(strings.NewReader(test.input), test.limit)
			if test.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrText) {
					t.Fatalf("ReadLimitedBody() error = %v, want error containing %q", err, test.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadLimitedBody() error = %v", err)
			}
			if string(body) != test.want {
				t.Fatalf("ReadLimitedBody() = %q, want %q", body, test.want)
			}
		})
	}
	if _, err := ReadLimitedBody(errorReader{}, 3); err == nil || !strings.Contains(err.Error(), "read response body: read failed") {
		t.Fatalf("ReadLimitedBody() error = %v", err)
	}
}

// TestParseRetryAfter は、Retry-Afterの秒数と日時を検証する
func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	maxSeconds := int64((1<<63 - 1) / time.Second)
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "seconds", value: "30", want: 30 * time.Second},
		{name: "max seconds", value: strconv.FormatInt(maxSeconds, 10), want: time.Duration(maxSeconds) * time.Second},
		{name: "date", value: now.Add(2 * time.Minute).Format(http.TimeFormat), want: 2 * time.Minute},
		{name: "empty", value: "", want: 0},
		{name: "invalid", value: "invalid", want: 0},
		{name: "past", value: now.Add(-time.Second).Format(http.TimeFormat), want: 0},
		{name: "overflow", value: strconv.FormatInt(maxSeconds+1, 10), want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ParseRetryAfter(test.value, now); got != test.want {
				t.Fatalf("ParseRetryAfter() = %s, want %s", got, test.want)
			}
		})
	}
}
