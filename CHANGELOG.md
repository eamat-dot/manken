# Changelog

このファイルには、利用者に影響する変更を記録する。
形式は [Keep a Changelog](https://keepachangelog.com/ja/1.1.0/) を基準とし、
バージョン番号には [Semantic Versioning](https://semver.org/lang/ja/) を使用する。

## [Unreleased]

### Added

- MADBのマンガ単行本をタイトルで検索する公開API
- 共通の書籍モデル、検索条件、検索結果、エラー分類を定義する `api` パッケージ
- `madb` だけで通常利用を完結できる共通型と定数のエイリアス
- MADBの版表示、単行本レーベル、参照先シリーズの名前、ID、URL
- 変換済み検索結果と受信したrawレスポンスを返す公開API
- 検索結果のJSON出力、カーソル、rawレスポンス保存を確認できるCLIデモ
