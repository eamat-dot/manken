# Google Booksプロバイダを実装する

## 背景

- 現状:
  - `manken` はMADBとopenBDを実装済みで、複数プロバイダを用途に応じて利用できるハブを目指している
  - TODO028 / RESEARCH028でGoogle Books API v1の現行仕様、検索条件、認証、主要レスポンス項目、利用条件を再調査した
  - Google Booksは漫画専用データベースではないが、タイトル・著者・ISBN・フリーワードで広く書誌候補を検索できる
- 課題:
  - 既存の `__sample/book-api` は旧 `CommonBook` 前提で、現在の `api.Book` / `NormalizedBook`、Raw response、共通エラー、カーソル仕様と一致しない
  - Google BooksはAPIキーが必要で、検索構文、ページング、ISBN参照上限、秘密情報の扱いをパッケージ固有仕様として明確にする必要がある
  - `saleInfo`、`accessInfo`、タイトルから推測する巻数・シリーズなどを初期実装へ混在させると、共通モデルの責務が広がりすぎる
- 変更が必要な理由:
  - RESEARCH028で採用範囲を整理できたため、現在の `manken` のパッケージ構造に沿ったGoogle Booksプロバイダを実装する

## 目的

- このTODOで達成すること:
  - `github.com/eamat-dot/manken/googlebooks` パッケージを追加する
  - APIキーを使用してGoogle Books Volumes APIを検索できるようにする
  - `SearchBooksRequest` のTitle、Author、FreeTextをGoogle Books検索構文へ安全に変換する
  - Limitと不透明なカーソルによるページングを提供する
  - ISBN-10 / ISBN-13を検証し、1件のISBNを1回のHTTPリクエストでGoogle Booksから参照できるようにする
  - 安全に意味を対応できるGoogle Books書誌項目だけを `api.Book` へ変換する
  - 検索成功レスポンスを `SearchBooksWithRawResponse` から取得できるようにする
  - Google Books固有仕様、APIキー、利用条件を利用者向け文書へ反映する
- 期待する利用者視点の結果:
  - Google Books APIキーを持つ利用者が、他プロバイダの認証情報を設定せず `googlebooks` だけを利用できる
  - タイトル・著者・フリーワード・ISBNから書誌候補を取得し、共通モデルと必要に応じてRaw responseを利用できる

## 根拠

- [`../research/028-google-books-api.md`](../research/028-google-books-api.md)
- [`../research/027-provider-capabilities.md`](../research/027-provider-capabilities.md)
- [`../spec.md`](../spec.md)
- [`../../ARCHITECTURE.md`](../../ARCHITECTURE.md)
- `__sample/book-api/internal/apis/google.go` はHTTPリクエスト例としてのみ参照し、そのまま移植しない

## 非目的

- 今回は扱わないこと:
  - Google Booksを漫画単行本だけへ完全に絞り込む判定ロジック
  - `saleInfo`、`accessInfo` の共通販売・閲覧モデル化
  - `NormalizedBook.Prices`、`Medium` をGoogle Booksの地域依存情報から設定すること
  - タイトル文字列からシリーズ、巻数、版表示、完結状態を推測すること
  - Google Books Volume IDによる直接取得API
  - OAuth 2.0、My Libraryなど利用者固有データ
  - 複数プロバイダの横断検索、重複統合、結果の並べ替え
  - 自動リトライ、キャッシュ、ログ、アクセス間隔制御、バックグラウンドgoroutine
  - 複数ISBNを1回のGoogle Books参照へまとめる独自バッチAPI
- 別TODOまたはbacklogで扱うこと:
  - Volume ID直接取得の必要性
  - 楽天Books / Kobo / DMM等を比較した販売情報共通モデル
  - Google Booksを含む横断検索の表示方法とBranding Guidelinesの整合確認

## 前提・制約

