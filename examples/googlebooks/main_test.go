package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteResult_DoesNotEscapeHTML は、CLI用JSONでURLの&をUnicodeエスケープしないことを検証する
func TestWriteResult_DoesNotEscapeHTML(t *testing.T) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = originalStdout })

	if code := writeResult(map[string]string{"url": "https://example.test/?a=1&b=2"}); code != 0 {
		t.Fatalf("writeResult() code = %d, want 0", code)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("stdout writer close error = %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("stdout reader close error = %v", err)
	}
	text := string(output)
	if !strings.Contains(text, `"https://example.test/?a=1&b=2"`) {
		t.Fatalf("output = %q, want literal ampersand URL", text)
	}
	if strings.Contains(text, `\u0026`) {
		t.Fatalf("output = %q, must not contain \\u0026", text)
	}
}

// TestWriteRawResponse_CreatesExactFile は、rawレスポンスを変更せず新規保存することを検証する
func TestWriteRawResponse_CreatesExactFile(t *testing.T) {
	body := []byte("{\r\n  \\\"items\\\": []\r\n}\n")
	path := filepath.Join(t.TempDir(), "__googlebooks-result.json")
	if err := writeRawResponse(path, body); err != nil {
		t.Fatalf("writeRawResponse() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("saved body = %q, want %q", got, body)
	}
}

// TestWriteRawResponse_RefusesExistingFile は、既存ファイルを上書きしないことを検証する
func TestWriteRawResponse_RefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "__googlebooks-result.json")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := writeRawResponse(path, []byte("replacement")); err == nil {
		t.Fatal("writeRawResponse() error = nil, want existing-file error")
	}
}
