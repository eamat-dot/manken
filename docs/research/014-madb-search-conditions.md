# MADB検索条件の拡張調査

## 1. 目的

MADBの漫画単行本検索について、次の実現方法を調査する。

- タイトル内の空白区切り語をAND検索する
- ISBNで検索する
- 著者名で検索する
- タイトル、著者、版表示、判型などを組み合わせて絞り込む
- 指定語を含む結果を除外する

調査日は2026年8月1日。実装と確定仕様はこの文書へ含めない。

## 2. 現行実装

`madb/query.go` は `schema:name` を対象とする `simple_query_string` を使用し、
`Title` 全体を二重引用符で囲んでいる。検索条件は `Title` だけであり、
`Title` は必須である。カーソルは整形後のタイトルとLimitを保持している。

`simple_query_string` は複数語の論理条件を確実に指定する用途には適さない。
MADBの環境では `うる星 + 復刻box` の `+` がANDとして機能せず、
片方だけに一致する多数の結果を返した。

## 3. 公式検索画面との比較

公式検索画面は、マンガ単行本の詳細検索で次の項目を提供している。

- フリーワード
- タイトル
- 巻・号
- 作者名
- 発行者名
- レーベル
- 本の形状など
- キーワード・タグ
- 備考

画面は公開SPARQLエンドポイントではなく、サイト内部の `/es-api/` へ
`title`、`author`、`shape` などを別パラメーターとして送信している。この内部APIは
公開APIとして案内されていないため、ライブラリから直接使用しない。

公式画面で `title=うる星 復刻box` を指定すると4件を返し、全件のタイトルに
両方の語が含まれた。

## 4. 公開SPARQLでの確認結果

### 4.1 AND検索

MADB Labは、Neptune全文検索の `query_string` と `AND` を使用する例を
公式に掲載している。

```text
queryType: query_string
query:     "うる星" AND "復刻box"
field:     schema:name
```

この条件を公開SPARQLエンドポイントで実行すると、公式検索画面と同じ4件を返した。
入力を空白で分割し、各語を引用句へ変換して `AND` で結ぶ方法を採用できる。

### 4.2 NOT検索

次の条件は、`うる星` に一致する結果から `復刻box` を含むタイトルを除外した。

```text
"うる星" AND NOT "復刻box"
```

`MINUS` と複数の全文検索 `SERVICE` を使う方法でも除外できたが、公式例に沿う
`query_string` の `NOT` の方がクエリを単純にできる。

利用者の入力を全文検索構文としてそのまま渡さず、ライブラリが各語を引用して
`AND` と `NOT` を組み立てる必要がある。

### 4.3 ISBN検索

ISBNは `schema:isbn` の文字列として格納されている。`M190399` では
チェックディジットが不正な `4088466361` と、正しい `9784088466361` の両方を
確認した。後者から変換した正しいISBN-10は `4088466365` である。ハイフンを含む値は
今回の確認では見つからず、空白を含む例は `9784778031404 (set)` のような
ISBNではない付記付き値だった。

入力からハイフンと空白を除き、チェックディジットを検証したISBNだけを検索する。
ISBN-10とISBN-13の対応値を生成できる場合は両方を `VALUES` へ指定する。
これにより、MADBが片方だけを持つ場合も検索できる。付記付きの不正値は一致させない。

### 4.4 著者名検索

著者は次の2経路に存在する。

- 単行本の `schema:creator` 文字列
- 単行本の `dcterms:creator` が参照するAgentの `rdfs:label`

2経路を `UNION` で結ぶ検索により、creator文字列を持たずAgent参照だけを持つ
`M830542` を `KotzDean` で取得できた。著者名検索は片方だけを対象にできない。

### 4.5 フリーワード検索

フィールドを省略する方法と `field "*"` は、MADBの公開エンドポイントで
`all shards failed` になった。検索対象は明示的に列挙する必要がある。

単行本リソース上の複数フィールドを列挙した `query_string` では、
`"うる星" AND "高橋留美子" AND "新装版"` がタイトル、creator、版表示を
またいで一致した。初期対象の候補は次のとおり。

- `schema:name`
- `schema:alternativeHeadline`
- `ma:seriesName`
- `schema:volumeNumber`
- `schema:creator`
- `schema:publisher`
- `schema:brand`
- `schema:version`
- `schema:isbn`
- `schema:description`
- `schema:keywords`
- `schema:size`

