# 楽天ブックス書籍検索APIの現行仕様とmankenでの利用範囲調査

## 1. 目的

楽天ブックス書籍検索APIの2026年8月時点の現行仕様を確認し、`manken` の
楽天Booksプロバイダで利用する検索能力、書誌変換、Raw response、商品・販売情報、
アフィリエイト情報の扱いを整理する。

この文書は2026年8月8日時点の調査結果であり、確定仕様や実装指示ではない。
後続の実装時には、API仕様と利用条件に変更がないか再確認する。

## 2. 結論

楽天ブックス書籍検索APIは、紙の漫画単行本を検索する次のプロバイダとして有力である。

- タイトル、ISBN、著者を専用条件で検索できる
- `booksGenreId` で漫画カテゴリへ絞り込める
- ISBN、タイトル、サブタイトル、シリーズ、著者、出版社、発売日などを書誌候補として取得できる
- 価格、販売状態、商品URL、画像、試し読みURL、レビューなどの商品情報も取得できる
- 検索にはApplication IDとAccess Keyが必須で、Affiliate IDは任意である
- Affiliate IDを指定した場合だけ `affiliateUrl` が返る
- 1ページ最大30件、最大100ページのページ番号方式である
- 汎用フリーワード検索と除外キーワード検索は、今回対象のBooks Book Search APIにはない

初期実装は「紙書籍の書誌検索＋楽天Booksの商品情報をRaw responseでも利用できる
プロバイダ」とするのが自然である。Affiliate IDを設定していない利用者にも検索を
利用可能にし、アフィリエイトURLは任意機能として扱う。

価格、在庫、レビュー、送料、アフィリエイトURLなどは書誌より更新頻度と利用条件が
厳しい。初期実装で新しい共通販売モデルを追加せず、共通モデルへ安全に対応できる
既存項目とRaw responseを分ける。

## 3. 調査方法と確認水準

次の根拠を区別して確認した。

1. 楽天ウェブサービスの現行公式APIリファレンス
2. 楽天ウェブサービスの利用ガイド、利用規約、ヘルプ
3. 2026年8月8日に有効な認証情報を使った実APIリクエスト
4. 2026年7月31日の既存調査
5. `__sample/book-api` の既存楽天Books実装

主な公式ページ:

- 楽天ブックス書籍検索API:
  https://webservice.rakuten.co.jp/documentation/books-book-search
- 楽天ウェブサービス利用ガイド:
  https://webservice.rakuten.co.jp/guide
- クレジット表示:
  https://webservice.rakuten.co.jp/guide/credit
- 利用規約:
  https://webservice.rakuten.co.jp/guide/rule
- 楽天ウェブサービス ヘルプ:
  https://webservice.faq.rakuten.net/hc/ja

実API確認では `.env` の `RAKUTEN_APP_ID`、`RAKUTEN_ACCESS_KEY`、
`RAKUTEN_AFFILIATE_ID` を使用した。認証値、完全なリクエストURL、
無加工レスポンスは保存していない。

## 4. APIの前提

### 4.1 正式名称、バージョン、エンドポイント

正式名称は「楽天ブックス書籍検索API」で、現行リファレンスのバージョンは
`2017-04-04` である。

REST/JSONのエンドポイント:

```text
https://openapi.rakuten.co.jp/services/api/BooksBook/Search/20170404
```

楽天ブックス総合検索APIより、書籍固有のタイトル、著者、出版社、ISBN、
書籍サイズ、ジャンルなどを指定できる詳細検索APIである。

### 4.2 認証情報

現行APIでは次の認証情報を区別する。

| 設定 | APIパラメータ・ヘッダー | 必須 | 用途 |
| --- | --- | --- | --- |
| `RAKUTEN_APP_ID` | `applicationId` | 必須 | アプリケーション識別 |
| `RAKUTEN_ACCESS_KEY` | `accessKey` | 必須 | APIアクセス認証 |
| `RAKUTEN_AFFILIATE_ID` | `affiliateId` | 任意 | アフィリエイトURL生成 |

