# Backlog

現在のAPIの対象外とした機能や、着手時期を決めていない作業を記録する。
実施が決まった項目は、範囲と受け入れ条件を定めたTODOへ移す。

## 検索機能

- マンガ単行本シリーズ検索
- 複数データ取得元の横断検索
  - 取得元をまたぐ検索条件とLimitの意味
  - 取得元をまたぐカーソルとページングの規則
- 検索結果の重複統合

## データ取得元

openBD着手前の共通契約と、取得元ごとの実装順を記録する。

openBDの前に、次の共通契約を順番に実装する。

1. ISBN参照APIを書誌検索APIから分離する
   （[`todo/done/020-separate-isbn-lookup-api.md`](todo/done/020-separate-isbn-lookup-api.md)）
2. 共通書籍モデルに並列タイトルを追加する
   （[`todo/done/021-add-parallel-titles.md`](todo/done/021-add-parallel-titles.md)）

共通契約の完了後、次の順序で取得元を実装する。

1. openBD（複数ISBN参照のみ。実装TODOは
   [`todo/019-implement-openbd-api.md`](todo/019-implement-openbd-api.md)）
2. Google Books API
3. 楽天Books API
4. 楽天Kobo API
5. Yahoo!ショッピング商品検索API（`bookfan` の紙書籍のみ）

楽天Booksと楽天Koboは、紙書籍と電子書籍で識別子、検索条件、変換規則が
異なるため、別の実装TODOにする。各取得元の詳細TODOは一度に作らず、直前の
取得元で確立したパッケージ構造、エラー処理、テスト方法を反映して順番に作る。

次の取得元は着手時期を決めていない。

- NDL 国立国会図書館サーチ
- DMM.com API（電子書籍）

取得元を追加する場合は、HTTPステータスと共通の `ErrorKind` の対応を比較する。

## 運用

- キャッシュ
- 自動リトライ
- クライアント側のレート制限

## インターフェース

- MCPサーバー
- MCP実装を同一モジュールと独立モジュールのどちらに置くかの比較
- Go向けMCP SDKの選定
- MCPツール名、説明、入出力Schemaの設計
- stdioとStreamable HTTPの提供範囲
