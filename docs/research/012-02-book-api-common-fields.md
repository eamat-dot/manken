# 書籍API共通項目の比較調査

## 1. 目的

Google Books API、openBD、Yahoo!ショッピング商品検索API、DMM.com API、
楽天ブックス書籍検索API、楽天Kobo電子書籍検索APIで取得できる項目を比較し、
`api.Book` を特定の取得元に依存しないモデルへ見直すための論点を整理する。

この文書は 2026-07-30 時点の調査結果であり、確定仕様や実装指示ではない。
`api.Book` の変更、各取得元パッケージの追加、既存APIの互換性対応は別TODOで扱う。

## 2. 調査対象

- 現在の共通モデル: [`api/model.go`](../../api/model.go)
- Google Books API:
  - https://developers.google.com/books/docs/v1/reference/volumes
- openBD:
  - https://openbd.jp/spec/
  - https://api.openbd.jp/v1/schema?pretty
  - 詳細メモ: [`009-openbd-api.md`](009-openbd-api.md)
- Yahoo!ショッピング商品検索API v3:
  - https://developer.yahoo.co.jp/webapi/shopping/v3/itemsearch.html
- DMM.com API:
  - https://affiliate.dmm.com/api/
  - DMM公式組織のGo SDK:
    https://github.com/dmmlabo/dmm-go-sdk/blob/master/api/product.go
- 楽天ブックス書籍検索API:
  - https://webservice.rakuten.co.jp/documentation/books-book-search
- 楽天Kobo電子書籍検索API:
  - https://webservice.rakuten.co.jp/documentation/kobo-ebook-search
- 実装済みサンプル:
  - [`__sample/book-api/internal/apis/google.go`](../../__sample/book-api/internal/apis/google.go)
  - [`__sample/book-api/internal/apis/rakuten.go`](../../__sample/book-api/internal/apis/rakuten.go)
  - [`__sample/book-api/internal/apis/kobo.go`](../../__sample/book-api/internal/apis/kobo.go)

## 3. 先に分けるべき概念

各APIは同じ「本」を返しているように見えるが、返す対象は同一ではない。

### 3.1 書誌レコード

本の内容、版、出版物としての識別情報を表す。

- Google Books
- openBD
- MADB

Google Booksは販売情報も返すが、中心となる `volumeInfo` は書誌情報である。

### 3.2 販売商品

ストアで販売される商品と、その時点の価格、在庫、商品ページを表す。

- Yahoo!ショッピング
- DMM.com
- 楽天ブックス
- 楽天Kobo

同じISBNの紙書籍でも販売店ごとに価格、在庫、商品URLが異なり得る。
電子書籍はISBNを持たず、ストア固有の商品番号だけを持つ場合がある。

販売商品APIから得られる項目も、すべてを同じ扱いにはしない。価格は単話版、
無料試読版などを判定する補助情報として取得元と取得日時を付けて保持する。
一方、在庫、ポイント、送料、配送予定などの販売運用情報は `Book` に含めない。

取得元が返した商品名や価格と、そこから正規化した書誌情報も同じ階層へ混在させず、
取得元の値と正規化結果を分ける。

## 4. API別の取得可能項目

ここでは検索結果1件に含まれる項目を中心に整理する。検索件数、ページ番号、
カーソルなど、検索結果全体の制御情報は `Book` の候補に含めない。

### 4.1 Google Books API

#### 識別・参照

- `id`: Google Books内の巻ID
- `etag`: レスポンス版を識別する不透明な値
- `selfLink`: APIリソースURL
- `volumeInfo.infoLink`: Google Books上の情報ページ
- `volumeInfo.canonicalVolumeLink`: 巻の正規ページ
- `volumeInfo.previewLink`: プレビューページ

#### 書誌

- `volumeInfo.title`
- `volumeInfo.subtitle`
- `volumeInfo.authors[]`
  - 公式仕様上は著者と編集者を含み、役割を区別しない
- `volumeInfo.publisher`
- `volumeInfo.publishedDate`
- `volumeInfo.description`
  - 簡単なHTMLを含む場合がある
- `volumeInfo.industryIdentifiers[]`
  - `ISBN_10`
  - `ISBN_13`
  - `ISSN`
  - `OTHER`
- `volumeInfo.pageCount`
- `volumeInfo.dimensions`
  - `height`
  - `width`
  - `thickness`
- `volumeInfo.printType`
  - `BOOK`
  - `MAGAZINE`
