# ISBN参照APIを書誌検索APIから分離

状態: 完了（2026年8月3日）

## 背景

- 現在の `SearchBooksRequest` は、タイトルや著者名などの検索条件とISBNを
  同じ構造体に持つ
- ISBNは検索語ではなく、特定の書籍を識別子から参照するための値である
- 現在の契約では、ISBNと他の検索条件をANDまたはORで組み合わせられるように
  読めるが、この組み合わせはmankenの利用方法として提供しない
- openBDは複数ISBNを1回のリクエストで取得できる一方、現在の単数
  `ISBN string` ではその機能を表現できない
- MADBでは同じISBNを持つ複数のリソースが存在し得る

## 目的

- 書誌条件による検索と、ISBNによる書籍参照を公開API上で分離する
- 複数ISBNを1回の呼び出しで指定できる共通の結果契約を定義する
- 未収録ISBNと、同じISBNに対応する複数書籍を入力ISBNごとに表現する
- openBD実装で再利用できるISBN検証と結果契約を先に確立する

## 非目的

- openBDパッケージの実装
- ISBNとタイトル、著者名、フリーワードの組み合わせ検索
- 複数取得元をまたぐISBN参照
- ISBN以外の識別子による参照API
- ISBN参照結果の取得元間統合や重複排除
- 並列タイトルや巻数の正規化

## 前提・制約

- `SearchBooksRequest` から `ISBN` を削除する破壊的変更とする
- 旧 `ISBN` フィールドや旧ISBN検索を維持する互換APIは追加しない
- 共通型は `api` パッケージに定義し、取得元パッケージからエイリアス公開する
- ISBNの形式とチェックディジットは外部サービスへ送信する前に検証する
- 取得元ごとの1回の最大指定件数は各パッケージ仕様で定義する
- 1件でも不正なISBNがある場合は、外部通信せず呼び出し全体を失敗させる
- 通常の単体テストから実サービスへ接続しない

## 公開契約

```go
type ISBNLookupResult struct {
	Items []ISBNLookupItem `json:"items"`
}

type ISBNLookupItem struct {
	RequestedISBN string `json:"requested_isbn"`
	Books         []Book `json:"books"`
}
```

取得元パッケージは次のメソッドを提供する。

```go
func (client *Client) LookupBooksByISBN(
	ctx context.Context,
	isbns []string,
) (ISBNLookupResult, error)

func (client *Client) LookupBooksByISBNWithRawResponse(
	ctx context.Context,
	isbns []string,
) (ISBNLookupResult, []byte, error)
```

### 入出力規則

- 1件以上のISBNを必須とする
- `RequestedISBN` は呼び出し側が指定した文字列を変更せず保持する
- `Items` は入力と同じ件数、同じ順序で返す
- 同じISBNが複数回指定された場合も、入力位置ごとに要素を返す
- 内部問い合わせではISBN-10と対応するISBN-13を同じ書籍として重複排除できる
- 内部で重複排除した場合も、結果は元の入力位置へ展開し直す
- 該当なしはエラーではなく、非nilの空の `Books` とする
- 同じISBNに複数書籍が対応する場合は、統合せず同じ要素の `Books` にすべて返す
- ISBN参照は `Limit` とカーソルを使用しない

## 対象範囲

### 共通API

- [x] `SearchBooksRequest` から `ISBN` を削除する
- [x] `ISBNLookupResult` と `ISBNLookupItem` を追加する
- [x] 空スライスをJSONの `[]` として出力する契約をテストする
- [x] 共通仕様からISBNと他条件の組み合わせ規則を削除する

### ISBN検証の共有

- [x] 現在のMADB用ISBN整形、検証、ISBN-10/13変換を取得元非依存の内部実装へ移す
- [x] ASCIIハイフン、Unicode空白、小文字末尾 `x` を扱う
- [x] 978で始まるISBN-13とISBN-10を相互に対応付ける
- [x] 979で始まるISBN-13からISBN-10を生成しない
- [x] MADBの既存ISBN検証結果を変更しない

### MADB

- [x] `madb.SearchBooks` からISBN検索を削除する
- [x] `LookupBooksByISBN` を追加する
- [x] 複数の入力ISBNを1回のSPARQL問い合わせへまとめる
- [x] 1つのISBNに一致する複数リソースを同じ `ISBNLookupItem.Books` へ返す
- [x] 入力ISBNごとに結果を安定した順序へそろえる
- [x] `LookupBooksByISBNWithRawResponse` を追加する
- [x] MADB固有の最大指定件数とエラー条件を仕様化する
- [x] ISBN検索を含む既存カーソルとの互換性を終了する

### 利用者向け文書

- [x] `docs/spec.md` にISBN参照APIの完全な契約を記載する
- [x] `docs/pkg/madb/spec.md` からISBN検索を分離し、ISBN参照として記載する
- [x] READMEとデモのISBN利用例を新しいメソッドへ変更する
- [x] `SearchBooks` の検索条件にISBNを含めないことを明記する

## テスト

- [x] 単一のISBN-10とISBN-13を参照できる
- [x] 複数ISBNを入力順で返す
- [x] ISBN-10と対応するISBN-13を同時指定しても各入力位置へ結果を返す
- [x] 重複した入力ISBNを各入力位置へ返す
- [x] 未収録ISBNを空の `Books` として返す
- [x] 同じISBNを持つ複数のMADBリソースを統合せず返す
- [x] 不正ISBNが混在する場合に外部通信しない
- [x] 空の入力を `invalid_argument` にする
- [x] Raw responseが受信した成功応答本文を変更せず返す
- [x] `SearchBooks` がISBNを受け取らない公開契約になっている

## 検証

- [x] `gofmt` を実行する
- [x] `go test -v ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] `golangci-lint run` を実行する
- [x] `go test -v -tags=integration ./madb` を実行する
- [x] 別のGoモジュールから新しいAPIをimportして実行する
- [x] `git diff --check` を実行する

## 受け入れ条件

- [x] ISBNと書誌検索条件が異なるメソッドへ分離されている
- [x] 複数ISBNの結果を入力ごとに対応付けられる
- [x] 未収録と複数書籍の両方を表現できる
- [x] MADBのISBN以外の検索結果とページングが退行しない
- [x] openBD実装が共通ISBN参照契約を再利用できる
- [x] 旧ISBN検索の互換コードを残していない

## リスク・懸念

- 公開API、MADBクエリ、カーソル、README、デモへ破壊的な変更が及ぶ
- ISBN-10とISBN-13の対応付けと入力位置への展開を誤ると、別の入力へ書籍を返す
- 取得元ごとに同じISBNへ対応する書籍件数と並び順が異なる

## 決定事項

- MADBで1回に受け付けるISBNは1件以上500件以下とする
- 1つのISBNに対応する複数MADBリソースはURI文字列昇順で個別に返す
- 1件から1,000件までの実測と500件採用理由は調査記録へ残す
