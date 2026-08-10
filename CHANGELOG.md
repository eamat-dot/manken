# Changelog

このファイルには、利用者に影響する変更を記録する。
形式は [Keep a Changelog](https://keepachangelog.com/ja/1.1.0/) を基準とし、
バージョン番号には [Semantic Versioning](https://semver.org/lang/ja/) を使用する。

## [Unreleased]

### Added

- 共通 `SearchBooksRequest.Publisher` によるMADB、Google Books、楽天Booksの出版社名検索
- MADBのマンガ単行本をタイトルで検索する公開API
- 共通の書籍モデル、検索条件、検索結果、エラー分類を定義する `api` パッケージ
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
- 取得元の通常URLと分離してアフィリエイトURLを保持する `BookSource.AffiliateURL`
- 楽天Booksの `itemPrice` を取得時点の税込JPY価格として共通価格情報へ変換
- 主タイトルと別言語または別文字体系のタイトルを保持する `ParallelTitles`
- 検索結果のJSON出力、カーソル、rawレスポンス保存を確認できるCLIデモ

### Changed

- 楽天Books検索を発売日の古い順へ変更し、商品形態による任意の絞り込みと複数著者の人物単位変換に対応
- ISBN条件を `SearchBooks` から削除し、ISBN専用の `LookupBooksByISBN` へ分離
- `Book`を、利用者向けの`Normalized`と取得元別の`Sources`を持つ構造へ
  破壊的に変更
- MADBの巻表示を、許可した構文に限って`Volume.Number`と`Volume.Label`へ変換
- MADBの出版社名に付記されたカナ読みを除去し、正規化後の重複を削除
- 欠落した任意の書誌項目をJSONから省略
