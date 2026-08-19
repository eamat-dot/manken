# DMMパッケージ仕様

`SearchSeriesRequest.DateFrom` と `DateTo` は、DMM商品 `date` の時期を `gte_date` と `lte_date` で絞り込む。開始は指定期間の `00:00:00`、終了は期間終端の `23:59:59` とし、タイムゾーン変換は行わない。FreeTextは引き続き必須で、日付条件はCursor照合に含める。

`dmm` はDMM.com Webサービス v3を使い、一般向けDMMブックス電子コミックのシリーズを探し、指定したシリーズに属する個別商品を取得する。import pathは `github.com/eamat-dot/manken/dmm` であり、FANZAは対象にしない。

## 公開APIとClient

`dmm` は `Client`、`Option`、`Book`、`BookSeries`、`Source`、`PublicationMedium`、`ErrorKind`、`Error` など、通常利用に必要な `model` の型をエイリアスとして公開する。`SeriesSearchItem` はDMM固有のシリーズ候補型である。`SourceDMM` はDMM.com Webサービス v3 ItemListを、`PublicationMediumDigital` は電子書籍を表す。

`NewClient(httpClient, dmm.WithAPIID(...), dmm.WithAffiliateID(...))` はAPI IDとAffiliate IDの両方を必要とする。ライブラリ本体は環境変数を読まない。認証値が空白だけ、または制御文字を含む場合は `invalid_argument` となる。`httpClient` がnilの場合は60秒のTimeoutを使い、呼び出し側から渡されたHTTPクライアントは変更しない。リダイレクトは自動追従しない。

`WithEndpoint` は通常使用しない。HTTPSの絶対URLを受け付け、テスト用のlocalhostまたはloopbackへのHTTPだけを許可する。user information、query、fragment付きURLは拒否する。

## シリーズ検索

`SearchSeries` は変換済みシリーズを、`SearchSeriesWithRawResponse` は同じ結果に加えて秘匿済みRaw responseを返す。両メソッドは `site=DMM.com`、`service=ebook`、`floor=comic`、`sort=rank`、`output=json` を固定してItemListを呼ぶ。

| `SearchSeriesRequest` | DMM       | 動作                                                        |
| --------------------- | --------- | ----------------------------------------------------------- |
| `FreeText`            | `keyword` | 必須。DMMのフリーワードとして空白を1つにして送る            |
| `ExcludedText`        | `keyword` | 空白区切りの各語を取得元の `-語` として送る                 |
| `Limit`               | `hits`    | 0は20、1から100を許可する                                   |
| `Cursor`              | `offset`  | 不透明Cursor経由で扱う                                      |

`FreeText` が空、または各条件の語に `|`、`"`、先頭 `-` を含む場合は、通信前に `invalid_argument` を返す。DMMのリテラルエスケープ方法が未確認のためである。

keyword応答itemは商品ではなく、そのitemが明示する唯一の `iteminfo.series` を `SeriesSearchItem` として返す。各要素は埋め込んだ `BookSeries` の名称、DMM内ID、`SourceDMM` をJSONの同じ要素へフラットに持つ。DMMがシリーズURLを明示しないためURLは空である。取得元の順序とページングを保ち、シリーズの推測、統合、並べ替えは行わない。seriesが欠落、複数、IDまたは名称が空の場合は `invalid_response` を返す。

`SeriesSearchItem` の `title`、`authors`、`publishers`、`subjects`、`images`、`sources` は、同じkeyword応答itemの代表商品から得る候補判別用の補助情報である。シリーズ自体の確定した属性ではない。`title` はitemの値を加工せず保持する。`authors` は `iteminfo.author.name` を返却順で保持する。`publishers` はこのpackageが固定するDMM電子コミックの `iteminfo.manufacture.name` を返却順で保持し、空名とmanufacturer IDは出力しない。`subjects` はgenre IDを文字列表現で `Scheme=dmm` のSubjectにする。`images` は有効なHTTP(S) URLを `large`、`list`、`small` の順で確認し、最初の1件だけを保持する。`sources` は代表商品の `content_id`、通常URL、アフィリエイトURLを個別Bookと同じ規則で保持する。これらのIDとURLはseries IDやseries URLではない。

## シリーズ内Book取得