`applicationId` と `accessKey` は組で必須である。Access Keyはクエリにも指定できるが、
URL、ログ、エラーメッセージへ露出させにくくするため、`manken` ではHTTPヘッダーの
`accessKey` を使用する候補とする。

Affiliate IDは検索認証ではない。設定しなくても同じ書籍検索APIを利用できる。
楽天の利用ガイドではApplication IDとAccess Keyはアプリごとに発行される一方、
Affiliate IDはデベロッパーごとに1つ発行され、登録した複数アプリで共用すると
案内されている。

したがって後続実装では、Affiliate ID未設定を設定エラーにしてはならない。

### 4.3 Raw responseと認証情報

成功レスポンスの公式出力項目にはApplication ID、Access Key、Affiliate IDの
入力値をそのまま返す項目はない。

ただし、次には認証・アフィリエイト関連情報が含まれ得る。

- 完全なリクエストURLには `applicationId` が含まれる
- Access Keyをクエリに指定した場合は完全なURLにも含まれる
- Affiliate IDを指定した場合はリクエストURLに `affiliateId` が含まれる
- Affiliate IDを指定したレスポンスには追跡用の `affiliateUrl` が含まれる

このため、実装ではAccess Keyをヘッダーへ置き、HTTPクライアントのエラーに
完全なURLを無条件で含めない。Raw response本文を返す機能と、リクエストURLを
診断情報として記録する機能は分けて考える。

## 5. レスポンス形式

### 5.1 `formatVersion`

`format` の既定値は `json` である。

`formatVersion` の既定値は1だが、JSONではバージョン2を使用する方が扱いやすい。

`formatVersion=1`:

```text
items[0].item.title
```

`formatVersion=2`:

```text
items[0].title
```

2026年8月8日の実APIでも `formatVersion=2` で正常に取得できた。
新規実装では明示的に `formatVersion=2` を指定する候補とする。

### 5.2 `elements`

`elements` へカンマ区切りで項目名を指定すると、返却項目を限定できる。

初期実装ではRaw responseを利用できることを優先し、帯域削減のために不用意に
`elements` を固定しない方がよい。必要性が確認できてから最適化する。

## 6. 検索条件

### 6.1 必須となる検索条件

次のうち1つ以上を指定する。

- `title`: 書籍タイトル
- `author`: 著者名
- `publisherName`: 出版社名
- `size`: 書籍サイズ
- `isbn`: ISBN
- `booksGenreId`: 楽天ブックスジャンルID

タイトル、著者、出版社は半角空白で区切った複数キーワードを指定できる。
公式リファレンスは複数キーワード検索を案内しているが、今回確認したページには
AND/ORの意味を明示する指定はない。

### 6.2 タイトル検索

`title` は書籍タイトル専用条件であり、現在の `SearchBooksRequest.Title` と
対応させる候補になる。

2026年8月8日に次を確認した。

```text
title=ふつつかな悪女ではございますが
booksGenreId=001001
hits=3
```

結果:

- HTTP 200
- `count=14`
- `pageCount=5`
- 先頭は漫画単行本
- 先頭商品に13桁ISBNあり

2026年7月31日の同タイトル・同ジャンル調査でも `count=14` であり、
漫画ジャンルの指定により同名の原作小説は先頭確認範囲へ混入しなかった。

ただし、APIが「漫画単行本だけ」「タイトル完全一致だけ」を保証するわけではない。
セット商品、特装版、関連商品などの後段判定は別に必要である。

### 6.3 ISBN検索

`isbn` は専用検索条件である。

2026年8月8日に `9784758088732` を指定すると、HTTP 200、`count=1` で
「上京コロシヤ娘(1)」を取得した。返却ISBNも入力と一致した。

