# madb パッケージ仕様

## 1. 目的

`madb` パッケージは、メディア芸術データベース（MADB）から漫画本の書誌情報を検索し、
`manken` パッケージの共通モデルへ変換する。

本仕様書は、MADBに固有のクライアントAPI、検索方法、項目対応、HTTP処理、
ページング、エラー分類を定義する。

共通モデルとMCPとの責務境界は、[manken API仕様](../../spec.md)で定義する。

## 2. パッケージ

### 2.1 確定事項

パッケージのimport pathは次のとおり。

```text
github.com/eamat-dot/manken/madb
```

パッケージ名には `mediaarts` ではなく `madb` を使用する。

MADBから取得した書籍の `Source` は、次の値とする。

```go
const SourceMADB manken.Source = "madb"
```

### 2.2 未確定事項

- `SourceMADB` を `manken` と `madb` のどちらで定義するか

## 3. 初期APIの対象

### 3.1 確定事項

初期APIは、MADBに登録されたマンガ単行本のタイトル検索を対象とする。

次の機能を提供する。

- マンガ単行本に限定したタイトル検索
- 取得件数の指定
- カーソルによる続きの取得
- SPARQL Results JSONから `manken.Book` への変換
- HTTPクライアントの差し替え
- MADBエンドポイントの差し替え

次の機能は初期APIに含めない。

- ISBN検索
- 著者名検索
- マンガ単行本シリーズの検索
- MADBの任意SPARQL実行
- キャッシュ
- 自動リトライ
- クライアント側のレート制限

### 3.2 未確定事項

- タイトルを完全一致、部分一致、全文検索のどれで照合するか
- タイトル検索でマンガ単行本シリーズも検索対象に含めるか

## 4. Client API

MADBの検索は、次のAPIで呼び出す予定である。

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)

