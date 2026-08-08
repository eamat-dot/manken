# CLIデモ

`examples` には、各書誌情報取得元を実サービスで確認するためのCLIデモを置く。
汎用の検索アプリケーションではない。

## MADB

`madb/main.go` は、`madb` パッケージでMADBの実サービスを検索・ISBN参照し、結果をJSONで
確認するためのCLIである。

### 基本的な使い方

リポジトリのルートで次を実行する。

```text
go run ./examples/madb -title "動物のおしゃべり" -limit 5
```

タイトルに「動物のおしゃべり」を含む単行本を5件まで検索し、共通書籍モデルへ
変換した結果を標準出力へJSONで出す。

### オプション

| オプション | 内容 |
| --- | --- |
| `-title` | タイトルに含める検索語 |
| `-author` | 著者名に含める検索語 |
| `-free-text` | 主要な書誌項目を横断して検索する語 |
| `-exclude` | 主要な書誌項目に含まれる場合、結果から除外する語 |
| `-limit` | 取得件数。1から100まで。0は既定値の20件 |
| `-cursor` | 前回の結果に含まれる `next_cursor` |
| `-raw-output` | MADBから受信した変換前レスポンスを保存する新規ファイル |

検索では `-title`、`-author`、`-free-text` の少なくとも1つを指定する。
ISBN参照では ISBN-10またはISBN-13を位置引数で1件以上500件以下指定し、検索条件、
`-exclude`、`-limit`、`-cursor` と併用しない。
オプションはISBNより前に指定する。
複数の検索条件を指定した場合の組み合わせと `-exclude` の対象項目は
[検索条件](../docs/pkg/madb/spec.md#6-検索条件)、ISBNの検証は
[ISBN参照](../docs/pkg/madb/spec.md#7-isbn参照)を参照する。

### 実行例

複数のISBNを入力順に参照する。

```text
go run ./examples/madb 9784088466361 9784990524302
```

著者名で検索する。

```text
go run ./examples/madb -author "佐々木倫子"
```

主要な書誌項目に「うる星」と「高橋留美子」を含み、「復刻box」または
「愛蔵版」を含まない単行本を検索する。

```text
go run ./examples/madb -free-text "うる星 高橋留美子" -exclude "復刻box 愛蔵版"
```

### 出力

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
    "contributors": [{
      "name": "佐々木倫子",
      "roles": ["author"]
    }],
    "publishers": ["白泉社"],
    "imprints": ["白泉社文庫"],
    "identifiers": [{"type": "isbn10", "value": "4592881486"}],
    "dates": [{"type": "published", "value": "1996-06-19"}]
  },
  "sources": [{
    "source": "madb",
    "id": "M292129",
    "url": "https://mediaarts-db.artmuseums.go.jp/id/M292129"
  }]
}
```

`normalized` は通常利用する共通書誌情報であり、MADB応答との差分や変換履歴を表すものではない。
`sources` は取得元を参照するための情報を表す。`authors` は表示や簡易利用向けの
著者名一覧、`contributors` は人物ごとの読みと役割を保持できる詳細情報である。
読みや共通役割を取得できない人物は、`reading` や `roles` を持たない
`contributor` として出力される場合がある。変換前のMADB応答を確認する場合は
`-raw-output` を使う。項目ごとの変換規則と欠落値の扱いは
[MADBパッケージ仕様](../docs/pkg/madb/spec.md#5-結果変換)を参照する。

終了コードは次のとおり。

| 終了コード | 状態 |
| --- | --- |
| `0` | 検索またはISBN参照とJSON出力に成功 |
| `1` | 検索、ISBN参照、保存、JSON出力のいずれかに失敗 |
| `2` | 必須条件がない、またはCLI引数が不正 |

### 次ページの取得

結果の `next_cursor` に値がある場合は、次回の `-cursor` へ指定する。
`-title`、`-author`、`-free-text`、`-exclude`、`-limit` は、初回に
指定したものをすべて同じ値で再指定する。

```text
go run ./examples/madb -title "動物のおしゃべり" -limit 5 -cursor "<next_cursor>"
```

カーソルと検索条件または取得件数が一致しない場合は入力エラーとなる。
カーソルの詳細は [MADBパッケージ仕様](../docs/pkg/madb/spec.md#8-limitとページング)を
参照する。

### 変換前レスポンスの保存

MADBから受信した変換前のSPARQL Results JSONを確認する場合は、存在しないファイルを
`-raw-output` へ指定する。

```text
go run ./examples/madb -title "動物のおしゃべり" -limit 5 -raw-output madb-raw.json
```

- 標準出力には共通書籍モデルへ変換した検索結果だけを出す
- `-raw-output` のファイルには、同じ検索で受信した本文を変更せず保存する
- 既存ファイルは上書きしない
- 書き込みまたはファイルを閉じる処理に失敗した場合は、不完全な新規ファイルを削除する
- 成功応答のJSON解析または共通書籍モデルへの変換に失敗した場合も、読み込み済みの
  変換前レスポンスを保存する
- 通信失敗、成功以外のHTTP応答、本文の読み込み失敗、4 MiBの上限超過では保存しない

`SearchBooksWithRawResponse`、`LookupBooksByISBNWithRawResponse` と本文上限の仕様は
[MADBパッケージ仕様](../docs/pkg/madb/spec.md#3-パッケージとclient)と
[HTTP仕様](../docs/pkg/madb/spec.md#10-http)を参照する。

## openBD

`openbd/main.go` は、`openbd` パッケージでopenBDから複数ISBNの書誌情報を参照し、結果をJSONで
確認するためのCLIである。

### 基本的な使い方

リポジトリのルートで次を実行する。

```text
go run ./examples/openbd 9784098515172 4592730933
```

指定したISBNを入力順に参照し、共通書籍モデルへ変換した結果を標準出力へJSONで出す。
ISBNはISBN-10またはISBN-13を1件以上1,000件以下指定する。

### オプション

| オプション | 内容 |
| --- | --- |
| `-raw-output` | openBDから受信した変換前レスポンスを保存する新規ファイル |

オプションはISBNより前に指定する。

```text
go run ./examples/openbd -raw-output openbd-raw.json 9784098515172 4592730933
```

### 出力

標準出力には `openbd.ISBNLookupResult` のJSONだけを出す。`items` は入力と同じ順序・件数になり、
該当なしのISBNも `books: []` として残る。警告、エラー、診断情報は標準エラー出力へ出す。

終了コードは次のとおり。

| 終了コード | 状態 |
| --- | --- |
| `0` | ISBN参照とJSON出力に成功 |
| `1` | ISBN参照、保存、JSON出力のいずれかに失敗 |
| `2` | ISBNが指定されていない、またはCLI引数が不正 |

### 変換前レスポンスの保存

`-raw-output` には、存在しないファイルを指定する。

- 標準出力には共通書籍モデルへ変換した結果だけを出す
- 指定ファイルには、同じ参照で受信した本文を変更せず保存する
- 既存ファイルは上書きしない
- 書き込みまたはファイルを閉じる処理に失敗した場合は、不完全な新規ファイルを削除する
- 成功応答のJSON解析または共通書籍モデルへの変換に失敗した場合も、読み込み済みの
  変換前レスポンスを保存する
- 通信失敗、成功以外のHTTP応答、本文の読み込み失敗、64 MiBの上限超過では保存しない

`LookupBooksByISBNWithRawResponse` と本文上限の仕様は
[openBDパッケージ仕様](../docs/pkg/openbd/spec.md#3-パッケージとclient)と
[HTTP仕様](../docs/pkg/openbd/spec.md#7-http)を参照する。

## Google Books

`googlebooks/main.go` は、`googlebooks` パッケージでGoogle Booksを検索・ISBN参照し、結果をJSONで
確認するためのCLIである。実行前にGoogle Books APIキーを環境変数へ設定する。

```powershell
$env:GOOGLE_BOOKS_API_KEY = "your-key"
```

### 基本的な使い方

リポジトリのルートで次を実行する。

```text
go run ./examples/googlebooks -title "動物のお医者さん" -limit 5
```

タイトルに「動物のお医者さん」を含む書籍候補を5件まで検索し、共通書籍モデルへ変換した結果を
標準出力へJSONで出す。Google Booksは漫画専用の取得元ではない。

### オプション

| オプション | 内容 |
| --- | --- |
| `-title` | タイトルに含める検索語 |
| `-author` | 著者名に含める検索語 |
| `-free-text` | 複数の書誌項目を対象にする検索語 |
| `-exclude` | Google Booksのfull-text検索対象から除外する語。複数語は空白で区切る。タイトル以外の説明文などに含まれる場合も除外されることがある |
| `-limit` | 検索件数。1から40まで。0は既定値の20件 |
| `-cursor` | 前回の結果に含まれる `next_cursor` |
| `-raw-output` | 検索またはISBN参照でGoogle Booksから受信した変換前レスポンスを保存する新規ファイル |

検索では `-title`、`-author`、`-free-text` の少なくとも1つを指定する。ISBN参照ではISBN-10または
ISBN-13を位置引数で1件だけ指定し、検索条件、`-limit`、`-cursor`とは併用しない。`-raw-output` は
ISBN参照でも使用できる。オプションはISBNより前に指定する。

### 実行例

著者名で検索する。

```text
go run ./examples/googlebooks -author "佐々木倫子"
```

複数の書誌項目を対象に検索する。

```text
go run ./examples/googlebooks -free-text "日本 漫画"
```

ISBNを1件参照する。

```text
go run ./examples/googlebooks 4088466365
```

変換前レスポンスも保存する。

```text
go run ./examples/googlebooks -raw-output googlebooks-isbn-raw.json 4088466365
```

次ページを取得するには、前回の `next_cursor` と同じ検索条件・Limitを指定する。

```text
go run ./examples/googlebooks -title "動物のお医者さん" -limit 5 -cursor "<next_cursor>"
```

### 出力と変換前レスポンスの保存

標準出力には検索時は `googlebooks.SearchBooksResult`、ISBN参照時は
`googlebooks.ISBNLookupResult` のJSONだけを出す。警告、エラー、診断情報は標準エラー出力へ出す。
ISBN参照の `items` は1要素で、`RequestedISBN` は入力文字列を保持し、該当なしも `books: []` として残る。

`-raw-output` に存在しないファイルを指定すると、検索またはISBN参照で受信した1回の成功レスポンス本文を
変更せず保存する。既存ファイルは上書きしない。JSON解析または共通書籍モデルへの変換に失敗した場合も、
読み込み済みの本文を保存する。通信失敗、成功以外のHTTP応答、本文読込失敗、16 MiB超過では保存しない。

検索条件、ISBN参照、変換項目、利用条件は
[Google Booksパッケージ仕様](../docs/pkg/googlebooks/spec.md)と
[Google Booksガイド](../docs/pkg/googlebooks/guide.md)を参照する。

## 楽天Books

`rakutenbooks/main.go` は、`rakutenbooks` パッケージで楽天Booksの紙書籍を検索・ISBN参照し、
結果をJSONで確認するためのCLIである。実行前にApplication IDとAccess Keyを環境変数へ設定する。
Affiliate IDは任意である。

```text
RAKUTEN_APP_ID=<Application ID>
RAKUTEN_ACCESS_KEY=<Access Key>
RAKUTEN_AFFILIATE_ID=<Affiliate ID。任意>
```

### 基本的な使い方

一般コミックをタイトル検索する。

```text
go run ./examples/rakutenbooks -title "動物のお医者さん" -limit 5
```

### オプション

| オプション | 内容 |
| --- | --- |
| `-title` | タイトルに含める検索語 |
| `-author` | 著者名に含める検索語 |
| `-genre` | 漫画区分。`general`、`bl`、`tl`。既定値は `general` |
| `-size` | 楽天Booksの商品形態。`0`は絞り込みなし、`1`から`10`は公式分類 |
| `-limit` | 検索件数。1から30まで。0は既定値の20件 |
| `-cursor` | 前回の結果に含まれる `next_cursor` |
| `-raw-output` | 検索またはISBN参照で楽天Booksから受信した変換前レスポンスを保存する新規ファイル |

検索では `-title` または `-author` の少なくとも1つを指定する。ISBN参照ではISBN-10または
ISBN-13を位置引数で1件だけ指定し、検索条件、`-limit`、`-cursor`とは併用しない。
楽天Booksではフリーワード検索と除外語検索は提供しない。

### 実行例

著者名で検索する。

```text
go run ./examples/rakutenbooks -author "佐々木倫子"
```

BLコミックを検索する。

```text
go run ./examples/rakutenbooks -genre bl -title "セブンティーンシロップス"
```

TLコミックを検索する。

```text
go run ./examples/rakutenbooks -genre tl -title "メロすぎ朔椰"
```

文庫を検索する。

```text
go run ./examples/rakutenbooks -size 2 -title "動物のお医者さん"
```

ISBNを1件参照する。

```text
go run ./examples/rakutenbooks 9784758088732
```

次ページを取得するには、前回の `next_cursor` と同じタイトル・著者・Limit・漫画区分・商品形態を指定する。

```text
go run ./examples/rakutenbooks -genre general -title "動物のお医者さん" -limit 5 -cursor "<next_cursor>"
```

### 出力と変換前レスポンスの保存

標準出力には検索時は `rakutenbooks.SearchBooksResult`、ISBN参照時は
`rakutenbooks.ISBNLookupResult` のJSONだけを出す。警告、エラー、診断情報は標準エラー出力へ出す。

変換済み結果では、`itemPrice` を取得時点の税込JPY価格として `normalized.prices` に出力する。
Affiliate IDを設定した場合は、通常商品URLとは別に `sources[].affiliate_url` へ楽天の
アフィリエイトURLを出力する。

`-raw-output` に存在しないファイルを指定すると、検索またはISBN参照で受信した1回の成功レスポンス本文を
変更せず保存する。既存ファイルは上書きしない。JSON解析または共通書籍モデルへの変換に失敗した場合も、
読み込み済みの本文を保存する。通信失敗、成功以外のHTTP応答、本文読込失敗、16 MiB超過では保存しない。

楽天ウェブサービスはApplication ID単位のリクエスト頻度、クレジット表示、取得データの保存・更新に
利用条件がある。CLIデモ自体はレート制御やキャッシュを行わない。
検索条件、漫画区分、ISBN参照、変換項目、利用条件は
[楽天Booksパッケージ仕様](../docs/pkg/rakutenbooks/spec.md)と
[楽天Booksガイド](../docs/pkg/rakutenbooks/guide.md)を参照する。
