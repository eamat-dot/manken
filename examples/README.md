# MADB CLIデモ

`demo-madb.go` は、`madb` パッケージでMADBの実サービスを検索・ISBN参照し、結果をJSONで
確認するための動作確認用CLIである。汎用の検索アプリケーションではない。

## 基本的な使い方

リポジトリのルートで次を実行する。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5
```

タイトルに「動物のおしゃべり」を含む単行本を5件まで検索し、共通書籍モデルへ
変換した結果を標準出力へJSONで出す。

## オプション

| オプション | 内容 |
| --- | --- |
| `-title` | タイトルに含める検索語 |
| `-isbn` | 参照するISBN-10またはISBN-13。500件まで繰り返し指定可能 |
| `-author` | 著者名に含める検索語 |
| `-free-text` | 主要な書誌項目を横断して検索する語 |
| `-exclude` | 主要な書誌項目に含まれる場合、結果から除外する語 |
| `-limit` | 取得件数。1から100まで。0は既定値の20件 |
| `-cursor` | 前回の結果に含まれる `next_cursor` |
| `-raw-output` | MADBから受信した変換前レスポンスを保存する新規ファイル |

検索では `-title`、`-author`、`-free-text` の少なくとも1つを指定する。
ISBN参照では `-isbn` を指定し、検索条件、`-exclude`、`-limit`、`-cursor` と併用しない。
複数の検索条件を指定した場合の組み合わせと `-exclude` の対象項目は
[検索条件](../docs/pkg/madb/spec.md#6-検索条件)、ISBNの検証は
[ISBN参照](../docs/pkg/madb/spec.md#7-isbn参照)を参照する。

## 検索例

複数のISBNを入力順に参照する。

```text
go run ./examples/demo-madb.go -isbn "978-4-08-846636-1" -isbn "9784990524302"
```

著者名で検索する。

```text
go run ./examples/demo-madb.go -author "佐々木倫子"
```

主要な書誌項目に「うる星」と「高橋留美子」を含み、「復刻box」または
「愛蔵版」を含まない単行本を検索する。

```text
go run ./examples/demo-madb.go -free-text "うる星 高橋留美子" -exclude "復刻box 愛蔵版"
```

## 出力

検索結果は `madb.SearchBooksResult`、ISBN参照結果は `madb.ISBNLookupResult` のJSONとして
標準出力へ出す。ISBN参照の `items` は入力と同じ順序・件数になり、該当なしの項目も
`books: []` として残る。
警告、エラー、診断情報は標準エラー出力へ出し、標準出力へJSON以外を混在させない。

1冊分の `normalized` と `sources` は次の形になる。

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

`normalized` は表記を共通形式にそろえた通常利用向けの書誌情報、`sources` は
その根拠となったMADBの値を表す。項目ごとの変換規則と欠落値の扱いは
[MADBパッケージ仕様](../docs/pkg/madb/spec.md#5-結果変換)を参照する。

終了コードは次のとおり。

| 終了コード | 状態 |
| --- | --- |
| `0` | 検索またはISBN参照とJSON出力に成功 |
| `1` | 検索、ISBN参照、保存、JSON出力のいずれかに失敗 |
| `2` | 必須条件がない、またはCLI引数が不正 |

## 次ページの取得

結果の `next_cursor` に値がある場合は、次回の `-cursor` へ指定する。
`-title`、`-author`、`-free-text`、`-exclude`、`-limit` は、初回に
指定したものをすべて同じ値で再指定する。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5 -cursor "<next_cursor>"
```

カーソルと検索条件または取得件数が一致しない場合は入力エラーとなる。
カーソルの詳細は [MADBパッケージ仕様](../docs/pkg/madb/spec.md#8-limitとページング)を
参照する。

## 変換前レスポンスの保存

MADBから受信した変換前のSPARQL Results JSONを確認する場合は、存在しないファイルを
`-raw-output` へ指定する。

```text
go run ./examples/demo-madb.go -title "動物のおしゃべり" -limit 5 -raw-output __madb-result.json
```

- 標準出力には共通書籍モデルへ変換した検索結果だけを出す
- `-raw-output` のファイルには、同じ検索で受信した本文を変更せず保存する
- 既存ファイルは上書きしない
- 書き込みまたはファイルを閉じる処理に失敗した場合は、不完全な新規ファイルを削除する
- 成功応答のJSON解析または共通書籍モデルへの変換に失敗した場合も、読み込み済みの
  変換前レスポンスを保存する
- 通信失敗、成功以外のHTTP応答、本文の読み込み失敗、4 MiBの上限超過では保存しない

`__` で始まるファイルはローカル確認用であり、このリポジトリではGit管理対象外となる。
`SearchBooksWithRawResponse`、`LookupBooksByISBNWithRawResponse` と本文上限の契約は
[MADBパッケージ仕様](../docs/pkg/madb/spec.md#3-パッケージとclient)と
[HTTP仕様](../docs/pkg/madb/spec.md#10-http)を参照する。
