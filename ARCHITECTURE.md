# manken アーキテクチャ

## 目的と範囲

この文書は、現在の `manken` モジュールを構成するパッケージ、依存方向、
検索処理の流れ、責務分担を実装と仕様書から要約する。
将来の実装を拘束する設計指針ではなく、実装または仕様書と食い違う場合は
実装と仕様書を正として、この文書を更新する。
公開型の詳細な仕様は [共通API仕様](docs/spec.md)、取得元固有の検索・変換規則は
[MADBパッケージ仕様](docs/pkg/madb/spec.md)と
[openBDパッケージ仕様](docs/pkg/openbd/spec.md)と
[Google Booksパッケージ仕様](docs/pkg/googlebooks/spec.md)と
[楽天Booksパッケージ仕様](docs/pkg/rakutenbooks/spec.md)を一次文書とする。
[楽天Koboパッケージ仕様](docs/pkg/rakutenkobo/spec.md)も一次文書とする。
[NDLサーチパッケージ仕様](docs/pkg/ndl/spec.md)も一次文書とする。
[Yahoo!ショッピングパッケージ仕様](docs/pkg/yahooshopping/spec.md)も一次文書とする。
[DMMパッケージ仕様](docs/pkg/dmm/spec.md)も一次文書とする。

現在のモジュールは共通モデルを定義する `model`、MADBを検索・参照する `madb`、
openBDをISBNで参照する `openbd`、Google Booksを検索・ISBN参照する `googlebooks`、
楽天Booksを検索・ISBN参照する `rakutenbooks`、楽天Koboを検索する `rakutenkobo`、国立国会図書館サーチを検索・ISBN参照する `ndl` を提供する。
Yahoo!ショッピングのTower紙書籍を検索・ISBN参照する `yahooshopping` も提供する。
DMMブックス電子コミックのシリーズを探索し、シリーズ内の個別商品を取得する `dmm` も提供する。
ルート `manken` は利用側が生成したprovider Clientを登録し、明示Sourceの共通検索またはISBN参照だけへ委譲する。複数の取得元をまとめる検索とMCPサーバーは提供しない。

## 構成と依存方向

```text
利用側・examples
       |
       +------> manken -----> madb/openbd/googlebooks/rakutenbooks/rakutenkobo/ndl/yahooshopping -----> model
       |           |
       |           `------> model
       |
       +------> madb -----> model
       |          |
       |          +------> internal/isbn
       |          |
       |          `------> MADB SPARQL Query Service
       |
       `------> openbd --> model
                  |
                  +------> internal/isbn
                  |
                  `------> openBD
       |
       +------> googlebooks -> model
       |                |
       |                +------> internal/isbn
       |                |
       |                `------> Google Books Volumes API
       |
       +------> rakutenbooks -> model
       |                |
       |                +------> internal/isbn
       |                |
       |                `------> 楽天ブックス書籍検索API
       |
       `------> rakutenkobo -> model
                        |
                        `------> 楽天Kobo電子書籍検索API

       `------> yahooshopping -> model
                         |
                         +------> internal/isbn
                         |
                         `------> Yahoo!ショッピング商品検索API
       |
       `------> dmm -----------> model
                         |
                         `------> DMM.com Webサービス v3 ItemList
       |
       `------> ndl -> model
                         |
                         +------> internal/isbn
                         |
                         `------> NDLサーチ SRU API

madb/openbd/googlebooks/rakutenbooks/rakutenkobo/ndl/yahooshopping/dmm
       `------> internal/titlemeta
madb/ndl/dmm
       `------> internal/daterange
madb/ndl
       `------> internal/authorrole