- 変更してよい範囲:
  - `googlebooks/` 新規パッケージ
  - `api.Source` のGoogle Books定数追加
  - `internal/isbn` の既存処理の再利用。Google固有仕様を共通処理へ混在させない
  - `examples/googlebooks/` と `examples/README.md`
  - `docs/pkg/googlebooks/spec.md` と `docs/pkg/googlebooks/guide.md`
  - `README.md`、`docs/spec.md`、`ARCHITECTURE.md`、`docs/backlog.md`、`docs/research/027-provider-capabilities.md`
  - 必要なら `CHANGELOG.md`
- 変更してはいけない範囲:
  - MADB / openBDの公開APIと既存動作
  - Google Books固有項目を理由なく `api.NormalizedBook` へ追加すること
  - APIキーをリポジトリへ記録すること
  - `__sample/book-api` を本実装として再利用・importすること
- 互換性要件:
  - 既存のMADB、openBD利用コードが変更なしで動作する
  - `api.Source` への追加は後方互換な定数追加に留める
- パフォーマンス要件:
  - 検索は1ページにつき1HTTPリクエストとする
  - ISBN参照は1件だけ受け付け、1 HTTPリクエストで完結させる

## 対象範囲

### パッケージ名と取得元

- import path: `github.com/eamat-dot/manken/googlebooks`
- `api.SourceGoogleBooks` を追加する
- JSONのSource値は `googlebooks` とする
- `googlebooks` パッケージから通常利用に必要な共通型、定数、エラーを型エイリアスとして参照できるようにする

### Clientと認証

公開APIは既存プロバイダと同じOption方式を基本とする。

```go
func NewClient(httpClient *http.Client, options ...Option) (*Client, error)
func WithAPIKey(apiKey string) Option
func WithEndpoint(endpoint string) Option
```

- APIキーは必須とし、空文字列または空白だけの場合は `invalid_argument`
- `NewClient(nil, WithAPIKey(key))` を利用可能にする
- APIキーはClient内部で保持し、Volumes APIへの `key` クエリパラメータにだけ使用する
- エラー文字列、ログ、カーソル、Raw responseへAPIキーを追加しない
- 完全なリクエストURLをエラーへ埋め込まない
- `WithEndpoint` はテスト用に絶対HTTP(S) URLへ差し替え可能とし、ユーザー情報、既存クエリ、フラグメントを許可しない
- `httpClient == nil` は既存プロバイダと同様に60秒タイムアウトのクライアントを使用する
- 呼び出し側の `http.Client` と `Transport` を変更しない
- Clientはリクエスト状態を保持せず、複数goroutineから安全に利用できるものとする

### SearchBooks

公開メソッド:

```go
func (client *Client) SearchBooks(
    ctx context.Context,
    request SearchBooksRequest,
) (SearchBooksResult, error)

func (client *Client) SearchBooksWithRawResponse(
    ctx context.Context,
    request SearchBooksRequest,
) (SearchBooksResult, []byte, error)
```

検索条件はGoogle BooksのRaw queryを呼び出し側から直接受け取らず、共通検索条件を安全な構文へ変換する。

- Title、Author、FreeTextの少なくとも1つを必須とする
- 各入力はUnicode空白で語へ分割し、Google検索演算子として解釈されないよう値を引用・エスケープする
- Titleの各語は `intitle:` を付ける
- Authorの各語は `inauthor:` を付ける
- FreeTextの各語は通常検索語として追加する
- 複数条件・複数語はGoogle BooksのAND相当になる形で組み合わせる
- `ExcludedText` はGoogle公式ドキュメントの除外検索例と実サービス確認を根拠に対応する
  - 各語をGoogle検索演算子として直接解釈させず、安全な `-term` 相当の条件へ変換する
  - `ExcludedText` だけの検索は許可せず、Title、Author、FreeTextの少なくとも1つを必須とする
  - 採用した動作を `docs/pkg/googlebooks/spec.md` とRESEARCH027へ反映する

Volumes listへは初期実装で次を固定する。

- `printType=books`
- `orderBy=relevance`
- `projection=full`
- `filter` は指定しない
- `langRestrict` は固定しない。日本語以外を暗黙に除外しない