- `volumeInfo.mainCategory`
- `volumeInfo.categories[]`
- `volumeInfo.language`
- `volumeInfo.contentVersion`
  - コンテンツの版識別子であり、書誌上の版表示とは限らない

#### 画像・評価

- `volumeInfo.imageLinks`
  - `smallThumbnail`
  - `thumbnail`
  - `small`
  - `medium`
  - `large`
  - `extraLarge`
- `volumeInfo.averageRating`
- `volumeInfo.ratingsCount`

#### 販売

- `saleInfo.country`
- `saleInfo.saleability`
- `saleInfo.onSaleDate`
- `saleInfo.isEbook`
- `saleInfo.listPrice.amount`
- `saleInfo.listPrice.currencyCode`
- `saleInfo.retailPrice.amount`
- `saleInfo.retailPrice.currencyCode`
- `saleInfo.buyLink`

販売情報は国によって変わり得る。

#### 閲覧・アクセス

- `accessInfo.country`
- `accessInfo.viewability`
- `accessInfo.embeddable`
- `accessInfo.publicDomain`
- `accessInfo.textToSpeechPermission`
- `accessInfo.epub.isAvailable`
- `accessInfo.epub.downloadLink`
- `accessInfo.pdf.isAvailable`
- `accessInfo.pdf.downloadLink`
- `accessInfo.webReaderLink`
- `accessInfo.accessViewStatus`
- `searchInfo.textSnippet`

`userInfo` と `accessInfo.downloadAccess` は利用者や端末に依存するため、
共通の書誌モデルには含めない。

#### 既存サンプルとの差

`__sample/book-api` はタイトル、著者、出版社、刊行日、説明、サムネイル、
プレビューURL、カテゴリだけを読み取っている。ISBN、サブタイトル、言語、
ページ数、寸法、販売情報などはサンプルでは未取得だが、APIからは取得できる。

### 4.2 openBD

openBDの応答は `summary`、`onix`、`hanmoto` に分かれる。詳細な全項目は
[`009-openbd-api.md`](009-openbd-api.md) に整理されているため、ここでは
共通モデルの検討に関係する概念をまとめる。

#### 識別

- ISBN
- `RecordReference`
- `ProductIdentifier`
- 関連旧版の識別子

#### 書誌

- タイトル
- サブタイトル
- 原書名
- シリーズ、コレクション
- 部編番号、巻次
- 版種別、版表示
- 著者、編集者などの寄与者
- 寄与者の役割
- 寄与者略歴
- 発行元、発売元
- 出版日、発売予定日
- 言語、対象国
- ページ数
- 判型、実寸
- 製本
- 商品形態、付録
- 内容紹介、目次、前書き、帯内容
- 主題、Cコード、NDC、ジャンル、キーワード
- 読者対象、成人指定

#### 画像・関連情報

- 表紙などの画像
- 試し読み有無
- 関連書、関連リンク
- 書評情報

#### 販売・供給

- 価格、通貨、価格適用日
- 出版状況
- 在庫状態
- 販売条件
- 重版予定日
- 絶版日

openBDはISBN検索専用であり、タイトル検索結果の1件として使う取得元ではない。
また、同じ概念が `summary`、`onix`、`hanmoto` に重複するため、取得元内で
優先順位と変換規則を決めてから共通モデルへ変換する必要がある。

### 4.3 Yahoo!ショッピング商品検索API

このAPIは書誌APIではなく、汎用の商品検索APIである。

#### 商品

- `hits.name`: 商品名
- `hits.description`: 商品説明
- `hits.headLine`: キャッチコピー
- `hits.code`: ストア管理の商品コード
- `hits.janCode`: JANコード
- `hits.releaseDate`: 発売日
- `hits.condition`: 新品または中古
- `hits.inStock`: 在庫有無
- `hits.url`: 商品URL
- `hits.image.small`
- `hits.image.medium`
- `hits.exImage`: 指定サイズの画像

#### 分類

- `hits.genreCategory`
  - ID
  - 名前
  - 階層
- `hits.parentGenreCategories[]`
- `hits.brand`
- `hits.parentBrands[]`

#### 価格・販売

- 税込価格、税抜価格
- 通常価格、セール価格、プレミアム会員価格
- 定価
- セール期間
- ポイント額、ポイント倍率
- 送料条件
- 支払方法
- 配送地域、締め時間、配送日数
- 商品レビュー件数、平均、URL