楽天BooksはISBN参照にも利用できる。ただし検索APIの結果として返るため、
後続実装では既存の `LookupBooksByISBN` として公開するか、検索capabilityだけに
含めるかを既存MADB・openBD・Google Booksとの一貫性を見て決める。

### 6.4 著者検索

`author` は著者専用条件である。

2026年8月8日に次を確認した。

```text
author=佐々木倫子
booksGenreId=001001
hits=3
```

結果はHTTP 200、`count=30` で、先頭に「新装版 動物のお医者さん（9）」が返った。
現在の `SearchBooksRequest.Author` と対応させる候補になる。

### 6.5 漫画への絞り込み

既存RESEARCH012-04で確認したジャンルIDは2026年8月時点でも実装候補として維持する。

| 区分 | `booksGenreId` |
| --- | --- |
| 一般コミック | `001001` |
| BLコミック | `001021002` |
| TLコミック | `001029002` |

書籍サイズには `size=9` の「コミック」もある。

漫画検索ではジャンルIDを主な条件とし、`size=9` を固定しない。漫画文庫など別サイズの
商品を落とし得るためである。一方、文庫だけを確認したい場合などには有用なので、TODO032では
`size` を楽天Books固有の任意フィルターとして公開する。共通 `SearchBooksRequest` には追加せず、
楽天BooksのClient Optionとして扱う。ISBN参照には適用しない。

### 6.6 フリーワードと除外条件

Books Book Search APIには、今回確認した範囲で次の専用条件はない。

- 書誌項目をまたぐ汎用 `keyword`
- NOTまたは除外キーワード

そのため、現在の共通検索条件との対応は次の候補になる。

| manken | 楽天Books |
| --- | --- |
| `Title` | `title` |
| `Author` | `author` |
| `FreeText` | 直接対応なし |
| `ExcludedText` | API側の直接対応なし |

`publisherName` は出版社専用条件として利用できる。MADBの `schema:publisher`、Google Booksの
`inpublisher:` とも共通化できる可能性が高いため、共通 `SearchBooksRequest.Publisher` の追加候補とする。
ただし共通API変更と既存プロバイダ修正を伴うため、TODO032には含めず別ブランチ・別TODOで扱う。

`ExcludedText` を取得後フィルタだけで実現すると、API上の `count` とページングの意味が
フィルタ後件数と一致しない。Google BooksやMADBと同じ除外capabilityがあるようには
扱わない。

## 7. 絞り込みと並び順

主な検索制御パラメータは次のとおりである。

| パラメータ | 値・範囲 | 用途 |
| --- | --- | --- |
| `hits` | 1〜30、既定30 | 1ページの件数 |
| `page` | 1〜100、既定1 | ページ番号 |
| `availability` | 0〜6 | 在庫・取り寄せ・予約状態 |
| `outOfStockFlag` | 0/1 | 販売不可商品を含めるか |
| `chirayomiFlag` | 0/1 | 試し読み可能商品のみにするか |
| `sort` | 複数の定義値 | 標準、売上、発売日、価格、レビュー等 |
| `limitedFlag` | 0/1 | 限定商品だけにするか |
| `genreInformationFlag` | 0/1 | ジャンル別件数情報を返すか |

`sort` の主な定義値:

- `standard`
- `sales`
- `+releaseDate`
- `-releaseDate`
- `+itemPrice`
- `-itemPrice`
- `reviewCount`
- `reviewAverage`

共通Sort型は現時点では追加しない。各プロバイダで並び順の意味が大きく異なるためである。
楽天Booksは `standard` の並びに利用者から見た規則性が乏しいため、TODO032では検索時の固定既定値を
`+releaseDate`（発売日の古い順）へ変更する。これは巻数順を保証するものではなく、新装版・特装版などが
混在すれば発売日順として並ぶ。販売状態やレビュー順など他のsort値の公開指定はTODO032の対象外とする。

## 8. ページング

ページングは `hits` と1始まりの `page` を使用する。

- `hits`: 1〜30
- `page`: 1〜100
- `pageCount`: 最大100