`printType=books` は雑誌を除外するだけで漫画判定には使用しない。Google Booksの返却順を維持し、独自の順位付けや並べ替えをしない。

### Limitとカーソル

- `Limit == 0` の既定値は20件とする
- 有効範囲は1〜40件とする
- 40件を超える値は外部通信前に `invalid_argument`
- Google Booksの `startIndex` は公開せず、`SearchBooksResult.NextCursor` に不透明なカーソルとして保持する
- カーソルは少なくともバージョン、次の `startIndex`、正規化済み検索条件と実効Limitのハッシュを保持する
- 別の検索条件またはLimitへカーソルを再利用した場合は `invalid_argument`
- APIキーをカーソルへ含めない
- `items` が空または続きがない場合は `NextCursor` を空にする
- `totalItems` と今回取得件数から続きがある場合だけ次カーソルを生成する
- 返却されたitemsの順序を維持する

### ISBN参照

公開メソッド:

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

- 入力は1件だけとする。Google Booksに複数ISBN一括参照APIがないため、1 ISBNと1 HTTPレスポンスを対応させる
- `internal/isbn` でISBN-10 / ISBN-13を検証し、問い合わせ用に対応するISBN-13へ統一する
- `q=isbn:<ISBN-13>`、`printType=books`、`projection=full`、`maxResults=40`、`startIndex=0` で1回だけ検索する
- 40件を超える追加ページは取得しない
- 返却Volumeの `industryIdentifiers` を検証し、要求ISBNと同一と確認できるVolumeだけを結果へ対応付ける
- 同じISBNに複数Volumeが対応する場合は勝手に統合せず、1レスポンス内の一致候補をすべて返す
- `ISBNLookupResult.Items` は1要素とし、`RequestedISBN` は入力文字列を変更しない
- 該当なしはエラーにせず、非nilの空 `Books` を返す
- `LookupBooksByISBNWithRawResponse` は、その1回の2xx成功レスポンス本文を変更せず返す

### Google Booksレスポンスの変換

初期実装では必要な範囲だけを持つ非公開レスポンス型を定義し、未知のJSON項目は無視する。欠落した任意項目を正常な応答として扱う。

`api.Book` への対応は次を基本とする。

| Google Books | 共通モデル | 変換規則 |
| --- | --- | --- |
| `id` | `Sources[0].ID` | Google Books Volume IDをそのまま使用 |
| `volumeInfo.canonicalVolumeLink` | `Sources[0].URL` | 有効なHTTP(S) URLなら優先 |
| `volumeInfo.infoLink` | `Sources[0].URL` | canonicalがない場合のフォールバック |
| 固定値 | `Sources[0].Source` | `SourceGoogleBooks` |
| `volumeInfo.title` | `Normalized.Title` | タイトル全体を分解せず保持 |
| `volumeInfo.subtitle` | `Normalized.Subtitle` | そのまま保持 |
| `volumeInfo.authors[]` | `Normalized.Authors` | 空要素を除き応答順を保持 |
| `volumeInfo.authors[]` | `Normalized.Contributors` | 同じ人物を役割なしContributorとして保持。著者/編集者を推測しない |
| `volumeInfo.publisher` | `Normalized.Publishers` | 空でなければ1件 |
| `volumeInfo.publishedDate` | `Normalized.Dates` | `BookDateTypePublished`。年/月/日など取得元精度を保持 |
| `industryIdentifiers` の `ISBN_10` / `ISBN_13` | `Normalized.Identifiers` | 共通ISBN検証を通った値だけ対応する型で設定。取得元にない対になるISBNを生成して追加しない |
| `volumeInfo.description` | `Normalized.Description` | HTMLを含む可能性も含め、取得元文字列をサニタイズ・要約せず保持 |
| `volumeInfo.language` | `Normalized.Languages` | 空でなければ1件 |
| `volumeInfo.categories[]` | `Normalized.Subjects` | `Scheme="google_books"`、Nameに取得元値。漫画判定には使用しない |
| `volumeInfo.pageCount` | `Normalized.PageCount` | 正の値だけ保持し、0以下は不明値として未設定にする |
| `volumeInfo.imageLinks` | `Normalized.Images` | 非空URLだけを安定したフィールド順で設定し、PurposeにGoogle Booksのフィールド名を入れる |

