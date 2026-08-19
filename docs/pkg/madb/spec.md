# madb パッケージ仕様

## 1. 目的

`madb` パッケージは、メディア芸術データベース（MADB）からマンガ単行本を
タイトル、著者名、主要な書誌項目で検索し、ISBNから書籍を参照して、`model`
パッケージで定義する共通モデルへ変換する。指定語を含む検索結果の除外にも対応する。

本仕様書は、MADB固有の検索、項目対応、HTTP処理、
ページング、エラー分類を定義する。

`DateFrom` と `DateTo` は `schema:datePublished` をMADB側で絞り込む。年、年月、年月日の精度に応じた半開区間を使い、指定精度より粗い取得値を推測して含めない。日付だけの検索と片側指定を受け付け、日付条件はCursor照合にも含める。

共通モデルの仕様は [manken API仕様](../../spec.md)、パッケージの責務境界は
[アーキテクチャ](../../../ARCHITECTURE.md)で定義する。

MADBの `schema:provider` 配下にある `schema:price` は、所蔵・提供元に付随する値であり、
共通 `Price` が表す書籍の定価または取得時点販売価格とは意味が異なるため変換しない。
所蔵情報も現在の取得対象外とする。

## 2. 参照資料

2026年7月29日、30日、8月3日に次の公式資料と実サービスを確認した。

