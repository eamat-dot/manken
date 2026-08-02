# 漫検 manken - 漫画の書誌情報を検索・ISBN参照する Go ライブラリ

`madb` パッケージで、メディア芸術データベース（MADB）から漫画の書誌情報を検索・参照する。
タイトル、著者名、フリーワードで単行本を検索するほか、複数のISBNから書籍を参照し、
データ取得元に依存しない共通書籍モデルで結果を受け取れる。

## 検索できる条件

- タイトル
- 著者名
- 複数の書誌項目を対象とするフリーワード
- 指定語を含む結果の除外

取得件数の指定とカーソルによるページングも利用できる。
ISBN-10またはISBN-13による参照は、検索とは独立したAPIで1回に500件まで指定できる。

## 必要な環境

Go 1.26.0以降を使用する。

## インストール

```text
go get github.com/eamat-dot/manken/madb
```

MADB検索に必要な `madb` パッケージを利用側のGoモジュールへ追加する。

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/eamat-dot/manken/madb"
)

func main() {
	client, err := madb.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.SearchBooks(
		context.Background(),
		madb.SearchBooksRequest{Title: "動物のおしゃべり"},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, book := range result.Books {
		fmt.Println(book.Normalized.Title, book.Normalized.Authors)
	}
}
```

タイトルに「動物のおしゃべり」を含む単行本を検索し、タイトルと著者を出力する。
ISBN参照は `client.LookupBooksByISBN(ctx, []string{"4088466365"})` のように呼び出す。
検索条件、ISBN参照、ページング、エラーの詳細は
[MADBパッケージ仕様](docs/pkg/madb/spec.md)を参照する。

## CLIデモ

`examples/demo-madb.go` で、MADBの実サービスを検索・参照してJSON結果を確認できる。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5
```

タイトルに「動物のおしゃべり」を含む単行本を5件まで検索する。
全オプションと操作方法は [MADB CLIデモ](examples/README.md) を参照する。

## MADBの利用について

取得データの利用には [MADB Lab利用規約](https://mediag.bunka.go.jp/madb_lab/user_terms/) が
適用される。データを加工して表示する場合は、加工したことを利用者へ示す必要がある。
通信条件とサービス変更時の注意は [MADBパッケージ仕様](docs/pkg/madb/spec.md) に記載する。

## ドキュメント

- [MADBパッケージ仕様](docs/pkg/madb/spec.md): MADB固有の検索、変換、通信、エラー
- [共通API仕様](docs/spec.md): 共通書籍モデルとAPIの契約
- [アーキテクチャ](ARCHITECTURE.md): パッケージ構成と依存関係
- [Changelog](CHANGELOG.md): 利用者に影響する変更

## 開発

```text
task mod-deps
task lint
task test
task build
task all
```

`task all` は依存関係の整理、lint、テスト、全パッケージのビルドを順に実行する。
通常のテストは実サービスへ接続しない。

MADBの実サービスを明示的に確認する場合は次を実行する。

```text
go test -v -tags=integration ./madb
```

## ライセンス

[MIT License](LICENSE)