次は初期実装で設定しない。

- `ParallelTitles`
- `TitleReading`
- `Series`
- `Volume`
- `EditionStatements`
- `IsFinalVolume`
- `Imprints`
- `Medium`
- `PhysicalSize`
- `Prices`
- rating関連
- `saleInfo`
- `accessInfo`
- `searchInfo`
- `contentVersion`

Volume IDはGoogle BooksのVolume itemでは必須とみなし、検索itemで空の場合は `invalid_response` とする。タイトルなど任意の書誌項目の欠落だけでは結果全体を失敗させない。

### Raw response

- `SearchBooksWithRawResponse` はGoogle Booksから受信した1回の検索2xx成功レスポンス本文を変更せず `[]byte` で返す
- `LookupBooksByISBNWithRawResponse` は1件のISBN参照で受信した1回の2xx成功レスポンス本文を変更せず `[]byte` で返す
- JSON解析または共通モデル変換に失敗した場合も、上限内で読み込み済みの2xx本文を返す
- 本文読込失敗、本文上限超過、2xx以外、通信失敗ではRaw本文を返さない
- APIキーはレスポンス本文へ追加しない。リクエストURLをRawとして返さない

### HTTPとエラー

- 既定エンドポイントは `https://www.googleapis.com/books/v1/volumes`
- GET、`Accept: application/json` を使用する
- 成功レスポンス本文の上限は16 MiBとする
- エラー本文の読込上限は64 KiBとする
- エラー本文を通常エラーメッセージや `api.Error` に保持しない
- HTTP 408、429、500〜599は `unavailable`
- その他の2xx以外は `upstream`
- 通信失敗・タイムアウトは `unavailable`
- 入力、APIキー、Client設定、カーソルの不正は `invalid_argument`
- JSON不正、本文上限、必須ID欠落、ページング矛盾は `invalid_response`
- `Retry-After` がある場合は既存プロバイダと同じ秒数形式 / HTTP-date形式を扱う
- `context.Canceled` と `context.DeadlineExceeded` を `errors.Is` で判定できる状態を維持する
- 自動リトライはしない

## 想定ユースケース

- 入力:
  - Google Books APIキー
  - Title / Author / FreeText / ExcludedText、Limit、Cursor
  - または1件のISBN
- 実行:
  - `googlebooks.Client` がGoogle Booksへ問い合わせ、取得元固有レスポンスを検証して共通モデルへ変換する
- 出力:
  - 検索は `SearchBooksResult`、ISBN参照は `ISBNLookupResult`
  - 検索では必要に応じて受信したRaw JSONも取得できる
- 失敗時の扱い:
  - 入力不正、外部サービスエラー、通信失敗、不正成功応答を共通 `ErrorKind` で判定できる
  - APIキーをエラーへ露出しない

## 処理方針

1. `api.SourceGoogleBooks` と `googlebooks` の公開エイリアス、Client / Optionの骨格を追加する
2. SearchBooksの入力検証、検索文字列生成、カーソル、HTTP処理を実装して単体テストを固める
3. Google Books非公開レスポンス型と安全な `api.Book` 変換を実装する
4. ISBN参照を既存 `internal/isbn` と同じ結果対応規則へ接続する
5. `integration` build tagで実サービス確認を追加し、ExcludedTextを含む代表検索の挙動を確認する
6. examplesとパッケージ仕様を作成し、README / 共通spec / ARCHITECTURE / backlog / capability比較を現在実装へ同期する
7. 全体検証後、TODOの全チェックが完了した場合だけ `docs/todo/done/` へ移動する

## 実施項目

### 調査・実装前確認