#### 販売者

- `hits.seller.sellerId`
- `hits.seller.name`
- `hits.seller.url`
- ストア評価

紙書籍は、JANコードへISBN-13と同じ数字が入る場合でも、API上の項目は
あくまでJANコードである。チェックディジットを検証せずISBNとして扱わない。

`bookfan` の実レスポンスでは、商品名の著者区切りと、商品説明内の `画:`、
`原作:`、`企画・原案:`、`出版社:`、`発売日:`、`シリーズ名等:`、`巻数:`
などに一定の表記が見つかった。すべてのYahoo!商品に適用する汎用推測ではなく、
ショップを限定した変換規則として採用できるか追加調査する。

`ebookjapan` はYahoo!ショッピング上に商品が存在する一方、今回の
商品検索API v3による `seller_id=ebookjapan` の実リクエストでは0件だった。
Yahoo!ショッピング商品検索APIでは電子書籍を検索できないものとして、
`ebookjapan` を実装対象から外す。

実レスポンスと変換候補は
[`011-yahoo-shopping-book-data.md`](011-yahoo-shopping-book-data.md) に分離する。

### 4.4 DMM.com API

DMMの公式APIリファレンスは今回の外部確認では正常表示できなかった。
以下はDMM公式組織のGo SDKが定義する商品情報API v3の応答型に基づく
暫定一覧である。実装前に、利用者がアクセスできる現行リファレンスと
DMMブックスの実レスポンスで再確認する必要がある。

#### 商品

- `content_id`
- `product_id`
- `maker_product`
- `title`
- `volume`
- `date`
- `category_name`
- `comment`
- `isbn`
- `jancode`
- `stock`
- `URL`
- `URLsp`

#### サービス・分類

- `service_name`、`service_code`
- `floor_name`、`floor_code`
- `iteminfo.maker[]`
- `iteminfo.label[]`
- `iteminfo.series[]`
- `iteminfo.keyword[]`
- `iteminfo.genre[]`
- `iteminfo.author[]`
- その他の商品種別向け寄与者・属性

`iteminfo` の各要素はIDと名前を持つ。DMMブックスで `maker` や `label` が
常に出版社や出版レーベルを意味するかは、実レスポンスの確認前に決めない。

#### 画像・サンプル・評価

- `imageURL.list`
- `imageURL.small`
- `imageURL.large`
- `sampleImageURL`
- `sampleMovieURL`
- レビュー件数、平均、レビューURL

#### 価格・販売

- `prices.price`
- `prices.price_all`
- `prices.list_price`
- 配信形式ごとの価格
- アフィリエイトURL

DMMの商品ID、コンテンツID、メーカー商品コードは別の識別子であり、
一律に `Book.ID` へまとめると情報が失われる。

### 4.5 楽天ブックス書籍検索API

#### 書誌

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

`contents` は全集やセット商品の収録内容であり、一般的な目次とは限らない。
`salesDate` は「上旬」「頃」「以降」などを含む場合があるため、日付型へ
強制変換せず原文を保持する必要がある。

#### 商品・販売

- `itemPrice`
- `itemUrl`
- `affiliateUrl`
- 3サイズの商品画像
- `chirayomiUrl`
- `availability`
- `postageFlag`
- `limitedFlag`
- `reviewCount`
- `reviewAverage`
- `booksGenreId`

`listPrice`、`discountRate`、`discountPrice` は公式仕様上0固定であり、
使用しないよう案内されている。

#### 既存サンプルとの差

`__sample/book-api` はタイトル、著者、出版社、発売日、シリーズ名、画像、
商品URL、説明、ジャンル、価格、ISBN、サイズの一部を構造体へ定義している。
ただし、共通型へはISBN、価格、サイズを渡していない。サブタイトル、読み、
収録内容、在庫、試し読み、レビューも未取得である。

### 4.6 楽天Kobo電子書籍検索API

#### 書誌・商品

- `title`
- `titleKana`
- `subTitle`
- `seriesName`
- `author`
- `authorKana`
- `publisherName`
- `itemNumber`
- `itemCaption`
- `salesDate`
- `koboGenreId`

Koboの商品番号はISBNとは限らない。公式の出力項目にはISBNがない。
言語は検索条件として指定できるが、検索結果1件の出力項目には含まれない。

#### 販売

- `itemPrice`
- `itemUrl`
- `affiliateUrl`
- 3サイズの商品画像
- `reviewCount`
- `reviewAverage`
- `salesType`
  - 通常商品
  - 予約商品

