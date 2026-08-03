# 漫検 manken - 漫画の書誌情報を検索・ISBN参照する Go ライブラリ

`madb` パッケージでメディア芸術データベース（MADB）から漫画の書誌情報を検索・参照し、
`openbd` パッケージでopenBDから複数ISBNの書誌情報を参照する。
どちらもデータ取得元に依存しない共通書籍モデルで結果を受け取れる。

## 検索できる条件

- タイトル
- 著者名
- 複数の書誌項目を対象とするフリーワード
- 指定語を含む結果の除外

取得件数の指定とカーソルによるページングも利用できる。
ISBN-10またはISBN-13による参照は、検索とは独立したAPIで行う。1回に指定できる件数は
MADBが500件、openBDが1,000件となる。

## 必要な環境

Go 1.26.0以降を使用する。

## インストール

```text
go get github.com/eamat-dot/manken/madb
go get github.com/eamat-dot/manken/openbd
```

使用するデータ取得元のパッケージを利用側のGoモジュールへ追加する。

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

openBDで複数のISBNを参照する場合は次のように呼び出す。

```go
package main

import (
	"context"
	"log"

	"github.com/eamat-dot/manken/openbd"
)

func main() {
	client, err := openbd.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.LookupBooksByISBN(
		context.Background(),
		[]string{"4088466365", "9784048689410"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d件の入力結果を取得", len(result.Items))
}
```

結果は入力ISBNと同じ件数、同じ順序で返る。未収録ISBNの `Books` は空になる。
変換項目と通信条件は [openBDパッケージ仕様](docs/pkg/openbd/spec.md)を参照する。

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

## openBDの利用について

openBDの書誌・書影などは、本の販促・紹介目的に限って利用できる。
[openBD API利用規約](https://openbd.jp/terms/)への同意と、削除要請への対応が必要となる。
現在のAPIは従来と同じURLと応答形式を維持しているが、代替書誌への移行により
項目の欠落や書影収録範囲の縮小がある。詳細は
[openBDパッケージ仕様](docs/pkg/openbd/spec.md)に記載する。

## ドキュメント

- [MADBパッケージ仕様](docs/pkg/madb/spec.md): MADB固有の検索、変換、通信、エラー
- [openBDパッケージ仕様](docs/pkg/openbd/spec.md): openBD固有のISBN参照、変換、通信、エラー
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
go test -v -tags=integration ./openbd
```

## ライセンス

[MIT License](LICENSE)
