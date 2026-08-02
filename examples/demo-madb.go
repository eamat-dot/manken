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

	"github.com/eamat-dot/manken/madb"
)

// stringList は、繰り返し指定された文字列オプションを入力順に保持する
type stringList []string

// String は、指定済みの文字列をカンマ区切りで返す
func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

// Set は、指定された文字列を末尾へ追加する
func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

// main は、MADB検索デモを実行して終了コードを設定する
func main() {
	os.Exit(run())
}

// run は、CLI引数に従ってMADBを検索し結果をJSONで出力する
func run() int {
	title := flag.String("title", "", "検索する漫画のタイトル（空白区切りの全語を含む）")
	var isbns stringList
	flag.Var(&isbns, "isbn", "参照する漫画のISBN-10またはISBN-13（繰り返し指定可）")
	author := flag.String("author", "", "検索する漫画の著者名（空白区切りの全語を含む）")
	freeText := flag.String("free-text", "", "主要な書誌項目を横断する検索語（空白区切りの全語を含む）")
	excludedText := flag.String("exclude", "", "主要な書誌項目から除外する語（空白区切りのいずれかを含む本を除外）")
	limit := flag.Int("limit", 0, "取得件数（1～100、0は既定値）")
	cursor := flag.String("cursor", "", "前回の検索結果に含まれるnext_cursor")
	rawOutput := flag.String("raw-output", "", "変換前のMADBレスポンスを保存する新規ファイル")
	flag.Parse()

	hasSearchCondition := strings.TrimSpace(*title) != "" ||
		strings.TrimSpace(*author) != "" || strings.TrimSpace(*freeText) != ""
	if len(isbns) == 0 && !hasSearchCondition {
		fmt.Fprintln(os.Stderr, "検索条件または -isbn を指定してください")
		flag.Usage()
		return 2
	}
	if len(isbns) != 0 && searchOptionWasSet() {
		fmt.Fprintln(os.Stderr, "-isbn は検索用の -title、-author、-free-text、-exclude、-limit、-cursor と併用できません")
		flag.Usage()
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := madb.NewClient(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MADBクライアントを作成できません: %v\n", err)
		return 1
	}

	if len(isbns) != 0 {
		return runISBNLookup(ctx, client, isbns, *rawOutput)
	}
	return runSearch(ctx, client, madb.SearchBooksRequest{
		Title:        *title,
		Author:       *author,
		FreeText:     *freeText,
		ExcludedText: *excludedText,
		Limit:        *limit,
		Cursor:       *cursor,
	}, *rawOutput)
}

// searchOptionWasSet は、ISBN参照と併用できない検索用フラグが明示されたか判定する
func searchOptionWasSet() bool {
	wasSet := false
	flag.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "title", "author", "free-text", "exclude", "limit", "cursor":
			wasSet = true
		}
	})
	return wasSet
}

// runSearch は、MADBの書誌検索を実行して結果と任意のrawレスポンスを出力する
func runSearch(ctx context.Context, client *madb.Client, request madb.SearchBooksRequest, rawOutput string) int {
	if rawOutput == "" {
		result, err := client.SearchBooks(ctx, request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "MADBの検索に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}

	result, rawResponse, searchErr := client.SearchBooksWithRawResponse(ctx, request)
	if rawResponse != nil {
		if err := writeRawResponse(rawOutput, rawResponse); err != nil {
			fmt.Fprintf(os.Stderr, "MADBのrawレスポンスを保存できません: %s: %v\n", rawOutput, err)
			if searchErr != nil {
				fmt.Fprintf(os.Stderr, "MADBの検索に失敗しました: %v\n", searchErr)
			}
			return 1
		}
	}
	if searchErr != nil {
		fmt.Fprintf(os.Stderr, "MADBの検索に失敗しました: %v\n", searchErr)
		return 1
	}

	return writeResult(result)
}

// runISBNLookup は、MADBのISBN参照を実行して結果と任意のrawレスポンスを出力する
func runISBNLookup(ctx context.Context, client *madb.Client, isbns []string, rawOutput string) int {
	if rawOutput == "" {
		result, err := client.LookupBooksByISBN(ctx, isbns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "MADBのISBN参照に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}

	result, rawResponse, lookupErr := client.LookupBooksByISBNWithRawResponse(ctx, isbns)
	if rawResponse != nil {
		if err := writeRawResponse(rawOutput, rawResponse); err != nil {
			fmt.Fprintf(os.Stderr, "MADBのrawレスポンスを保存できません: %s: %v\n", rawOutput, err)
			if lookupErr != nil {
				fmt.Fprintf(os.Stderr, "MADBのISBN参照に失敗しました: %v\n", lookupErr)
			}
			return 1
		}
	}
	if lookupErr != nil {
		fmt.Fprintf(os.Stderr, "MADBのISBN参照に失敗しました: %v\n", lookupErr)
		return 1
	}
	return writeResult(result)
}

// writeResult は、結果をインデント付きJSONとして標準出力へ書き込む
func writeResult(result any) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "検索結果をJSONで出力できません: %v\n", err)
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

	removeErr := os.Remove(path)
	return errors.Join(writeErr, removeErr)
}

// writeAndCloseRawResponse は、rawレスポンスを書き込んで出力先を閉じる
func writeAndCloseRawResponse(writer io.WriteCloser, body []byte) error {
	written, writeErr := writer.Write(body)
	if writeErr == nil && written != len(body) {
		writeErr = io.ErrShortWrite
	}
	closeErr := writer.Close()
	return errors.Join(writeErr, closeErr)
}
