# 漫検 manken - 漫画のタイトル・著者・ISBNなどを検索する Go ライブラリ

漫画本の書誌情報を検索するためのGoライブラリ。
複数のGoプロジェクトから利用できる共通モデルと検索APIを提供する。

## 現在利用できる機能

- メディア芸術データベース（MADB）のマンガ単行本をタイトル、著者名、ISBN、
  フリーワードで検索する
- 指定した語を含む単行本を検索結果から除外する
- 1回に取得する件数を1件から100件まで指定する
- 検索結果のカーソルを使って続きを取得する
- MADBの書誌情報を取得元に依存しない `madb.Book` として受け取る
- 正規化した書誌情報と、根拠となったMADBの元値を分けて取得する
- タイトル読み、ページ数、紙書籍の大きさを取得する
- 数値巻、版表示、単行本レーベル、参照先シリーズのIDとURLを取得する
- 入力、外部サービス、通信、レスポンス解析のエラーを分類する

キャッシュ、自動リトライ、汎用CLIアプリケーション、MCPサーバーは含めない。

## 初期実装の制約

- `Authors` は主要な創作者の役割を確定できるcreatorと、
  役割表記のないcreatorを含む。未知または不正なcreatorは推測せず、
  MADBの元値だけに残す
- MADBに単一の正規タイトルを示す情報がない場合、安定ソートした先頭を
  `Normalized.Title` の暫定値にする
- 許可した構文に一致しない巻表示は数値化せず、MADBの元値だけを保持する
- 検索結果はMADBリソースURIの昇順であり、関連度、刊行日、巻数では並べ替えない

これらの共通規格は、他の書籍検索APIが提供する項目と検索方式を調査してから再評価する。

## 必要な環境

Go 1.26.0以降を使用する。

## インストール

```text
go get github.com/eamat-dot/manken/madb
```

## 使い方

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
		madb.SearchBooksRequest{
			Title: "動物のおしゃべり",
			Limit: 5,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, book := range result.Books {
		fmt.Println(
			book.Normalized.Title,
			book.Normalized.Authors,
			book.Normalized.Identifiers,
		)
	}
}
```

`Limit` が0の場合は20件取得する。続きがある場合は `NextCursor` を次の
`SearchBooksRequest.Cursor` へそのまま指定する。次ページでも、すべての検索条件と
`Limit` を最初のリクエストと同じ値にする。

ISBNだけで検索する場合は、`Title` の代わりに `ISBN` へISBN-10またはISBN-13を
指定する。ハイフンと空白は省略できる。`Title` と `ISBN` の両方を指定すると、
両方に一致する単行本を検索する。

著者名で検索する場合は `Author` を指定する。MADBのcreator文字列とAgent参照の
両方を対象にする。複数語はすべてを含む著者名に一致し、タイトルやISBNも指定すると
すべての条件に一致する単行本を検索する。

主要な書誌項目を横断して検索する場合は `FreeText` を指定する。空白区切りの全語を
含む単行本を、タイトル、creator、出版社、版表示などから検索する。

指定語を含む単行本を除外する場合は、正の検索条件とともに `ExcludedText` を指定する。
空白区切りの語を1つでも含む単行本を除外する。Agent参照だけに存在する著者名は
除外対象にならない。

`madb.NewClient(nil)` はタイムアウト60秒のHTTPクライアントを使用する。
独自のタイムアウトやTransportが必要な場合は、設定済みの `*http.Client` を渡す。

## CLIデモ

`examples/demo-madb.go` は、任意のタイトル、著者名、ISBN、フリーワード、除外語を
指定してMADBの実サービスを検索する動作確認用のCLIデモである。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5
```

```text
go run ./examples/demo-madb.go -isbn "978-4-08-846636-1"
```

```text
go run ./examples/demo-madb.go -author "佐々木倫子"
```

```text
go run ./examples/demo-madb.go -free-text "うる星 高橋留美子" -exclude "復刻box 愛蔵版"
```