- [x] RESEARCH028、共通spec、MADB / openBDのClient・エラー・Raw response・カーソル実装を確認する
- [x] 現在のGoogle Books公式仕様とRESEARCH028に実装を妨げる差異がないことを確認する
- [x] 現在の未コミットTODO028関連変更を保持し、上書き・巻き戻ししない

### 公開API・Client

- [x] `api.SourceGoogleBooks` とJSON値 `googlebooks` を追加する
- [x] `googlebooks` パッケージで通常利用に必要な共通型・定数・エラーをエイリアス公開する
- [x] `NewClient`、`WithAPIKey`、`WithEndpoint` を実装する
- [x] APIキー未設定、nil Option、不正endpointを外部通信前に `invalid_argument` とする
- [x] APIキーをエラー、カーソル、Raw responseへ露出させない

### タイトル・著者・フリーワード検索

- [x] `SearchBooks` と `SearchBooksWithRawResponse` を実装する
- [x] Title / Author / FreeTextの少なくとも1条件を必須にする
- [x] 利用者入力をGoogle Books検索演算子として直接解釈せず、安全な検索式へ変換する
- [x] Titleを `intitle:`、Authorを `inauthor:`、FreeTextを通常検索語へ変換する
- [x] `printType=books`、`orderBy=relevance`、`projection=full` を設定する
- [x] `langRestrict` と電子書籍filterを初期実装で固定しない
- [x] Google Booksの返却順を維持し、独自の順位付けをしない
- [x] ExcludedTextはGoogle公式ドキュメントの除外検索例と実サービス確認を根拠に、安全な除外条件へ変換して対応する

### Limit・カーソル

- [x] Limitの既定20、範囲1〜40を実装する
- [x] `startIndex` を隠蔽する不透明カーソルを実装する
- [x] カーソルを正規化済み検索条件と実効Limitへ関連付ける
- [x] 異なる条件・Limit、壊れたカーソル、未知バージョンを `invalid_argument` とする
- [x] 続きがない場合は `NextCursor` を空にする

### ISBN参照

- [x] `LookupBooksByISBN` を実装する
- [x] ISBN入力を1件だけに制限する
- [x] ISBN-10 / ISBN-13の検証・正規化に `internal/isbn` を再利用する
- [x] ISBN検索を `maxResults=40`、`startIndex=0` の1 HTTPリクエストにする
- [x] 返却VolumeのISBNが要求ISBNと一致することを確認してから結果へ対応付ける
- [x] `RequestedISBN` に元の入力文字列を保持する
- [x] 該当なしを非nilの空 `Books` として返す
- [x] `LookupBooksByISBNWithRawResponse` で1回の成功レスポンス本文を返す

### レスポンス変換

- [x] 必要最小限の非公開Google Booksレスポンス型を実装する
- [x] Volume ID、canonical/info URL、title、subtitle、authors、publisher、publishedDateを仕様どおり変換する
- [x] Google Booksのauthorsを役割なしContributorとしても保持し、役割を推測しない
- [x] ISBN_10 / ISBN_13は検証済みの取得元値だけをIdentifiersへ設定する
- [x] descriptionはHTMLを含む場合も取得元文字列を保持する
- [x] language、categories、pageCount、imageLinksを仕様どおり変換する
- [x] タイトルからSeries / Volume / EditionStatements等を推測しない
- [x] saleInfo / accessInfo / Prices / Medium / PhysicalSizeを初期共通化しない
- [x] 欠落した任意項目を正常として扱い、Volume ID欠落など必須構造の不正だけを `invalid_response` とする

### HTTP・エラー・秘密情報

- [x] 既定endpoint、GET、Accept JSON、APIキーqueryを実装する
- [x] 成功本文16 MiB、エラー本文64 KiBの上限を実装する
- [x] HTTPステータスを共通 `ErrorKind` へ分類する
- [x] `Retry-After`、context cancel / deadlineを既存プロバイダと同じ規則で扱う
- [x] 外部エラー本文とAPIキーを公開エラーへ含めない
- [x] 自動リトライ、ログ、キャッシュ、内部並行実行を追加しない

### 単体テスト