最大到達件数は条件上30件×100ページで3,000件である。ただし、検索結果の変動により
ページ間の内容が変わる可能性がある。

現在の `SearchBooksRequest.Cursor` へ対応させる場合、カーソルには少なくとも
「次のpage」を不透明な文字列として持たせる候補がある。`hits` が途中で変わると
ページ境界が変わるため、Google Booksと同様にカーソルから継続条件を復元できる形が
望ましい。

## 9. 0件とエラー

### 9.1 0件

2026年8月8日に存在しないタイトルを検索した結果:

- HTTP 200
- `count=0`
- `pageCount=0`
- `items` は空

公式エラー表にはHTTP 404 `not_found` も記載されているが、通常の検索0件が必ず
404になるわけではない。後続実装では、HTTP 200かつ `count=0` を正常な空結果として
扱う。

### 9.2 公式エラー

現行公式リファレンスの主なHTTPエラー:

| HTTP | APIエラー | 意味 | 共通エラー候補 |
| ---: | --- | --- | --- |
| 400 | `wrong_parameter` | 不正・不足パラメータ | invalid request |
| 404 | `not_found` | API側でデータなし | not found候補。ただし検索0件と分ける |
| 429 | `too_many_requests` | リクエスト超過 | rate limit |
| 500 | `system_error` | 楽天側内部エラー | provider/server error |
| 503 | `service_unavailable` | メンテナンス・過負荷 | unavailable |

既存の共通 `ErrorKind` との正確な対応は実装TODOで現行定義を確認して決める。
HTTP 404をISBN参照の未発見と同一視する場合も、実APIのISBN未発見挙動を追加確認する。

## 10. 実APIで確認した代表ケース

2026年8月8日、`formatVersion=2`、Access KeyをHTTPヘッダーに指定し、
1リクエストごとに1秒以上間隔を空けて確認した。

| ケース | HTTP | `count` | 確認内容 |
| --- | ---: | ---: | --- |
| タイトル+一般コミック | 200 | 14 | 漫画、ISBN、商品URLを取得 |
| ISBN | 200 | 1 | 入力ISBNと一致する商品を取得 |
| 著者+一般コミック | 200 | 30 | 著者検索が有効 |
| タイトル+一般コミック、Affiliate IDあり | 200 | 14 | 同じ検索結果で `affiliateUrl` が出現 |
| 存在しないタイトル | 200 | 0 | 正常な空結果 |

Affiliate IDなしのタイトル検索と、Affiliate IDありの同条件検索では、
`count` と先頭商品は同じだった。

Affiliate IDなし:

- `affiliateUrl` は空
- `itemUrl` のホストは `books.rakuten.co.jp`

Affiliate IDあり:

- `affiliateUrl` が返る
- アフィリエイトURLのホストは `hb.afl.rakuten.co.jp`
- 通常の `itemUrl` も引き続き返る

この実測からも、Affiliate IDは検索機能の必須設定ではなく、アフィリエイトリンクを
必要とする利用者だけが設定する構成にできる。

## 11. 主要な取得項目

### 11.1 書誌・分類

公式出力には次がある。

- `title`
- `titleKana`
- `subTitle`
- `subTitleKana`
- `seriesName`
- `seriesNameKana`
- `contents`
- `contentsKana`
- `author`
- `authorKana`
- `publisherName`
- `size`
- `isbn`
- `itemCaption`
- `salesDate`
- `booksGenreId`

注意点:

- `contents` は全集・セットなどの収録内容で、一般的な目次とは限らない
- `salesDate` は年月日だけでなく「上旬」「中旬」「下旬」「頃」「以降」等を含み得る
- `booksGenreId` は複数所属時に `/` 区切りで最下位ジャンルIDが返る
- `author` と `authorKana` は単一文字列で、複数寄与者の区切りや役割を公式出力表は定義していない