#### 既存サンプルとの差

`__sample/book-api` はタイトル、著者、出版社、発売日、シリーズ名、画像、
商品URL、説明、ジャンル、価格を構造体へ定義している。サブタイトル、
タイトルと著者の読み、商品番号、レビュー、販売タイプは未取得である。

楽天BooksとKoboで漫画、BL、TLを分けるジャンルIDと検索条件は
[`012-rakuten-comic-genres.md`](012-rakuten-comic-genres.md) に分離する。

## 5. 共通項目の対応状況

記号の意味は次のとおり。

- `◎`: 明示的な専用項目がある
- `○`: 近い項目はあるが、役割や意味が十分に分かれない
- `△`: 商品名や説明に候補があり、取得元を限定した変換規則の調査が必要
- `-`: 確認した出力項目にない

| 概念 | Google | openBD | Yahoo! | DMM | 楽天Books | 楽天Kobo | MADB |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 取得元内ID | ◎ | ◎ | ◎ | ◎ | ○ | ◎ | ◎ |
| タイトル | ◎ | ◎ | ○ | ◎ | ◎ | ◎ | ◎ |
| サブタイトル | ◎ | ◎ | △ | - | ◎ | ◎ | ◎ |
| シリーズ名 | - | ◎ | ○ | ◎ | ◎ | ◎ | ◎ |
| シリーズID | - | ○ | △ | ◎ | - | - | ◎ |
| 巻数 | △ | ◎ | ○ | ◎ | △ | △ | ◎ |
| 寄与者名 | ○ | ◎ | ○ | ◎ | ○ | ○ | ○ |
| 寄与者の役割 | - | ◎ | ○ | - | - | - | ◎ |
| 出版社 | ◎ | ◎ | ○ | △ | ◎ | ◎ | ◎ |
| 発行レーベル | - | ◎ | △ | ○ | - | - | ◎ |
| ISBN-10 | ◎ | - | - | ○ | ○ | - | ◎ |
| ISBN-13 | ◎ | ◎ | ○ | ○ | ◎ | - | ◎ |
| ISBN以外の識別子 | ◎ | ◎ | ◎ | ◎ | - | ◎ | - |
| 書誌上の版表示 | - | ◎ | △ | - | - | - | ◎ |
| 出版日 | ◎ | ◎ | - | △ | - | - | ◎ |
| 発売日 | ◎ | ◎ | ◎ | ○ | ◎ | ◎ | ○ |
| 説明・内容紹介 | ◎ | ◎ | ○ | ○ | ◎ | ◎ | - |
| 表紙・商品画像 | ◎ | ◎ | ◎ | ◎ | ◎ | ◎ | - |
| 言語 | ◎ | ◎ | - | - | - | - | - |
| 主題・ジャンル | ◎ | ◎ | ◎ | ◎ | ◎ | ◎ | - |
| ページ数 | ◎ | ◎ | - | - | - | - | - |
| 判型・寸法 | ◎ | ◎ | - | ○ | ◎ | - | - |
| 紙・電子などの形式 | ◎ | ◎ | △ | ○ | ◎ | ◎ | - |
| 価格・通貨 | ◎ | ◎ | ◎ | ◎ | ◎ | ◎ | - |
| 在庫・販売状態 | ◎ | ◎ | ◎ | ◎ | ◎ | ○ | - |
| 商品・購入URL | ◎ | - | ◎ | ◎ | ◎ | ◎ | - |
| 試し読みURL | ◎ | ○ | △ | ○ | ◎ | - | - |
| レビュー集計 | ◎ | ○ | ◎ | ◎ | ◎ | ◎ | - |

この表の `○` と `△` は、そのまま実装可能という意味ではない。ショップ固有の
表示規則を使う場合も、実例、例外、失敗時の扱いを取得元パッケージ内で決める。

## 6. 現在の `api.Book` の問題

### 6.1 `ID` の種類を表せない

現在は取得元の識別子を1つだけ `ID` に設定する。DMMは `content_id`、
`product_id`、`maker_product`、ISBN、JANを同時に返し、Googleも巻IDと
複数種別の業界識別子を返す。1項目では識別子の種類と複数性を保持できない。

### 6.2 ISBNだけが特別扱いされている