- [x] APIキー、Option、endpoint、nil Client / nil Contextの入力検証をテストする
- [x] Title / Author / FreeTextと複数条件の検索式をテストする
- [x] 検索演算子・引用符等を利用者入力から注入できないことをテストする
- [x] Limit境界、カーソル往復、条件・Limit不一致、壊れたカーソルをテストする
- [x] 0件、1件、複数件、次ページあり/なしをテストする
- [x] NormalizedBookの各採用項目と非採用項目をテストする
- [x] roleless Contributor、ISBN不正値の除外、任意項目欠落をテストする
- [x] ISBN 1件・複数・重複・ISBN10/13同一書籍・該当なし・複数ページをテストする
- [x] ISBN検索で要求ISBNと一致しないVolumeを結果へ混在させないことをテストする
- [x] Raw responseが2xx本文を変更せず返し、解析失敗時も本文を返すことをテストする
- [x] 408 / 429 / 5xx、その他HTTPエラー、通信失敗、本文上限、不正JSONをテストする
- [x] APIキーがエラー文字列へ出ないことを明示的にテストする
- [x] 通常テストが実Google Booksへ接続しないことを維持する

### 実サービス確認

- [x] `integration` build tagのGoogle Books統合テストを追加する
- [x] `GOOGLE_BOOKS_API_KEY` が設定されている場合、タイトル・著者・ISBN・フリーワード検索を確認する
- [x] `printType=books`、Limit=40、startIndexによるページングを代表例で確認する
- [x] ExcludedTextの除外構文をGoogle公式ドキュメントと実サービスで確認し、共通条件として採用して仕様へ反映する
- [x] APIキーが利用できない環境では統合テストを明示的にskipし、単体テストと混同しない

### examples

- [x] `examples/googlebooks` に公開APIだけを使用するCLIデモを追加する
- [x] APIキーはコマンドライン引数へ直接渡さず、`GOOGLE_BOOKS_API_KEY` 環境変数から読む
- [x] タイトル、著者、フリーワード、除外条件、Limit、Cursor、ISBNを動作確認できる構成にする
- [x] 検索と1件のISBN参照のRaw responseを既存デモと同様に安全にファイル出力できるようにする
- [x] examples用テストを追加し、`examples/README.md` に全オプションと使用例を記載する

### ドキュメント

- [x] `docs/pkg/googlebooks/spec.md` を作成し、公開API、検索条件、Limit、カーソル、ISBN、変換、Raw、HTTP、エラーを現在仕様として記載する
- [x] `docs/pkg/googlebooks/guide.md` を作成し、APIキーの準備、最小利用例、環境変数を使うデモ実行方法を利用者向けに記載する
- [x] Google Books Terms / Brandingへの公式リンクと、APIキー、attribution / link、保存・キャッシュ、課金形態、結果順の注意をguideを中心に利用判断に必要な粒度で記載する
- [x] `docs/spec.md` のSource一覧を現在実装へ同期する
- [x] `README.md` の対応プロバイダ、導入条件、インストール、主要機能、最小利用例をGoogle Books対応へ更新する
- [x] `ARCHITECTURE.md` の現在のパッケージ構成と依存図を更新する
- [x] `docs/research/027-provider-capabilities.md` のGoogle Booksを実装済み状態へ更新する
- [x] `docs/backlog.md` から完了したGoogle Books実装候補を除き、今回対象外にした将来項目だけを必要に応じて残す
- [x] `CHANGELOG.md` の現行運用に照らして追加が必要なら利用者向け変更として記載する
- [x] README、spec、examples READMEを差分だけでなく全体として読み、文書の役割や記述の重複が不自然でないことを確認する

## 検証