公式仕様は複数寄与者の区切りを定義していないが、サンプルプログラム作成時の実データ確認で
`author` と `authorKana` が複数人の場合に `/` 区切りで返ることを確認している。
TODO032では `/` で人物単位へ分割し、各要素の前後空白だけを除去する。氏名内部の半角空白、
全角空白、カンマなどは表記揺れとして残し、この変更では統一しない。Contributorの役割も推測しない。

### 11.2 `size` の公式表記と実レスポンス差

検索入力の `size` は整数で、`9` がコミックである。公式の出力表も1〜10の番号で
サイズを説明している。

一方、2026年8月8日のJSON実レスポンスでは `size` に数値ではなく日本語文字列
`"コミック"` が返った。

したがって実装のレスポンス型は、現行実データに合わせて文字列として扱う候補とする。
公式表の番号へ逆変換する前提を置かない。

### 11.3 商品・販売

公式出力には次がある。

- `itemPrice`: 税込価格
- `itemUrl`: 通常の商品URL
- `affiliateUrl`: Affiliate ID指定時だけ返る
- `smallImageUrl`
- `mediumImageUrl`
- `largeImageUrl`
- `chirayomiUrl`
- `availability`
- `postageFlag`
- `limitedFlag`
- `reviewCount`
- `reviewAverage`

`listPrice`、`discountRate`、`discountPrice` は2013年11月28日以降0固定と案内され、
使用しないよう明記されている。正規化候補にしない。

## 12. `NormalizedBook` への変換候補

現行共通モデルへ、項目名だけでなく意味を確認できるものだけを対応させる。

### 12.1 初期変換候補

| 楽天Books | manken候補 | 判断 |
| --- | --- | --- |
| `title` | `Normalized.Title` | 書籍タイトルとして明示 |
| `titleKana` | `Normalized.TitleReading` | タイトル読みとして明示 |
| `subTitle` | `Normalized.Subtitle` | 専用項目あり |
| `seriesName` | `Normalized.Series[].Name` | 現行実装では保持するが、作品シリーズ以外の出版コレクション・レーベル相当値が混在する可能性がある |
| `publisherName` | `Normalized.Publishers` | 出版社として明示 |
| `isbn` | `Normalized.Identifiers` | ISBNとして明示。形式検証して種別を決める |
| `salesDate` | `Normalized.Dates` の発売日 | 原文精度を保持する |
| `itemCaption` | `Normalized.Description` | 商品説明として明示。ただし利用条件を確認して表示する |
| `booksGenreId` | `Normalized.Subjects` | 楽天Books固有Schemeの分類コードとして保持可能 |
| `itemPrice` | `Normalized.Prices` | 税込の取得時点価格として意味は明確。ただし保存・更新条件あり |
| 画像URL | `Normalized.Images` | 商品画像として保持可能。利用条件に従う |

`itemPrice` は `PriceTypeCurrent`、通貨JPY、税込、取得日時付きで `Prices` へ変換できる。
TODO031のレビュー時にこの対応を採用した。価格情報のキャッシュ期限が24時間のため、guideで
利用条件を明示する。

`seriesName` は `シリウスKC`、`HOWLコミックス` などレーベル・出版コレクションに見える値を
含み得る一方、情報を捨てると利用者やLLMが版・系統を判断しにくくなる。TODO032では現行の
`Normalized.Series` への保持を変更しない。未実装プロバイダの同種項目が出そろった後に、
`Series` 自体の意味、出版コレクション、Imprintとの境界を共通モデルとして再検討する。

### 12.2 実装前に追加確認する項目

#### 著者

複数人の `author` / `authorKana` が `/` 区切りで返ることはサンプルプログラム作成時に確認済みである。
TODO032では `author` を `/` で分割して `Authors` と人物単位の `Contributors` へ変換する。
`authorKana` も同様に分割し、著者と読みの件数が一致する場合だけ同じ位置の人物へ `Reading` として対応付ける。
件数が一致しない場合は読みを推測で割り当てない。氏名内部の空白・カンマ等はこの変更では正規化しない。

