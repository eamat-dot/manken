# madb パッケージ仕様

## 1. 目的

`madb` パッケージは、メディア芸術データベース（MADB）からマンガ単行本を
タイトルで検索し、`api` パッケージで定義する共通モデルへ変換する。

本仕様書は、初期実装で使用するMADB固有の検索、項目対応、HTTP処理、
ページング、エラー分類を定義する。

共通モデルとMCPとの責務境界は、[manken API仕様](../../spec.md)で定義する。

## 2. 参照資料

2026年7月29日と30日に次の公式資料と実サービスを確認した。

- [MADBの概要とSPARQLクエリサービス](https://mediaarts-db.artmuseums.go.jp/about)
- [MADBクラス定義](https://mediaarts-db.artmuseums.go.jp/data/class/)
- [データセットおよびSPARQLクエリサービスの利用方法](https://mediag.bunka.go.jp/madb_lab/lod/howto/)
- [SPARQLクエリサービスとサンプル](https://mediag.bunka.go.jp/madb_lab/lod/sparql/)
- [MADBメタデータスキーマ仕様書v1.2](https://github.com/mediaarts-db/dataset/blob/main/doc/MADB%E3%83%A1%E3%82%BF%E3%83%87%E3%83%BC%E3%82%BF%E3%82%B9%E3%82%AD%E3%83%BC%E3%83%9E%E4%BB%95%E6%A7%98%E6%9B%B8.pdf)
- [MADB Lab利用規約](https://mediag.bunka.go.jp/madb_lab/user_terms/)
- [Amazon Neptune全文検索パラメーター](https://docs.aws.amazon.com/neptune/latest/userguide/full-text-search-parameters.html)

公式GitHubのスキーマはv1.2が最新である。実装時は、現在の名前空間である
`https://mediaarts-db.artmuseums.go.jp/` を使用する。

## 3. パッケージとClient

import pathは次のとおり。

```text
github.com/eamat-dot/manken/madb
```

公開APIは次の形とする。

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)

func WithEndpoint(endpoint string) Option

func (client *Client) SearchBooks(
	ctx context.Context,
	request madb.SearchBooksRequest,
) (madb.SearchBooksResult, error)

func (client *Client) SearchBooksWithRawResponse(
	ctx context.Context,
	request madb.SearchBooksRequest,
) (madb.SearchBooksResult, []byte, error)
```

共通型と共通エラーの実体は `api` パッケージに定義する。`madb` は通常利用に
必要な型と定数をエイリアスとして公開し、利用側は `madb` だけをimportして
検索、結果の参照、分類済みエラーの判定を行える。

- `Option` は外部から独自実装できない不透明な設定型とする
- `Client` は設定だけを保持し、検索条件や結果を保持しない
- 検索ごとに新しい `http.Request` を作成する
- 呼び出し側の `http.Client` と `Transport` を変更しない
- `Client` は複数goroutineから安全に利用できるものとする
- ログ出力、goroutineの開始、自動リトライ、キャッシュを行わない
- `httpClient` が `nil` の場合は、タイムアウト60秒のクライアントを使用する

`SearchBooksWithRawResponse` は、変換済み検索結果に加えて、MADBから受信した
成功レスポンス本文を変更せず `[]byte` で返す。本文は呼び出し元が所有し、
変更しても `Client` や変換済み検索結果へ影響しない。

2xx応答の本文を上限内で読み込んだ後、JSON解析またはbindingから検索結果への
変換に失敗した場合は、読み込み済み本文とエラーを同時に返す。本文読込の失敗、
本文上限の超過、2xx以外のHTTP応答、通信失敗では本文を返さない。
`SearchBooks` は同じ検索処理を使用し、受信本文を呼び出し元へ返さない。

既定エンドポイントは次のとおり。

```text
https://mediaarts-db.artmuseums.go.jp/sparql
```

`WithEndpoint` は絶対URLを受け付ける。スキームは `http` または `https`、
ホストは必須とし、ユーザー情報、クエリ、フラグメントを受け付けない。

## 4. 検索対象と項目対応

マンガ単行本は次のクラスで識別する。

```text
https://mediaarts-db.artmuseums.go.jp/data/class#MangaBook
```

クラス識別子は `cm101` である。マンガ単行本シリーズを示す
`class:MangaBookSeries` は検索結果に含めない。

| MADBの取得元 | `madb` の非公開型 | `madb.Book` |
| --- | --- | --- |
| `schema:identifier` | `ID` | `ID` |
| 言語タグなしの `schema:name` | `Titles` | `Titles` |
| 言語タグなしの `schema:alternativeHeadline` | `Subtitles` | `Subtitles` |
| 言語タグなしの `ma:seriesName` | `SeriesNames` | `SeriesNames` |
| `schema:isPartOf` が参照するシリーズの言語タグなし `schema:name` | `SeriesNames` | `SeriesNames` |
| 参照先シリーズの `schema:identifier` | `SeriesID` | `SeriesID` |
| 参照先シリーズURI | `SeriesResourceURI` | `SeriesURL` |
| `schema:volumeNumber` | `VolumeNumber` | `VolumeNumber` |
| 言語タグなしの `schema:version` | `Versions` | `EditionStatements` |
| `schema:creator` とcreator Agent | `Creators` | `Authors` |
| `schema:publisher` | `Publishers` | `Publishers` |
| 言語タグなしの `schema:brand` | `Brands` | `Imprints` |
| `schema:isbn` | `ISBNs` | `ISBN10s`、`ISBN13s` |
| `schema:datePublished` | `PublishedDate` | `PublishedDate` |
| 固定値 | 対応なし | `madb.SourceMADB` |
| マンガ単行本URI | `ResourceURI` | `SourceURL` |

リソースURIは次の規則を持つ。

```text
https://mediaarts-db.artmuseums.go.jp/id/ + MADB ID
```

マンガ単行本のMADB IDは `M` と数字で構成される。`ID` には
`schema:identifier` の `M...` を設定し、`SourceURL` にはリソースURIを設定する。

`schema:isPartOf` が参照するリソースは
`class:MangaBookSeries` に限定する。マンガ単行本シリーズのMADB IDは
`C` と数字で構成される。参照先の `schema:identifier` を `SeriesID`、
参照先リソースURIを `SeriesURL` に設定する。

## 5. 結果変換

MADBレスポンスは、まず `madb` パッケージの非公開型へ変換する。
非公開型はMADBの項目構造を保持し、`madb.Book` をそのままレスポンスの
受取先として使用しない。

概念上の非公開型は次の形とする。実装時に型を分割してもよいが、
`Creators` と `Authors` の境界は維持する。

```go
type sourceBook struct {
	ID                string
	Titles            []string
	Subtitles         []string
	SeriesNames       []string
	SeriesID          string
	SeriesResourceURI string
	VolumeNumber      string
	Versions          []string
	Creators          []string
	Brands            []string
	Publishers        []string
	ISBNs             []string
	PublishedDate     string
	ResourceURI       string
}
```

SPARQL bindingを `sourceBook` へ集約した後、`madb` パッケージ内の変換処理で
`madb.Book` を組み立てる。MADB固有の役割除去、ISBN検証、欠落値処理は
この変換処理が担当する。

### 5.1 複数値

`Titles`、`Subtitles`、`SeriesNames`、`EditionStatements`、`Authors`、
`Publishers`、`Imprints`、`ISBN10s`、`ISBN13s` は、値がない場合も
空スライスにする。

- 空文字列を除外する
- 完全に同じ値だけを重複除去する
- Goの文字列昇順で並べ、レスポンス順に依存させない
- 表記揺れ、異体字、読み、役割の違いを同一値と推測しない
- タイトルやシリーズ名から欠落項目を補完しない

実データでは、言語タグなしの値だけでも複数タイトル、副題、シリーズ名が存在する。
単一の正規タイトルを示すプロパティはないため、任意の1件を選ばずすべて返す。

### 5.2 CreatorsからAuthorsへの変換

`schema:creator` は文字列であり、次のような役割を含む。

```text
[著]手塚治虫
[原作]梶原一騎
[作画]...
```

`sourceBook.Creators` には、`dcterms:creator` が参照する全Agentの
`rdfs:label` を格納する。Agent参照がない場合は、言語タグなしの
`schema:creator` を格納する。

`sourceBook` から `madb.Book` へ変換するときに、著者、原作者、作画担当などを
役割で分けず、名前だけを `Authors` へ設定する。値の先頭にある `[著]`、
`[原作]`、`[作画]` などの角括弧部分を役割として取り除く。

値が `[` で始まり `]` を含む間は、先頭から最初の `]` までを繰り返し取り除く。
前後の空白を除いた結果が空の場合は使用しない。役割ごとの分類は
`madb.Book` へ含めない。

Agent参照と `schema:creator` の各役割には、RDF上で1対1の対応がない。
Agent名が1件でも存在する場合は全Agent名を優先するため、解説者などの
非著者役割が `Authors` に含まれる場合がある。初期実装では名前から役割を
推測して除外せず、他の書籍検索APIの責任主体モデルを調査してから規則を再評価する。

### 5.3 publisher

`schema:publisher` は文字列であり、発行、発売、読みなどを含む場合がある。
初期APIでは役割や読みを解析せず、取得値をそのまま返す。

### 5.4 ISBN

`schema:isbn` は0件以上存在する。ASCIIのハイフンと空白を除いた後、
ISBN-10またはISBN-13のチェックディジットを検証する。

- 正しいISBN-10は大文字の `X` を含む10文字として `ISBN10s` へ設定する
- 正しいISBN-13は13桁として `ISBN13s` へ設定する
- 検証に失敗した値は返さない
- 同じ種別が複数ある場合もすべて返す

### 5.5 刊行日と巻数

`schema:datePublished` は `YYYY`、`YYYY-MM`、`YYYY-MM-DD` など、
精度の異なる文字列である。日付として再解釈せず、取得値をそのまま保持する。
空文字列は欠落として扱う。不正に見える値も推測で修正しない。

`schema:volumeNumber` も文字列として保持する。複数の異なる値が返された場合は、
スキーマの0または1件という契約に反するため `invalid_response` とする。

### 5.6 版表示、単行本レーベル、シリーズ

マンガ単行本へ直接記録された言語タグなしの `schema:version` を
`sourceBook.Versions` へ格納し、`EditionStatements` として返す。
版表示は0件以上存在し、複数の異なる値もすべて返す。値を通常版、新装版、
愛蔵版、完全版、文庫版などの独自分類へ変換しない。

マンガ単行本へ直接記録された言語タグなしの `schema:brand` を
`sourceBook.Brands` へ格納し、`Imprints` として返す。`ja-hrkt` などの
言語タグ付きの読みは返さない。レーベルらしくない値が含まれていても、
文字列の内容から除外または修正しない。

`schema:isPartOf` が参照する `class:MangaBookSeries` から、次を取得する。

- 言語タグなしの `schema:name`
- `schema:identifier`
- シリーズリソースURI

シリーズの `schema:name` は、単行本へ直接記録された言語タグなしの
`ma:seriesName` と合わせて `SeriesNames` へ設定する。完全に同じ名前だけを
重複除去する。参照先が欠落する場合、`SeriesID` と `SeriesURL` は空文字列にする。

異なる複数の参照先シリーズ、シリーズID、シリーズURIが返された場合は、
スキーマの0または1件という契約に反するため `invalid_response` とする。
参照先シリーズの `schema:brand` と `schema:version` は単行本の値として返さない。
単行本の版表示またはレーベルが欠落しても、参照先シリーズ、タイトル、
出版社、判型から補完しない。

## 6. タイトル検索

タイトル検索にはAmazon Neptuneの全文検索拡張を使用する。

```text
queryType: simple_query_string
field:     schema:name
```

検索語は全文検索構文として受け付けない。次の順序で文字列を生成する。

1. 前後のUnicode空白を取り除く
2. 全文検索層向けに `\` と `"` をエスケープする
3. 値全体を二重引用符で囲み、フレーズ検索にする
4. SPARQL文字列リテラル向けに引用符、バックスラッシュ、制御文字をエスケープする

引用句を使用することで、確認した検索語では `schema:name` の部分一致と同じ
30件を取得した。`match` は同じ検索語で929件となり、初期APIには広すぎるため
使用しない。引用符、バックスラッシュ、空白、`+`、`-`、`|` を含む入力でも、
クエリ構造を変更させず正常応答を得られることを確認した。

検索結果は必ず `rdf:type class:MangaBook` で絞り込む。大文字小文字、
Unicode正規化、表記揺れはライブラリ側で変換せず、全文検索サービスの解析に従う。

全文検索の既定の最大結果窓は10,000件である。これを超える一致結果の
完全なページングは保証しない。

## 7. Limitとページング

`Limit` の既定値は20、最大値は100とする。

- `0` は既定値20として扱う
- 1から100までは指定値を使用する
- 負数または101以上は `invalid_argument` とする
- 次ページ判定のため、リソースを `Limit + 1` 件取得する

ページングにはMADBリソースURIの昇順によるキーセット方式を使用する。
初回は先頭から取得し、次ページでは直前ページの末尾URIより大きいURIを取得する。
`OFFSET` は、検索中の追加データによって位置がずれるため使用しない。

カーソルは、バージョン、末尾URI、Limit、検索語のSHA-256を含むJSONを
パディングなしBase64 URL形式で符号化する。

- 形式不正、未対応バージョン、URI不正は `invalid_argument` とする
- カーソルのLimitまたは検索語がリクエストと一致しない場合は
  `invalid_argument` とする
- 有効期間は設けない
- カーソルは改ざん防止を目的とした署名を持たない
- MADB更新中の完全なスナップショット一貫性は保証しない

URI順の部分一致検索を2回実行して同一順序になることと、全文検索でも
末尾URIを指定した次ページが重複なく直後のURIから始まることを確認した。

カーソル境界は `STR(?resource)` と末尾URI文字列を比較する。SPARQLのIRI同士を
`>` で直接比較しない。並び順は引き続きリソースURIの昇順とする。

## 8. SPARQLレスポンスの組み立て

ページ対象のリソースURIをサブクエリで先に確定し、そのリソースに対する各項目を
外側のクエリで取得する。外側のbinding数に `LIMIT` を適用しない。

1冊に対する複数bindingはGo側でまとめる。SPARQLの `GROUP_CONCAT` は、
区切り文字を実データと区別できないため使用しない。

SPARQL Results JSONのbindingは、要求した変数が欠落することを正常な状態として
扱う。未知の変数は無視するが、既知の変数の型が契約と異なる場合は
`invalid_response` とする。

## 9. HTTP

既定エンドポイントはGETとPOSTに対応する。初期実装は、検索語をURLへ含めず、
長いクエリでもURL長の制約を受けないPOSTを使用する。

```text
Content-Type: application/x-www-form-urlencoded
Accept: application/sparql-results+json
```

フォームの `query` パラメーターへSPARQLを設定する。正常応答は
`application/sparql-results+json; charset=UTF-8` である。

- 成功レスポンス本文の上限は4 MiBとする
- エラーレスポンス本文の読込上限は64 KiBとする
- 上限超過は `invalid_response` とする
- 成功レスポンス本文は1回だけ読み込み、rawレスポンスと結果変換に共用する
- 外部サービスのエラー本文を公開エラーメッセージへ含めない
- エラー本文を `madb.Error` に保持しない

100件と次ページ判定用1件の項目取得では、705 bindings、約214 KiBの応答を
確認した。

正常応答と400応答で、次のレート制限ヘッダーを確認した。

```text
X-RateLimit-Limit: 300
X-RateLimit-Remaining: ...
```

期間を示す公式説明と `Retry-After` 付き429応答は確認できなかった。
ライブラリは自動リトライやアクセス間隔の制御を行わない。

## 10. エラー

`madb.Error.Operation` には次の値を使用する。

```text
madb.NewClient
madb.SearchBooks
```

| 状態 | `ErrorKind` |
| --- | --- |
| 入力、Limit、カーソル、Client設定の不正 | `invalid_argument` |
| HTTP 408、429、500から599 | `unavailable` |
| その他の成功以外のHTTPステータス | `upstream` |
| 一時的な通信失敗、タイムアウト | `unavailable` |
| 成功本文の上限超過、JSONまたはbindingの不正 | `invalid_response` |

無効なSPARQLでは、HTTP 400と `application/json` のエラー本文が返ることを
確認した。ライブラリが生成したSPARQLの不正は利用者入力の誤りではないため、
HTTP 400を `invalid_argument` へ変換しない。

`Retry-After` は秒数形式とHTTP-date形式の両方を解析する。解析できない値は
無視する。`context.Canceled` と `context.DeadlineExceeded` は、
ラップ後も `errors.Is` で判定できるようにする。

## 11. 利用条件とサービス変更

SPARQL Query Serviceで取得したデータの利用には、MADBの利用規約が適用される。
利用者向けドキュメントには出典を記載し、データを加工して表示する場合は
加工した旨を記載する。

MADBは技術サポートを提供せず、サービスやコンテンツを予告なく変更する場合がある。
エンドポイント、名前空間、全文検索設定は将来変更される可能性がある。

## 12. 実レスポンス確認例

調査用クエリは、認証情報を使用せず、公式エンドポイントへPOSTした。

| リソースまたは検索 | 確認内容 |
| --- | --- |
| `M521557` | タイトル、creator、publisher、ISBN-13、年月精度の刊行日 |
| `C51001` | `dcterms:creator` が参照するAgent名 |
| `M1032577` | 1冊から複数Agentへの関連 |
| `M1080236` | creator、publisher、ISBN、刊行日、巻数の欠落 |
| `M190399` | ISBN-10とISBN-13の併存 |
| `M215486` | 複数publisherと役割、読みの混在 |
| `M380671` | `[通常版]` の版表示と単行本レーベル |
| `M377325` | 新装版の版表示と単行本レーベルの欠落 |
| `M354575` | 愛蔵版の版表示 |
| `M358795` | 完全版の版表示 |
| `M296746` | 文庫版の版表示 |
| `M292127`、`M292128`、`M292129` | 同じタイトルに対する異なるシリーズ参照とレーベル |
| `M292129` の公式サイト | 単行本レーベルと読み、`C262212` のシリーズ表示、版表示の欠落 |
| `M335551`、`M346749` | 1冊に複数の版表示 |
| `M1079781` | 単行本と参照先シリーズで異なるレーベル |
| `動物のおしゃべり` | 全文検索、部分一致、複数回実行、次ページ |
| `おたがね\ : オタがためカネはなる` | バックスラッシュを含む検索 |
| UUIDを含む存在しない題名 | HTTP 200と空のbindings |

2026年7月30日にTODO011の実装結果を実サービスで確認した。

- `動物のお医者さん` の検索で `M292127`、`M292128`、`M292129` が
  それぞれ異なる `SeriesID` と `Imprints` を返した
- `プロジェクトX挑戦者たち` の検索で、`M335551` が
  `コミック版` と `第2版` の両方を `EditionStatements` として返した
- `スリーＺメン` の検索で、`M1079781` の `Imprints` は
  単行本の `藤子不二雄全集` だけを返し、参照先シリーズの
  `虫コミックス` は混入しなかった
- `動物のおしゃべり` を100件で検索した応答は、次ページ判定用の1冊を含め
  37,588バイトであり、4 MiBの本文上限内だった

実レスポンスは更新されるためリポジトリへ固定保存せず、確認対象のID、
検索語、件数、判断結果を本仕様書へ記録する。