func (client *Client) SearchBooks(
	ctx context.Context,
	request manken.SearchBooksRequest,
) (manken.SearchBooksResult, error)
```

### 4.1 確定事項

- `Client` は設定だけを保持し、検索ごとの条件や結果を保持しない
- 検索ごとに新しい `http.Request` を作成する
- `context.Context` のキャンセルと期限を尊重する
- 呼び出し側から渡された `http.Client` と `Transport` を変更しない
- `Client` は複数goroutineから安全に利用できるものとする
- ライブラリ内部からログを出力しない
- ライブラリ内部でgoroutineを開始しない
- 自動リトライとキャッシュは行わない
- 不正なクライアント設定は `NewClient` でエラーにする

### 4.2 未確定事項

- `httpClient` が `nil` の場合に使用するタイムアウト
- `NewClient` が返す設定エラーの型

## 5. Endpoint Option

既定のMADBエンドポイントは次のとおり。

```text
https://mediaarts-db.artmuseums.go.jp/sparql
```

エンドポイントの差し替えには、次のオプションを使用する予定である。

```go
func WithEndpoint(endpoint string) Option
```

### 5.1 確定事項

- `WithEndpoint` はテストと、公式エンドポイント変更時の差し替えに使用する
- URLは `NewClient` で検証する
- `http` または `https` 以外のスキームは受け付けない

### 5.2 未確定事項

- URLにクエリ文字列またはフラグメントが含まれる場合の扱い
- テスト専用の非公開設定手段で代替できるか

## 6. MADB項目とBook

現時点で対応が確認できている候補は次のとおり。

| `manken.Book` | MADBプロパティ |
| --- | --- |
| `Title` | `schema:name` |
| `Subtitle` | `schema:alternativeHeadline` |
| `SeriesName` | `ma:seriesName` |
| `VolumeNumber` | `schema:volumeNumber` |
| `Authors` | `schema:creator` |
| `Publishers` | `schema:publisher` |

### 6.1 確定事項

- MADBに存在しない値をタイトルなどから推測しない
- `schema:creator` から著者名として扱う値を抽出し、`Authors` へ変換する
- creatorとpublisherが複数ある場合は、1冊の `Book` へまとめる
- 重複する著者名とpublisherは除外する
- ISBNは形式を確認できた値だけを `ISBN10` または `ISBN13` へ設定する
- 刊行日は精度を変更せず文字列として保持する
- MADBの対象を参照できるURLを `SourceURL` へ設定する

### 6.2 未確定事項

- マンガ単行本を示す正確なクラスURI
- MADB内の識別子とリソースURIのどちらを `ID` に使用するか
- creatorの表示名を取得するために必要な関連
- creatorに著者以外が含まれる場合の `Authors` への変換規則
- ISBN-10、ISBN-13、刊行日の取得元プロパティ
- `Subtitle` と `SeriesName` を初期APIで確実に取得できるか
- 複数値の並び順

## 7. タイトル検索

### 7.1 確定事項

- 検索対象をマンガ単行本に限定する
- `Title` はSPARQL構文ではなくデータ値として扱う
- タイトルに引用符やバックスラッシュが含まれてもクエリ構造を変更させない
- 同じリソースの複数bindingは1冊の結果へまとめる
- 結果の順序を安定させる

### 7.2 未確定事項

- MADBの全文検索機能と `schema:name` の部分一致のどちらを使用するか
- 大文字小文字、空白、Unicode正規化の扱い
- 安定した順序に使用するソートキー
- 1冊へまとめる処理をSPARQLとGoのどちらで行うか

## 8. ページング

### 8.1 確定事項

- 初回検索では空のカーソルを受け付ける
- 続きがある場合は、不透明な文字列として `NextCursor` を返す
- 呼び出し側がカーソルの内容を解析または生成することを前提にしない
- 不正形式または検索条件と一致しないカーソルは入力エラーにする
- カーソルは永続的な識別子として扱わない

### 8.2 未確定事項

- `LIMIT` と `OFFSET` を使用するか
- 次ページ判定のために `Limit + 1` 件を取得するか
- カーソルに保持する値と符号化形式
- カーソルへタイトルとLimitを関連付ける方法
- カーソルに有効期間を設けるか
- MADBの更新中も安定したページングを保証できるか

## 9. HTTP

### 9.1 確定事項

- SPARQL Results JSONを要求する
- 成功以外のHTTPステータスをエラーとして扱う
- エラー本文の読込量を制限する
- 成功レスポンスも無制限に読み込まない
- リダイレクト、プロキシ、TLSの方針は `http.Client` に従う
- URLやエラーへ検索語を不用意に含めない
- `Retry-After` がある場合は、呼び出し側が参照できる形で保持する

### 9.2 未確定事項

- GETとPOSTのどちらを使用するか
- リクエストのContent-TypeとAccept
- 成功レスポンス本文の読込上限
- エラー本文の読込上限
- HTTP 429で返される `Retry-After` の形式
- MADB固有のタイムアウトまたはアクセス頻度に関する制約

## 10. エラー

### 10.1 確定事項

- MADBへ送信できない入力は `manken.ErrorKindInvalidArgument` とする
- MADBが返したエラー応答は `manken.ErrorKindUpstream` または
  `manken.ErrorKindUnavailable` とする
- 一時的な通信失敗は `manken.ErrorKindUnavailable` とする
- 成功応答を解析できない場合は `manken.ErrorKindInvalidResponse` とする
- コンテキストのキャンセルと期限切れを元のエラーとして判定可能にする
- 外部サービスのレスポンス本文を通常のエラーメッセージへそのまま含めない

### 10.2 未確定事項

- MADBのHTTPステータスと `ErrorKind` の対応
- 不正なSPARQL応答で返されるHTTPステータスと本文
- 診断用に保持するエラー本文の形式
- `Operation` に使用する値

## 11. 現時点の未確定事項一覧

1. マンガ単行本を示すクラスURI
2. タイトル検索の一致方法
3. 各 `Book` フィールドに対応するMADBプロパティ
4. 複数値の順序と重複除去規則
5. 安定した検索順序とページング方法
6. タイムアウトとレスポンス上限
7. HTTPリクエストとレスポンスの詳細
8. MADBのHTTPエラーと `manken.ErrorKind` の対応
