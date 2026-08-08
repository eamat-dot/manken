package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eamat-dot/manken/rakutenbooks"
)

// TestParseGenre は、CLIの漫画区分名を公開定数へ変換できることを確認する
func TestParseGenre(t *testing.T) {
	for _, test := range []struct {
		value string
		want  rakutenbooks.ComicGenre
	}{
		{"general", rakutenbooks.ComicGenreGeneral},
		{"bl", rakutenbooks.ComicGenreBL},
		{"tl", rakutenbooks.ComicGenreTL},
	} {
		got, err := parseGenre(test.value)
		if err != nil || got != test.want {
			t.Fatalf("parseGenre(%q) = %q, %v; want %q", test.value, got, err, test.want)
		}
	}
	if _, err := parseGenre("other"); err == nil {
		t.Fatal("parseGenre(other) error = nil")
	}
}

// TestWriteResult_DoesNotEscapeHTML は、CLI用JSONでURLの&をUnicodeエスケープしないことを確認する
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
	if !strings.Contains(text, `"https://example.test/?a=1&b=2"`) || strings.Contains(text, `\u0026`) {
		t.Fatalf("output = %q", text)
	}
}

// TestWriteRawResponse_CreatesExactFile は、Raw responseを変更せず新規保存することを確認する
func TestWriteRawResponse_CreatesExactFile(t *testing.T) {
	body := []byte("{\r\n  \\\"Items\\\": []\r\n}\n")
	path := filepath.Join(t.TempDir(), "__rakutenbooks-result.json")
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

// TestWriteRawResponse_RefusesExistingFile は、既存ファイルを上書きしないことを確認する
func TestWriteRawResponse_RefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "__rakutenbooks-result.json")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := writeRawResponse(path, []byte("replacement")); err == nil {
		t.Fatal("writeRawResponse() error = nil, want existing-file error")
	}
}
