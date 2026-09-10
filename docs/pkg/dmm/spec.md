# DMMパッケージ仕様

## 1. 概要

`dmm` パッケージはDMM.com Webサービス v3を使い、一般向けDMMブックス電子コミックのシリーズを探し、指定したシリーズに属する個別商品を取得する。FANZAは対象にしない。

インポートパスは `github.com/eamat-dot/manken/dmm` である。

## 2. 公開APIとClient

`dmm` は `Client`、`Option`、`Book`、`BookSeries`、`Source`、`PublicationMedium`、`ErrorKind`、`Error` など、通常利用に必要な `model` の型をエイリアスとして公開する。`SeriesSearchItem` はDMM固有のシリーズ候補型である。`SourceDMM` はDMM.com Webサービス v3 ItemListを、`PublicationMediumDigital` は電子書籍を表す。

`NewClient(httpClient, dmm.WithAPIID(...), dmm.WithAffiliateID(...))` はAPI IDとAffiliate IDの両方を必要とする。ライブラリ本体は環境変数を読まない。認証値が空白だけ、または制御文字を含む場合は `invalid_argument` となる。

`httpClient` が `nil` の場合は60秒のタイムアウトを使い、呼び出し側から渡された `http.Client` は変更しない。リダイレクトは自動追従しない。

`WithEndpoint` は通常使用しない。HTTPSの絶対URLを受け付け、テスト用の `localhost` またはループバックアドレスへのHTTPだけを許可する。ユーザー情報、クエリ、フラグメント付きURLは拒否する。

## 3. シリーズ検索

`SearchSeries` は変換済みシリーズを返す。`SearchSeriesWithRawResponse` は同じ結果に加えて、認証情報を秘匿したRaw responseを返す。

両メソッドはItemListを次の固定条件で呼び出す。

- `site=DMM.com`
- `service=ebook`
- `floor=comic`
- `sort=rank`
- `output=json`

| `SearchSeriesRequest` | DMM       | 動作                                             |
| --------------------- | --------- | ------------------------------------------------ |
| `FreeText`            | `keyword` | 必須。DMMのフリーワードとして空白を1つにして送る |
| `ExcludedText`        | `keyword` | 空白区切りの各語を取得元の `-語` として送る      |
| `Limit`               | `hits`    | 0は20、1から100を許可する                        |
| `Cursor`              | `offset`  | 不透明なCursor経由で扱う                         |

`SearchSeriesRequest.DateFrom` と `DateTo` は、DMM商品 `date` の時期を `gte_date` と `lte_date` で絞り込む。開始は指定期間の `00:00:00`、終了は期間終端の `23:59:59` とし、タイムゾーン変換は行わない。`FreeText` は引き続き必須で、日付条件はCursor照合に含める。

`FreeText` が空、または各条件の語に `|`、`"`、先頭 `-` を含む場合は、通信前に `invalid_argument` を返す。DMMのリテラルを安全にエスケープする方法が確認できていないためである。

キーワード検索のレスポンスに含まれる各itemは商品として返さず、そのitemが明示する唯一の `iteminfo.series` を `SeriesSearchItem` として返す。各要素は埋め込んだ `BookSeries` の名称、DMM内ID、`SourceDMM` をJSONの同じ要素へフラットに持つ。DMMがシリーズURLを明示しないためURLは空である。

取得元の順序とページングを保ち、シリーズの推測、統合、並べ替えは行わない。`series` が欠落、複数、またはID・名称が空の場合は `invalid_response` を返す。

`SeriesSearchItem` の `title`、`authors`、`publishers`、`subjects`、`images`、`sources` は、同じキーワード検索のitemに含まれる代表商品から取得し、シリーズ候補を見分けるために使う補助情報である。シリーズ自体の確定した属性ではない。

- `title` はitemの値を加工せず保持する
- `authors` は `iteminfo.author.name` を返却順で保持する
- `publishers` は、このパッケージが固定するDMM電子コミックの `iteminfo.manufacture.name` を返却順で保持し、空名とmanufacturer IDは出力しない
- `subjects` はgenre IDを文字列で表し、`Scheme=dmm` の `Subject` にする
- `images` は有効なHTTP(S) URLを `large`、`list`、`small` の順で確認し、最初の1件だけを保持する
- `sources` は代表商品の `content_id`、通常URL、アフィリエイトURLを個別の `Book` と同じ規則で保持する

これらのIDとURLはシリーズIDやシリーズURLではない。

## 4. シリーズ内の `Book` 取得

