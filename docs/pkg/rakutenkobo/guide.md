# 楽天Koboガイド

`rakutenkobo` パッケージは、楽天Koboから電子コミックを検索し、商品情報や書誌情報を取得するためのパッケージです。ISBN参照には対応していません。

## 準備するもの

楽天ウェブサービスで発行したApplication IDとAccess Keyが必要です。Affiliate IDは任意で、楽天アフィリエイトのURLが必要な場合だけ設定します。

認証情報は環境変数やシークレット管理機能で保管し、ソースコードへ直接書かないことを推奨します。`manken` は環境変数から自動では読み込みません。

## 直接利用する

`rakutenkobo` パッケージを直接インポートして利用できます。

```go
import "github.com/eamat-dot/manken/rakutenkobo"
```

最小限のタイトル検索は次のように書けます。

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/eamat-dot/manken/rakutenkobo"
)

func main() {
	// 認証情報を指定して楽天Kobo Clientを初期化する
	client, err := rakutenkobo.NewClient(nil,
		rakutenkobo.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
		rakutenkobo.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 楽天Koboでタイトル検索する
	result, err := client.SearchBooks(
		context.Background(),
		rakutenkobo.SearchRequest{Title: "ふつつかな悪女ではございますが"},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d books found", len(result.Books))
}
```

Affiliate IDを使う場合は、Client作成時に `WithAffiliateID` を追加してください。

```go
// Affiliate IDを含めて楽天Kobo Clientを初期化する
client, err := rakutenkobo.NewClient(nil,
	rakutenkobo.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
	rakutenkobo.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
	rakutenkobo.WithAffiliateID(os.Getenv("RAKUTEN_AFFILIATE_ID")),
)
```

## 電子コミックを検索する

タイトル、著者名、出版社名、商品キーワードを指定できます。除外語も利用できますが、除外語だけで検索することはできません。

```go
// 楽天Koboで著者名を指定して検索する
result, err := client.SearchBooks(
	context.Background(),
	rakutenkobo.SearchRequest{Author: "尾羊英"},
)
```

既定では一般コミックを検索します。BLまたはTLを検索する場合は、Client作成時に `WithComicGenre` を指定してください。1回の検索で複数の漫画区分をまとめて検索することはありません。

電子書籍の商品検索なので、単話、分冊、合本、無料版などが検索結果に含まれる場合があります。

## Raw responseを取得する

`SearchBooksWithRawResponse` を使うと、変換済みの結果と楽天Koboから受信した成功レスポンス本文を取得できます。

## 利用条件

楽天Koboの商品情報、画像、価格、アフィリエイトURLなどを表示・保存・再利用する場合は、利用時点の最新条件を確認してください。クレジット表示も楽天ウェブサービスの現行案内に従ってください。

- [楽天ウェブサービス利用ガイド（公式）](https://webservice.rakuten.co.jp/guide)
- [クレジット表示（公式）](https://webservice.rakuten.co.jp/guide/credit)
- [楽天ウェブサービス利用規約（公式）](https://webservice.rakuten.co.jp/guide/rule)

`manken` は利用方法が各規約に適合することを保証しません。

## 詳細仕様

検索条件、除外語、漫画区分、カーソル、変換、Raw response、通信、エラーの完全な仕様は[manken 楽天Koboパッケージ仕様](spec.md)を参照してください。
