# Changelog

このファイルには、利用者に影響する変更を記録する。
形式は [Keep a Changelog](https://keepachangelog.com/ja/1.1.0/) を基準とし、
バージョン番号には [Semantic Versioning](https://semver.org/lang/ja/) を使用する。

## [Unreleased]

### Added

- 国立国会図書館サーチの全国書誌を検索し、1件のISBN参照とrawレスポンス取得に対応する `ndl` パッケージ
- タワーレコード Yahoo!店に限定してYahoo!ショッピングの紙書籍商品を検索し、1件のISBN参照とrawレスポンス取得に対応する `yahooshopping` パッケージ
- 登録したprovider Clientへ取得元を明示して検索・ISBN参照を委譲するルート `manken.Client` facade
- MADBとNDLサーチの定型出典・クレジット情報を返す `Attributions()` と、成功結果のJSONに含まれる `attributions`
- APIキーなしでMADBとNDLサーチを実行できるCLI examples
- 共通 `SearchRequest.DateFrom` / `DateTo` によるNDL、MADBの時期検索と、DMM `SearchSeriesRequest.DateFrom` / `DateTo` による商品 `date` 絞り込み
- 共通 `SearchRequest.Publisher` によるMADB、Google Books、楽天Booksの出版社名検索
- MADBのマンガ単行本をタイトルで検索する公開API
- 共通の書籍モデル、検索条件、検索結果、エラー分類を定義する `model` パッケージ
- `madb` だけで通常利用を完結できる共通型と定数のエイリアス
- MADBの版表示、単行本レーベル、参照先シリーズの名前、ID、URL
- MADBのタイトル読み、ページ数、紙書籍の大きさ
- 変換済み検索結果と受信したrawレスポンスを返す公開API
- 入力順と元のISBN表記を保持して、最大500件のISBNを一括参照する公開API
- openBDから最大1,000件のISBNを一括参照する `openbd` パッケージ
- Google Booksを検索し、1件のISBN参照と検索・ISBN参照のrawレスポンス取得に対応する `googlebooks` パッケージ
- 楽天Booksの一般・BL・TLコミックを検索し、1件のISBN参照とrawレスポンス取得に対応する `rakutenbooks` パッケージ
- 楽天Koboの一般・BL・TLコミックをタイトル・著者・出版社・商品キーワード・除外語で検索する `rakutenkobo` パッケージ
- 楽天Koboの商品番号、著者読み、取得時点価格、通常URL・アフィリエイトURL、画像、rawレスポンスの取得
- DMMブックスの一般向け電子コミックをシリーズ探索し、シリーズIDで個別商品を取得して秘匿済みrawレスポンスを取得する `dmm` パッケージ
- DMMシリーズ候補を代表商品由来のタイトル、著者、出版社、genre、画像、商品参照先で判別する `SeriesSearchItem`
- DMMが明示する作品または刊行物のシリーズを保持する `BookSeries`
- 取得元の通常URLと分離してアフィリエイトURLを保持する `BookSource.AffiliateURL`
- 楽天Booksの `itemPrice` を取得時点の税込JPY価格として共通価格情報へ変換
- NDLの書誌価格とopenBDの確認済みONIX価格を `ListPrice` へ変換
- NDLのextentにある単純な書籍サイズと、タイトルの `＜完＞` / `<完>` 完結表示

### Changed

- 共通 `Book` から `Imprints` とJSON `imprints` を削除し、MADBの `schema:brand` を `PublicationSeries` へ統合する破壊的変更
- `PublicationSeries` を名称の文字列配列へ簡略化し、共通モデルからReadingを削除する破壊的変更
- 初回リリース前の共通系列モデルを再編し、旧 `Series` を `BookSeries` と `PublicationSeries` へ分離
- DMM電子コミックのmanufacturer名をシリーズ内個別Bookの出版社として保持
- 楽天Books検索を発売日の古い順へ変更し、商品形態による任意の絞り込みと複数著者の人物単位変換に対応
- ISBN条件を `SearchBooks` から削除し、ISBN専用の `LookupBooksByISBN` へ分離
- 共通 `Book` を平坦化し、書誌情報を `Book` 直下から参照する構造へ変更。取得元タイトルは非破壊で保持する
- 識別子、日付、価格を用途別フィールドへ整理し、表紙を `CoverURL`、書籍サイズを `Size` として単純化
- Contributorの共通役割を固定enumから人間向け日本語文字列へ変更し、`Reading` を維持
- 共通packageを `api` から `model` へ、検索条件を `SearchRequest` / `Query` / `Exclude` へ整理
- NDLの出版時期検索を固有 `SearchOptions.From` / `Until` から共通 `SearchRequest.DateFrom` / `DateTo` へ移行
- MADBの巻表示を、許可した構文に限って`Volume.Number`と`Volume.Label`へ変換
- MADBの出版社名に付記されたカナ読みを除去し、正規化後の重複を削除
- 欠落した任意の書誌項目をJSONから省略
