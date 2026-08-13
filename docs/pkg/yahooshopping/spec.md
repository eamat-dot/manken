# Yahoo!ショッピングパッケージ仕様

## 1. 範囲

`github.com/eamat-dot/manken/yahooshopping` は、Yahoo!ショッピング商品検索API v3のうち、
Towerが販売する紙書籍だけを検索・ISBN参照する。ほかのストア、電子書籍、在庫・送料・レビューの
共通化は行わない。

## 2. Client

`NewClient` は必須の `WithClientID` と任意の `WithEndpoint` を受け取る。ライブラリは環境変数を読まない。
Client IDはリクエストの `appid` として送信するが、Raw response、Cursor、公開エラーには含めない。
redirectは追従せず、成功本文は16 MiBまでとする。

## 3. 検索とISBN参照

`SearchBooks` と `SearchBooksWithRawResponse` は、`Title`、`Author`、`Publisher`、`FreeText` の非空値を
空白区切りの `query` にして送信する。これらは専用書誌検索ではなく商品キーワードによる近似検索である。
通常検索は常に `seller_id=tower`、`genre_category_id=10251`、`image_size=600` を送信する。`ExcludedText` は未対応であり、
通信前に `invalid_argument` を返す。Limitは1から100、0は20である。Cursorは検索条件と実効Limitに結び付き、
次のリクエストでも `start + results <= 1000` を満たす位置だけを表す。

`LookupBooksByISBN` とRaw response版はISBNを1件だけ受け付ける。ISBN-10はISBN-13へ変換し、
`jan_code`、`seller_id=tower`、`image_size=600` を送信する。ISBN参照にはカテゴリを送信しない。返却JANが要求ISBN-13と
一致した商品だけを返す。

## 4. 変換

Towerかつコミックカテゴリの通常検索結果だけを変換する。全巻セットと全巻セット表記は除外する。
`description` のTower固有ラベルから変換する。`タイトル:`をTitleの一次候補、`タイトルカナ:`を主Titleが`タイトル:`から変更されずに
採用された場合だけTitleReading、
`レーベル:`をPublishers、`発売日:`をReleased日付として扱う。発売日は数値の`releaseDate`より明示ラベルを優先し、
取得元の精度を維持する。`アーティスト:`は役割不明の不完全な寄与者表示であり、原作者などの人物や他の寄与者を
`他`として省略することがある。安全に`、`で分離できる表示済みの人物だけをAuthorsと役割なしContributorsへ設定する。
`他`は人物として扱わず、`アーティストカナ:`は分離後の人数と順序が一致する場合だけContributor.Readingへ設定する。
表示にない人物・役割は推測しない。

`タイトル:`の末尾にある`(15)`、` 17`、` 1巻 (1)`等の明確な巻表示だけをVolumeへ分離する。先頭の`完全版`、
`新装版`、`特装版`、`愛蔵版`は、末尾巻表示を除いた残りのタイトルが明確な場合だけEditionStatementsへ設定する。
巻数がタイトル途中にある、副題やコミックス名との境界が不明、または版表示が重複する場合は、タイトルを分離しない。整数化できる巻数の
Volume.Labelは10進数表記にそろえ、`上`、`下`のような整数化できない巻表示は取得元の表記を保持する。`タイトル:`の巻数または版表示を分離した場合と、
`タイトル:`がないため外側の商品名をTitleとして使う場合は、タイトルカナとの対応を推測せずTitleReadingを設定しない。

有効なISBN-13のJANだけをISBN Identifierへ、ISBN-13でない場合は8桁または13桁のASCII数字だけをJAN Identifierへ変換する。価格は取得時点の
JPY価格であり、税込かどうかは`priceLabel.taxable`がtrueまたはfalseの場合だけ設定する。欠落またはnullでは未設定にする。
商品コードとURLは `BookSource`、`image.small`と`image.medium`、`image_size=600`で要求した`exImage`は `Images`として返す。
`exImage`はURLがある場合だけ追加し、幅・高さが取得できた場合だけその値を設定する。`exImage`の欠落はリクエスト失敗ではない。
媒体は紙書籍として返す。
Raw responseには変換しない販売情報を保持する。

複数の取得元でISBNを基に書誌情報を補う場合は、上位層の取得元組合せで扱う。`yahooshopping.Client`はNDL、openBD、
楽天Booksなどを呼び出さず、結果を統合しない。

## 5. 利用条件

Yahoo!デベロッパーネットワークのClient IDが必要であり、公式の1クエリ/秒とクレジット表示要件に従う。
商品検索API v3のRaw responseを永続保存できるかは公式文書だけでは断定しない。保存・キャッシュ・再配布を行う前に
最新の公式条件を確認する。