#### `size`

実レスポンスは `"コミック"` だったため、`PhysicalSize.Name` として現行どおり保持する。
検索入力の数値コードは出力文字列とは別概念として扱い、TODO032で楽天Books固有の任意フィルターを追加する。

#### 紙媒体

楽天ブックス書籍検索APIは紙書籍の商品検索に用いるため、漫画ジャンルへ限定した結果を
`PublicationMediumPrint` とする候補がある。サイズ種別に書籍以外の物理媒体も含まれるため、
無条件に全結果を紙とするより検索条件と結果を確認して設定する。

### 12.3 初期共通化を見送る項目

次はRaw responseへ残し、現行の共通モデルへ無理に入れない。

- `availability`
- `postageFlag`
- `limitedFlag`
- `reviewCount`
- `reviewAverage`
- `chirayomiUrl`
- `contents` / `contentsKana`

在庫、送料、限定販売、レビューは書誌ではなく販売運用情報である。
`chirayomiUrl` も通常の商品参照URLとは用途が異なるため、現時点ではRawに残す。

## 13. 取得元IDとURL

### 13.1 取得元内ID

Books Book Search APIの公式出力項目には、楽天Booksの商品専用IDとして明示された
安定IDが見当たらない。

ISBNを `BookSource.ID` へ流用すると、取得元固有IDと標準書誌識別子の意味が混ざる。
ISBNは `Identifiers` に保持し、楽天固有IDを公式に確認できない限り `BookSource.ID` を
空にする候補とする。

商品URLのpathからIDらしい文字列を抽出して独自IDにしない。

### 13.2 通常商品URL

`itemUrl` は楽天ブックスの商品ページURLで、2026年8月8日の実APIでは
`books.rakuten.co.jp` を指した。

通常の商品参照URLとして `BookSource.URL` へ設定する候補になるが、書誌参照URLと
販売商品URLの責務はbacklogでも未整理である。楽天Booksの実装だけを理由に共通型を
拡張せず、現行 `BookSource.URL` の意味と整合する範囲で使用する。

### 13.3 アフィリエイトURL

`affiliateUrl` はAffiliate IDを指定した場合だけ返る。通常の `itemUrl` とは別情報である。

TODO031のレビュー時に、販売系プロバイダでも共通利用できる取得元参照情報として
`BookSource.AffiliateURL` を追加する方針を採用した。

- `BookSource.URL`: 通常の商品参照URL
- `BookSource.AffiliateURL`: アフィリエイト用URL

アフィリエイトURLがある場合も通常URLを置き換えず、両方を別項目として保持する。

## 14. Raw response

楽天Booksは商品・販売情報を多く持つため、LLM/MCP用途でもRaw responseの価値が高い。
初期実装でも、実装済みプロバイダと同様にNormalizedとRawを別経路で利用できる構成を
維持する候補とする。

ただしRawには次の注意がある。

- Affiliate IDを指定すると `affiliateUrl` が含まれる
- 価格・販売可能情報は変動し、キャッシュ期間の条件がある
- その他の楽天由来情報にも保存期間の条件がある
- 利用者がRawを永続保存する場合も楽天ウェブサービスの利用条件を確認する必要がある

ライブラリがRawを返せることと、利用者が無期限に保存・再配布できることは同じではない。

## 15. 利用制限と表示・保存条件

楽天Booksは技術的に取得できる項目だけでなく、利用条件を利用者向け文書へ案内する必要が
あるプロバイダである。

### 15.1 リクエスト制限

楽天ウェブサービスのヘルプでは、1つの `application_id` につき
「1秒に1回以下」のリクエストとするよう案内している。

2026年8月8日の調査リクエストも1秒以上間隔を空けた。

後続実装では、ライブラリ内部で自動的な全プロバイダ横断連打を行う場合にこの制限を
超えない設計が必要になる。単体Clientで待機を強制するか、呼び出し側責務とするかは
実装TODOで決める。

