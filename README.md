# 漫検 manken - 漫画の書誌情報を検索・取得する Go ライブラリ

`manken` は、漫画の書誌情報を検索・取得し、データ取得元に依存しない共通書籍モデルで
扱うためのGoライブラリである。利用側は、必要なプロバイダのパッケージだけを導入できる。

## 対応プロバイダ

| パッケージ      | データ取得元                                                                                                                  | 利用できる機能                                                               | 利用開始に必要なもの                                                |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| `madb`          | [メディア芸術データベース（MADB）](https://mediaarts-db.artmuseums.go.jp/) / [MADB Lab](https://mediag.bunka.go.jp/madb_lab/) | タイトル・著者名・出版社名などによる検索、ISBNによる取得                     | APIキー・アカウント登録は不要。MADB Lab利用規約の確認が必要         |
| `openbd`        | [openBD](https://openbd.jp/)                                                                                                  | ISBNによる取得                                                               | APIキー・アカウント登録は不要。openBD API利用規約への同意が必要     |
| `googlebooks`   | [Google Books](https://books.google.com/)                                                                                     | タイトル・著者名・出版社名・フリーワードによる検索、1件のISBNによる取得      | Google Books APIキーとTerms / Brandingの確認が必要                  |
| `rakutenbooks`  | [楽天ブックス](https://books.rakuten.co.jp/)                                                                                  | 一般・BL・TLコミックのタイトル・著者名・出版社名検索、1件のISBNによる取得    | Application IDとAccess Keyが必要。Affiliate IDは任意                |
| `rakutenkobo`   | [楽天Kobo](https://books.rakuten.co.jp/e-book/)                                                                               | 一般・BL・TLコミックのタイトル・著者名・出版社名・商品キーワード・除外語検索 | Application IDとAccess Keyが必要。Affiliate IDは任意                |
| `yahooshopping` | [Yahoo!ショッピング](https://shopping.yahoo.co.jp/)                                                                           | Tower固定の紙コミック商品キーワード検索、1件のISBNによる取得                 | Yahoo!ショッピングClient IDとクレジット表示要件の確認が必要         |
| `ndl`           | [国立国会図書館サーチ](https://ndlsearch.ndl.go.jp/)                                                                          | 完成済み全国書誌のタイトル・著者・出版社・フリーワード検索、1件のISBN参照    | APIキー不要。NDLサーチAPIの利用表示と書誌データの利用条件確認が必要 |

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
- 楽天Booksは楽天ウェブサービスの利用条件に従う。リクエスト頻度、クレジット表示、
  データの保存・更新条件は [楽天Booksガイド](docs/pkg/rakutenbooks/guide.md) を確認する。
- 楽天Koboも楽天ウェブサービスの利用条件に従う。Application ID / Access Key、クレジット表示、
  電子書籍の商品情報の扱いは [楽天Koboガイド](docs/pkg/rakutenkobo/guide.md) を確認する。
- Yahoo!ショッピングはTower固定の商品検索であり、タイトル・著者・出版社は専用書誌検索ではない。
  公式の1クエリ/秒とクレジット表示要件、Raw responseの保存条件の未確定性は
  [Yahoo!ショッピングパッケージ仕様](docs/pkg/yahooshopping/spec.md)を確認する。
- NDLサーチAPIを利用するサイトやアプリケーションでは、その利用を表示する。全国書誌情報を二次利用する場合は
  [NDLサーチガイド](docs/pkg/ndl/guide.md)の表示・利用条件と大量アクセス時の注意を確認する。

## 主な機能

### MADB

- タイトルによる検索
- 著者名による検索
- 出版社名による検索
- 複数の書誌項目を対象とするフリーワード検索
- 指定語を含む結果の除外
- 取得件数の指定とカーソルによるページング
- ISBN-10またはISBN-13による書誌情報の取得

### openBD

- 複数のISBN-10またはISBN-13による書誌情報の取得

### Google Books

- タイトル、著者名、出版社名、複数の書誌項目を対象にするフリーワードによる検索
- 指定語を含む結果の除外
- 取得件数の指定とカーソルによるページング
- ISBN-10またはISBN-13を1件指定した書誌情報の取得
- 変換済みの検索・ISBN参照結果と受信したrawレスポンスの取得

### 楽天Books

- 一般・BL・TLコミックを区分したタイトル・著者名・出版社名検索と楽天Booksの商品形態による任意の絞り込み
- 取得件数の指定とカーソルによるページング
- ISBN-10またはISBN-13を1件指定した書誌情報の取得
- 取得時点の税込販売価格を共通価格情報として取得
- Affiliate IDを任意設定し、通常商品URLとは別のアフィリエイトURLを取得
- 変換済み結果と受信したrawレスポンスの取得

### 楽天Kobo

- 一般・BL・TLコミックを区分したタイトル・著者名・出版社名・商品キーワード検索
- 指定語を含む結果の除外
- 取得件数の指定とカーソルによるページング
- Kobo商品番号、取得時点の税込販売価格、商品URL、画像の取得
- Affiliate IDを任意設定し、通常商品URLとは別のアフィリエイトURLを取得
- 変換済み結果と受信したrawレスポンスの取得

### NDLサーチ

- 完成済み全国書誌を対象にするタイトル・著者名・出版社名・フリーワード検索
- 出版時期、件名、内容記述によるNDL固有の絞り込み
- NDC 726.1 / NDLC Y84による漫画候補の既定絞り込みと個別解除
- ISBN-10またはISBN-13を1件指定した書誌情報の取得
- 取得件数の指定とカーソルによるページング
- DC-NDL v3から変換した巻、版、シリーズ、読み、分類などと受信したraw XMLの取得

## 必要な環境

Go 1.26.0以降を使用する。

## インストール

```text
go get github.com/eamat-dot/manken/madb
go get github.com/eamat-dot/manken/openbd
go get github.com/eamat-dot/manken/googlebooks
go get github.com/eamat-dot/manken/rakutenbooks
go get github.com/eamat-dot/manken/rakutenkobo
go get github.com/eamat-dot/manken/ndl
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

楽天Booksを使う場合はApplication IDとAccess Keyを設定する。Affiliate IDは任意である。

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/eamat-dot/manken/rakutenbooks"
)

func main() {
    client, err := rakutenbooks.NewClient(nil,
        rakutenbooks.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
        rakutenbooks.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }
    result, err := client.SearchBooks(
        context.Background(),
        rakutenbooks.SearchBooksRequest{Title: "動物のお医者さん"},
    )
    if err != nil {
        log.Fatal(err)
    }
    for _, book := range result.Books {
        log.Println(book.Normalized.Title)
    }
}
```

既定は一般コミックを検索する。BL・TLの指定、Affiliate ID、利用条件は
[楽天Booksパッケージ仕様](docs/pkg/rakutenbooks/spec.md)と
[楽天Booksガイド](docs/pkg/rakutenbooks/guide.md)を参照する。

楽天Koboも同じApplication ID / Access Key方式でClientを作成し、電子書籍を検索できる。
ISBN参照は提供しない。検索条件と除外語の組み合わせ、商品番号の扱いは
[楽天Koboパッケージ仕様](docs/pkg/rakutenkobo/spec.md)と
[楽天Koboガイド](docs/pkg/rakutenkobo/guide.md)を参照する。

NDLサーチは認証情報なしで利用できる。タイトル検索とISBN参照は次のように呼び出す。

```go
client, err := ndl.NewClient(nil)
if err != nil {
    log.Fatal(err)
}
result, err := client.SearchBooks(
    context.Background(),
    ndl.SearchBooksRequest{Title: "動物のお医者さん"},
)
```

NDLサーチは漫画専用の取得元ではなく、タイトル検索は関連タイトルも対象にする。検索条件、変換項目、利用条件は
[NDLサーチパッケージ仕様](docs/pkg/ndl/spec.md)と[NDLサーチガイド](docs/pkg/ndl/guide.md)を参照する。

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

楽天BooksのApplication IDとAccess Keyを環境変数へ設定済みの場合は、次のデモを実行できる。

```text
go run ./examples/rakutenbooks -title "動物のお医者さん" -limit 5
```

同じ環境変数で楽天Koboを利用できるアプリでは、電子コミック検索も実行できる。

```text
go run ./examples/rakutenkobo -title "ふつつかな悪女ではございますが" -limit 5
```

認証情報なしでNDLサーチのデモを実行できる。

```text
go run ./examples/ndl -title "動物のお医者さん" -limit 5
```

## ドキュメント

- [MADBパッケージ仕様](docs/pkg/madb/spec.md): MADB固有の検索、変換、通信、エラー
- [openBDパッケージ仕様](docs/pkg/openbd/spec.md): openBD固有のISBN参照、変換、通信、エラー
- [Google Booksパッケージ仕様](docs/pkg/googlebooks/spec.md): Google Books固有の検索、ISBN参照、変換、通信、エラー
- [Google Booksガイド](docs/pkg/googlebooks/guide.md): APIキー、デモ、利用条件
- [楽天Booksパッケージ仕様](docs/pkg/rakutenbooks/spec.md): 楽天Books固有の検索、漫画区分、ISBN参照、変換、通信、エラー
- [楽天Booksガイド](docs/pkg/rakutenbooks/guide.md): 認証情報、Affiliate ID、デモ、利用条件
- [楽天Koboパッケージ仕様](docs/pkg/rakutenkobo/spec.md): 楽天Kobo固有の検索、変換、通信、エラー
- [楽天Koboガイド](docs/pkg/rakutenkobo/guide.md): 認証情報、Affiliate ID、利用条件
- [NDLサーチパッケージ仕様](docs/pkg/ndl/spec.md): NDLサーチ固有の検索、ISBN参照、変換、通信、エラー
- [NDLサーチガイド](docs/pkg/ndl/guide.md): 利用表示、書誌データの二次利用条件、大量アクセス時の注意
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
go test -v -tags=integration ./rakutenbooks
go test -v -tags=integration ./rakutenkobo
go test -v -tags=integration ./ndl
```

## ライセンス

[MIT License](LICENSE)
