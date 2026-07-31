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
| `schema:identifier` | `ID` | `Sources[0].ID` |
| 言語タグなしの `schema:name` | `Titles` | `Normalized.Title`、`Sources[0].Values.Titles` |
| `ja-hrkt` の `schema:name` | `TitleKana` | `Normalized.TitleKana`、`Sources[0].Values.TitleKana` |
| 言語タグなしの `schema:alternativeHeadline` | `Subtitles` | `Normalized.Subtitle`、`Sources[0].Values.Subtitles` |
| 言語タグなしの `ma:seriesName` | `SeriesNames` | `Normalized.Series`、`Sources[0].Values.SeriesNames` |
| 参照先シリーズの言語タグなし `schema:name` | `RelatedSeriesNames` | `Normalized.Series`、`Sources[0].Values.SeriesNames` |
| 参照先シリーズの `schema:identifier` | `SeriesID` | `Normalized.Series[].ID` |
| 参照先シリーズURI | `SeriesResourceURI` | `Normalized.Series[].URL` |
| `schema:volumeNumber` | `VolumeNumber` | `Normalized.Volume`、`Sources[0].Values.Volume` |
| 言語タグなしの `schema:version` | `Versions` | `Normalized.EditionStatements`、`Sources[0].Values.Editions` |
| `schema:creator` とcreator Agent | `Creators`、`AgentNames` | `Normalized.Authors`、`Sources[0].Values.Authors` |
| `schema:publisher` | `Publishers` | `Normalized.Publishers`、`Sources[0].Values.Publishers` |
| 言語タグなしの `schema:brand` | `Brands` | `Normalized.Imprints`、`Sources[0].Values.Imprints` |
| `schema:isbn` | `ISBNs` | `Normalized.Identifiers`、`Sources[0].Values.ISBNs` |
| `schema:datePublished` | `PublishedDate` | `Normalized.Dates`、`Sources[0].Values.PublishedDate` |
| `schema:numberOfPages` | `PageCount` | `Normalized.PageCount`、`Sources[0].Values.PageCount` |
| `schema:size` | `Size` | `Normalized.PhysicalSize`、`Sources[0].Values.Size` |
| 固定値 | 対応なし | `Sources[0].Source`、シリーズの `Source` |
| マンガ単行本URI | `ResourceURI` | `Sources[0].URL` |

リソースURIは次の規則を持つ。

```text
https://mediaarts-db.artmuseums.go.jp/id/ + MADB ID
```

マンガ単行本のMADB IDは `M` と数字で構成される。`Sources[0].ID` には
`schema:identifier` の `M...` を設定し、`Sources[0].URL` にはリソースURIを設定する。

