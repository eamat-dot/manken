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

	"github.com/eamat-dot/manken/openbd"
)

// main は、openBD ISBN参照デモを実行して終了コードを設定する
func main() {
	os.Exit(run())
}

// run は、CLI引数に従ってopenBDをISBNで参照し結果をJSONで出力する
func run() int {
	rawOutput := flag.String("raw-output", "", "変換前のopenBDレスポンスを保存する新規ファイル")
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: demo-openbd [options] ISBN...")
		flag.PrintDefaults()
	}
	flag.Parse()

	isbns := flag.Args()
	if len(isbns) == 0 {
		fmt.Fprintln(os.Stderr, "ISBNを1件以上指定してください")
		flag.Usage()
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := openbd.NewClient(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openBDクライアントを作成できません: %v\n", err)
		return 1
	}

	if *rawOutput == "" {
		result, err := client.LookupBooksByISBN(ctx, isbns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "openBDのISBN参照に失敗しました: %v\n", err)
			return 1
		}
		return writeResult(result)
	}

	result, rawResponse, lookupErr := client.LookupBooksByISBNWithRawResponse(ctx, isbns)
	if rawResponse != nil {
		if err := writeRawResponse(*rawOutput, rawResponse); err != nil {
			fmt.Fprintf(os.Stderr, "openBDのrawレスポンスを保存できません: %s: %v\n", *rawOutput, err)
			if lookupErr != nil {
				fmt.Fprintf(os.Stderr, "openBDのISBN参照に失敗しました: %v\n", lookupErr)
			}
			return 1
		}
	}
	if lookupErr != nil {
		fmt.Fprintf(os.Stderr, "openBDのISBN参照に失敗しました: %v\n", lookupErr)
		return 1
	}

	return writeResult(result)
}

// writeResult は、結果をインデント付きJSONとして標準出力へ書き込む
func writeResult(result any) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "ISBN参照結果をJSONで出力できません: %v\n", err)
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
