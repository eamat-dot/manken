package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/eamat-dot/manken/ndl"
)

// main は、NDLサーチデモを実行して終了コードを設定する
func main() { os.Exit(run()) }

// run は、CLI引数に従ってNDLサーチを検索またはISBNで参照する
func run() int {
	title := flag.String("title", "", "タイトルに含める検索語")
	author := flag.String("author", "", "著者名に含める検索語")
	publisher := flag.String("publisher", "", "出版社名に含める検索語")
	query := flag.String("query", "", "複数の書誌項目を対象にする検索語")
	from := flag.String("from", "", "出版年月日の開始。YYYY、YYYY-MM、YYYY-MM-DD")
	to := flag.String("to", "", "出版年月日の終了。YYYY、YYYY-MM、YYYY-MM-DD")
	subject := flag.String("subject", "", "件名に含める検索語")
	description := flag.String("description", "", "内容記述に含める検索語")
	mangaNDC := flag.Bool("manga-ndc", true, "NDC 726.1の漫画分類フィルタを有効にする")
	mangaNDLC := flag.Bool("manga-ndlc", true, "NDLC Y84の漫画分類フィルタを有効にする")
	limit := flag.Int("limit", 0, "検索件数。1から500まで。0は既定値の20件")
	cursor := flag.String("cursor", "", "前回の結果に含まれる next_cursor")
	rawOutput := flag.String("raw-output", "", "検索またはISBN参照の変換前NDLサーチXMLを保存する新規ファイル")
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: ndl [options] [ISBN]")
		flag.PrintDefaults()
	}
	flag.Parse()
	isbns := flag.Args()
	if err := validateArguments(isbns, *title, *author, *publisher, *query, *from, *to, *subject, *description, searchOptionWasSet()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		return 2
	}
	client, err := ndl.NewClient(nil, ndl.WithMangaNDCFilter(*mangaNDC), ndl.WithMangaNDLCFilter(*mangaNDLC))
	if err != nil {
		fmt.Fprintf(os.Stderr, "NDLサーチクライアントを作成できません: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if len(isbns) == 1 {
		result, raw, err := client.LookupBooksByISBNWithRawResponse(ctx, isbns)
		return finish(result, raw, err, *rawOutput)
	}
	request := ndl.SearchRequest{Title: *title, Author: *author, Publisher: *publisher, Query: *query, DateFrom: *from, DateTo: *to, Limit: *limit, Cursor: *cursor}
	options := ndl.SearchOptions{Subject: *subject, Description: *description}
	result, raw, err := client.SearchBooksWithOptionsAndRawResponse(ctx, request, options)
	return finish(result, raw, err, *rawOutput)
}

// validateArguments は、検索条件とISBN位置引数の組み合わせを検証する
func validateArguments(isbns []string, title, author, publisher, query, from, to, subject, description string, searchOptionSet bool) error {
	if len(isbns) > 1 {
		return errors.New("ISBN参照は1件だけ指定してください")
	}
	if len(isbns) == 1 && searchOptionSet {
		return errors.New("ISBN参照は検索条件、-limit、-cursor、NDL固有検索フラグ、漫画分類フラグと併用できません")
	}
	if len(isbns) == 0 && !anyNonEmpty(title, author, publisher, query, from, to, subject, description) {
		return errors.New("検索条件またはISBNを1件指定してください")
	}
	return nil
}

// searchOptionWasSet は、ISBN参照と併用できない検索用フラグが明示されたか判定する
func searchOptionWasSet() bool {
	wasSet := false
	flag.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "title", "author", "publisher", "query", "from", "to", "subject", "description", "manga-ndc", "manga-ndlc", "limit", "cursor":
			wasSet = true
		}
	})
	return wasSet
}

// anyNonEmpty は、空白以外の検索条件が一つでもあるか判定する
func anyNonEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

// finish は、Raw responseを保存してから結果または失敗を出力する
func finish(result any, raw []byte, operationErr error, rawOutput string) int {
	if rawOutput != "" && raw != nil {
		if err := writeRawResponse(rawOutput, raw); err != nil {
			fmt.Fprintf(os.Stderr, "NDLサーチのrawレスポンスを保存できません: %s: %v\n", rawOutput, err)
			return 1
		}
	}
	if operationErr != nil {
		fmt.Fprintf(os.Stderr, "NDLサーチへの問い合わせに失敗しました: %v\n", operationErr)
		return 1
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "結果をJSONで出力できません: %v\n", err)
		return 1
	}
	return 0
}

// writeRawResponse は、rawレスポンスを既存ファイルを上書きせず保存する
func writeRawResponse(path string, body []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return errors.Join(err, os.Remove(path))
	}
	return nil
}
