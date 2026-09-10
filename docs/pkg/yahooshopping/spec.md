# Yahoo!ショッピングパッケージ仕様

## 1. 範囲

`yahooshopping` パッケージは、Yahoo!ショッピング商品検索API v3のうち、Towerが販売する紙書籍だけを検索・ISBN参照する。

ほかのストアと電子書籍は対象にしない。在庫・送料・レビューは `Book` へ変換しない。書誌変換はTower固有の `description` ラベルと商品表記を前提にするため、`seller_id` を利用者設定として公開せず、別ストアへ同じ変換規則を推測して適用しない。

## 2. Client

`NewClient` は必須の `WithClientID` と任意の `WithEndpoint` を受け取る。ライブラリは環境変数を読まない。

Client IDはリクエストの `appid` として送信するが、Raw response、Cursor、公開エラーには含めない。リダイレクトは追従せず、成功本文は16 MiBまでとする。

## 3. 検索とISBN参照

`SearchBooks` と `SearchBooksWithRawResponse` は、`Title`、`Author`、`Publisher`、`Query` の空でない値を空白区切りの `query` にして送信する。これらは専用の書誌検索ではなく、商品キーワードによる近似検索である。

通常検索では常に次のパラメーターを送信する。

- `seller_id=tower`
- `genre_category_id=10251`
- `image_size=600`

`Exclude` は未対応であり、指定すると通信前に `invalid_argument` を返す。`Limit` は1から100、0は20である。Cursorは検索条件と実効Limitに結び付き、次のリクエストでも `start + results <= 1000` を満たす位置だけを表す。

`DateFrom` と `DateTo` は、Tower書籍の時期検索に利用できる取得元条件を確認できないため、通信前に `invalid_argument` を返す。`sale_start_*` と取得後の絞り込みは使わない。

`LookupBooksByISBN` と `LookupBooksByISBNWithRawResponse` はISBNを1件だけ受け付ける。ISBN-10はISBN-13へ変換し、`jan_code`、`seller_id=tower`、`image_size=600` を送信する。ISBN参照にはカテゴリを送信しない。返却されたJANが要求ISBN-13と一致する商品だけを返す。

## 4. `Book` への変換

通常検索では、Towerかつコミックカテゴリの商品だけを変換する。全巻セットと、全巻セットを示す商品表記は除外する。

Tower固有の `description` ラベルは次のように扱う。

- `タイトル:` は `Title` の第一候補とする
- `タイトルカナ:` は、`タイトル:` を変更せず `Title` に採用した場合だけ `TitleReading` とする
- `レーベル:` は `Publishers` とする
- `発売日:` は `ReleaseDate` とする

`レーベル:` は出版社名を返す場合があるため、ラベル名だけを根拠に `PublicationSeries` や `BookSeries` へ変換しない。この規則はTower固定の範囲だけに適用する。

`releaseDate` はstring以外に `null` やUnix秒数値が返る場合があるため、発売日は型と表記が明示された `発売日:` ラベルを優先し、取得元の精度を維持する。

`アーティスト:` は役割不明の不完全な寄与者表示であり、原作者などの人物や他の寄与者を `他` として省略することがある。安全に `、` で分離できる表示済みの人物だけを `Authors` と役割なしの `Contributors` へ設定する。`他` は人物として扱わない。`アーティストカナ:` は分離後の人数と順序が一致する場合だけ `Contributor.Reading` へ設定する。表示にない人物・役割は推測しない。

Tower固有のタイトル選択後、安全なタイトル構文から、明確な巻表示、確認済み版表示、巻表示へ隣接する完結表示を `Volume`、`Editions`、`IsFinalVolume` へ補う。分冊、単話、セット、合本、無料試読、`Vol.N` は `Volume` へ推測しない。

解析結果を得ても `Title` は `タイトル:` の取得元文字列をそのまま保持し、巻数や版表示を除去しない。括弧付き数字は、後続が空、確認済み版表示、または安全な完結表示だけの場合に `Volume` へ採用する。整数化できる巻数の `Volume.Label` は10進数表記にそろえ、`上巻`、`下巻` のような整数化できない巻表示も末尾の `巻` を除いた `上`、`下` へ正規化する。

`タイトル:` がある場合だけ、同じ説明内の `タイトルカナ:` を `TitleReading` へ設定する。`タイトル:` がなく外側の商品名を `Title` として使う場合は、タイトルカナとの対応を推測しない。

有効なISBN-13のJANだけを `ISBN13` へ設定する。ISBN-13でない場合は、有効なチェックディジットを持つ8桁または13桁のASCII数字だけを `JAN` へ変換する。

価格は取得時点のJPY価格として `CurrentPrice` に設定する。税込かどうかは `priceLabel.taxable` が `true` または `false` の場合だけ保持し、欠落または `null` では未設定にする。

商品コードとURLは `BookSource` に保持する。表紙画像は `image_size=600` で要求した `exImage`、`image.medium`、`image.small` の順で利用可能な最初のURLを `CoverURL` として返す。`exImage` の欠落はリクエスト失敗ではない。媒体は紙書籍として返す。

Raw responseには `Book` へ変換しない販売情報を保持する。

複数の取得元からISBNを基に情報を補う処理は呼び出し側で行う。`yahooshopping.Client` はNDL、openBD、楽天Booksなどを内部で呼び出さず、結果を統合しない。

## 5. 利用条件

Yahoo!デベロッパーネットワークのClient IDが必要であり、公式の1クエリ/秒とクレジット表示要件に従う。

商品検索API v3のRaw responseを永続保存できるかは公式文書だけでは断定しない。保存・キャッシュ・再配布を行う前に、最新の公式条件を確認する。
