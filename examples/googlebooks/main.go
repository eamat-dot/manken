package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/eamat-dot/manken/googlebooks"
)

// main は、Google Booksデモを実行して終了コードを設定する
func main() {
	os.Exit(run())
}

// run は、CLI引数に従ってGoogle Booksを検索またはISBNで参照する
func run() int {
	title := flag.String("title", "", "タイトルに含める検索語")
	author := flag.String("author", "", "著者名に含める検索語")
	freeText := flag.String("free-text", "", "複数の書誌項目を対象にする検索語")
	exclude := flag.String("exclude", "", "検索結果から除外する語")
	limit := flag.Int("limit", 0, "検索件数。1から40まで。0は既定値の20件")
	cursor := flag.String("cursor", "", "前回の結果に含まれる next_cursor")
	rawOutput := flag.String("raw-output", "", "検索またはISBN参照の変換前Google Booksレスポンスを保存する新規ファイル")
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: demo-googlebooks [options] [ISBN]")
		flag.PrintDefaults()
	}
	flag.Parse()

	apiKey := os.Getenv("GOOGLE_BOOKS_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GOOGLE_BOOKS_API_KEY を設定してください")
		return 2
	}
	isbns := flag.Args()
	if len(isbns) > 0 && (*title != "" || *author != "" || *freeText != "" || *exclude != "" || *limit != 0 || *cursor != "") {
		fmt.Fprintln(os.Stderr, "ISBN参照は検索条件、-limit、-cursor と併用できません")
		return 2
	}
	if len(isbns) == 0 && *title == "" && *author == "" && *freeText == "" {
		fmt.Fprintln(os.Stderr, "検索条件またはISBNを1件以上指定してください")
		flag.Usage()
		return 2
	}

	client, err := googlebooks.NewClient(nil, googlebooks.WithAPIKey(apiKey))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Google Booksクライアントを作成できません: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if len(isbns) > 0 {
		if *rawOutput == "" {
			result, err := client.LookupBooksByISBN(ctx, isbns)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Google BooksのISBN参照に失敗しました: %v\n", err)
				return 1
			}
			return writeResult(result)
		}
		result, raw, lookupErr := client.LookupBooksByISBNWithRawResponse(ctx, isbns)
		if raw != nil {
			if err := writeRawResponse(*rawOutput, raw); err != nil {
				fmt.Fprintf(os.Stderr, "Google Booksのrawレスポンスを保存できません: %s: %v\n", *rawOutput, err)
				return 1
			}
		}
		if lookupErr != nil {
			fmt.Fprintf(os.Stderr, "Google BooksのISBN参照に失敗しました: %v\n", lookupErr)
			return 1
		}
		return writeResult(result)
	}
	request := googlebooks.SearchBooksRequest{Title: *title, Author: *author, FreeText: *freeText, ExcludedText: *exclude, Limit: *limit, Cursor: *cursor}
	if *rawOutput == "" {
		result, err := client.SearchBooks(ctx, request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Google Booksの検索に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}
	result, raw, searchErr := client.SearchBooksWithRawResponse(ctx, request)
	if raw != nil {
		if err := writeRawResponse(*rawOutput, raw); err != nil {
			fmt.Fprintf(os.Stderr, "Google Booksのrawレスポンスを保存できません: %s: %v\n", *rawOutput, err)
			return 1
		}
	}
	if searchErr != nil {
		fmt.Fprintf(os.Stderr, "Google Booksの検索に失敗しました: %v\n", searchErr)
		return 1
	}
	return writeResult(result)
}

// writeResult は、結果をインデント付きJSONとして標準出力へ書き込む
func writeResult(result any) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "結果をJSONで出力できません: %v\n", err)
		return 1
	}
	return 0
}

// writeRawResponse は、受信したrawレスポンスを新規ファイルへそのまま書き込む
func writeRawResponse(path string, body []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	writeErr := writeAndCloseRawResponse(file, body)
	if writeErr == nil {
		return nil
	}
	return errors.Join(writeErr, os.Remove(path))
}

// writeAndCloseRawResponse は、rawレスポンスを書き込んで出力先を閉じる
func writeAndCloseRawResponse(writer io.WriteCloser, body []byte) error {
	written, writeErr := writer.Write(body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	return errors.Join(writeErr, writer.Close())
}