`ISBN10s` と `ISBN13s` は便利だが、ISSN、JAN、ストア商品番号、
出版社商品コードを追加するたびにフィールドが増える。取得元固有IDとは別に、
種別付きの識別子コレクションが必要である。

### 6.3 `Authors` が寄与者の役割を失う

openBDは著者、原作者、作画、編集などの役割を提供できる。MADBも元データには
役割が含まれる。Googleは著者と編集者を同じ配列で返すため不明な場合もある。
一般化する場合は、役割が分かる取得元の情報を捨てず、不明な取得元では
役割なしとして保持できる形が必要である。

コミカライズ作品では、著者表示が「原作者、作画、キャラクター原案」と
「作画、原作者、キャラクター原案」のどちらになる場合もある。この順序は
表示や同一候補の比較に使う情報であり、役割ごとに並べ替えてはならない。

`Contributors` は取得元が示す順序を維持した配列とし、各要素へ役割を付ける。
それとは別に、既存互換と役割なしの順序付き表示のため `Authors` も残す案を
検討する。取得元が順序を提供しない場合に意味のある順序を推測しない。

### 6.4 シリーズの名前と識別子の対応が崩れる

`SeriesNames` は複数だが、`SeriesID` と `SeriesURL` は単数である。
複数シリーズが返った場合に、どの名前がどのIDとURLへ対応するか表せない。

### 6.5 `PublishedDate` に出版日と発売日が混在する

- Google: `publishedDate`
- openBD: 複数の役割を持つ出版・発売日
- 楽天Books/Kobo: `salesDate`
- Yahoo!: `releaseDate`
- DMM: `date`
- MADB: `datePublished`

これらをすべて `PublishedDate` に入れると、同じフィールドでも意味が異なる。
少なくとも出版日と販売開始日を区別する必要がある。

### 6.6 `SourceURL` に参照URLと商品URLが混在する

MADBのリソースURI、GoogleのAPIリソースURL、Googleの情報ページ、
ストアの商品購入ページは用途が異なる。出典レコードのURLと販売ページURLを
分ける必要がある。

### 6.7 画像、説明、形式、言語を保持できない

複数の取得元が明示的に返す次の項目を現在は保持できない。

- 内容紹介
- 表紙画像
- 言語
- 主題、ジャンル
- ページ数
- 判型、寸法
- 紙書籍、電子書籍などの形式

### 6.8 漫画本への限定は意図したドメイン境界である

`manken` は漫画単行本を検索するライブラリであるため、`Book` と検索型の
「漫画本」という説明は維持する。Google Books、Yahoo!ショッピング、
楽天Books、楽天Koboのタイトル検索では、原作小説や関連書を避けるため、
取得元が提供するカテゴリや形式で漫画単行本へ絞り込む。

openBDのようにISBNというユニークな識別子で1冊を取得する処理では、
タイトル検索時のカテゴリ絞り込みを要求しない。ISBNで取得した対象が
漫画単行本かを検証するか、そのまま返すかは別途決める。

## 7. 推奨するモデル境界

### 7.1 `Book` に置く情報

- 正規化に使用した取得元と取得元内レコードID
- 正規化前の取得元のタイトル、読み、巻数、版表示など
- 正規化したタイトル、読み、サブタイトル
- シリーズと巻数
- 版表示
- 寄与者と役割
- 出版社、発行レーベル
- 種別付き識別子
- 出版日
- 内容紹介
- 表紙画像
- 言語
- 主題、ジャンル
- ページ数、判型、寸法
- 紙、電子などの形式
- 定価、取得時点の価格
- 出典レコードの参照URL

価格は購入機能のためではなく、単話版、無料試読版などを判定する補助情報として
保持する。定価と取得時点の販売価格を区別し、変動する価格には取得日時と取得元を
関連付ける。

### 7.2 `Book` に置かない情報

- 在庫、予約、販売終了などの状態
- 新品、中古
- 送料、配送
- 試し読み、サンプルURL
- 商品レビュー集計
- ポイント、会員別価格
- アフィリエイトURL
- 検索条件、検索カーソル
- Raw response本体

取得元固有の全項目は、取得元パッケージのレスポンス型またはRaw responseに残す。

## 8. モデル案

取得元の情報、正規化済み書誌情報、Raw responseを三層に分ける。

```go
type Book struct {
	Normalized NormalizedBook
	Sources    []BookSource
}
```

- `NormalizedBook`: 利用者が検索、表示、比較に使う正規化済み書誌情報
- `BookSource.Values`: 正規化に使用した取得元の値
- Raw response: 別メソッドで返す無加工のAPIレスポンス