`SearchBooksBySeries` は指定したDMMシリーズ内の個別商品を `[]Book` として返す。`SearchBooksBySeriesWithRawResponse` は同じ結果に加えて秘匿済みRaw responseを返す。両メソッドは `site=DMM.com`、`service=ebook`、`floor=comic`、`sort=rank`、`output=json`、`article=series`、`article_id` を固定してItemListを呼ぶ。

| `SearchBooksBySeriesRequest` | DMM          | 動作                                      |
| ---------------------------- | ------------ | ----------------------------------------- |
| `SeriesID`                   | `article_id` | 必須。DMMが明示したシリーズIDを指定する   |
| `Limit`                      | `hits`       | 0は20、1から100を許可する                 |
| `Cursor`                     | `offset`     | 不透明Cursor経由で扱う                    |

空または空白だけの `SeriesID` は `invalid_argument` となる。返却itemは、要求したIDと一致する唯一の `iteminfo.series` を持つ必要がある。seriesの欠落、複数、ID・名称の欠落、要求IDとの不一致は `invalid_response` となる。

シリーズ内BookはDMMが返したitem順を維持する。古い順または巻数昇順を安全に指定できる取得元sortを確認できておらず、`number` の意味も巻数に限定できないため、ライブラリ側でreverseや巻数sortを行わない。これによりRaw responseのitem順と変換済みBooksの順序を一致させる。

個別Bookはitem固有の `title`、`content_id`、通常URL、アフィリエイトURL、author、manufacturer、genre、imageURL、電子媒体を保持する。`content_id` は `BookSource.ID`、`URL` と `affiliateURL` はそれぞれ通常URLとアフィリエイトURLとして、有効なHTTP(S) URLだけを保持する。`BookSeries` には検証済みのDMM series ID、名称、`SourceDMM` を1件設定する。`PublicationSeries` は設定しない。

authorは返却順でAuthorsと役割なしContributorsへ入れる。manufacturerの空でないnameは返却順でPublishersへ入れ、manufacturer IDは保持しない。genreの数値IDは文字列表現で `Scheme=dmm` のSubjectにする。`imageURL` オブジェクトの有効なHTTP(S) URLは、`large`、`list`、`small` の順で確認し、最初の1件だけを `CoverURL` へ保持する。titleは非破壊で保持し、安全なタイトル構文からVolume、Editions、IsFinalVolumeを補う場合がある。`number` は巻、号、話などが混在するためVolumeへ変換しない。`volume`、`date`、`prices.price` は、現行調査で共通 `Volume`、日付フィールド、`Price` と同じ意味だと安全に確定できていないため変換しない。ISBNも変換しない。ISBN参照APIと `cid` による単一商品参照APIは提供しない。

## Cursorとページング

`SearchSeries` と `SearchBooksBySeries` のCursorは、検索種別、実効検索条件、実効Limit、次offsetに関連付け、API IDとAffiliate IDを保存しない。シリーズ検索用Cursorをシリーズ内Book取得へ流用することはできない。DMMの1始まり `offset` は公開しない。次offsetが50000を超える場合はCursorを返さない。itemsを含む成功応答の `first_position` が要求offsetと一致しない場合は `invalid_response` を返す。

## Raw responseとHTTPエラー

Raw response版は、成功JSONを再エンコードし、Clientへ設定したAPI IDとAffiliate IDをJSON全体から固定の秘匿値へ置換して返す。URL query内のpercent-encodingされた認証値も秘匿する。HTTP(S)絶対URLではない文字列をURLとして解析・再エンコードしない。JSON numberは精度を保って再エンコードする。JSON破損時はRaw responseを返さない。

成功本文は16 MiBまで読み込み、上限超過またはBodyのClose失敗は `invalid_response` とする。2xx以外の本文は64 KiBまで読み捨て、読み捨てまたはCloseの失敗を分類済みエラーの原因として保持する。HTTP 408、429、5xxと通信失敗は `unavailable`、その他のHTTP失敗とDMM本文statusエラーは `upstream`、JSON不正、ページング矛盾、シリーズ構造の矛盾は `invalid_response` である。`Retry-After` は安全に解釈できる秒数または将来のHTTP-dateだけを `Error.RetryAfter` へ保持する。

公開エラーとCursorには認証情報を含めない。自動リトライ、キャッシュ、内部レートリミッター、ログ、バックグラウンド処理は行わない。