検索結果は、MADBのデータを `madb.Book` へ変換した
`madb.SearchBooksResult` のJSONとして標準出力へ出す。

JSON内の1冊分は次の形になる。

```json
{
  "normalized": {
    "title": "動物のお医者さん",
    "series": [{
      "name": "動物のお医者さん",
      "id": "C262212",
      "url": "https://mediaarts-db.artmuseums.go.jp/id/C262212",
      "source": "madb"
    }],
    "volume": {"number": 8, "label": "8"},
    "authors": ["佐々木倫子"],
    "publishers": ["白泉社"],
    "imprints": ["白泉社文庫"],
    "identifiers": [{"type": "isbn10", "value": "4592881486"}],
    "dates": [{"type": "published", "value": "1996-06-19"}]
  },
  "sources": [{
    "source": "madb",
    "id": "M292129",
    "url": "https://mediaarts-db.artmuseums.go.jp/id/M292129",
    "values": {
      "titles": ["動物のお医者さん"],
      "series_names": ["動物のお医者さん"],
      "volume": "第8巻",
      "authors": ["[著]佐々木倫子"],
      "publishers": ["白泉社　∥　ハクセンシャ"],
      "imprints": ["白泉社文庫"],
      "isbns": ["4592881486"],
      "published_date": "1996-06-19"
    }
  }]
}
```

`normalized` は通常利用する共通書誌情報、`sources` は正規化の根拠となった
MADBの値を表す。欠落した任意項目はJSONへ出力しない。版表示やレーベルの
内容から欠落値を推測または分類しない。出版社名の `∥` より後ろがカナ読みだけの
場合は、`normalized.publishers` から読みを除去して重複を取り除く。

変換前のSPARQL Results JSONも確認する場合は、未作成の保存先を
`-raw-output` へ指定する。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5 -raw-output __madb-result.json
```

標準出力には変換済み検索結果だけを出し、`__madb-result.json` には同じ検索で
受信したrawレスポンスを変更せず保存する。既存ファイルは上書きしない。
`__` で始まるファイルはローカル確認用であり、Git管理対象外となる。

2xx応答のJSON解析や検索結果への変換に失敗した場合も、読み込み済みの
rawレスポンスは保存する。HTTPエラー本文や4 MiBの上限を超えた本文は保存しない。

結果の `next_cursor` に値がある場合は、`-cursor` へ指定すると次ページを取得できる。
`-title`、`-isbn`、`-author`、`-free-text`、`-exclude`、`-limit` は、初回に
指定したものをすべて同じ値で再指定する。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5 -cursor "<next_cursor>"
```

## MADBの利用について

このライブラリは、[メディア芸術データベース](https://mediaarts-db.artmuseums.go.jp/)の
[SPARQLクエリサービス](https://mediag.bunka.go.jp/madb_lab/lod/sparql/)を使用する。
取得データの利用には[MADB Lab利用規約](https://mediag.bunka.go.jp/madb_lab/user_terms/)が
適用される。データを加工して表示する場合は、加工したことを利用者へ示す必要がある。

MADBは技術サポートを提供しておらず、サービス、エンドポイント、データ、
全文検索設定が予告なく変更される可能性がある。このライブラリは自動リトライ、
キャッシュ、アクセス間隔の制御を行わない。

## 設計

- [共通API仕様](docs/spec.md)
- [MADBパッケージ仕様](docs/pkg/madb/spec.md)
- [構想](docs/concept.md)
- [実装TODO](docs/todo/)

データ取得元に依存しない型の実体は `api` パッケージへ置く。
`madb` パッケージは通常利用に必要な型と定数をエイリアスとして公開し、
MADB固有の通信と変換も担当する。MADBだけを利用する場合、`api` を直接
importする必要はない。

ルートは将来、複数の取得元を呼び出す公開ファサードに使用する。
ファサードと複数データ取得元の横断検索は未実装である。

将来MCPサーバーから呼び出せるようにするが、MCP SDKやトランスポートは
検索ライブラリへ依存させない。MCPサーバー自体は未実装である。

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