- [x] `gofmt` またはプロジェクトで採用している整形を実行する
- [x] `go test -v ./...` を実行する
- [x] `go test -run TestDoesNotExist ./...` または利用可能な現行build確認を実行する
- [x] `golangci-lint run` または `task lint` の現行lintを実行する
- [x] `GOOGLE_BOOKS_API_KEY` が利用可能なら `go test -v -tags=integration ./googlebooks` を実行する。利用不可ならskip理由をTODOへ記録する
- [x] CodexPro / Gitの差分確認で意図しない秘密情報、`__sample` の改変、TODO028成果の巻き戻しがないことを確認する
- [x] 通常テストと外部API結合確認の結果を分けて記録する

## 受け入れ条件

- [x] 別Goモジュールから `github.com/eamat-dot/manken/googlebooks` をimportできる
- [x] `WithAPIKey` を設定したClientでTitle / Author / FreeText検索ができる
- [x] Limit 1〜40とNextCursorでGoogle Booksの次ページを取得できる
- [x] ExcludedTextはGoogle公式ドキュメントと実測結果を根拠に、安全な除外条件へ変換して対応している
- [x] Google BooksではISBN入力を1件に制限し、1件の `ISBNLookupResult.Items` と1回のHTTPレスポンスを対応付ける
- [x] Google Books固有レスポンス型が公開APIへ露出していない
- [x] 初期変換項目とRawに残す項目をパッケージ仕様から確認できる
- [x] APIキーがエラー、Raw response、カーソル、ドキュメント例へ漏れていない
- [x] Google Booksの結果をライブラリ内部で独自に並べ替えていない
- [x] saleInfo / accessInfo / 漫画判定 / 巻数推測など非目的の機能を混在させていない
- [x] README、共通spec、パッケージspec、Google Books guide、ARCHITECTURE、examples READMEが実装と一致している
- [x] MADB / openBDを含む通常テストとlintが成功する

## リスク・懸念

- Google Booksの検索結果は漫画以外の書籍を含むため、プロバイダ実装が漫画判定まで保証したように説明しない
- `volumeInfo.authors` は著者と編集者の役割を区別できないため、共通役割を推測しない
- Google Booksの書誌項目は欠落する場合があり、検索語や地域によって結果が変化する
- `saleInfo` / `accessInfo` は地域依存であり、書誌情報と同じ安定性を前提にしない
- Branding GuidelinesはGoogle Books検索結果の並べ替え・改変を制限するため、将来の横断検索設計に影響する
- APIキーはURL queryへ入るため、HTTPエラー構築時にURL文字列を含めると秘密情報漏えいになる
- ISBN参照は1入力1検索が基本となり、openBDよりHTTPリクエスト数が多い

## 未決事項

なし。

## 実行記録

- 2026-08-07: `gofmt`、`go test -v ./...`、`go test -run TestDoesNotExist ./...`、
  `golangci-lint run`、`go build -v ./...`、`git diff --check` が成功した
- 2026-08-07: `go test -v -tags=integration ./googlebooks` は
  `GOOGLE_BOOKS_API_KEY is not set` と表示してskipした。通常テストは`httptest`だけを使い、
  実Google Booksへ接続していない
- 2026-08-07: `task dev` 経由で `GOOGLE_BOOKS_API_KEY` を設定し、`go test -v -tags=integration ./googlebooks` が成功した。
  タイトル・著者・フリーワード・ISBN検索、Limit=40、NextCursorによる次ページ取得をGoogle Books実サービスで確認した
- 2026-08-07: `__sandbox/googlebooks-import-check` に独立 `go.mod` を作成し、`replace github.com/eamat-dot/manken => ../..` で
  `github.com/eamat-dot/manken/googlebooks` をimportする `go test ./...` が成功した。`__sandbox` はGit管理対象外とする
- 2026-08-07: ChatGPTレビューで、`http.Client.Do` の通信エラーがAPIキー付きURLを
  `*url.Error` 経由で公開エラーへ含める可能性を確認した。URLを除去して根本原因だけを保持するよう修正し、
  APIキー非露出の回帰テストを追加した。`go test ./googlebooks` と `go test ./...` は成功した
