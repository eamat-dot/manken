# Backlog

初期APIの対象外とした機能や、着手時期を決めていない作業を記録する。
実施が決まった項目は、範囲と受け入れ条件を定めたTODOへ移す。

## 検索機能

- マンガ単行本シリーズ検索
- 複数データ取得元の横断検索
- 検索結果の重複統合

## データ取得元

- NDL 国立国会図書館サーチ
- Google Books API
- openBD (ISBN検索のみ)
- Yahoo!ショッピングAPI（ebookjapan）
- DMM.com API (電子書籍)
- 楽天Books API / 楽天Kobo API

MADB,NDL,Google Books あたりはレスポンスの表記ゆれが大きい
楽天ウェブサービスは2026年の大幅な仕様変更で運用ハードルが上がった


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
