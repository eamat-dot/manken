# MADB・NDLサーチのCLI examples

`examples` には、`madb` と `ndl` パッケージを実サービスへ接続して試すためのCLI exampleがあります。どちらもAPIキーやアカウント登録は不要ですが、実行時にはインターネット接続が必要です。

コマンドは、cloneした `manken` repositoryのルートで実行してください。

```text
go run ./examples/madb -title "動物のおしゃべり" -limit 5
go run ./examples/ndl -title "動物のお医者さん" -limit 5
```

検索・ISBN参照の結果は、整形したJSONとして標準出力へ出します。エラーや診断情報は標準エラーへ出すため、標準出力をファイルや別のコマンドへ渡せます。

## MADB

`examples/madb` は、メディア芸術データベース（MADB）の漫画単行本を検索し、複数のISBNをまとめて参照できます。

### 検索する

検索では、`-title`、`-author`、`-publisher`、`-query`、`-from`、`-to` の少なくとも1つを指定します。

```text
go run ./examples/madb -author "佐々木倫子"
go run ./examples/madb -publisher "白泉社" -from 2020 -to 2024
go run ./examples/madb -query "うる星 高橋留美子" -exclude "復刻box 愛蔵版"
```

### ISBNで参照する

ISBN-10またはISBN-13を1件以上500件以下で指定します。複数指定した場合も、結果の `items` は入力と同じ順序・件数になります。オプションはISBNより前に指定してください。

```text
go run ./examples/madb 9784088466361 9784990524302
```

検索用オプションとISBNは併用できません。検索条件の組み合わせ、ISBNの検証、カーソルの詳細は[MADBパッケージ仕様](../docs/pkg/madb/spec.md)を参照してください。

### 主なオプション

| オプション    | 内容                                                   |
| ------------- | ------------------------------------------------------ |
| `-title`      | タイトルに含める検索語                                 |
| `-author`     | 著者名に含める検索語                                   |
| `-publisher`  | 出版社名に含める検索語                                 |
| `-query`      | 主要な書誌項目を横断して検索する語                     |
| `-exclude`    | 主要な書誌項目に指定語を含む結果を除外                 |
| `-from`       | 出版時期の開始。`YYYY`、`YYYY-MM`、`YYYY-MM-DD`        |
| `-to`         | 出版時期の終了。`-from` と併用する場合は同じ精度で指定 |
| `-limit`      | 取得件数。1～100。`0` は既定値の20件                   |
| `-cursor`     | 前回の結果に含まれる `next_cursor`                     |
| `-raw-output` | 変換前のMADBレスポンスを保存する新規ファイル           |

全オプションは次のコマンドで確認できます。このコマンドは実サービスへ接続しません。

```text
go run ./examples/madb -h
```

## NDLサーチ

`examples/ndl` は、国立国会図書館サーチの全国書誌から書籍を検索し、1件のISBNを参照できます。検索は既定で `NDC 726.1` と `NDLC Y84` の分類を使い、漫画に絞り込みます。

### 検索する

検索では、`-title`、`-author`、`-publisher`、`-query`、`-from`、`-to`、`-subject`、`-description` の少なくとも1つを指定します。

```text
go run ./examples/ndl -description "ハムテル" -limit 5
go run ./examples/ndl -publisher "白泉社" -from 2020 -to 2024
```

古い漫画など、NDLC Y84が付与されていない書誌も探す場合は、NDLCによる絞り込みだけを無効にできます。

```text
go run ./examples/ndl -title "動物のお医者さん" -manga-ndlc=false -limit 5
```

### ISBNで参照する

ISBN-10またはISBN-13を1件指定します。ISBN参照では漫画分類による絞り込みを行いません。

```text
go run ./examples/ndl 9784088466361
```

検索用オプションとISBNは併用できません。検索条件、分類による絞り込み、ISBN参照の詳細は[NDLサーチパッケージ仕様](../docs/pkg/ndl/spec.md)を参照してください。

### 主なオプション

| オプション     | 内容                                                     |
| -------------- | -------------------------------------------------------- |
| `-title`       | タイトルに含める検索語                                   |
| `-author`      | 著者名に含める検索語                                     |
| `-publisher`   | 出版社名に含める検索語                                   |
| `-query`       | 複数の書誌項目を対象にする検索語                         |
| `-from`        | 出版年月日の開始。`YYYY`、`YYYY-MM`、`YYYY-MM-DD`        |
| `-to`          | 出版年月日の終了。`-from` と併用する場合は同じ精度で指定 |
| `-subject`     | 件名に含める検索語                                       |
| `-description` | 内容記述に含める検索語                                   |
| `-manga-ndc`   | NDC 726.1の漫画分類フィルタ。既定値は `true`             |
| `-manga-ndlc`  | NDLC Y84の漫画分類フィルタ。既定値は `true`              |
| `-limit`       | 取得件数。1～500。`0` は既定値の20件                     |
| `-cursor`      | 前回の結果に含まれる `next_cursor`                       |
| `-raw-output`  | 変換前のNDLサーチXMLを保存する新規ファイル               |

全オプションは次のコマンドで確認できます。このコマンドは実サービスへ接続しません。

```text
go run ./examples/ndl -h
```

## 変換前レスポンスを保存する

`-raw-output` に存在しないファイルを指定すると、同じ検索またはISBN参照で受信した変換前レスポンスを保存します。

```text
go run ./examples/madb -title "動物のおしゃべり" -limit 5 -raw-output madb-raw.json
go run ./examples/ndl -title "動物のお医者さん" -limit 5 -raw-output ndl-raw.xml
```

- MADBはSPARQL Results JSON、NDLサーチはXMLを保存します。
- 既存ファイルは上書きしません。
- 標準出力には変換済み結果のJSONだけを出します。
- レスポンスの解析や `Book` への変換に失敗した場合も、受信済みの本文を保存できる場合があります。

Raw response APIの詳細と本文サイズの上限は、[MADBガイド](../docs/pkg/madb/guide.md)と[NDLサーチガイド](../docs/pkg/ndl/guide.md)から各パッケージ仕様を確認してください。

## 出典・クレジット情報と利用条件

MADBとNDLサーチでは、成功した検索・ISBN参照結果のJSONに `attributions` が含まれます。MADBは取得データの出典情報、NDLサーチはAPI利用表示と全国書誌情報の出典・ライセンス情報を返します。同じ定型情報は、ライブラリから `madb.Attributions()` または `ndl.Attributions()` を呼び出して通信なしで取得することもできます。

`attributions` は表示・保存に利用できる情報ですが、それだけで各サービスの利用条件への適合を保証するものではありません。取得データを利用する際は、用途、加工、保存、表示方法などに適用される最新の公式条件を確認してください。

- [メディア芸術データベース利用規約](https://mediaarts-db.artmuseums.go.jp/user_terms)
- [NDLサーチ APIのご利用について](https://ndlsearch.ndl.go.jp/help/api)
- [API提供対象データプロバイダ一覧](https://ndlsearch.ndl.go.jp/help/api/provider)
