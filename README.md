# 漫検 manken - 漫画の書誌情報を検索・取得する Go ライブラリ

`manken` は、漫画の書誌情報を検索・取得し、データ取得元に依存しない共通書籍モデルで
扱うためのGoライブラリである。利用側は、必要なプロバイダのパッケージだけを導入できる。

## 対応プロバイダ

| パッケージ | データ取得元 | 利用できる機能 | 利用開始に必要なもの |
| --- | --- | --- | --- |
| `madb` | [メディア芸術データベース（MADB）](https://mediaarts-db.artmuseums.go.jp/) / [MADB Lab](https://mediag.bunka.go.jp/madb_lab/) | タイトル・著者名などによる検索、ISBNによる取得 | APIキー・アカウント登録は不要。MADB Lab利用規約の確認が必要 |
| `openbd` | [openBD](https://openbd.jp/) | ISBNによる取得 | APIキー・アカウント登録は不要。openBD API利用規約への同意が必要 |
| `googlebooks` | [Google Books](https://books.google.com/) | タイトル・著者名・フリーワードによる検索、1件のISBNによる取得 | Google Books APIキーとTerms / Brandingの確認が必要 |

### 利用前の確認

- MADBのデータを利用する場合は出典を記載する。編集・加工した場合は、その旨も記載する。
  詳細は [MADB Lab利用規約](https://mediag.bunka.go.jp/madb_lab/user_terms/) を確認する。
- openBDの書誌・書影などは、本の紹介・販促目的に限って利用できる。取得データの任意改変は
  認められておらず、削除要請を受けた場合は対応が必要となる。詳細は
  [openBD API利用規約](https://openbd.jp/terms/) を確認する。
- openBDは、収録されていないISBNや、一部の書誌項目・書影がない書籍を含む。取得結果の扱いは
  [openBDパッケージ仕様](docs/pkg/openbd/spec.md)を参照する。
- Google Booksは漫画専用の取得元ではない。検索結果の表示、保存、課金形態には
  [Google Booksガイド](docs/pkg/googlebooks/guide.md)のTerms / Branding上の注意が適用される。

## 主な機能

### MADB

- タイトルによる検索
- 著者名による検索
- 複数の書誌項目を対象とするフリーワード検索
- 指定語を含む結果の除外
- 取得件数の指定とカーソルによるページング
- ISBN-10またはISBN-13による書誌情報の取得

### openBD

- 複数のISBN-10またはISBN-13による書誌情報の取得

### Google Books

- タイトル、著者名、複数の書誌項目を対象にするフリーワードによる検索
- 指定語を含む結果の除外
- 取得件数の指定とカーソルによるページング
- ISBN-10またはISBN-13を1件指定した書誌情報の取得
- 変換済みの検索・ISBN参照結果と受信したrawレスポンスの取得

## 必要な環境

Go 1.26.0以降を使用する。

## インストール

```text
go get github.com/eamat-dot/manken/madb
go get github.com/eamat-dot/manken/openbd
go get github.com/eamat-dot/manken/googlebooks
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

Google Booksを使う場合は、利用側でAPIキーを安全に読み込んでClientへ渡す。

```go
client, err := googlebooks.NewClient(nil,
    googlebooks.WithAPIKey(os.Getenv("GOOGLE_BOOKS_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}
result, err := client.SearchBooks(
    context.Background(),
    googlebooks.SearchBooksRequest{Title: "動物のお医者さん"},
)
```

Google Booksは漫画以外も返す。検索条件、ISBN参照、利用条件は
[Google Booksパッケージ仕様](docs/pkg/googlebooks/spec.md)と
[Google Booksガイド](docs/pkg/googlebooks/guide.md)を参照する。

## CLIデモ

`examples/madb` で、MADBの実サービスを検索・参照してJSON結果を確認できる。

```text
go run ./examples/madb -title "動物のおしゃべり" -limit 5
```

タイトルに「動物のおしゃべり」を含む単行本を5件まで検索する。
全オプションと操作方法は [MADB CLIデモ](examples/README.md) を参照する。

Google BooksのAPIキーを環境変数へ設定済みの場合は、次のデモを実行できる。

```text
go run ./examples/googlebooks -title "動物のお医者さん" -limit 5
```

## ドキュメント

- [MADBパッケージ仕様](docs/pkg/madb/spec.md): MADB固有の検索、変換、通信、エラー
- [openBDパッケージ仕様](docs/pkg/openbd/spec.md): openBD固有のISBN参照、変換、通信、エラー
- [Google Booksパッケージ仕様](docs/pkg/googlebooks/spec.md): Google Books固有の検索、ISBN参照、変換、通信、エラー
- [Google Booksガイド](docs/pkg/googlebooks/guide.md): APIキー、デモ、利用条件
- [共通API仕様](docs/spec.md): 共通書籍モデルとAPI仕様
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

各取得元の実サービスを明示的に確認する場合は次を実行する。

```text
go test -v -tags=integration ./madb
go test -v -tags=integration ./openbd
go test -v -tags=integration ./googlebooks
```

## ライセンス

[MIT License](LICENSE)