`SearchBooksBySeries` は指定したDMMシリーズ内の個別商品を `[]Book` として返す。`SearchBooksBySeriesWithRawResponse` は同じ結果に加えて、認証情報を秘匿したRaw responseを返す。

両メソッドはItemListを次の固定条件で呼び出す。

- `site=DMM.com`
- `service=ebook`
- `floor=comic`
- `sort=rank`
- `output=json`
- `article=series`
- `article_id=<SeriesID>`

| `SearchBooksBySeriesRequest` | DMM          | 動作                                    |
| ---------------------------- | ------------ | --------------------------------------- |
| `SeriesID`                   | `article_id` | 必須。DMMが明示したシリーズIDを指定する |
| `Limit`                      | `hits`       | 0は20、1から100を許可する               |
| `Cursor`                     | `offset`     | 不透明なCursor経由で扱う                |

空または空白だけの `SeriesID` は `invalid_argument` となる。返却itemは、要求したIDと一致する唯一の `iteminfo.series` を持つ必要がある。`series` の欠落、複数、ID・名称の欠落、要求IDとの不一致は `invalid_response` となる。

シリーズ内の `Book` はDMMが返したitem順を維持する。古い順または巻数昇順を安全に指定できる取得元の `sort` を確認できておらず、`number` の意味も巻数に限定できないため、ライブラリ側で逆順化や巻数順への並べ替えを行わない。これによりRaw responseのitem順と変換済み `Books` の順序を一致させる。

個別の `Book` は、item固有の `title`、`content_id`、通常URL、アフィリエイトURL、author、manufacturer、genre、imageURL、電子媒体を保持する。

- `content_id` は `BookSource.ID` に設定する
- `URL` と `affiliateURL` は、それぞれ通常URLとアフィリエイトURLとして、有効なHTTP(S) URLだけを保持する
- `BookSeries` には検証済みのDMMシリーズID、名称、`SourceDMM` を1件設定する
- `PublicationSeries` は設定しない
- authorは返却順で `Authors` と役割なしの `Contributors` へ入れる
- manufacturerの空でないnameは返却順で `Publishers` へ入れ、manufacturer IDは保持しない
- genreの数値IDは文字列で表し、`Scheme=dmm` の `Subject` にする
- `imageURL` の有効なHTTP(S) URLは `large`、`list`、`small` の順で確認し、最初の1件だけを `CoverURL` へ保持する

`title` は非破壊で保持し、安全なタイトル構文から `Volume`、`Editions`、`IsFinalVolume` を補う場合がある。`number` は巻、号、話などが混在するため `Volume` へ変換しない。

`volume`、`date`、`prices.price` は、現行調査で `Volume`、日付フィールド、`Price` と同じ意味だと安全に確定できていないため変換しない。ISBNも変換しない。ISBN参照APIと `cid` による単一商品参照APIは提供しない。

## 5. Cursorとページング

`SearchSeries` と `SearchBooksBySeries` のCursorは、検索種別、実効検索条件、実効Limit、次の `offset` に関連付け、API IDとAffiliate IDを保存しない。シリーズ検索用Cursorをシリーズ内 `Book` 取得へ流用することはできない。

DMMの1始まりの `offset` は公開しない。次の `offset` が50000を超える場合はCursorを返さない。itemを含む成功レスポンスの `first_position` が要求 `offset` と一致しない場合は `invalid_response` を返す。

## 6. Raw responseとHTTPエラー

Raw responseを返すメソッドは成功レスポンスのJSONを再エンコードし、Clientへ設定したAPI IDとAffiliate IDをJSON全体から固定の秘匿値へ置換して返す。URLクエリ内のパーセントエンコードされた認証値も秘匿する。HTTP(S)絶対URLではない文字列をURLとして解析・再エンコードしない。JSONのnumberは精度を保って再エンコードする。JSONが破損している場合はRaw responseを返さない。

成功本文は16 MiBまで読み込み、上限超過またはBodyのClose失敗は `invalid_response` とする。2xx以外の本文は64 KiBまで読み捨て、読み捨てまたはCloseの失敗を分類済みエラーの原因として保持する。

- HTTP 408、429、5xxと通信失敗は `unavailable`
- その他のHTTP失敗とDMMレスポンス本文のstatusエラーは `upstream`
- JSON不正、ページング矛盾、シリーズ構造の矛盾は `invalid_response`

`Retry-After` は安全に解釈できる秒数または将来のHTTP-dateだけを `Error.RetryAfter` へ保持する。

公開エラーとCursorには認証情報を含めない。自動リトライ、キャッシュ、内部レートリミッター、ログ、バックグラウンド処理は行わない。