```

- `model`
  - 取得元に依存しない書籍モデル、検索条件、検索結果、エラー分類を定義する
  - 外部サービスのレスポンス型や通信処理へ依存しない
- `manken`
  - provider Clientの生成、認証、provider固有検索条件を扱わず、登録済みClientを明示Sourceのroleへ委譲する
  - providerから `manken` への依存を持たず、結果、Raw response、provider由来のエラーを変更しない
- `madb`
  - 入力検証、SPARQL生成、HTTP通信、レスポンス解析、共通モデルへの変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - MADB固有の中間表現と変換規則をパッケージ外へ公開しない
- `openbd`
  - ISBN入力検証、HTTP通信、応答対応の検証、共通モデルへの変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - openBD固有の中間表現とONIXコードをパッケージ外へ公開しない
- `googlebooks`
  - Google Booksの検索文字列生成、HTTP通信、ページング、ISBN参照、共通モデルへの変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - Google Books固有レスポンス型、APIキー、販売・閲覧情報をパッケージ外へ公開しない
- `rakutenbooks`
  - 楽天Booksのタイトル・著者・出版社検索、漫画区分、HTTP通信、ページング、ISBN参照、共通モデルへの変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - 取得時点価格とアフィリエイトURLは共通モデルへ変換し、楽天Books固有レスポンス型、認証情報、在庫等はパッケージ外へ公開しない
- `rakutenkobo`
  - 楽天Koboの検索、漫画区分、HTTP通信、ページング、共通モデルへの変換を担当する
  - 電子書籍の商品番号、取得時点価格、アフィリエイトURLは共通モデルへ変換し、ISBN推測や楽天Kobo固有レスポンス型、認証情報は公開しない
- `ndl`
  - SRU CQL生成、HTTP通信、Cursor、ISBN参照、DC-NDL v3の解析、共通モデルへの変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - 出版年月日の範囲は共通 `model.SearchRequest.DateFrom` / `DateTo` から `from` / `until` へ変換し、`subject` / `description` はNDL固有の `SearchOptions` として公開する
  - 安全に変換できる書誌価格と既知の著者役割は共通モデルへ変換し、所蔵・個体情報、JPNO、未知の役割・典拠情報は公開しない
- `yahooshopping`
  - Tower固定の紙書籍商品検索、ISBN参照、HTTP通信、Cursor、共通モデルへの変換を担当する
  - Client ID、ストア固有レスポンス型、在庫・送料・レビューは公開しない
- `dmm`
  - DMMブックス電子コミックのシリーズ探索、シリーズ内個別商品取得、HTTP通信、Cursor、共通モデルへの限定的な変換を担当する
  - API ID、Affiliate ID、DMM固有レスポンス型、`prices.price`・`number`・`date` 等の未共通化項目は公開しない
- `internal/isbn`
  - ISBNの整形、チェックディジット検証、ISBN-10とISBN-13の相互変換を担当する
  - 取得元パッケージ間で再利用できるが、モジュール外へ公開しない
- `internal/titlemeta`
  - 取得元タイトルを変更せず、安全に判断できる巻表示、版表示、完結表示を補助フィールドへ抽出する
  - MADB、openBD、Google Books、楽天Books、楽天Kobo、NDL、Yahoo!ショッピング、DMMから再利用し、モジュール外へ公開しない
- `internal/daterange`
  - `DateFrom` / `DateTo` の形式、暦日、精度、順序を検証し、指定期間の境界を計算する
  - MADB、NDL、DMMから再利用し、取得元固有の検索式・queryへの変換は各providerが担当する
- `internal/authorrole`
  - `Authors` へ含める共通の主要創作者役割を判定する
  - MADBとNDLから再利用し、取得元固有の役割表記から共通日本語役割名への変換は各providerが担当する
- `examples`
  - `madb`、`openbd`、`googlebooks`、`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` の公開APIを使う動作確認用CLIを置く
  - ライブラリの一部として再利用する内部処理は置かない

`model` は取得元パッケージを参照しない。利用側が単一の取得元だけを使う場合は
`madb`、`openbd`、`googlebooks`、`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` の必要なパッケージだけをimportでき、
共通型を直接扱う用途では `model` をimportできる。

## MADBの検索処理の流れ

次の流れはMADBのタイトル・著者名・フリーワード検索にだけ適用する。

```text
SearchRequest
       |
       v
入力検証・検索条件の正規化
       |
       v
カーソルの検証 ------ 条件またはLimit不一致なら入力エラー
       |
       v
SPARQL生成 -> HTTP POST -> 上限付きレスポンス読込
       |
       v
SPARQL Results JSONの解析・1冊単位への集約
       |
       v
MADB固有値からmodel.Bookへの変換
       |
       v
SearchBooksResultと次ページカーソル
```

検索条件はライブラリが安全なSPARQLと全文検索式へ変換する。利用者入力を
SPARQLや全文検索の構文として直接扱わない。ページングはMADBリソースURI順の
キーセット方式を使い、カーソルを正規化済み検索条件と取得件数へ関連付ける。

ISBN参照では、入力順と元文字列を保持したままISBN-10・ISBN-13の候補を作り、
重複除去した候補を1回のSPARQLへまとめる。応答のリソースと一致ISBNを集約後、
元の各入力位置へ個別の `Book` を展開する。検索用のLimitとカーソルは使わない。

openBDのISBN参照では、入力をISBN-13へ統一して重複除去し、1回のGETへまとめる。
応答配列の件数、順序、ISBNを検証してから元の各入力位置へ結果を展開する。

Google Booksの検索では、共通検索条件を引用済みのGoogle Books検索式へ変換し、
`startIndex` を検索条件とLimitに関連付けた不透明カーソルへ隠蔽する。ISBN参照は1件だけを受け付け、
1回のHTTPリクエストで取得した応答から、要求ISBNと一致するVolumeだけを返す。

## 書籍データの境界

`model.Book` は共通書誌情報を直下に保持し、`Sources` に取得元への参照を保持する。

- `Title` は取得元から採用した完全なタイトル文字列を保持し、巻数や版表示を別フィールドへ抽出しても短縮しない
- ISBN、日付、価格は用途別のフィールドへ置き、種別付きの汎用配列を介さない
- 表紙は `CoverURL`、書籍サイズは `Size` として、provider間で一貫して使える範囲だけを共通化する
- `Sources` は取得元、取得元内ID、通常の参照URL、アフィリエイトURLを保持し、取得元固有のレスポンス本文は保持しない

取得元に存在しない情報や項目間の対応は推測しない。完全なレスポンス本文が必要な場合は、取得元パッケージの `WithRawResponse` メソッドを使い、共通モデルへ取り込まない。

## 通信と実行制御の境界

各取得元の `Client` は、呼び出し側が渡した `context.Context` と `http.Client` を尊重する。
現行ライブラリはバックグラウンド処理を開始せず、ログ、キャッシュ、自動リトライ、
アクセス間隔の制御を行わない。必要な場合は呼び出し側が担当する。

成功レスポンス本文は上限付きで1回読み込み、結果変換とrawレスポンス返却に共用する。
HTTPエラー本文は公開エラーへそのまま含めない。エラーは `model.Error` の分類と標準の
エラーチェーンを通して利用側が判定できるようにする。

## 拡張時の現在の想定

現時点では、新しいデータ取得元を追加する場合、取得元ごとのパッケージが通信、
固有レスポンス、共通モデルへの変換を所有する構成を想定している。`model` には、
複数の取得元で共有できると確認した仕様だけを置く。

MCPなど別のインターフェースを追加する場合は、検索ライブラリの公開入出力を利用し、
SDK、ツール定義、構造化入出力、トランスポートを別の層で扱う案を想定している。
未実装機能と着手順は [Backlog](docs/backlog.md) で管理する。