- [MADBの概要とSPARQLクエリサービス](https://mediaarts-db.artmuseums.go.jp/about)
- [MADBクラス定義](https://mediaarts-db.artmuseums.go.jp/data/class/)
- [データセットおよびSPARQLクエリサービスの利用方法](https://mediag.bunka.go.jp/madb_lab/lod/howto/)
- [SPARQLクエリサービスとサンプル](https://mediag.bunka.go.jp/madb_lab/lod/sparql/)
- [MADBメタデータスキーマ仕様書v1.2](https://github.com/mediaarts-db/dataset/blob/main/doc/MADB%E3%83%A1%E3%82%BF%E3%83%87%E3%83%BC%E3%82%BF%E3%82%B9%E3%82%AD%E3%83%BC%E3%83%9E%E4%BB%95%E6%A7%98%E6%9B%B8.pdf)
- [MADB Lab利用規約](https://mediag.bunka.go.jp/madb_lab/user_terms/)
- [Amazon Neptune全文検索パラメーター](https://docs.aws.amazon.com/neptune/latest/userguide/full-text-search-parameters.html)

クエリには `https://mediaarts-db.artmuseums.go.jp/` 名前空間を使用する。

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
	request madb.SearchRequest,
) (madb.SearchBooksResult, error)

func (client *Client) SearchBooksWithRawResponse(
	ctx context.Context,
	request madb.SearchRequest,
) (madb.SearchBooksResult, []byte, error)

func (client *Client) LookupBooksByISBN(
	ctx context.Context,
	isbns []string,
) (madb.ISBNLookupResult, error)

func (client *Client) LookupBooksByISBNWithRawResponse(
	ctx context.Context,
	isbns []string,
) (madb.ISBNLookupResult, []byte, error)
```

共通型と共通エラーの実体は `model` パッケージに定義する。`madb` は通常利用に
必要な型と定数をエイリアスとして公開し、利用側は `madb` だけをimportして
検索、結果の参照、分類済みエラーの判定を行える。

- `Option` は外部から独自実装できない不透明な設定型とする
- `Client` は設定だけを保持し、検索条件や結果を保持しない
- 検索ごとに新しい `http.Request` を作成する
- 呼び出し側の `http.Client` と `Transport` を変更しない
- `Client` は複数goroutineから安全に利用できるものとする
- ログ出力、goroutineの開始、自動リトライ、キャッシュを行わない
- `httpClient` が `nil` の場合は、タイムアウト60秒のクライアントを使用する

`SearchBooksWithRawResponse` と `LookupBooksByISBNWithRawResponse` は、変換済み結果に
加えて、MADBから受信した成功レスポンス本文を変更せず `[]byte` で返す。本文は
呼び出し元が所有し、変更しても `Client` や変換済み結果へ影響しない。

2xx応答の本文を上限内で読み込んだ後、JSON解析またはbindingから検索結果への
変換に失敗した場合は、読み込み済み本文とエラーを同時に返す。本文読込の失敗、
本文上限の超過、2xx以外のHTTP応答、通信失敗では本文を返さない。
Raw responseを持たないメソッドも同じ通信と変換処理を使用し、受信本文を
呼び出し元へ返さない。

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

| MADBの取得元                                | `madb` の非公開型        | `madb.Book`                              |
| ------------------------------------------- | ------------------------ | ---------------------------------------- |
| `schema:identifier`                         | `ID`                     | `Sources[0].ID`                          |
| 言語タグなしの `schema:name`                | `Titles`                 | `Title`                                  |
| `ja-hrkt` の `schema:name`                  | `TitleKana`              | `TitleReading`                           |
| 言語タグなしの `schema:alternativeHeadline` | `Subtitles`              | `Subtitle`                               |
| 言語タグなしの `ma:seriesName`              | `SeriesNames`            | `BookSeries`                             |
| 参照先シリーズの言語タグなし `schema:name`  | `RelatedSeriesNames`     | `BookSeries`                             |
| 参照先シリーズの `schema:identifier`        | `SeriesID`               | `BookSeries[].ID`                        |
| 参照先シリーズURI                           | `SeriesResourceURI`      | `BookSeries[].URL`                       |
| `schema:volumeNumber`                       | `VolumeNumber`           | `Volume`                                 |
| 言語タグなしの `schema:version`             | `Versions`               | `Editions`                               |
| `schema:creator` とcreator Agent            | `Creators`、`AgentNames` | `Authors`、`Contributors`                |
| `schema:publisher`                          | `Publishers`             | `Publishers`                             |
| 言語タグなしの `schema:brand`               | `Brands`                 | `PublicationSeries`                      |
| `schema:isbn`                               | `ISBNs`                  | `ISBN10`、`ISBN13`                       |
| `schema:datePublished`                      | `PublishedDate`          | `PublishedDate`                          |
| `schema:numberOfPages`                      | `PageCount`              | `PageCount`                              |
| `schema:size`                               | `Size`                   | `Size`                                   |
| 固定値                                      | 対応なし                 | `Sources[0].Source`、シリーズの `Source` |
| マンガ単行本URI                             | `ResourceURI`            | `Sources[0].URL`                         |

中央列はMADB応答を集約するための非公開実装型を示し、公開APIではない。

リソースURIは次の規則を持つ。

```text
https://mediaarts-db.artmuseums.go.jp/id/ + MADB ID
```

マンガ単行本のMADB IDは `M` と数字で構成される。`Sources[0].ID` には
`schema:identifier` の `M...` を設定し、`Sources[0].URL` にはリソースURIを設定する。

`schema:isPartOf` が参照するリソースは
`class:MangaBookSeries` に限定する。マンガ単行本シリーズのMADB IDは
`C` と数字で構成される。参照先の `schema:identifier` とリソースURIは、
対応する `BookSeries` の `ID` と `URL` に設定する。

## 5. 結果変換

MADBのSPARQL Results JSONを取得元固有の非公開表現へ読み込み、同じリソースの
bindingを集約してから共通の `madb.Book` へ変換する。MADB固有の役割表記、ISBN、
欠落値の扱いはこの節で定義する。

### 5.1 複数値と単数値の選択

`Titles`、`TitleKana`、`Subtitles`、シリーズ名、版表示、著者、出版社、レーベル、ISBNは、
空文字列を除外し、完全に同じ値だけを重複除去してGoの文字列昇順にする。

- 空文字列を除外する
- 完全に同じ値だけを重複除去する
- Goの文字列昇順で並べ、レスポンス順に依存させない
- 表記揺れ、異体字、読み、役割の違いを同一値と推測しない
- タイトルやシリーズ名から、Volume、Editions、IsFinalVolume以外の欠落項目を補完しない。明示VolumeまたはEditionsが欠落する場合だけ、安全なタイトル構文から補う場合がある

実データでは、言語タグなしの値だけでも複数タイトル、副題、シリーズ名が存在する。
単一の正規値を示すプロパティがないため、`Title` と
`Subtitle` には安定ソート後の先頭を暫定値として設定する。
選択しなかった候補を確認する必要がある場合は、リクエスト単位で返すRaw responseを使用する。

タイトル読みは、すべてのUnicode空白を除去して同じ値になった候補を数える。
最多の候補が1つに決まる場合だけ `TitleReading` へ設定し、最多候補が
同数の場合は空にする。

### 5.2 creator文字列からAuthors・Contributorsへの変換

`schema:creator` は文字列であり、次のような役割を含む。

```text
[著]手塚治虫
[原作]梶原一騎
[作画]...
```

`sourceBook.Creators` には言語タグなしの `schema:creator`、
`sourceBook.AgentNames` には `dcterms:creator` が参照する全Agentの
`rdfs:label` を別々に格納する。creator文字列とAgentにはRDF上の1対1対応が
ないため、名前の一致や配列位置から両者を結び付けない。人物名と読みの対応も
確認できないため、MADBでは `Contributor.Reading` を設定しない。

言語タグなしcreator文字列が1件以上ある場合は、役割表記を解析して著者と寄与者を作る。
creator文字列がない場合だけ、全Agent名を著者と役割なしの寄与者として使用する。
どちらも空文字列を除外し、完全一致で重複除去してGoの文字列昇順にする。

#### 5.2.1 MADB役割の対応

先頭の角括弧内が次の表記と完全一致する場合だけ、共通役割へ変換する。

| MADB役割                                                                                                                | 共通役割                   |
| ----------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `著`、`著者`、`作`、`共著`、`ほか著`、`他著`                                                                            | `著者`                     |
| `原作`、`原案`、`共原作`                                                                                                | `原作`                     |
| `脚本`、`シナリオ`、`構成`、`脚色`、`文`、`ストーリー`、`ライター`                                                      | `脚本`                     |
| `漫画`、`作画`、`画`、`劇画`、`まんが`、`絵`、`comic`、`Comic`、`COMIC`、`comics`、`コミック`、`マンガ`、`アーティスト` | `作画`                     |
| `キャラクター原案`                                                                                                      | `キャラクター原案`         |
| `キャラクターデザイン`                                                                                                  | `キャラクターデザイン`     |
| `編`、`編集`                                                                                                            | `編集`                     |
| `訳`                                                                                                                    | `翻訳`                     |
| `監修`                                                                                                                  | `監修`                     |
| `解説`                                                                                                                  | `解説`                     |
| `カバーデザイン`、`装丁`、`装幀`、`デザイン`                                                                            | `デザイン`                 |

`・` で結ばれた複合役割は、すべての構成要素が上表へ対応する場合だけ、
複数の共通役割へ変換する。1つでも未知の構成要素があれば、複合役割全体を
未知として扱う。同じ共通役割になる構成要素は1件へまとめる。

`Authors` へ含める共通役割は `著者`、`原作`、`脚本`、`作画`、
`キャラクター原案`、`キャラクターデザイン` とする。
`編集`、`翻訳`、`監修`、`解説`、`デザイン` は `Contributors` だけに含める。

#### 5.2.2 creator文字列の解析

creator文字列は次の順序で解析する。

1. 前後のUnicode空白を除く
2. 先頭が `[役割]` の場合、最初の `]` より後ろを人物名の候補にする
3. 役割全体を前項の共通役割へ対応できるか確認する
4. 人物名候補が `[` で始まる場合は、最初に対応する `]` との1組だけを除き、
   角括弧内と後続文字列を連結する
5. 前後のUnicode空白を除き、空でなければ人物名として使用する

先頭に役割表記がないcreator文字列は、文字列全体を人物名として `Authors` と
役割なしの `Contributors` に含める。既知役割を持つ人物は `Contributors` に含め、
`著者`、`原作`、`脚本`、`作画`、`キャラクター原案`、
`キャラクターデザイン` のいずれかを持つ場合だけ `Authors` に含める。役割表記があるが
対応できない人物は、役割なしの `Contributors` にだけ含める。

閉じ角括弧がない、角括弧が不正に重なる、人物名が空になる値からは、`Authors` と
`Contributors` を作らない。値を確認する必要がある場合はRaw responseを使用する。

同じ人物名から複数の既知役割を得た場合は、人物名の完全一致で1つの
`Contributor` にまとめる。人名の異体字、別名、空白差を同一人物と推測しない。

#### 5.2.3 順序

MADBのRDFはcreatorの順序を持たない。`Creators` と `AgentNames` はGoの文字列昇順で
安定化し、その順に `Authors` と `Contributors` を組み立てる。同じ名前をまとめた
場合は最初に現れた位置を維持し、役割はcreator内の出現順で重複除去する。
この順序は取得元の表示順や役割の優先順位を表さない。

`M292132` の `[著]佐々木倫子` と `[解説]藤原新也` は、佐々木倫子だけを
`Authors` に含め、両名をそれぞれ `著者`、`解説` の
`Contributors` として返す。

### 5.3 publisher

`schema:publisher` は文字列であり、発行、発売、読みなどを含む場合がある。
`Publishers` では、`∥` より後ろがカタカナ、空白、中黒、長音記号だけで
構成される場合に限り、区切り以降を出版社名の読みとして除去する。除去後に同じに
なった出版社名は1件へまとめる。`発行元 ∥ 発売元` など、区切り後がカナ読みでは
ない値は変更しない。

### 5.4 ISBN

`schema:isbn` は0件以上存在する。ASCIIのハイフンと空白を除いた後、
ISBN-10またはISBN-13のチェックディジットを検証する。

- 正しいISBN-10は大文字の `X` を含む10文字として `ISBN10` へ設定する
- 正しいISBN-13は13桁として `ISBN13` へ設定する
- 検証に失敗した値は `ISBN10` / `ISBN13` へ設定しない
- 同じ種別が複数ある場合もすべて返す

### 5.5 刊行日と巻数

`schema:datePublished` は `YYYY`、`YYYY-MM`、`YYYY-MM-DD` など、
精度の異なる文字列である。`PublishedDate` に取得値をそのまま保持し、日付として再解釈しない。
空文字列は欠落として扱う。不正に見える値も推測で修正しない。

文字列全体が許可した整数構文へ一致する場合は `Volume.Number` と
10進数の `Label`を設定する。`上`、`中`、`下`、`前編`、`後編`、小数巻、
`別巻`、`外伝`、`番外編`は `Label`だけを設定する。解析できない値は正規化せず、
`Volume` を設定しない。複数の異なる値が返された場合は、スキーマの0または1件という
保証事項に反するため `invalid_response` とする。

### 5.6 ページ数と大きさ

`schema:numberOfPages` は、数字だけ、または数字の後ろに `p` が付く表記だけを
`PageCount` の整数へ変換する。解析できない値はページ数を設定しない。

`schema:size` の文字列は `Size` にそのまま保持する。整数または小数の
センチメートル表記を認識できる場合だけ `Medium` を `print` とする。
判型名など解析できない値から寸法や媒体を推測しない。

### 5.7 版表示、出版側グループ表示、シリーズ

マンガ単行本へ直接記録された言語タグなしの `schema:version` を
`Editions` として返す。
版表示は0件以上存在し、複数の異なる値もすべて返す。値を通常版、新装版、
愛蔵版、完全版、文庫版などの独自分類へ変換しない。

マンガ単行本へ直接記録された言語タグなしの `schema:brand` を
`PublicationSeries` として返す。`ja-hrkt` などの言語タグ付きの読みは返さない。
空文字列を除外し、完全一致の重複を除去してGo文字列昇順で返す。レーベルらしくない値が
含まれていても、文字列の内容から除外、修正、再結合、再分類しない。たとえば
`M530976`（ISBN `9784253121989`）では、NDL保存済みRawの
`dcndl:seriesTitle` が `A.L.C.SELECTION. アラ還 愛子ときどき母` という1値である一方、
現行MADBは `A`、`C`、`L`、`SELECTION`、`アラ還 愛子ときどき母` の5つの
`schema:brand` を返す。取得元データを `A.L.C.SELECTION` へ再結合したり、
`アラ還 愛子ときどき母` を `BookSeries` へ移したりしない。

`schema:isPartOf` が参照する `class:MangaBookSeries` から、次を取得する。

- 言語タグなしの `schema:name`
- `schema:identifier`
- シリーズリソースURI

シリーズの `schema:name` は、単行本へ直接記録された言語タグなしの
`ma:seriesName` と分けて `RelatedSeriesNames` に保持する。`BookSeries`では、
参照先シリーズ名とID、URL、`SourceMADB`を同じ要素へ設定する。同じ名前が
直接指定にもある場合は1要素へまとめる。参照先が欠落する場合、直接指定された
名前だけをID、URLなしで返す。

異なる複数の参照先シリーズ、シリーズID、シリーズURIが返された場合は、
スキーマの0または1件という保証事項に反するため `invalid_response` とする。
参照先シリーズの `schema:brand` と `schema:version` は単行本の値として返さない。
単行本の版表示または出版側グループ表示が欠落しても、参照先シリーズ、タイトル、
出版社、判型から補完しない。

## 6. 検索条件

`Title`、`Author`、`Publisher`、`Query`、`DateFrom`、`DateTo` の少なくとも1つを正条件として必須とする。
`Exclude` だけの検索は `invalid_argument` とする。
複数を指定した場合は、それぞれの条件をAND結合する。検索結果は必ず
`rdf:type class:MangaBook` で絞り込む。

`Title`、`schema:creator`、`Query`、独立した`Exclude`の全文検索では、
`?resource` を返すNeptune FTS候補を `Neptune#fts.entity_id` の昇順で取得する。このentity IDは
MADB書籍リソースURIに対応し、ページングで使用するリソースURI昇順と同じ順序とする。

`Author` のAgent `rdfs:label`検索では、FTSは `?searchAgent` を返す。この経路のentity IDは
Agent URIに対応するため、Agent候補をentity ID昇順で取得してから
`?resource dcterms:creator ?searchAgent` へ結合する。最終的な書籍のページングは、ほかの検索と同じく
MADB書籍リソースURI昇順で行う。

### 6.1 タイトル検索

タイトル検索にはAmazon Neptuneの全文検索拡張を使用する。

```text
queryType: query_string
field:     schema:name
```

検索語は全文検索構文として受け付けない。次の順序で文字列を生成する。

1. `strings.Fields` と同じ規則でUnicode空白を区切りとして検索語へ分割する
2. 各語の `\` と `"` を全文検索の引用句向けにエスケープする
3. 各語を二重引用符で囲み、` AND ` で結ぶ
4. 完成した全文検索式をSPARQL文字列リテラル向けにエスケープする

例えば `うる星 復刻box` は `"うる星" AND "復刻box"` として検索する。
引用符、バックスラッシュ、`AND`、`OR`、`NOT`、`+`、`-` を含む入力も
各語の引用句内に閉じ込め、利用者入力によって全文検索構文を変更させない。
タイトルの検索語数にライブラリ独自の上限は設けない。検索語数の増加により
応答時間が長くなる場合は、呼び出し元がContextまたはHTTPクライアントの
タイムアウトで制御する。

大文字小文字、Unicode正規化、表記揺れはライブラリ側で変換せず、
全文検索サービスの解析に従う。

全文検索の完全性は、最終的に返すマンガ単行本件数ではなく、FTS raw候補に適用される
OpenSearch result windowに制約される。実行計画がFTS候補を先に評価する場合、RDFの
`class:MangaBook` とPublisherの条件はFTS候補の後に評価されるため、最終Book件数がresult window未満でも、
FTS raw候補がwindowを超える検索の完全なページングは保証しない。

### 6.2 著者名検索

著者名検索は、次の2経路を対象にする。

- 単行本の `schema:creator` 文字列
- 単行本の `dcterms:creator` が参照するAgentの `rdfs:label`

両経路を `UNION` し、同じ単行本が両方に一致した場合は `DISTINCT` で1件にまとめる。
Agent名に一致した場合は、Agent URIから `dcterms:creator` を逆引きして単行本URIを得る。
著者役割による絞り込みや、creator文字列とAgentの1対1対応付けは行わない。

著者名はタイトルと同じ規則でUnicode空白を区切りとして検索語へ分割し、各語を
引用して `AND` で結ぶ。利用者入力の全文検索演算子は構文として解釈しない。
著者名の検索語数にライブラリ独自の上限は設けず、呼び出し元がContextまたは
HTTPクライアントのタイムアウトで制御する。

Agent参照の逆引きはcreator文字列だけの検索より遅くなる可能性がある。
既定のHTTPクライアントでは60秒を上限とする。

### 6.3 出版社名検索

出版社名検索は、単行本リソースのRDF `schema:publisher` 値に対する大文字小文字を区別しない
部分一致とする。Neptune全文検索は使用しない。FTS候補を先に絞る方式ではresult windowの後に
Publisher条件が評価され、該当書籍を取りこぼす実例があるため、出版社条件はRDF値へ直接適用する。

`Publisher` はUnicode空白を区切りとして検索語へ分割し、各語がpublisher値のいずれかに
含まれる書籍を検索する。複数の検索語はAND条件とし、各語は別々のpublisher値に一致してよい。
例えば、`小学館` は `小学館` と `小学館　∥　ショウガクカン` の双方に一致し、`小学` と
`ショウガクカン` も同じ値への部分一致として扱う。利用者入力はSPARQL文字列リテラルとして
安全に扱い、SPARQL構文として解釈しない。

`Title`、`Author`、`Query` と同時指定した場合は、すべての正条件をAND結合する。

`Query` の検索対象にも `schema:publisher` を含める。`Publisher` はその対象を限定する
専用条件であり、`Query` の対象項目を変更しない。

### 6.4 フリーワード検索

フリーワード検索は、漫画単行本リソース上の次の文字列項目を対象にする。

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

Neptune全文検索の `query_string` に上記の `field` をすべて明示する。フィールドの
省略と `*` は使用しない。`Query` はタイトルと同じ規則でUnicode空白を区切りに
検索語へ分割し、各語を安全な引用句にして `AND` で結ぶ。各語は同じ項目に存在する
必要はなく、複数の対象項目をまたいで一致してよい。

Agentの `rdfs:label` は別リソースにあるため対象外とする。Agent参照だけに存在する
著者名を漏れなく検索する場合は `Author` を使用する。`Title`、`Author`、`Publisher` と
同時指定した場合は、フリーワードを含むすべての条件をAND結合する。

検索語数にライブラリ独自の上限は設けず、呼び出し元がContextまたはHTTPクライアントの
タイムアウトで制御する。対象フィールドはMADBの全文検索インデックス変更により
利用できなくなる可能性がある。`schema:size` は寸法文字列であり、一般的な判型名が
常に存在するとは限らない。

### 6.5 除外検索

`Exclude` は、6.4のフリーワード検索と同じ12項目を対象にする。
Agentの `rdfs:label` は対象外とし、Agent参照だけに存在する著者名では除外しない。

除外文字列はUnicode空白で検索語へ分割し、各語を安全な引用句へ変換する。
どれか1語でも含む単行本を除外する。利用者入力の `AND`、`OR`、`NOT`、引用符、
バックスラッシュは全文検索構文として解釈しない。

`Query` がある場合は、フリーワードのAND式へ除外語をそれぞれ `AND NOT` で
追加する。例えば `Query` が `うる星`、`Exclude` が `復刻box 愛蔵版` の
場合は、次の式を生成する。

```text
"うる星" AND NOT "復刻box" AND NOT "愛蔵版"
```

`Query` がない場合は、フリーワード対象の12項目に対して除外語を `OR` で
結合した全文検索を実行し、その一致結果をSPARQLの `MINUS` で除く。タイトルと
著者名のどちらの正条件でも同じ除外範囲を維持する。

除外語数にライブラリ独自の上限は設けず、呼び出し元がContextまたはHTTPクライアントの
タイムアウトで制御する。一般的な語の除外を `MINUS` で実行すると応答時間が長くなる
可能性がある。

## 7. ISBN参照

`LookupBooksByISBN` と `LookupBooksByISBNWithRawResponse` は、1件以上500件以下の
ISBNを受け付ける。空入力または501件以上は、外部通信せず `invalid_argument` とする。
1件でも不正なISBNがある場合も、呼び出し全体を `invalid_argument` とする。
500件はこのパッケージの公開上限であり、MADBサービスの問い合わせ上限を表すものではない。

各入力は次の順序で整形して検証する。

1. ASCIIハイフンとUnicode空白を除く
2. 末尾の小文字 `x` を大文字 `X` にする
3. ISBN-10またはISBN-13の文字種とチェックディジットを検証する
4. ISBN-10から978で始まるISBN-13を生成する
5. 978で始まるISBN-13からISBN-10を生成する

ISBN-13は978または979で始まる13桁だけを受け付ける。979で始まるISBN-13から
ISBN-10は生成しない。付記付きの `9784778031404 (set)` は有効なISBNとして扱わない。

入力値と生成できた対応値は問い合わせ全体で重複を除いて文字列昇順にし、1回の
SPARQLへ指定する。

```text
VALUES ?matchedISBN { "4088466365" "9784088466361" }
?resource schema:isbn ?matchedISBN .
```

結果の `Items` は入力と同じ件数、同じ順序で返す。`RequestedISBN` は入力文字列を
変更せず保持する。重複入力やISBN-10と対応するISBN-13を同時指定した場合も、
元の入力位置ごとに結果を返す。該当なしの `Books` は非nilの空スライスとする。

同じISBNに複数のMADBリソースが一致する場合は、値を1冊へ統合せず、リソースURIの
文字列昇順で別々の `Book` として返す。ISBN参照はLimitとカーソルを使用しない。

## 8. Limitとページング

`Limit` の既定値は20、最大値は100とする。

- `0` は既定値20として扱う
- 1から100までは指定値を使用する
- 負数または101以上は `invalid_argument` とする
- 次ページ判定のため、リソースを `Limit + 1` 件取得する

ページングにはMADBリソースURIの昇順によるキーセット方式を使用する。
初回は先頭から取得し、次ページでは直前ページの末尾URIより大きいURIを取得する。
`OFFSET` は、検索中の追加データによって位置がずれるため使用しない。

`?resource` を返す全文検索では、FTS候補を書籍リソースURIに対応するentity ID昇順で取得した後に、
同じリソースURI順でページ境界を判定する。`schema:creator`の文字列経路はこの順序とCursorが一致する。
Agent `rdfs:label`検索では、Agent URIに対応するentity ID順で候補を取得してから書籍リソースへ結合するため、
候補順と書籍リソースURI Cursorは一致しない。FTSはRDFのJOIN前に`batchSize`、`maxResults`、OpenSearch
result windowの範囲で候補を取得し得るため、このAgent経路はCursorを最後まで進めても書籍の完全列挙を
保証しない。Neptuneの既定`batchSize` と既定`maxResults`を使用し、ライブラリはこれらを固定値で指定しない。

カーソル形式のバージョンは2とする。カーソルは、バージョン、末尾URI、Limit、
正規化済み全検索条件のSHA-256を含むJSONを
パディングなしBase64 URL形式で符号化する。

検索条件のハッシュには空の `isbns` 配列を含める。現在の検索APIはISBN条件を
持たないため、この値は常に空とする。

- 形式不正、未対応バージョン、URI不正は `invalid_argument` とする
- カーソルのLimitまたは正規化済み検索条件がリクエストと一致しない場合は
  `invalid_argument` とする
- 有効期間は設けない
- カーソルは改ざん防止を目的とした署名を持たない
- MADB更新中の完全なスナップショット一貫性は保証しない

URI順の部分一致検索を2回実行して同一順序になることと、全文検索でも
末尾URIを指定した次ページが重複なく直後のURIから始まることを確認した。

カーソル境界は `STR(?resource)` と末尾URI文字列を比較する。SPARQLのIRI同士を
`>` で直接比較しない。並び順は引き続きリソースURIの昇順とする。

## 9. SPARQLレスポンスの組み立て

ページ対象のリソースURIをサブクエリで先に確定し、そのリソースに対する各項目を
外側のクエリで取得する。外側のbinding数に `LIMIT` を適用しない。

ISBN参照は、リソースURIと実際に一致した `matchedISBN` をサブクエリで確定する。
外側のクエリにLimitを設けず、Go側でリソース単位に集約してから入力位置へ対応付ける。

1冊に対する複数bindingはGo側でまとめる。SPARQLの `GROUP_CONCAT` は、
区切り文字を実データと区別できないため使用しない。

SPARQL Results JSONのbindingは、要求した変数が欠落することを正常な状態として
扱う。未知の変数は無視するが、既知の変数の型が仕様と異なる場合は
`invalid_response` とする。

## 10. HTTP

既定エンドポイントはGETとPOSTに対応する。SPARQLをURLへ含めず、長いクエリでも
URL長の制約を受けないPOSTを使用する。

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

## 11. エラー

`madb.Error.Operation` には次の値を使用する。

```text
madb.NewClient
madb.SearchBooks
madb.LookupBooksByISBN
```

| 状態                                                            | `ErrorKind`        |
| --------------------------------------------------------------- | ------------------ |
| 入力ISBN、入力件数、検索条件、Limit、カーソル、Client設定の不正 | `invalid_argument` |
| HTTP 408、429、500から599                                       | `unavailable`      |
| その他の成功以外のHTTPステータス                                | `upstream`         |
| 一時的な通信失敗、タイムアウト                                  | `unavailable`      |
| 成功本文の上限超過、JSONまたはbindingの不正                     | `invalid_response` |

無効なSPARQLでは、HTTP 400と `application/json` のエラー本文が返ることを
確認した。ライブラリが生成したSPARQLの不正は利用者入力の誤りではないため、
HTTP 400を `invalid_argument` へ変換しない。

`Retry-After` は秒数形式とHTTP-date形式の両方を解析する。解析できない値は
無視する。`context.Canceled` と `context.DeadlineExceeded` は、
ラップ後も `errors.Is` で判定できるようにする。

## 12. 利用条件とサービス変更

SPARQL Query Serviceで取得したデータの利用には、MADBの利用規約が適用される。
利用者向けドキュメントには出典を記載し、データを加工して表示する場合は
加工した旨を記載する。

MADBは技術サポートを提供せず、サービスやコンテンツを予告なく変更する場合がある。
エンドポイント、名前空間、全文検索設定は将来変更される可能性がある。