- 2026-08-07: `-term` の実サービス確認で、`intitle:\"動物のお医者さん\"` が `totalItems=300`、
  `intitle:\"動物のお医者さん\" -動物` が `totalItems=0` となり、除外構文が検索結果へ作用することを確認した。
  最終レビューでGoogle公式ドキュメントにも除外検索例が明記されていることを再確認し、共通 `ExcludedText` を安全な除外条件へ変換して対応した
- 2026-08-07: CLIデモの整形済みJSONでURL中の `&` が `\u0026` へHTMLエスケープされることを確認した。
  `json.Encoder.SetEscapeHTML(false)` をGoogle Books、MADB、openBDの各デモへ設定し、Google Booksには回帰テストを追加した。
  `go test ./examples/googlebooks ./examples/madb ./examples/openbd` と `go test ./...` は成功した
- 2026-08-07: ISBN参照で `-raw-output` を利用できない問題を受け、Google Books固有のISBN参照を1件入力・1 HTTPリクエストへ単純化した。
  `LookupBooksByISBNWithRawResponse` を追加し、CLIでもISBN参照と `-raw-output` を併用可能にした。複数ISBNは通信前に `invalid_argument` とする。
  ISBN検索は `maxResults=40`、`startIndex=0` の1レスポンスだけを扱い、Raw本文との1対1対応を維持する。`go test ./...` は成功した
- 2026-08-07: ISBN実レスポンスで、ISBN一致Volumeの `pageCount` が0、ISBNを持たない関連Volumeでは正の値となる例を確認した。
  別Volumeからページ数を補完・マージせず、Google Booksの `pageCount <= 0` は不明値として `Normalized.PageCount` を未設定にするよう変更した。
- 2026-08-07: mainマージ前の最終レビューでGoogle公式「Using the API」に `q=-term` の除外検索が明記されていることを再確認した。
  それまでの「公式仕様で保証されない」という判断を訂正し、`ExcludedText` の各語を安全な除外条件へ変換して対応した。
  README、共通spec、Google Books spec、RESEARCH027/028、examples READMEを同期し、`go test ./...` は成功した。
  `go test -v -tags=integration ./googlebooks` もコンパイル・通常テストは成功し、実サービス部分はこの環境にAPIキーがないためskipした。
- 2026-08-08: ローカル実サービスで著者 `佐々木倫子`、`ExcludedText=動物` を確認したところ `totalItems=0` となった。
  Google Booksの除外はタイトルだけでなくfull-text query全体に作用し、通常検索結果の「月館の殺人」も説明文に『動物のお医者さん』を含むため除外対象になり得る。
  この取得元固有の広い除外範囲をGoogle Books spec、examples README、RESEARCH028へ明記し、integrationテストも選択的除外を前提としない検証へ修正した。
- 2026-08-08: merge前の `task all` で `googlebooks/api_external_test.go` の公開APIシグネチャ確認がstaticcheck `QF1011` に該当した。
  明示型付きの空代入を、期待シグネチャを引数に要求するテストヘルパーへ置き換え、型変更をコンパイルで検出する意図を維持した。修正後の `go test ./...` は成功した。

## AIへの入力メモ（任意）

- 会話で決まった前提:
  - TODO028の調査を私が完了し、Google Books実装はCodexへ引き継ぐ
  - Google Booksは漫画専用の主検索元ではなく、広い書誌検索を補うプロバイダとして扱う
  - `NormalizedBook` とRaw responseは役割の違う出力として両方維持する
  - 利用しないプロバイダの認証情報は要求しない
- 優先度:
  - Google Booksを3番目の実装済み取得元として追加する
- 先に決めた論点:
  - パッケージ名は `googlebooks`
  - 必須APIキーは既存Option方式の `WithAPIKey` で設定する
  - 初期検索は `printType=books`、`orderBy=relevance`、`projection=full`、`langRestrict` なし
  - Limit既定20、最大40
  - ISBN入力は1件だけとし、1 HTTPリクエストで参照する
  - ISBN参照は `LookupBooksByISBNWithRawResponse` で1回の成功レスポンス本文を取得できる
  - saleInfo / accessInfo / 巻数等の推測は初期共通化しない
