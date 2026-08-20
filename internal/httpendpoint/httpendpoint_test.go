package httpendpoint

import "testing"

// TestValidate は、HTTP(S) endpointの共通構造制約を検証する
func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErrText string
	}{
		{name: "HTTPS", value: "https://example.test/path"},
		{name: "HTTP", value: "http://example.test/path"},
		{name: "relative URL", value: "/path", wantErrText: "endpoint must use http or https and include a host"},
		{name: "unsupported scheme", value: "ftp://example.test/path", wantErrText: "endpoint must use http or https and include a host"},
		{name: "missing host", value: "https:///path", wantErrText: "endpoint must use http or https and include a host"},
		{name: "user information", value: "https://user@example.test/path", wantErrText: "endpoint must not include user information, query, or fragment"},
		{name: "query", value: "https://example.test/path?x=1", wantErrText: "endpoint must not include user information, query, or fragment"},
		{name: "force query", value: "https://example.test/path?", wantErrText: "endpoint must not include user information, query, or fragment"},
		{name: "fragment", value: "https://example.test/path#part", wantErrText: "endpoint must not include user information, query, or fragment"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.value)
			if test.wantErrText == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErrText {
				t.Fatalf("Validate() error = %v, want %q", err, test.wantErrText)
			}
		})
	}
}

// TestValidateHTTPSOrLoopback は、HTTPSまたはloopback HTTPの制約を検証する
func TestValidateHTTPSOrLoopback(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErrText string
	}{
		{name: "HTTPS", value: "https://example.test/path"},
		{name: "HTTP rejected", value: "http://example.test/path", wantErrText: "endpoint must use https unless its host is loopback"},
		{name: "localhost HTTP", value: "http://LOCALHOST/path"},
		{name: "IPv4 loopback HTTP", value: "http://127.0.0.1/path"},
		{name: "IPv6 loopback HTTP", value: "http://[::1]/path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateHTTPSOrLoopback(test.value)
			if test.wantErrText == "" {
				if err != nil {
					t.Fatalf("ValidateHTTPSOrLoopback() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantErrText {
				t.Fatalf("ValidateHTTPSOrLoopback() error = %v, want %q", err, test.wantErrText)
			}
		})
	}
}