### 15.2 キャッシュ期間

楽天ウェブサービスのヘルプでは一般的なキャッシュ期間を次のように案内している。

- 商品の価格情報・販売可能情報: 24時間
- その他の情報: 3か月

また、価格または販売可能情報を表示する場合は少なくとも1週間に1回再取得して刷新し、
1時間に1回以上更新しない場合は更新日時と所定の注意事項を表示するよう案内している。

この条件は、Raw responseや価格をファイルへ保存する利用例にも影響する。
`manken` のguideでは「APIから返した値を利用者が保存する場合の条件」として案内する必要が
ある。

### 15.3 クレジット表示

楽天のクレジット表示ガイドでは、楽天APIを使用するすべてのアプリで指定のバッジまたは
テキストによるクレジット表示が必要と案内している。

ライブラリ自体にUIがない場合でも、楽天Books providerを利用して結果を表示する
アプリケーション側が要件を把握できるようguideに公式ページへの導線を置く。

### 15.4 利用目的と収益化

楽天ウェブサービス利用規約と公式ヘルプは、APIで取得した情報の利用目的や収益化について
制約を設けている。

公式ヘルプでは、楽天APIを利用した有償ツール販売は、楽天アフィリエイト以外の方法で
API利用から収入を得ることに当たり規約違反になると案内している。

また、楽天ウェブサービスで取得した商品と他社Webサービスの商品を同じ画面へ併置すること
自体は、提供元を表示すれば可能と案内されている。一方、楽天APIから取得した情報を使って
楽天以外のアフィリエイトリンクから収益を得る用途には制約がある。

`manken` は複数プロバイダを扱うため、この点は特に重要である。楽天由来データと他社由来
データの出典を失わず、利用者へ楽天の現行規約確認を促す必要がある。

この調査は規約の法的解釈を確定するものではない。利用形態ごとの可否が重要になる場合は、
楽天の現行規約・ヘルプを一次情報として確認する。

## 16. 既存資料・既存サンプルとの差

### 16.1 RESEARCH012系

2026年7月31日の主要な調査結果は現在も利用できる。

- 一般コミック `001001`
- BLコミック `001021002`
- TLコミック `001029002`
- タイトル+ジャンル検索が漫画単行本候補の取得に有効
- ISBNを取得できる
- 楽天BooksはKoboの `NGKeyword` に相当する除外条件を持たない

TODO030で追加・明確化した主な点:

- Application IDに加えてAccess Keyも現行APIでは必須
- Affiliate IDは任意
- `formatVersion=2` を現行実APIで確認
- ISBN・著者検索を2026年8月8日に再実測
- `hits=1..30`、`page=1..100`
- 0件がHTTP 200 + `count=0` となる実例
- 商品価格・販売可能情報のキャッシュ条件
- 1 Application IDあたり1秒1回以下のリクエスト制限
- クレジット表示と利用目的に関する条件

### 16.2 `__sample/book-api`

既存サンプルでそのまま参考にできる点:

- `BooksBook/Search/20170404` エンドポイント
- `applicationId` の利用
- Access KeyをHTTPヘッダーで送る構成
- ISBNを他の検索条件より優先する分岐の考え方

変更・再設計が必要な点:

- Affiliate IDの任意設定がない
- `formatVersion=2` を明示していないため、旧来の `Items[].Item` 構造を前提にしている
- `author` を `/` で無条件に分割しているが、公式仕様上の区切り保証を確認できていない
- 検索ごとにジャンル名APIを呼ぶため、1秒1回以下のリクエスト制限と相性が悪い
- 既知の漫画ジャンルIDはローカル定数で足りるため、通常検索ごとのジャンルAPI呼び出しは不要
- 楽天固有の商品・販売情報と共通書誌を分離していない
- `PublishedDate` に `salesDate` を直接入れており、発売日と出版日を区別していない

## 17. mankenでのcapability評価

2026年8月8日時点の評価:

