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

// main は、MADBタイトル検索デモを実行して終了コードを設定する
func main() {
	os.Exit(run())
}

// run は、CLI引数に従ってMADBを検索し結果をJSONで出力する
func run() int {
	title := flag.String("title", "", "検索する漫画のタイトル（必須）")
	limit := flag.Int("limit", 0, "取得件数（1～100、0は既定値）")
	cursor := flag.String("cursor", "", "前回の検索結果に含まれるnext_cursor")
	rawOutput := flag.String("raw-output", "", "変換前のMADBレスポンスを保存する新規ファイル")
	flag.Parse()

	if strings.TrimSpace(*title) == "" {
		fmt.Fprintln(os.Stderr, "検索するタイトルを -title で指定してください")
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

	request := madb.SearchBooksRequest{
		Title:  *title,
		Limit:  *limit,
		Cursor: *cursor,
	}

	if *rawOutput == "" {
		result, err := client.SearchBooks(ctx, request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "MADBの検索に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}

	result, rawResponse, searchErr := client.SearchBooksWithRawResponse(ctx, request)
	if rawResponse != nil {
		if err := writeRawResponse(*rawOutput, rawResponse); err != nil {
			fmt.Fprintf(os.Stderr, "MADBのrawレスポンスを保存できません: %s: %v\n", *rawOutput, err)
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

// writeResult は、検索結果をインデント付きJSONとして標準出力へ書き込む
func writeResult(result madb.SearchBooksResult) int {
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
