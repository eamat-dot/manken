# 漫検 manken - 漫画のタイトル・著者・ISBNなどを検索する Go ライブラリ

漫画本の書誌情報を検索するためのGoライブラリ。
複数のGoプロジェクトから利用できる共通モデルと検索APIを提供する。

## 開発状況

現在は仕様策定中であり、Goパッケージと検索機能は未実装。
最初のデータ取得元として、メディア芸術データベース（MADB）への対応を予定している。

初期APIでは、マンガ単行本のタイトル検索を対象とする。
ISBN検索、著者名検索、キャッシュ、自動リトライ、CLI、MCPサーバーは含めない。

## 設計

- [共通API仕様](docs/spec.md)
- [MADBパッケージ仕様](docs/pkg/madb/spec.md)
- [構想](docs/concept.md)
- [実装TODO](docs/todo/)

データ取得元に依存しない型はルートの `manken` パッケージへ置き、
MADB固有の通信と変換は `madb` パッケージへ分離する予定。

将来MCPサーバーから呼び出せるようにするが、MCP SDKやトランスポートは
検索ライブラリへ依存させない。

## 開発

Goモジュール作成後は、次のタスクを使用する。

```text
task mod-deps
task lint
task test
task build
task all
```

`task all` は依存関係の整理、lint、テスト、全パッケージのビルドを順に実行する。

## ライセンス

[MIT License](LICENSE)
