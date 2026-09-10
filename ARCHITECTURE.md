# manken アーキテクチャ

## 目的と範囲

この文書は、現在の `manken` モジュールを構成するパッケージ、依存方向、
検索処理の流れ、責務分担を実装と仕様書から要約する。
将来の実装を拘束する設計指針ではなく、実装または仕様書と食い違う場合は
実装と仕様書を正として、この文書を更新する。
公開型の詳細な仕様は [manken API仕様](docs/spec.md)、取得元固有の検索・変換規則は
[MADBパッケージ仕様](docs/pkg/madb/spec.md)と
[openBDパッケージ仕様](docs/pkg/openbd/spec.md)と
[Google Booksパッケージ仕様](docs/pkg/googlebooks/spec.md)と
[楽天Booksパッケージ仕様](docs/pkg/rakutenbooks/spec.md)を一次文書とする。
[楽天Koboパッケージ仕様](docs/pkg/rakutenkobo/spec.md)も一次文書とする。
[NDLサーチパッケージ仕様](docs/pkg/ndl/spec.md)も一次文書とする。
[Yahoo!ショッピングパッケージ仕様](docs/pkg/yahooshopping/spec.md)も一次文書とする。
[DMMパッケージ仕様](docs/pkg/dmm/spec.md)も一次文書とする。

現在のモジュールは `Book`、検索条件、検索結果、エラー型を定義する `model`、MADBを検索・参照する `madb`、
openBDをISBNで参照する `openbd`、Google Booksを検索・ISBN参照する `googlebooks`、
楽天Booksを検索・ISBN参照する `rakutenbooks`、楽天Koboを検索する `rakutenkobo`、国立国会図書館サーチを検索・ISBN参照する `ndl` を提供する。
Yahoo!ショッピングのTower紙書籍を検索・ISBN参照する `yahooshopping` も提供する。
DMMブックス電子コミックのシリーズを探索し、シリーズ内の個別商品を取得する `dmm` も提供する。
ルート `manken` は利用側が生成したprovider Clientを登録し、呼び出し時に指定された `Source` の検索またはISBN参照へ委譲する。複数の取得元をまとめる検索とMCPサーバーは提供しない。

## 構成と依存方向

### 公開パッケージの依存関係

```text
利用側
  |
  +-----> manken --------> facade対応provider -----> model
  |
  +-----> 各provider package ---------------------> model
  |
  `-----> model
```

`manken` は、ルートパッケージが公開する検索またはISBN参照に対応するproviderだけを登録・委譲対象とする。
各provider packageは直接利用でき、`model` の型だけを扱う場合は `model` も直接利用できる。

### providerと外部サービス

| provider        | 外部サービス                    |
| --------------- | ------------------------------- |
| `madb`          | MADB SPARQL Query Service       |
| `openbd`        | openBD                          |
| `googlebooks`   | Google Books Volumes API        |
| `rakutenbooks`  | 楽天ブックス書籍検索API         |
| `rakutenkobo`   | 楽天Kobo電子書籍検索API         |
| `ndl`           | NDLサーチ SRU API               |
| `yahooshopping` | Yahoo!ショッピング商品検索API   |
| `dmm`           | DMM.com Webサービス v3 ItemList |

### internalパッケージ

| internal package        | 利用provider                                                                                  | 役割                                      |
| ----------------------- | --------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `internal/isbn`         | `madb`、`openbd`、`googlebooks`、`rakutenbooks`、`ndl`、`yahooshopping`                       | ISBNの検証、変換、正準化                  |
| `internal/titlemeta`    | `madb`、`openbd`、`googlebooks`、`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` | 安全なタイトルメタデータ抽出              |
| `internal/httpresponse` | `madb`、`openbd`、`googlebooks`、`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` | HTTPレスポンス本文とRetry-Afterの共通処理 |
| `internal/httpendpoint` | `madb`、`openbd`、`googlebooks`、`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` | HTTP endpointの共通検証                   |
| `internal/daterange`    | `madb`、`ndl`、`dmm`                                                                          | 日付範囲の検証と境界計算                  |
| `internal/authorrole`   | `madb`、`ndl`                                                                                 | 共通の主要創作者役割の判定                |

### 各パッケージの責務

- `model`
  - 取得元に依存しない `Book` モデル、検索条件、検索結果、エラー分類を定義する
  - 外部サービスのレスポンス型や通信処理へ依存しない
- `manken`
  - provider Clientの生成、認証、provider固有検索条件を扱わず、登録済みClientへ呼び出し時に指定された `Source` の操作を委譲する
  - providerから `manken` への依存を持たず、結果、Raw response、provider由来のエラーを変更しない
- `madb`
  - 入力検証、SPARQL生成、HTTP通信、レスポンス解析、`model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - MADB固有の中間表現と変換規則をパッケージ外へ公開しない
- `openbd`
  - ISBN入力検証、HTTP通信、応答対応の検証、`model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - openBD固有の中間表現とONIXコードをパッケージ外へ公開しない
- `googlebooks`
  - Google Booksの検索文字列生成、HTTP通信、ページング、ISBN参照、`model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - Google Books固有レスポンス型、APIキー、販売・閲覧情報をパッケージ外へ公開しない