巻数は数値化できる部分を `*int`、上巻、下巻などを文字列で保持する。
`Authors` と `Contributors` は取得元の順序を維持する。ページ数は未取得と0を
区別し、価格は単話版や無料試読版を判定する補助情報として保持する。

項目一覧、補助型、正規化規則、現在の `api.Book` からの変更点は
[`013-book-source-and-normalized-model.md`](013-book-source-and-normalized-model.md)
に分離する。

タイトル、巻数、版表示の実例と分解候補は
[`014-title-volume-edition-examples.md`](014-title-volume-edition-examples.md)
に分離する。

合意したモデルを初期実装へ移す作業は
[`012-redesign-common-book-model.md`](../todo/done/012-redesign-common-book-model.md)
に分離する。

## 9. 推奨する進め方

1. Yahoo!の `bookfan` に対象を限定し、表記規則、カテゴリ、例外率を
   追加調査する。
2. 楽天BooksとKoboは、調査済みの一般、BL、TLジャンルIDを使い、
   取得後の厳密な採用条件を確定する。
3. その他の取得元について、漫画単行本へ限定する検索条件を確定する。
4. 各取得元について、書誌レコードIDと販売商品IDを確定する。
5. `NormalizedBook`、`BookSource`、`Identifier`、`Contributor`、`Series`、
   `Volume`、`Price` の最小契約を決める。
6. 既存 `api.Book` は移行期間を設けず、破壊的に置き換える。
7. `docs/spec.md` へ確定した共通契約だけを反映する。
8. 取得元ごとに非公開のレスポンス型と変換規則を別TODOへ分ける。

Google、楽天Books、楽天Koboの既存サンプルはHTTP呼び出しと実レスポンス確認に
再利用できる。ただし、サンプルの `CommonBook` は表示用に項目を絞った型であり、
新しい共通契約の根拠としてそのまま移植しない。

電子書籍のタイトル検索には、初期実装では楽天Koboを使用する。
Yahoo!ショッピングの `ebookjapan` は商品検索APIで取得できないため採用しない。

DMM.com APIはアフィリエイト登録済みで承認待ちだが、承認されない可能性がある。
承認結果が出るまで調査と実装を保留し、共通モデルの初期見直し対象から外す。
承認された場合は、実レスポンスと利用条件を確認してから追加採用を判断する。

## 10. 調査時点の結論

- `api.Book` は取得元の値と正規化した書誌情報を分ける必要がある。
- APIの全レスポンスは `Book` に埋め込まず、別メソッドで返す。
- `ISBN10s`、`ISBN13s` は種別付き識別子へ一般化する余地が大きい。
- 順序付きの `Authors` を維持し、順序と役割を保持する `Contributors` を
  併設する案が適している。
- `SeriesNames` と単一の `SeriesID`、`SeriesURL` は構造上の不整合がある。
- `PublishedDate` と `SourceURL` は取得元によって意味が変わるため分割が必要である。
- 説明、画像、言語、主題、形式は複数APIで共通して取得でき、追加候補となる。
- 価格は単話版や無料試読版を判定する補助情報として `Book` に保持する。
- 在庫、ポイント、送料などの販売運用情報は `Book` に含めない。
- Yahoo!の表示解析は全面禁止せず、ショップ固有の安定した規則として
  検証できた項目だけを変換候補とする。
- Yahoo!タイトル検索では漫画カテゴリによる絞り込みが必要である。
- ISBN検索ではタイトル検索用のカテゴリ絞り込みを要求しない。
- Yahoo!ショッピングは `bookfan` の紙書籍検索に限定し、
  `ebookjapan` の電子書籍検索には使用しない。
- 電子書籍のタイトル検索には、初期実装では楽天Koboを使用する。
- 楽天BooksとKoboはジャンル検索APIを通常検索から外し、一般、BL、TLの
  ジャンルIDを取得元別の非公開定数として保持する。
- 一般、BL、TLは利用者が明示的に1区分を選び、カテゴリ間を自動遷移しない。
  この指定は検索経路を単純にするために使い、ジャンル併記作品を除外する
  条件にはしない。
- アダルトコミックを明示検索する機能は初期対象外とするが、通常の検索結果へ
  混在した成人向け作品を判定して除外しない。
- DMMはアフィリエイト承認結果が出るまで調査と実装を保留する。