| capability | 評価 | 根拠 |
| --- | --- | --- |
| タイトル検索 | ○ 実測済み | `title` 専用条件 |
| ISBN検索 | ○ 実測済み | `isbn` 専用条件 |
| 著者検索 | ○ 実測済み | `author` 専用条件 |
| フリーワード検索 | × | Books Book Searchに汎用keywordなし |
| 除外条件 | × | 専用NOT/NGKeywordなし |
| 漫画絞り込み | ○ 実測済み | `booksGenreId`、`size=9`候補 |
| ページング | ○ 仕様確認済み | `hits` 1〜30、`page` 1〜100 |
| NormalizedBook | ○ 候補多数 | 書誌専用項目が多い |
| Raw response | ○ 実測済み | JSON `formatVersion=2` |
| 価格 | ○ | 税込 `itemPrice` |
| 販売状態 | ○ | `availability` 等 |
| 通常商品URL | ○ 実測済み | `itemUrl` |
| 画像 | ○ | 3サイズ |
| アフィリエイトURL | ○ 任意・実測済み | Affiliate ID指定時 `affiliateUrl` |

## 18. 後続の楽天Books実装TODOへの入力

後続実装では少なくとも次を対象候補とする。

### Client設定

- Application ID: 必須
- Access Key: 必須、HTTPヘッダーで送る
- Affiliate ID: 任意
- HTTP Clientの差し替え

Affiliate ID未設定でもClientを作成・検索できることをテストする。

### 検索

- `Title` -> `title`
- `Author` -> `author`
- 漫画ジャンルをプロバイダ内部条件として利用
- `FreeText`、`ExcludedText` を楽天Booksが未対応であることを明確にする
- Limitを1〜30へ変換する規則
- `page` を不透明Cursorへ変換する規則
- 0件を正常な空結果として扱う

### ISBN参照

- `isbn` を使った参照
- ISBN形式検証
- 1 ISBNあたりの検索結果の扱い
- ISBN未発見時の実レスポンス追加確認

### レスポンス

- `formatVersion=2`
- `size` を文字列で受けられる実装
- 欠落可能項目を正常として扱う
- APIレスポンスに楽天固有IDがないことを前提に、URLからIDを推測しない

### 変換

- タイトル、読み、サブタイトル、シリーズ、出版社、ISBN、発売日、説明、ジャンル
- 複数著者の分割規則を実例で追加確認
- `salesDate` の精度を保持し、完全な日付へ補完しない
- `itemPrice` は税込・JPY・取得時点価格として扱う
- アフィリエイトURLは `BookSource.URL` に入れず、`BookSource.AffiliateURL` に分離する

### Raw response

- NormalizedとRawを別経路で利用可能にする
- Affiliate IDあり・なしの双方をテストする
- 完全なリクエストURLをエラーやログへ出さない
- 楽天由来データの保存条件をguideへ案内する

### 利用者向け文書

- Application IDとAccess Keyが必須
- Affiliate IDは任意
- 1秒1回以下のリクエスト制限
- クレジット表示への導線
- データの保存・更新条件への導線
- 楽天アフィリエイト利用時の公式条件への導線

## 19. 未決事項

TODO030時点の未決事項のうち、TODO031で次を確定した。

- `itemPrice` は `Normalized.Prices` の取得時点価格として共通化する
- 通常商品URLは `BookSource.URL`、アフィリエイトURLは `BookSource.AffiliateURL` に分離する
- `size=9` は固定せず、一般・BL・TLのジャンルIDをClientで選択する
- 1秒1回以下の制御はライブラリ内部で行わず、呼び出し側へ委ねる

今後の確認事項は次のとおり。

- `author` / `authorKana` の複数寄与者区切りをどこまで安全に分解できるか
- `size` の出力が常に日本語文字列か、条件によって数値等が返ることがあるか
- 在庫、送料、レビューなど未共通化の販売情報をどこまで共通モデルへ含めるか
