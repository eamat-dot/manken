# openbd パッケージ仕様

## 1. 目的

`openbd` パッケージは、複数のISBNからopenBDの書誌情報を1回で参照し、
`api` パッケージで定義する共通書籍モデルへ変換する。

本仕様書は、openBD固有のISBN参照、項目対応、HTTP処理、エラー分類を定義する。
共通モデルの仕様は [manken API仕様](../../spec.md)、パッケージの責務境界は
[アーキテクチャ](../../../ARCHITECTURE.md)で定義する。

## 2. 参照資料とサービスの状態

2026年8月3日に次の公式資料と実サービスを確認した。

- [openBD公式サイト](https://openbd.jp/)
- [書誌APIデータ仕様](https://openbd.jp/spec/)
- [openBD API利用規約](https://openbd.jp/terms/)
- [openBD API（バージョン1）の提供終了について](https://openbd.jp/news/20230725.html)
- [代替書誌情報の提供開始について](https://openbd.jp/news/20230829.html)

従来のopenBD APIバージョン1は提供終了が告知されている。現在の同一エンドポイントは、
国立国会図書館の書誌情報を従来のONIX形式へ簡易変換した代替APIを含む。
プログラム上の互換性は維持されているが、項目の欠落と書影収録範囲の縮小がある。

## 3. パッケージとClient

import pathは次のとおり。

```text
github.com/eamat-dot/manken/openbd
```

公開APIは次の形とする。

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)

func WithEndpoint(endpoint string) Option

func (client *Client) LookupBooksByISBN(
	ctx context.Context,
	isbns []string,
) (openbd.ISBNLookupResult, error)

func (client *Client) LookupBooksByISBNWithRawResponse(
	ctx context.Context,
	isbns []string,
) (openbd.ISBNLookupResult, []byte, error)
```

共通型と共通エラーの実体は `api` パッケージに定義する。`openbd` は通常利用に
必要な型と定数をエイリアスとして公開し、利用側は `openbd` だけをimportして
ISBN参照、結果の参照、分類済みエラーの判定を行える。

- `Option` は外部から独自実装できない不透明な設定型とする
- `Client` は接続設定だけを保持し、リクエストや結果を保持しない
- リクエストごとに新しい `http.Request` を作成する
- 呼び出し側の `http.Client` と `Transport` を変更しない
- `Client` は複数goroutineから安全に利用できるものとする
- ログ出力、goroutineの開始、自動リトライ、キャッシュを行わない
- `httpClient` が `nil` の場合は、タイムアウト60秒のクライアントを使用する

既定エンドポイントは次のとおり。

```text
https://api.openbd.jp/v1/get
```

`WithEndpoint` は絶対URLを受け付ける。スキームは `http` または `https`、
ホストは必須とし、ユーザー情報、クエリ、フラグメントを受け付けない。

## 4. ISBN入力と結果対応

ISBN参照は1件以上1,000件以下の入力を受け付ける。空入力、1,001件以上、
不正なISBNが1件でもある場合は外部通信せず `invalid_argument` とする。

各入力は共通ISBN処理で検証し、問い合わせには対応するISBN-13を使用する。
ISBN-10と対応するISBN-13、同じISBNの重複入力は問い合わせ時に1件へまとめる。
問い合わせ順は、入力で正規化後のISBN-13が最初に現れた順序とする。

結果の `Items` は入力と同じ件数、同じ順序で返す。`RequestedISBN` は入力文字列を
変更せず保持する。該当なしを表す `null` は、対応する `Books` が非nilの空スライスに
なる。同じ書誌を重複指定した場合は、問い合わせ結果を元の各入力位置へ展開する。

成功応答は問い合わせISBNと同じ件数、同じ順序のJSON配列でなければならない。
非 `null` 要素では、`onix.RecordReference`、ISBN-13を表す
`onix.ProductIdentifier.IDValue`、`summary.isbn` のうち存在する値を検証する。
少なくとも1つが問い合わせISBNと一致し、存在するISBN値が互いに矛盾しない場合だけ
有効とする。件数不一致、ISBN不一致、ISBNを確認できない要素は
`invalid_response` とする。

## 5. 項目対応

openBD応答は非公開型へ変換してから `Book` を組み立てる。未知のJSON項目は無視し、
欠落した任意項目は有効な応答として扱う。

| openBDの取得元 | `openbd.Book` |
| --- | --- |
| 検証済みISBN-13 | `Sources[0].ID`、`Normalized.Identifiers` |
| ONIXの商品階層タイトル、`summary.title` | `Normalized.Title`、`Sources[0].Values.Titles` |
| ONIXのタイトル `collationkey` | `Normalized.TitleKana`、`Sources[0].Values.TitleKana` |
| ONIXのSubtitle | `Normalized.Subtitle`、`Sources[0].Values.Subtitles` |
| ONIXのCollection、`summary.series` | `Normalized.Series`、`Sources[0].Values.SeriesNames` |
| ONIXのContributor | `Normalized.Authors`、`Normalized.Contributors`、`Sources[0].Values.Authors` |
| ONIXのImprint、Publisher、`summary.publisher` | `Normalized.Publishers`、`Sources[0].Values.Publishers` |
| ONIXのPublishingDate、`hanmoto.dateshuppan`、`summary.pubdate` | `Normalized.Dates`、`Sources[0].Values.PublishedDate` |
| `summary.cover` | `Normalized.Images` |
| 固定値 | `Sources[0].Source` |

`Sources[0].ID` は検証済みISBN-13とする。openBDは応答に書籍ごとの参照ページURLを
返さないため、`Sources[0].URL` は空にする。

### 5.1 採用順序

タイトル、タイトル読み、サブタイトル、シリーズ、寄与者、出版社、出版日は、
意味を確認できるONIX項目を優先する。ONIX項目がない場合だけ、対応する
`summary` または `hanmoto` の値を使用する。

- タイトルは `TitleType=01` の商品階層 `TitleElementLevel=01` を優先する
- シリーズはCollection階層のタイトルを優先し、なければ `summary.series` を使う
- 発行元を表すONIXのImprint、発売元を表すPublisherを、この順で出版社候補にする
- ONIXの出版社候補がない場合だけ `summary.publisher` を使う
- 出版日は役割を対応付けられるONIX日付、`hanmoto.dateshuppan`、
  `summary.pubdate` の順で最初の値を使う
- 表紙画像は `summary.cover` だけを使用する

取得元の空文字列を候補から除き、完全に同じ値だけを重複除去する。文字列の空白、
句読点、人名表記、日付精度は変更しない。正規化に使用した主要な元値は
`Sources[0].Values` に残す。

### 5.2 タイトルと巻数

`summary.volume` とCollection階層の `PartNumber` は、出版社コレクション内の
刊行番号を含むため巻数へ使用しない。

タイトルは次の条件をすべて満たす場合だけ、主タイトル、並列タイトル、巻数へ分ける。

- ` = ` で2項目に分かれ、右側にラテン文字がある場合は右側を並列タイトル候補にする
- 末尾の巻表示が1から3桁の数字、`第N巻`、`N巻`、`(N)`、`（N）`、`#N` の
  いずれかに完全一致する
- 並列タイトルがある場合は、その末尾だけから巻表示を取り除く
- 並列タイトルがない場合は、シリーズ値が存在するときだけタイトル末尾を分解する

安全に分解できない場合は、取得元タイトル全体を `Normalized.Title` に設定する。
完全な取得元タイトルは、分解結果にかかわらず `Sources[0].Values.Titles` に残す。
タイトル読みは独立して扱う。タイトルと同じ巻数を読みの末尾から分離できる場合だけ
分解後の読みを設定し、対応を確認できない場合は `Normalized.TitleKana` を空にする。
取得元の読みは `Sources[0].Values.TitleKana` に残す。

### 5.3 寄与者

ONIXの `SequenceNumber` が数値として解釈できる寄与者を番号順にし、番号がない要素は
応答順で後ろに置く。人物名は `PersonName.content` を変更せず使用する。

対応する役割は次のとおり。

| ONIXコード | 共通役割 |
| --- | --- |
| `A01` | `author` |
| `A03`、`A14`、`A45` | `writer` |
| `A07`、`A12`、`A35`、`A46`、`A47` | `artist` |
| `A38` | `original_creator` |
| `A36` | `designer` |
| `B01` | `editor` |
| `B06` | `translator` |

役割が空の場合は人物名を `Authors` に含め、`Contributors` には含めない。
役割が1件以上ある場合は、対応できた役割だけを `Contributors` に設定する。
`author`、`original_creator`、`writer`、`artist` のいずれかへ対応した人物だけを
`Authors` に含める。未知の役割から共通役割を推測しない。

ONIXの寄与者がない場合、空でない `summary.author` を分割せず1件の
`Authors` として使用する。

### 5.4 変換しない項目

ISBN、タイトル、タイトル読み、サブタイトル、並列タイトル、巻数、シリーズ、
寄与者、出版社、出版日、表紙画像を共通モデルへ設定する。

説明、主題、言語、ページ数、判型、物理寸法、価格、ONIX SupportingResource、
版表示は共通モデルへ設定しない。これらはRaw responseから確認できる。

## 6. Raw response

`LookupBooksByISBNWithRawResponse` は、変換済み結果に加えてopenBDから受信した
成功レスポンス本文を変更せず `[]byte` で返す。JSON解析または結果変換に失敗した
場合も、上限内で読み込み済みの2xx本文を返す。

本文読込の失敗、本文上限の超過、2xx以外のHTTP応答、通信失敗では本文を返さない。
Raw responseを持たないメソッドも同じ通信と変換処理を使用する。

## 7. HTTP

ISBN-13をカンマ区切りにし、GETの `isbn` クエリパラメーターへ設定する。

```text
Accept: application/json
```

- 成功レスポンス本文の上限は64 MiBとする
- エラーレスポンス本文の読込上限は64 KiBとする
- 外部サービスのエラー本文を公開エラーメッセージへ含めない
- エラー本文を `openbd.Error` に保持しない
- 自動リトライとアクセス間隔制御は行わない

## 8. エラー

`openbd.Error.Operation` には次の値を使用する。

```text
openbd.NewClient
openbd.LookupBooksByISBN
```

| 状態 | `ErrorKind` |
| --- | --- |
| ISBN、入力件数、Client設定の不正 | `invalid_argument` |
| HTTP 408、429、500から599 | `unavailable` |
| その他の成功以外のHTTPステータス | `upstream` |
| 通信失敗、タイムアウト | `unavailable` |
| 本文上限超過、JSON、配列件数、ISBN対応の不正 | `invalid_response` |

`Retry-After` は秒数形式とHTTP-date形式を解析する。`context.Canceled` と
`context.DeadlineExceeded` は、ラップ後も `errors.Is` で判定できるようにする。

## 9. 利用条件とサービス変更

openBDのデータは本の販促・紹介目的に限って利用できる。利用者はopenBDの
利用規約へ同意し、削除要請やサービス変更へ対応する必要がある。

利用規約はデータの任意改変を禁止している。本パッケージは取得元値とRaw responseを
保持し、安全性を確認した形式分解だけを共通モデルへ設定するが、具体的な利用方法が
規約へ適合することを保証しない。利用者が用途と表示方法を確認する。

代替APIは従来のURLと応答形式を維持しているが、提供期間、収録範囲、項目、書影は
将来変更される可能性がある。
