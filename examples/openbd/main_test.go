package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteRawResponse_CreatesExactFile は、rawレスポンスを変更せず新規保存することを検証する
func TestWriteRawResponse_CreatesExactFile(t *testing.T) {
	body := []byte("[\r\n  null\r\n]\n")
	path := filepath.Join(t.TempDir(), "__openbd-result.json")

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
	path := filepath.Join(t.TempDir(), "__openbd-result.json")
	original := []byte("existing investigation data")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := writeRawResponse(path, []byte("replacement")); err == nil {
		t.Fatal("writeRawResponse() error = nil, want existing-file error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing file = %q, want %q", got, original)
	}
}

// TestWriteRawResponse_CreateFailure は、出力先を作成できない場合にエラーを返すことを検証する
func TestWriteRawResponse_CreateFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "__openbd-result.json")
	if err := writeRawResponse(path, []byte("[]")); err == nil {
		t.Fatal("writeRawResponse() error = nil, want create error")
	}
}