`schema:isPartOf` が参照するリソースは
`class:MangaBookSeries` に限定する。マンガ単行本シリーズのMADB IDは
`C` と数字で構成される。参照先の `schema:identifier` とリソースURIは、
対応する `Normalized.Series` の `ID` と `URL` に設定する。

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
	TitleKana         []string
	Subtitles         []string
	SeriesNames       []string
	RelatedSeriesNames []string
	SeriesID          string
	SeriesResourceURI string
	VolumeNumber      string
	Versions          []string
	Creators          []string
	Brands            []string
	Publishers        []string
	ISBNs             []string
	PublishedDate     string
	PageCount         string
	Size              string
	ResourceURI       string
}
```

SPARQL bindingを `sourceBook` へ集約した後、`madb` パッケージ内の変換処理で
`madb.Book` を組み立てる。MADB固有の役割除去、ISBN検証、欠落値処理は
この変換処理が担当する。

### 5.1 複数値と単数値の選択

`Titles`、`TitleKana`、`Subtitles`、シリーズ名、版表示、著者、出版社、レーベル、ISBNは、
空文字列を除外し、完全に同じ値だけを重複除去してGoの文字列昇順にする。
全候補は `Sources[0].Values` に残す。

- 空文字列を除外する
- 完全に同じ値だけを重複除去する
- Goの文字列昇順で並べ、レスポンス順に依存させない
- 表記揺れ、異体字、読み、役割の違いを同一値と推測しない
- タイトルやシリーズ名から欠落項目を補完しない

実データでは、言語タグなしの値だけでも複数タイトル、副題、シリーズ名が存在する。
単一の正規値を示すプロパティがないため、`Normalized.Title` と
`Normalized.Subtitle` には安定ソート後の先頭を暫定値として設定する。
この選択で失われる候補はなく、すべて取得元値に残る。

タイトル読みは、すべてのUnicode空白を除去して同じ値になった候補を数える。
最多の候補が1つに決まる場合だけ `Normalized.TitleKana` へ設定し、最多候補が
同数の場合は空にする。空白除去前の全候補は `Sources[0].Values.TitleKana` に残す。

### 5.2 CreatorsからAuthorsへの変換

`schema:creator` は文字列であり、次のような役割を含む。

```text
[著]手塚治虫
[原作]梶原一騎
[作画]...
```

`sourceBook.Creators` には言語タグなしの `schema:creator`、
`sourceBook.AgentNames` には `dcterms:creator` が参照する全Agentの
`rdfs:label` を別々に格納する。creator文字列とAgentにはRDF上の1対1対応が
ないため、名前の一致や配列位置から両者を結び付けない。

言語タグなしcreator文字列が1件以上ある場合は、役割表記を含む全creator文字列を
`Sources[0].Values.Authors` に変更せず残す。creator文字列がない場合だけ、
全Agent名を取得元の著者表示として残す。どちらも空文字列を除外し、完全一致で
重複除去してGoの文字列昇順にする。

#### 5.2.1 MADB役割の対応

先頭の角括弧内が次の表記と完全一致する場合だけ、共通役割へ変換する。

| MADB役割 | 共通役割 |
| --- | --- |
| `著`、`著者`、`作`、`共著`、`ほか著`、`他著` | `author` |
| `原作`、`原案`、`共原作` | `original_creator` |
| `脚本`、`シナリオ`、`構成`、`脚色`、`文`、`ストーリー`、`ライター` | `writer` |
| `漫画`、`作画`、`画`、`劇画`、`まんが`、`絵`、`comic`、`Comic`、`COMIC`、`comics`、`コミック`、`マンガ`、`アーティスト` | `artist` |
| `キャラクター原案` | `character_creator` |
| `キャラクターデザイン` | `character_designer` |
| `編`、`編集` | `editor` |
| `訳` | `translator` |
| `監修` | `supervisor` |
| `解説` | `commentator` |
| `カバーデザイン`、`装丁`、`装幀`、`デザイン` | `designer` |

`・` で結ばれた複合役割は、すべての構成要素が上表へ対応する場合だけ、
複数の共通役割へ変換する。1つでも未知の構成要素があれば、複合役割全体を
未知として扱う。同じ共通役割になる構成要素は1件へまとめる。

`Authors` へ含める共通役割は `author`、`original_creator`、`writer`、
`artist`、`character_creator`、`character_designer` とする。
`editor`、`translator`、`supervisor`、`commentator`、`designer` は
`Contributors` だけに含める。

#### 5.2.2 creator文字列の解析

creator文字列は次の順序で解析する。

1. 前後のUnicode空白を除く
2. 先頭が `[役割]` で、役割全体を前項の共通役割へ対応できるか確認する
3. 対応できた場合だけ、最初の `]` より後ろを人物名の候補にする
4. 人物名候補が `[` で始まる場合は、最初に対応する `]` との1組だけを除き、
   角括弧内と後続文字列を連結する
5. 前後のUnicode空白を除き、空でなければ人物名として使用する

先頭役割を対応できない値、閉じ角括弧がない値、人物名が空になる値からは、
`Authors` と `Contributors` を作らない。取得元値とRaw responseには残す。

先頭に役割がないcreator文字列は、文字列全体を人物名として `Authors` に含める。
役割を推測できないため `Contributors` には含めない。creator文字列がなく
Agent名だけがある場合も、全Agent名を `Authors` に含め、`Contributors` は
作らない。

同じ人物名から複数の既知役割を得た場合は、人物名の完全一致で1つの
`Contributor` にまとめる。人名の異体字、別名、空白差を同一人物と推測しない。

#### 5.2.3 順序

MADBのRDFはcreatorの順序を持たない。`Creators` と `AgentNames` はGoの文字列昇順で
安定化し、その順に `Authors` と `Contributors` を組み立てる。同じ名前をまとめた
場合は最初に現れた位置を維持し、役割はcreator内の出現順で重複除去する。
この順序は取得元の表示順や役割の優先順位を表さない。

`M292132` の `[著]佐々木倫子` と `[解説]藤原新也` は、佐々木倫子だけを
`Authors` に含め、両名をそれぞれ `author`、`commentator` の
`Contributors` として返す。

### 5.3 publisher

`schema:publisher` は文字列であり、発行、発売、読みなどを含む場合がある。
`Normalized.Publishers` では、`∥` より後ろがカタカナ、空白、中黒、長音記号だけで
構成される場合に限り、区切り以降を出版社名の読みとして除去する。除去後に同じに
なった出版社名は1件へまとめる。`発行元 ∥ 発売元` など、区切り後がカナ読みでは
ない値は変更しない。取得した全表記は `Sources[0].Values.Publishers` に残す。

### 5.4 ISBN

`schema:isbn` は0件以上存在する。ASCIIのハイフンと空白を除いた後、
ISBN-10またはISBN-13のチェックディジットを検証する。

- 正しいISBN-10は大文字の `X` を含む10文字として `Identifiers` へ設定する
- 正しいISBN-13は13桁として `Identifiers` へ設定する
- 検証に失敗した値は `Normalized.Identifiers` へ設定しない
- 検証前の全値は `Sources[0].Values.ISBNs` に残す
- 同じ種別が複数ある場合もすべて返す

### 5.5 刊行日と巻数

`schema:datePublished` は `YYYY`、`YYYY-MM`、`YYYY-MM-DD` など、
精度の異なる文字列である。日付として再解釈せず、取得値をそのまま保持する。
空文字列は欠落として扱う。不正に見える値も推測で修正しない。

`schema:volumeNumber` の元表記は `Sources[0].Values.Volume` に保持する。
文字列全体が許可した整数構文へ一致する場合は `Normalized.Volume.Number` と
10進数の `Label`を設定する。`上`、`中`、`下`、`前編`、`後編`、小数巻、
`別巻`、`外伝`、`番外編`は `Label`だけを設定する。解析できない値は正規化せず、
元表記だけを返す。複数の異なる値が返された場合は、スキーマの0または1件という
契約に反するため `invalid_response` とする。

### 5.6 ページ数と大きさ

`schema:numberOfPages` は、数字だけ、または数字の後ろに `p` が付く表記だけを
`Normalized.PageCount` の整数へ変換する。解析できない値はページ数を設定しない。

`schema:size` の元表記は `Sources[0].Values.Size` に残す。整数または小数第1位までの
センチメートル表記を認識し、先頭の値を高さ、`×` より後ろを幅としてミリメートルへ
変換する。認識できた場合は `Normalized.PhysicalSize` と `print` の
`Normalized.Medium` を設定する。判型名など解析できない値は推測で寸法へ変換しない。

### 5.7 版表示、単行本レーベル、シリーズ

マンガ単行本へ直接記録された言語タグなしの `schema:version` を
`sourceBook.Versions` へ格納し、`Normalized.EditionStatements` と
`Sources[0].Values.Editions` として返す。
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
`ma:seriesName` と分けて `RelatedSeriesNames` に保持する。`Normalized.Series`では、
参照先シリーズ名とID、URL、`SourceMADB`を同じ要素へ設定する。同じ名前が
直接指定にもある場合は1要素へまとめる。参照先が欠落する場合、直接指定された
名前だけをID、URLなしで返す。

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
| `M292132` | `[著]佐々木倫子` と `[解説]藤原新也` の役割分離 |
| `M521385` | Agent参照がなく、creator文字列だけがある単行本 |
| `M830542` | creator文字列がなく、Agent参照だけがある単行本 |
| `M850396` | 役割なしcreator文字列とAgent参照がある単行本 |
| `M809985` | `[[著]]近江のこ` という不正な角括弧表記 |
| `M196958`、`M208098`、`M208454`、`M213006` | 2つ目の角括弧が人名の一部であるcreator文字列 |
| `M190399` | ISBN-10とISBN-13の併存 |
| `M215486` | 複数publisherと役割、読みの混在 |
| `M380671` | `[通常版]` の版表示と単行本レーベル |
| `M377325` | 新装版の版表示と単行本レーベルの欠落 |
| `M354575` | 愛蔵版の版表示 |
| `M358795` | 完全版の版表示 |
| `M296746` | 文庫版の版表示 |
| `M292127`、`M292128`、`M292129` | 同じタイトルに対する異なるシリーズ参照とレーベル |
| `M292129` の公式サイト | 単行本レーベルと読み、`C262212` のシリーズ表示、版表示の欠落 |
| `M292141` | タイトル読み3件、`197p` のページ数、`17.3cm × 10.6cm` の大きさ、出版社名と読み |
| `M335551`、`M346749` | 1冊に複数の版表示 |
| `M1079781` | 単行本と参照先シリーズで異なるレーベル |
| `動物のおしゃべり` | 全文検索、部分一致、複数回実行、次ページ |
| `おたがね\ : オタがためカネはなる` | バックスラッシュを含む検索 |
| UUIDを含む存在しない題名 | HTTP 200と空のbindings |

2026年7月30日にTODO011の実装結果を実サービスで確認した。

- `動物のお医者さん` の検索で `M292127`、`M292128`、`M292129` が
  それぞれ異なるシリーズIDと `Normalized.Imprints` を返した
- `プロジェクトX挑戦者たち` の検索で、`M335551` が
  `コミック版` と `第2版` の両方を `Normalized.EditionStatements` として返した
- `スリーＺメン` の検索で、`M1079781` の `Normalized.Imprints` は
  単行本の `藤子不二雄全集` だけを返し、参照先シリーズの
  `虫コミックス` は混入しなかった
- `動物のおしゃべり` を100件で検索した応答は、次ページ判定用の1冊を含め
  37,588バイトであり、4 MiBの本文上限内だった

2026年8月1日にTODO013の実装結果を実サービスで確認した。

- `動物のお医者さん` の検索で `M292132` の `Normalized.Authors` は
  佐々木倫子だけを返した
- 同じ結果の `Normalized.Contributors` は、佐々木倫子を `author`、
  藤原新也を `commentator` として返した
- `Sources[0].Values.Authors` は `[著]佐々木倫子` と `[解説]藤原新也` を
  変更せず返した
- build tag付きの実サービス統合テスト一式とデモCLIの両方で同じ結果を確認した

実レスポンスは更新されるためリポジトリへ固定保存せず、確認対象のID、
検索語、件数、判断結果を本仕様書へ記録する。
