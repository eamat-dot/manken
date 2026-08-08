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

	"github.com/eamat-dot/manken/rakutenbooks"
)

// main は、楽天Booksデモを実行して終了コードを設定する
func main() {
	os.Exit(run())
}

// run は、CLI引数に従って楽天Booksを検索またはISBNで参照する
func run() int {
	title := flag.String("title", "", "タイトルに含める検索語")
	author := flag.String("author", "", "著者名に含める検索語")
	genreName := flag.String("genre", "general", "検索する漫画区分: general, bl, tl")
	limit := flag.Int("limit", 0, "検索件数。1から30まで。0は既定値の20件")
	cursor := flag.String("cursor", "", "前回の結果に含まれる next_cursor")
	rawOutput := flag.String("raw-output", "", "検索またはISBN参照の変換前楽天Booksレスポンスを保存する新規ファイル")
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: demo-rakutenbooks [options] [ISBN]")
		flag.PrintDefaults()
	}
	flag.Parse()

	applicationID := os.Getenv("RAKUTEN_APP_ID")
	accessKey := os.Getenv("RAKUTEN_ACCESS_KEY")
	if applicationID == "" || accessKey == "" {
		fmt.Fprintln(os.Stderr, "RAKUTEN_APP_ID と RAKUTEN_ACCESS_KEY を設定してください")
		return 2
	}
	genre, err := parseGenre(*genreName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	isbns := flag.Args()
	if len(isbns) > 0 && (*title != "" || *author != "" || *limit != 0 || *cursor != "") {
		fmt.Fprintln(os.Stderr, "ISBN参照は検索条件、-limit、-cursor と併用できません")
		return 2
	}
	if len(isbns) == 0 && *title == "" && *author == "" {
		fmt.Fprintln(os.Stderr, "タイトル、著者、またはISBNを1件指定してください")
		flag.Usage()
		return 2
	}

	options := []rakutenbooks.Option{
		rakutenbooks.WithApplicationID(applicationID),
		rakutenbooks.WithAccessKey(accessKey),
		rakutenbooks.WithComicGenre(genre),
	}
	if affiliateID := os.Getenv("RAKUTEN_AFFILIATE_ID"); affiliateID != "" {
		options = append(options, rakutenbooks.WithAffiliateID(affiliateID))
	}
	client, err := rakutenbooks.NewClient(nil, options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "楽天Booksクライアントを作成できません: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if len(isbns) > 0 {
		return runISBNLookup(ctx, client, isbns, *rawOutput)
	}
	request := rakutenbooks.SearchBooksRequest{Title: *title, Author: *author, Limit: *limit, Cursor: *cursor}
	return runSearch(ctx, client, request, *rawOutput)
}

// parseGenre は、CLIの漫画区分名を楽天Booksの公開定数へ変換する
func parseGenre(value string) (rakutenbooks.ComicGenre, error) {
	switch value {
	case "general":
		return rakutenbooks.ComicGenreGeneral, nil
	case "bl":
		return rakutenbooks.ComicGenreBL, nil
	case "tl":
		return rakutenbooks.ComicGenreTL, nil
	default:
		return "", fmt.Errorf("-genre は general, bl, tl のいずれかを指定してください: %s", value)
	}
}

// runISBNLookup は、ISBN参照を実行して必要ならRaw responseを保存する
func runISBNLookup(ctx context.Context, client *rakutenbooks.Client, isbns []string, rawOutput string) int {
	if rawOutput == "" {
		result, err := client.LookupBooksByISBN(ctx, isbns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "楽天BooksのISBN参照に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}
	result, raw, lookupErr := client.LookupBooksByISBNWithRawResponse(ctx, isbns)
	if raw != nil {
		if err := writeRawResponse(rawOutput, raw); err != nil {
			fmt.Fprintf(os.Stderr, "楽天Booksのrawレスポンスを保存できません: %s: %v\n", rawOutput, err)
			return 1
		}
	}
	if lookupErr != nil {
		fmt.Fprintf(os.Stderr, "楽天BooksのISBN参照に失敗しました: %v\n", lookupErr)
		return 1
	}
	return writeResult(result)
}

// runSearch は、書籍検索を実行して必要ならRaw responseを保存する
func runSearch(ctx context.Context, client *rakutenbooks.Client, request rakutenbooks.SearchBooksRequest, rawOutput string) int {
	if rawOutput == "" {
		result, err := client.SearchBooks(ctx, request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "楽天Booksの検索に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}
	result, raw, searchErr := client.SearchBooksWithRawResponse(ctx, request)
	if raw != nil {
		if err := writeRawResponse(rawOutput, raw); err != nil {
			fmt.Fprintf(os.Stderr, "楽天Booksのrawレスポンスを保存できません: %s: %v\n", rawOutput, err)
			return 1
		}
	}
	if searchErr != nil {
		fmt.Fprintf(os.Stderr, "楽天Booksの検索に失敗しました: %v\n", searchErr)
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
