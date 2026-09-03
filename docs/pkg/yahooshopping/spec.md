# Yahoo!ショッピングパッケージ仕様

## 1. 範囲

`github.com/eamat-dot/manken/yahooshopping` は、Yahoo!ショッピング商品検索API v3のうち、
Towerが販売する紙書籍だけを検索・ISBN参照する。ほかのストア、電子書籍、在庫・送料・レビューの
共通化は行わない。書誌変換はTower固有の `description` ラベルと商品表記を前提にするため、
`seller_id` を利用者設定として公開せず、別ストアへ同じ変換規則を推測適用しない。

## 2. Client

`NewClient` は必須の `WithClientID` と任意の `WithEndpoint` を受け取る。ライブラリは環境変数を読まない。
Client IDはリクエストの `appid` として送信するが、Raw response、Cursor、公開エラーには含めない。
redirectは追従せず、成功本文は16 MiBまでとする。

## 3. 検索とISBN参照

`SearchBooks` と `SearchBooksWithRawResponse` は、`Title`、`Author`、`Publisher`、`Query` の非空値を
空白区切りの `query` にして送信する。これらは専用書誌検索ではなく商品キーワードによる近似検索である。
通常検索は常に `seller_id=tower`、`genre_category_id=10251`、`image_size=600` を送信する。`Exclude` は未対応であり、
通信前に `invalid_argument` を返す。Limitは1から100、0は20である。Cursorは検索条件と実効Limitに結び付き、

`DateFrom` と `DateTo` はTower書籍の時期検索に利用できる取得元条件を確認できないため、通信前に `invalid_argument` を返す。`sale_start_*` と取得後の絞り込みは使わない。
次のリクエストでも `start + results <= 1000` を満たす位置だけを表す。

`LookupBooksByISBN` とRaw response版はISBNを1件だけ受け付ける。ISBN-10はISBN-13へ変換し、
`jan_code`、`seller_id=tower`、`image_size=600` を送信する。ISBN参照にはカテゴリを送信しない。返却JANが要求ISBN-13と
一致した商品だけを返す。

## 4. 変換

Towerかつコミックカテゴリの通常検索結果だけを変換する。全巻セットと全巻セット表記は除外する。
`description` のTower固有ラベルから変換する。`タイトル:`をTitleの一次候補、`タイトルカナ:`を主Titleが`タイトル:`から変更されずに
採用された場合だけTitleReading、
`レーベル:`をPublishers、`発売日:`をReleaseDateとして扱う。`レーベル:` は出版社名を返す場合があるため、
ラベル名だけを根拠に `PublicationSeries` や `BookSeries` へ変換しない。この規則はTower固定の範囲だけに適用する。
`releaseDate` はstring以外にnullやUnix秒数値が返る場合があるため、発売日は型と表記が明示された `発売日:` ラベルを優先し、
取得元の精度を維持する。`アーティスト:`は役割不明の不完全な寄与者表示であり、原作者などの人物や他の寄与者を
`他`として省略することがある。安全に`、`で分離できる表示済みの人物だけをAuthorsと役割なしContributorsへ設定する。
`他`は人物として扱わず、`アーティストカナ:`は分離後の人数と順序が一致する場合だけContributor.Readingへ設定する。
表示にない人物・役割は推測しない。

Tower固有のタイトル選択後、共通の保守的なタイトル解析で明確な巻表示、確認済み版表示、巻表示へ隣接する完結表示をVolume、Editions、IsFinalVolumeへ補う。分冊、単話、セット、合本、無料試読、Vol.NはVolumeへ推測しない。
解析結果を得ても `Title` は `タイトル:` の取得元文字列をそのまま保持し、巻数や版表示を除去しない。括弧付き数字は後続が空、確認済み版表示、または安全な完結表示だけの場合にVolumeへ採用する。整数化できる巻数の
Volume.Labelは10進数表記にそろえ、`上巻`、`下巻`のような整数化できない巻表示も末尾の`巻`を除いた`上`、`下`へ正規化する。`タイトル:`がある場合だけ同じ説明内の`タイトルカナ:`をTitleReadingへ設定し、
`タイトル:`がないため外側の商品名をTitleとして使う場合はタイトルカナとの対応を推測しない。

有効なISBN-13のJANだけを `ISBN13` へ、ISBN-13でない場合は有効なチェックディジットを持つ8桁または13桁のASCII数字だけを `JAN` へ変換する。価格は取得時点の
JPY価格として `CurrentPrice` に設定し、税込かどうかは`priceLabel.taxable`がtrueまたはfalseの場合だけ保持する。欠落またはnullでは未設定にする。
商品コードとURLは `BookSource`、表紙画像は `image_size=600` で要求した `exImage`、`image.medium`、`image.small` の順で利用可能な最初のURLを `CoverURL` として返す。
`exImage`の欠落はリクエスト失敗ではない。
媒体は紙書籍として返す。
Raw responseには変換しない販売情報を保持する。

複数の取得元でISBNを基に書誌情報を補う場合は、上位層の取得元組合せで扱う。`yahooshopping.Client`はNDL、openBD、
楽天Booksなどを呼び出さず、結果を統合しない。

## 5. 利用条件

Yahoo!デベロッパーネットワークのClient IDが必要であり、公式の1クエリ/秒とクレジット表示要件に従う。
商品検索API v3のRaw responseを永続保存できるかは公式文書だけでは断定しない。保存・キャッシュ・再配布を行う前に
最新の公式条件を確認する。