Agentの `rdfs:label` は別リソースにある。各語について単行本フィールドとAgent名を
`UNION` し、その結果をAND結合する方法は動作したが、2語でも約23秒かかった。
通常のフリーワード検索へこの経路を常時含めると、MADBの60秒制限へ近づきやすい。

そのため、フリーワードは単行本リソース上の項目を対象とし、Agent参照だけの著者を
漏れなく検索する場合は専用の著者名条件を使用する。タイトル、著者、版表示または
判型で絞る場合は、`Title`、`Author`、`FreeText` を同時指定してAND結合する。

## 5. 当時推奨した公開条件

次の案は2026年8月1日時点の調査結果であり、TODO020でISBNを専用の
`LookupBooksByISBN` へ分離したため、現在の公開契約ではない。

既存の `SearchBooksRequest` へ、次の文字列条件を段階的に追加する。

```go
type SearchBooksRequest struct {
	Title        string `json:"title"`
	ISBN         string `json:"isbn"`
	Author       string `json:"author"`
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}
```

- `Title`、`Author`、`FreeText` はUnicode空白で語に分割し、各語をAND結合する
- 異なる正条件もAND結合する
- `ISBN` は1つのISBNを受け取り、構造化条件として照合する
- `ExcludedText` はフリーワードと同じ単行本フィールドから、各語に一致する本を除外する
- `ExcludedText` だけの検索は禁止し、正条件を1つ以上必須とする
- 全文検索の演算子を利用者入力として公開しない

既存のタイトルだけの呼び出しはソース互換を維持する。`Title` は単独では必須でなくし、
いずれかの正条件を必須にする。カーソルは全検索条件の正規化済み表現とLimitへ結び付け、
形式をバージョン2へ更新する。カーソルは不透明な一時値のため、バージョン1を継続利用
できるようにする互換処理は追加しない。

## 6. リスクと制約

- `query_string` は厳格な構文なので、引用符、バックスラッシュ、制御文字を
  全文検索層とSPARQL層の両方でエスケープする必要がある
- 検索語や全文検索フィールドを増やすほど、MADBの推定実行時間60秒制限へ近づく
- フリーワードの対象フィールドは将来のMADBスキーマ変更に追随する必要がある
- フリーワードだけでは、Agent参照にしかない著者名を検索できない
- ISBNの付記付き不正値は、正しいISBN入力から検索できない
- MADBは同一ISBNを複数リソースへ持つ場合があるため、ISBN検索も複数件を返し得る

## 7. 複数ISBN参照の追加調査

2026年8月3日に、複数のISBN候補を1つの `VALUES ?matchedISBN` へまとめ、
一致したリソースとISBNを同時に取得できることを実サービスで確認した。
ISBN `9784990524302` は `M409358`、`M409359`、`M409360` の3リソースに一致した。
このため、1つの書籍へ値を統合せず、リソースURI昇順の個別結果として扱う。

1回の問い合わせ件数を変えた実測は次のとおり。時間と応答サイズは調査時点の
単発測定であり、サービス性能の保証値ではない。

| 入力ISBN数 | 応答時間 | 応答サイズ |
| ---: | ---: | ---: |
| 1 | 約0.23秒 | 約5.9 KiB |
| 10 | 約0.20秒 | 約82.9 KiB |
| 50 | 約0.27秒 | 約285 KiB |
| 100 | 約0.29から0.36秒 | 約317から391 KiB |
| 250 | 約0.35秒 | 約684 KiB |
| 500 | 約0.46秒 | 約1.18 MiB |
| 1,000 | 約0.96秒 | 約2.23 MiB |

1,000件でも成功したが、MADBには複数ISBN指定件数の公式上限が確認できず、
入力展開後のクエリと応答には書誌件数による変動がある。通常の応答時間と4 MiB本文上限に
余裕を残し、openBDの将来上限1,000件とも区別するため、MADBの公開上限は500件とした。

## 8. 参照先

- [MADB Lab: データセットおよびSPARQLクエリサービスの利用方法](https://mediag.bunka.go.jp/madb_lab/lod/howto/)
- [メディア芸術データベースについて](https://mediaarts-db.artmuseums.go.jp/about)
- [Amazon Neptune: Full-text search parameters](https://docs.aws.amazon.com/neptune/latest/userguide/full-text-search-parameters.html)
- [OpenSearch: Simple query string](https://docs.opensearch.org/latest/query-dsl/full-text/simple-query-string/)