- `rakutenbooks`
  - 楽天Booksのタイトル・著者・出版社検索、漫画区分、HTTP通信、ページング、ISBN参照、`model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - 取得時点価格とアフィリエイトURLは `model.Book` へ変換し、楽天Books固有レスポンス型、認証情報、在庫等はパッケージ外へ公開しない
- `rakutenkobo`
  - 楽天Koboの検索、漫画区分、HTTP通信、ページング、`model.Book` への変換を担当する
  - 電子書籍の商品番号、取得時点価格、アフィリエイトURLは `model.Book` へ変換し、ISBN推測や楽天Kobo固有レスポンス型、認証情報は公開しない
- `ndl`
  - SRU CQL生成、HTTP通信、Cursor、ISBN参照、DC-NDL v3の解析、`model.Book` への変換を担当する
  - 通常利用に必要な `model` の型と定数を型エイリアスとして公開する
  - 出版年月日の範囲は `model.SearchRequest.DateFrom` / `DateTo` から `from` / `until` へ変換し、`subject` / `description` はNDL固有の `SearchOptions` として公開する
  - 安全に変換できる書誌価格と既知の著者役割は `model.Book` へ変換し、所蔵・個体情報、JPNO、未知の役割・典拠情報は公開しない
- `yahooshopping`
  - Tower固定の紙書籍商品検索、ISBN参照、HTTP通信、Cursor、`model.Book` への変換を担当する
  - Client ID、ストア固有レスポンス型、在庫・送料・レビューは公開しない
- `dmm`
  - DMMブックス電子コミックのシリーズ探索、シリーズ内個別商品取得、HTTP通信、Cursor、`model.Book` への限定的な変換を担当する
  - API ID、Affiliate ID、DMM固有レスポンス型、`prices.price`・`number`・`date` 等の未共通化項目は公開しない
- `internal/isbn`
  - ISBNの整形、チェックディジット検証、ISBN-10とISBN-13の相互変換、および問い合わせ用ISBN-13への正準化を担当する
  - 取得元パッケージ間で再利用できるが、モジュール外へ公開しない
- `internal/httpresponse`
  - 上限付きHTTPレスポンス本文の読み込みと`Retry-After`の安全な待機時間への変換を担当する
  - すべてのproviderから再利用し、取得元、認証情報、`model.Error`へ依存せず、モジュール外へ公開しない
- `internal/httpendpoint`
  - HTTPエンドポイントのURL構造を検証し、必要なproviderではHTTPSまたはループバックHTTPに制限する
  - すべてのproviderから再利用し、取得元、認証情報、`model`へ依存せず、モジュール外へ公開しない
- `internal/titlemeta`
  - 取得元タイトルを変更せず、安全に判断できる巻表示、版表示、完結表示を補助フィールドへ抽出する
  - MADB、openBD、Google Books、楽天Books、楽天Kobo、NDL、Yahoo!ショッピング、DMMから再利用し、モジュール外へ公開しない
- `internal/daterange`
  - `DateFrom` / `DateTo` の形式、暦日、精度、順序を検証し、指定期間の境界を計算する
  - MADB、NDL、DMMから再利用し、取得元固有の検索式・queryへの変換は各providerが担当する
- `internal/authorrole`
  - `Authors` へ含める主要な創作者役割を判定する
  - MADBとNDLから再利用し、取得元固有の役割表記から `Contributors[].Roles` で使う日本語の役割名への変換は各providerが担当する
`model` は取得元パッケージを参照しない。利用側が単一の取得元だけを使う場合は
`madb`、`openbd`、`googlebooks`、`rakutenbooks`、`rakutenkobo`、`ndl`、`yahooshopping`、`dmm` の必要なパッケージだけをインポートでき、
`model` の型を直接扱う用途では `model` をインポートできる。

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

Google Booksの検索では、`SearchRequest` の条件を引用済みのGoogle Books検索式へ変換し、
`startIndex` を検索条件とLimitに関連付けた不透明カーソルへ隠蔽する。ISBN参照は1件だけを受け付け、
1回のHTTPリクエストで取得した応答から、要求ISBNと一致するVolumeだけを返す。

## 書籍データの境界

`model.Book` は、タイトルや著者などの書誌情報に加え、価格や表紙画像URLを直下に保持し、`Sources` に取得元への参照を保持する。ルートパッケージと各providerパッケージの `Book` は `model.Book` のtype aliasである。

- `Title` は取得元から採用した完全なタイトル文字列を保持し、巻数や版表示を別フィールドへ抽出しても短縮しない
- ISBN、日付、価格は用途別のフィールドへ置き、種別付きの汎用配列を介さない
- 表紙は `CoverURL`、書籍サイズは `Size` として、provider間で一貫して使える範囲だけを共通化する
- `Sources` は取得元、取得元内ID、通常の参照URL、アフィリエイトURLを保持し、取得元固有のレスポンス本文は保持しない

取得元に存在しない情報や項目間の対応は推測しない。完全なレスポンス本文が必要な場合は、取得元パッケージの `WithRawResponse` メソッドを使い、`model.Book` へ取り込まない。

## 通信と実行制御の境界

各取得元の `Client` は、呼び出し側が渡した `context.Context` と `http.Client` を尊重する。
現行ライブラリはバックグラウンド処理を開始せず、ログ、キャッシュ、自動リトライ、
アクセス間隔の制御を行わない。必要な場合は呼び出し側が担当する。

成功レスポンス本文は上限付きで1回読み込み、結果変換とRaw responseの返却に共用する。
HTTPエラー本文は公開エラーへそのまま含めない。エラーは `model.Error` の分類と標準の
エラーチェーンを通して利用側が判定できるようにする。

## 拡張時の現在の想定

現時点では、新しいデータ取得元を追加する場合、取得元ごとのパッケージが通信、
固有レスポンス、`model.Book` への変換を所有する構成を想定している。`model` には、
複数の取得元で共有できると確認した仕様だけを置く。

MCPなど別のインターフェースを追加する場合は、検索ライブラリの公開入出力を利用し、
SDK、ツール定義、構造化入出力、トランスポートを別の層で扱う案を想定している。
