# manken (漫画検索ライブラリ)

`manken` は、日本国内の書誌データベースや書店・電子書籍サービスから、漫画単行本を中心とした書誌情報・販売情報を検索・取得するGo言語向けライブラリです。

AniListなどの海外向けメタデータサービスでは、ローマ字表記を中心に扱うことがあります。`manken` は、**日本語のタイトルや著者名を含む国内向けの漫画情報**（出版社、ISBN、発売日、価格、商品URLなど）を取得することを目的としています。

## 主な特徴

- **国内の漫画情報を検索**: 日本国内の書誌データベースや書店・電子書籍サービスから、漫画単行本を中心に検索します。

- **漫画向けに絞り込み**: 書籍検索は、取得元で利用できる分類やカテゴリを使って、可能な範囲で漫画に絞り込みます。

- **ISBNで書籍を参照**: ISBN指定時は漫画向けの絞り込みをせず、該当する書籍情報を参照します。

> Google Booksなど、漫画だけに絞り込めない取得元もあります。電子書籍では単話・分冊・合本などが含まれる場合があります

## 必要な環境

Go 1.26.0以降

## インストール

```text
go get github.com/eamat-dot/manken
```

## クイックスタート

利用したいプロバイダのClientを作成して `manken.Client` に登録し、検索時に `Source` を指定して呼び出します。

### 1. 書籍のタイトル検索

以下は、MADB（メディア芸術データベース）をプロバイダとして登録し、タイトルで検索する例です。

```go
package main

import (
	"context"
	"log"

	"github.com/eamat-dot/manken"
	"github.com/eamat-dot/manken/madb"
)

func main() {
	// プロバイダの初期化
	provider, err := madb.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	// ルートClientへ登録
	client, err := manken.NewClient(manken.WithMADBClient(provider))
	if err != nil {
		log.Fatal(err)
	}

	// MADBを指定して書籍を検索
	result, err := client.SearchBooks(
		context.Background(),
		manken.SourceMADB,
		manken.SearchRequest{Title: "動物のおしゃべり"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

### 2. ISBNでの参照

登録済みのプロバイダと `Source` を指定して、ISBNから書籍情報を取得することも可能です。

```go
	result, err := client.LookupBooksByISBN(
		context.Background(),
		manken.SourceMADB,
		[]string{"4-08-846636-5"},
	)
```

## 対応データソース（Providers）

プロバイダごとの機能対応表および認証情報の要否です。

| データ取得元                                                       | パッケージ      | 対象                  | 書籍検索           | ISBN参照 | APIキー等            |
| ------------------------------------------------------------------ | --------------- | --------------------- | ------------------ | -------- | -------------------- |
| [国立国会図書館サーチ](https://ndlsearch.ndl.go.jp/)               | `ndl`           | 全国書誌              | 漫画に絞り込み     | ○       | **不要**             |
| [メディア芸術データベース](https://mediaarts-db.artmuseums.go.jp/) | `madb`          | 漫画書誌              | ○                 | ○       | **不要**             |
| [openBD](https://openbd.jp/)                                       | `openbd`        | 書誌                  | -                  | ○       | **不要**             |
| [Google Books APIs](https://books.google.com/)                     | `googlebooks`   | 書誌                  | 書籍全般           | ○       | API Key              |
| [Yahoo!ショッピング](https://store.shopping.yahoo.co.jp/tower/)    | `yahooshopping` | 紙書籍 _(タワレコ店)_ | 漫画に絞り込み     | ○       | Client ID            |
| [楽天ブックス](https://books.rakuten.co.jp/)                       | `rakutenbooks`  | 紙書籍                | 漫画に絞り込み     | ○       | App ID, Access Key   |
| [楽天Kobo](https://books.rakuten.co.jp/e-book/)                    | `rakutenkobo`   | 電子書籍              | 漫画に絞り込み     | -        | App ID, Access Key   |
| [DMMブックス](https://book.dmm.com/)                               | `dmm`           | 電子書籍              | シリーズ検索・取得 | -        | API ID, Affiliate ID |

### プロバイダに関する注意事項

- **書籍検索の挙動**: `Google Books` はジャンル指定が不可のため書籍全般が対象になります。\
  `Yahoo!ショッピング` は「タワーレコード Yahoo!店」の商品に限定して検索します。\
  `DMMブックス` はシリーズを検索し、シリーズ内の商品を取得します。

- **表記の揺れ・内容差**: プロバイダによってタイトル副題の扱いや著者名の姓名区切り、電子書籍の単位（単話・合本・無料版等）の扱いが異なります。

- **プロバイダの利用条件**: 各プロバイダの規約（クレジット表示、リクエスト頻度、アフィリエイト条件など）は利用時点の公式情報をご確認ください。
  - **国立国会図書館サーチ**: [NDLサーチ APIのご利用について](https://ndlsearch.ndl.go.jp/help/api)

  - **メディア芸術データベース**: [MADB Lab利用規約](https://mediag.bunka.go.jp/madb_lab/user_terms/)

  - **openBD**: [openBD API利用規約](https://openbd.jp/terms/)

  - **Google Books**: [Google Books API Terms of Service](https://developers.google.com/books/terms) / [Branding Guidelines](https://developers.google.com/books/branding)

  - **Yahoo!ショッピング**: [Yahoo!デベロッパーネットワーク ご利用ガイド](https://developer.yahoo.co.jp/start/) / [クレジット表示](https://developer.yahoo.co.jp/attribution/)

  - **楽天**: [楽天ウェブサービス利用規約](https://webservice.rakuten.co.jp/guide/rule) / [クレジット表示](https://webservice.rakuten.co.jp/guide/credit)

  - **DMMブックス**: [DMMアフィリエイト](https://affiliate.dmm.com/) / [DMMウェブサービス利用規約](https://terms.dmm.com/affiliate_web_service/)

## ドキュメント

- **共通仕様**: [manken API仕様](docs/spec.md)

- **開発者向けガイド**: [アーキテクチャ](ARCHITECTURE.md) / [変更履歴](CHANGELOG.md)

### 各パッケージの使い方

- [NDLサーチ](docs/pkg/ndl/guide.md)
- [MADB](docs/pkg/madb/guide.md)
- [openBD](docs/pkg/openbd/guide.md)
- [Google Books](docs/pkg/googlebooks/guide.md)
- [楽天Books](docs/pkg/rakutenbooks/guide.md)
- [楽天Kobo](docs/pkg/rakutenkobo/guide.md)
- [Yahoo!ショッピング](docs/pkg/yahooshopping/guide.md)
- [DMMブックス](docs/pkg/dmm/guide.md)

## 開発

```bash
go test -v ./...
go build -v ./...
golangci-lint run

```

## ライセンス

[MIT License](LICENSE)

> **Note**: 本ライブラリ自体はMITライセンスですが、取得した書誌データ・画像・APIの利用には各データ提供元の利用規約が適用されます。
