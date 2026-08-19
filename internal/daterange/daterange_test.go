package daterange

import (
	"testing"
	"time"
)

// TestParse は、日付範囲の形式、精度、境界を検証する
func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		wantEnd  string
		wantExcl string
		wantErr  bool
	}{
		{name: "year", from: "2024", to: "2024", wantEnd: "2024-12-31T23:59:59Z", wantExcl: "2025-01-01T00:00:00Z"},
		{name: "month", from: "2024-02", to: "2024-02", wantEnd: "2024-02-29T23:59:59Z", wantExcl: "2024-03-01T00:00:00Z"},
		{name: "day", from: "2024-02-29", to: "2024-02-29", wantEnd: "2024-02-29T23:59:59Z", wantExcl: "2024-03-01T00:00:00Z"},
		{name: "one side", from: "2024"},
		{name: "invalid day", from: "2024-02-30", wantErr: true},
		{name: "different precision", from: "2024", to: "2024-01", wantErr: true},
		{name: "reversed", from: "2024-02", to: "2024-01", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.from, test.to)
			if (err != nil) != test.wantErr {
				t.Fatalf("Parse(%q, %q) error = %v", test.from, test.to, err)
			}
			if test.wantErr || test.wantEnd == "" {
				return
			}
			if got.ToEnd().Format(time.RFC3339) != test.wantEnd || got.ToExclusive().Format(time.RFC3339) != test.wantExcl {
				t.Fatalf("range = %#v, end = %s, exclusive = %s", got, got.ToEnd().Format(time.RFC3339), got.ToExclusive().Format(time.RFC3339))
			}
		})
	}
}
