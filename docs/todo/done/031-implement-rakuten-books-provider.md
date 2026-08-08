# 楽天Booksプロバイダを実装する

## 背景

- 現状:
  - `manken` はMADB、openBD、Google Booksを実装済みで、複数プロバイダを用途に応じて利用するハブを目指している
  - TODO030 / RESEARCH030で楽天ブックス書籍検索APIの現行仕様、認証、検索条件、販売情報、アフィリエイト情報、利用条件を再調査した
  - 楽天Booksはタイトル、著者、ISBNの専用検索と漫画ジャンル絞り込みを利用でき、紙書籍の商品情報も取得できる
- 課題:
  - 既存の `__sample/book-api` は旧共通型を前提としており、現在の `api.Book`、Raw response、共通エラー、カーソル仕様と一致しない
  - Application IDとAccess Keyは必須だがAffiliate IDは任意であり、検索認証とアフィリエイト利用を分離する必要がある
  - 一般コミック、BL、TLは別ジャンルIDであり、共通 `SearchBooksRequest` に区分指定がないため、取得元固有の検索区分を安全に設定する方法が必要になる
  - 価格、在庫、アフィリエイトURLなどは書誌より変動が大きく利用条件も異なるため、初期実装で共通書誌へ混在させる範囲を限定する必要がある
- 変更が必要な理由:
  - RESEARCH030の調査結果を、現在の `manken` のパッケージ構造に沿った楽天Booksプロバイダとして実装するため

## 目的

- このTODOで達成すること:
  - `github.com/eamat-dot/manken/rakutenbooks` パッケージを追加する
  - Application IDとAccess Keyを使用して楽天ブックス書籍検索APIを検索できるようにする
  - Affiliate IDを任意設定とし、未設定でも検索できるようにする
  - Title、Author、Limit、Cursorを楽天Booksの検索条件へ変換する
  - 一般コミック、BLコミック、TLコミックを楽天Books固有のClient Optionで1区分ずつ選択できるようにする
  - ISBN-10 / ISBN-13を検証し、1件のISBNを1回のHTTPリクエストで参照できるようにする
  - 推測なしで対応できる書誌項目だけを `api.Book` へ変換する
  - 検索とISBN参照の成功レスポンスをRaw responseとして取得できるようにする
  - 楽天Books固有仕様、認証、Affiliate ID、レート制限、表示・保存条件を利用者向け文書へ反映する
- 期待する利用者視点の結果:
  - 楽天Booksだけを利用するために他プロバイダの認証情報を要求されない
  - Affiliate IDを持たない利用者もタイトル・著者・ISBN検索を利用できる
  - Affiliate IDを設定した利用者は、Raw responseから楽天が返した `affiliateUrl` を利用できる
  - 一般コミック、BL、TLの検索対象を明示して紙書籍の書誌候補を取得できる

## 根拠

- [`../../research/030-rakuten-books-api.md`](../../research/030-rakuten-books-api.md)
- [`../../research/027-provider-capabilities.md`](../../research/027-provider-capabilities.md)
- [`../../research/012-04-rakuten-comic-genres.md`](../../research/012-04-rakuten-comic-genres.md)
- [`../../spec.md`](../../spec.md)
- [`../../../ARCHITECTURE.md`](../../../ARCHITECTURE.md)
- `__sample/book-api/internal/apis/rakuten.go` はHTTPリクエスト例としてのみ参照し、そのまま移植しない

## 非目的

- 今回は扱わないこと:
  - 楽天Kobo APIの実装
  - 一般・BL・TLの3区分を1回の `SearchBooks` で内部横断して統合すること
  - `FreeText` または `ExcludedText` を後段フィルターだけで擬似対応すること
  - 商品タイトルから巻数、版、特装版、完結状態を推測すること
  - `author` 文字列を非公式な区切り規則で複数人物へ分解すること
  - 在庫、送料、レビュー、試し読みURLの共通モデル追加
  - 自動リトライ、キャッシュ、ログ、内部レートリミッター、バックグラウンドgoroutine
  - 複数プロバイダの横断検索、重複統合、販売情報共通モデルの確定
- 別TODOまたはbacklogで扱うこと:
  - 楽天Koboプロバイダ
  - DMM、楽天Kobo等を比較した在庫、送料、レビュー等の追加販売情報モデル
  - 複数漫画区分を横断する検索APIが必要になった場合の共通インターフェース設計

## 前提・制約

- 変更してよい範囲:
  - `rakutenbooks/` 新規パッケージ
  - `api.Source` の楽天Books定数追加
  - `internal/isbn` の既存処理の再利用
  - `examples/rakutenbooks/` と `examples/README.md`
  - `docs/pkg/rakutenbooks/spec.md` と `docs/pkg/rakutenbooks/guide.md`
  - `README.md`、`docs/spec.md`、`ARCHITECTURE.md`、`docs/backlog.md`、`docs/research/027-provider-capabilities.md`
  - 必要なら `.env.example` と `CHANGELOG.md`
- 変更してはいけない範囲:
  - MADB、openBD、Google Booksの公開APIと既存動作
  - 楽天固有の販売項目を理由なく `api.NormalizedBook` へ追加すること
  - 認証情報をリポジトリ、エラー、カーソル、Raw responseへ追加すること
  - `__sample/book-api` を本実装としてimportすること
- 互換性要件:
  - 既存プロバイダ利用コードが変更なしで動作する
  - `api.Source` への追加は後方互換な定数追加に留める
- パフォーマンス要件:
  - 検索は1ページにつき1HTTPリクエストとする
  - ISBN参照は1件だけ受け付け、1HTTPリクエストで完結させる
  - ライブラリ内部で待機や直列化を強制せず、楽天の「1 Application IDあたり1秒1回以下」の制限を利用者へ明示する

## 対象範囲

### 公開API・Client

- import pathは `github.com/eamat-dot/manken/rakutenbooks`
- `api.SourceRakutenBooks` のJSON値は `rakutenbooks`
- 既存プロバイダと同様に共通型、定数、エラーを型エイリアスで参照可能にする
- Clientは次のOptionを提供する
  - `WithApplicationID(string)` 必須
  - `WithAccessKey(string)` 必須
  - `WithAffiliateID(string)` 任意。空値を明示設定した場合は入力エラーとし、Option自体を指定しなければ未設定として扱う
  - `WithComicGenre(ComicGenre)` 任意。既定は一般コミック
  - `WithEndpoint(string)` テスト用
- `ComicGenre` は一般コミック、BLコミック、TLコミックだけを公開定数として提供し、内部で確認済み `booksGenreId` へ対応付ける
- Clientはリクエスト状態を保持せず、呼び出し側の `http.Client` / Transportを変更しない

### SearchBooks

- `SearchBooks` と `SearchBooksWithRawResponse` を提供する
- TitleまたはAuthorの少なくとも1つを必須にする
- `FreeText`、`ExcludedText` が非空なら外部通信前に `invalid_argument` とする
- `title` と `author` へ利用者文字列を値として渡し、Raw queryを受け取らない
- 選択された `ComicGenre` を `booksGenreId` に必ず設定する
- `format=json`、`formatVersion=2`、`sort=standard` を明示する
- 楽天の返却順を維持し、独自に並べ替えない

### Limit・カーソル

- `Limit == 0` の既定値は20件とする
- 有効範囲は1〜30件とする
- `page` は公開せず、次ページ番号を不透明なCursorに保持する
- Cursorはバージョン、次ページ番号、実効Limit、正規化済み検索条件と選択ジャンルのハッシュを保持する
- 異なる検索条件、Limit、ジャンルへCursorを再利用した場合は `invalid_argument`
- `pageCount` と現在ページから続きがある場合だけ `NextCursor` を生成し、100ページを超えない

### ISBN参照

- `LookupBooksByISBN` と `LookupBooksByISBNWithRawResponse` を提供する
- 入力は1件だけとする
- `internal/isbn` でISBN-10 / ISBN-13を検証し、問い合わせ用ISBN-13へ統一する
- `isbn=<ISBN-13>`、`formatVersion=2`、`hits=30`、`page=1` で1回だけ問い合わせる
- ISBN参照では漫画ジャンルを固定せず、入力ISBNに一致する紙書籍を参照する
- 返却ISBNを検証し、要求ISBNと一致する商品だけを結果へ含める
- `RequestedISBN` は入力文字列を保持する
- 0件はエラーにせず非nilの空 `Books` を返す

### レスポンス変換

初期変換では次を対象とする。

- `itemUrl` -> `BookSource.URL`。`affiliateUrl` は `BookSource.AffiliateURL` に分離する
- `BookSource.ID` は楽天Books固有の安定IDを確認できないため空にする
- `title` -> `Title`
- `titleKana` -> `TitleReading`
- `subTitle` -> `Subtitle`
- `seriesName` -> `Series[].Name`
- `author` -> `Authors` の1要素。非公式な `/` 分割をしない
- `author` / `authorKana` -> 役割なし `Contributor` 1要素として名前と読みを保持する
- `publisherName` -> `Publishers`
- `isbn` -> 検証済み `Identifiers`
- `salesDate` -> `BookDateTypeReleased`。原文精度を保持する
- `itemCaption` -> `Description`
- `booksGenreId` -> `/` で分割した楽天Books固有Schemeの `Subjects`
- `size` -> `PhysicalSize.Name`
- Books Book Searchの紙書籍結果 -> `PublicationMediumPrint`
- small / medium / large画像URL -> `Images`

`itemPrice` は取得時点の税込JPY価格として `Normalized.Prices`、`affiliateUrl` は `BookSource.AffiliateURL` へ変換する。`availability`、`postageFlag`、`limitedFlag`、レビュー、`chirayomiUrl`、`contents` は共通モデルへ追加せず、Raw responseから利用できる状態を維持する。

### Raw response・HTTP・エラー

- 検索とISBN参照の1回の2xx成功レスポンス本文を変更せず返す
- JSON解析・共通モデル変換失敗時も、上限内で読み込んだ2xx本文は返す
- 成功本文16 MiB、エラー本文64 KiBを上限とする
- `applicationId` と任意 `affiliateId` はquery、Access KeyはHTTP `accessKey` ヘッダーへ設定する
- 完全なリクエストURLを公開エラーへ埋め込まない
- HTTP 429、500〜599は `unavailable`、その他2xx以外は `upstream` を基本とする
- 通信失敗・タイムアウトは `unavailable`
- 入力、Client設定、Cursor不正は `invalid_argument`
- JSON不正、本文上限、ページング矛盾は `invalid_response`
- context cancel / deadlineを `errors.Is` で判定可能な状態を維持する
- 自動リトライしない

### 対象外

- 楽天ブックス総合検索API、ジャンル検索API、Kobo API
- 価格・在庫情報のキャッシュ機構
- Affiliate URLを通常商品URLへ置き換えること
- API取得値の永続保存をライブラリ側で行うこと

## 想定ユースケース

- 入力:
  - 必須のApplication IDとAccess Key
  - 任意のAffiliate ID
  - Title / Author、Limit、Cursor
  - 必要に応じてClient作成時の一般 / BL / TL区分
  - または1件のISBN
- 実行:
  - `rakutenbooks.Client` が楽天Booksへ問い合わせ、取得元固有レスポンスを検証して共通モデルへ変換する
- 出力:
  - 検索は `SearchBooksResult`、ISBN参照は `ISBNLookupResult`
  - 必要に応じて受信したRaw JSONも取得できる
  - Affiliate ID設定時の `affiliateUrl` はRaw responseに保持される
- 失敗時の扱い:
  - 入力不正、非対応検索条件、外部サービスエラー、通信失敗、不正成功応答を共通 `ErrorKind` で判定できる
  - 認証値を公開エラーへ露出しない

## 処理方針

1. `api.SourceRakutenBooks` と `rakutenbooks` の公開エイリアス、Client / Optionを追加する
2. SearchBooksの入力検証、ジャンル選択、Limit、Cursor、HTTP処理を実装して単体テストを固める
3. 非公開レスポンス型と安全な `api.Book` 変換を実装する
4. ISBN参照を `internal/isbn` と接続する
5. `integration` build tagで実サービス確認を追加し、認証、3ジャンル、ISBN、Affiliate IDあり・なしを確認する
6. examples、package spec / guideを作成し、README、共通spec、ARCHITECTURE、backlog、capability比較を同期する
7. 全体検証と差分レビュー後、全チェックが完了した場合だけTODOを `done/` へ移動する

## 実施項目

### 調査・実装前確認

- [x] RESEARCH030、RESEARCH027、共通spec、Google BooksのClient・エラー・Raw response・Cursor実装を確認する
- [x] `size=9` が一般・BL・TLの代表商品を取得できることを実測しつつ、漫画文庫等の別判型を落とすため固定条件にはしないと判断する
- [x] 一般・BL・TLは確認済みジャンルIDを1検索1区分で使い、既定を一般コミックとする

### 公開API・Client

- [x] `api.SourceRakutenBooks` とJSON値 `rakutenbooks` を追加する
- [x] `rakutenbooks` パッケージで通常利用に必要な共通型・定数・エラーをエイリアス公開する
- [x] `NewClient`、`WithApplicationID`、`WithAccessKey`、`WithAffiliateID`、`WithComicGenre`、`WithEndpoint` を実装する
- [x] 必須認証未設定、nil Option、不正genre、不正endpointを通信前に `invalid_argument` とする
- [x] Affiliate ID未設定でもClient作成と検索を可能にする
- [x] Application ID / Access Key等の認証値を公開エラー、Cursor、Raw responseへライブラリ側から追加しない。楽天Booksが返すAffiliate URLはそのまま保持する

### 検索・ページング

- [x] `SearchBooks` と `SearchBooksWithRawResponse` を実装する
- [x] Title / Authorの少なくとも1条件を必須にする
- [x] `FreeText` / `ExcludedText` を非対応として通信前に拒否する
- [x] 選択した漫画ジャンル、`formatVersion=2`、`sort=standard` をリクエストへ設定する
- [x] Limitの既定20、範囲1〜30を実装する
- [x] `page` を隠蔽する不透明Cursorを実装し、検索条件・Limit・ジャンルへ関連付ける
- [x] 続きがない場合または100ページ到達時は `NextCursor` を空にする
- [x] 楽天Booksの返却順を維持し、独自の並べ替えをしない

### ISBN参照

- [x] `LookupBooksByISBN` と `LookupBooksByISBNWithRawResponse` を実装する
- [x] ISBN入力を1件だけに制限し、ISBN-10 / ISBN-13を検証・正規化する
- [x] ISBN参照を1HTTPリクエストにし、検索ジャンルを固定しない
- [x] 返却ISBNが要求ISBNと一致する商品だけを対応付ける
- [x] `RequestedISBN` を保持し、該当なしを非nilの空 `Books` とする

### レスポンス変換

- [x] 必要最小限の非公開楽天Booksレスポンス型を実装する
- [x] 通常商品URL、タイトル、読み、サブタイトル、シリーズ、出版社、ISBN、発売日、説明を仕様どおり変換する
- [x] `author` / `authorKana` を分割せず1人分の表示文字列・読みとして保持し、役割を推測しない
- [x] 楽天ジャンルID、判型名、紙媒体、3サイズ画像を仕様どおり変換する
- [x] `BookSource.ID` をURL等から推測しない
- [x] `itemPrice` を取得時点価格、Affiliate URLを取得元参照情報として共通化し、在庫・レビュー・試し読み等は初期共通化しない
- [x] 欠落任意項目を正常として扱い、不正なページングやJSON等だけを `invalid_response` とする

### HTTP・エラー・秘密情報

- [x] Application IDをquery、Access KeyをHTTPヘッダー、Affiliate IDを設定時だけqueryへ付加する
- [x] 成功本文16 MiB、エラー本文64 KiBの上限を実装する
- [x] HTTP・通信・contextエラーを共通 `ErrorKind` へ分類する
- [x] リクエストURLや認証値を公開エラーへ含めない
- [x] 自動リトライ、内部レート制御、ログ、キャッシュを追加しない

### 単体テスト

- [x] Client Option、必須/任意認証、endpoint、nil Client / nil Contextをテストする
- [x] Title / Author、非対応FreeText / ExcludedText、ジャンル切替をテストする
- [x] Limit境界、Cursor往復、条件・Limit・ジャンル不一致、壊れたCursorをテストする
- [x] 0件、複数件、次ページあり/なし、100ページ境界をテストする
- [x] NormalizedBookの採用項目・非採用項目と任意項目欠落をテストする
- [x] ISBN入力検証、一致・不一致・該当なし、Raw responseをテストする
- [x] Affiliate IDの有無でquery、`BookSource.AffiliateURL`、Raw `affiliateUrl` の扱いが変わることをテストする
- [x] 429 / 5xx、その他HTTPエラー、通信失敗、本文上限、不正JSONをテストする
- [x] 認証値がエラー文字列へ出ないことを明示的にテストする
- [x] 通常テストが実楽天Booksへ接続しないことを維持する

### 実サービス確認

- [x] `integration` build tagの楽天Books統合テストを追加する
- [x] 必須認証を使い、タイトル・著者・ISBN検索を実サービスで確認する
- [x] 一般・BL・TLの各ジャンルで代表検索を実サービスで確認する
- [x] Affiliate IDあり・なしを実サービスで確認し、未設定でも検索可能、設定時は `BookSource.AffiliateURL` とRawに `affiliateUrl` が返ることを確認する
- [x] 実サービス呼び出しは1秒以上間隔を空ける
- [x] 認証情報が利用できない環境ではintegrationテストを明示的にskipする

### examples

- [x] `examples/rakutenbooks` に公開APIだけを使用するCLIデモを追加する
- [x] 認証情報はコマンドライン引数へ直接渡さず環境変数から読む
- [x] タイトル、著者、一般/BL/TL、Limit、Cursor、ISBNを動作確認できる構成にする
- [x] Affiliate IDは環境変数がある場合だけClientへ設定する
- [x] 検索とISBN参照のRaw responseを安全にファイル出力できるようにする
- [x] examples用テストと `examples/README.md` の全オプション・使用例を追加する

### ドキュメント

- [x] `docs/pkg/rakutenbooks/spec.md` を作成し、公開API、検索条件、ジャンル、Limit、Cursor、ISBN、変換、Raw、HTTP、エラーを現在仕様として記載する
- [x] `docs/pkg/rakutenbooks/guide.md` を作成し、認証情報、Affiliate ID、最小利用例、デモ、1秒1回制限、クレジット表示、保存・更新条件を案内する
- [x] `docs/spec.md` のSource一覧を同期する
- [x] `README.md` の対応プロバイダ、導入条件、主要機能、最小利用例を同期する
- [x] `ARCHITECTURE.md` の現在のパッケージ構成と依存関係を同期する
- [x] `docs/research/027-provider-capabilities.md` の楽天Booksを実装済み状態へ更新する
- [x] `docs/backlog.md` から完了した楽天Books実装候補を除き、将来項目だけを残す
- [x] `.env.example` の楽天認証・Affiliate ID案内が実装と一致することを確認する
- [x] `CHANGELOG.md` に利用者向けの楽天Booksパッケージ追加を記載する
- [x] README、spec、guide、examples READMEを全体として読み、役割や重複が不自然でないことを確認する

## 検証

- [x] gitignoredの `go/format.Source` 検証を `go test -v .` で実行し、TODO031で変更したGoファイルがgofmt相当の整形済みであることを確認する
- [x] `go test -v ./...` を実行する
- [x] `go test -run TestDoesNotExist ./...` を実行して全パッケージのコンパイルを確認する
- [x] `go test -vet=all ./...` を実行する
- [x] `go test -v -tags=integration ./rakutenbooks` を実行し、認証情報がプロセス環境にない場合のskip動作を確認する
- [x] gitignoredの実サービス検証で `.env` の楽天認証情報を読み込み、タイトル・著者・ISBN・一般/BL/TL・Affiliate IDあり/なしを確認する
- [x] gitignoredの別Goモジュールから `github.com/eamat-dot/manken/rakutenbooks` をimportして `go test -v .` を実行する
- [x] CodexProの差分確認で秘密情報、`__sample` の改変、既存プロバイダの意図しない変更がないことを確認する

CodexProのsafe bashでは `go build -v ./...` と `golangci-lint run` が許可されなかったため、
ビルド確認は `go test -run TestDoesNotExist ./...`、静的検査は `go test -vet=all ./...` で行った。

## 受け入れ条件

- [x] 別Goモジュールから `github.com/eamat-dot/manken/rakutenbooks` をimportできる
- [x] Application IDとAccess Keyを設定したClientでTitle / Author検索ができる
- [x] Affiliate ID未設定でも検索でき、設定時だけ楽天へAffiliate IDを送信する
- [x] 一般コミック、BL、TLをClient Optionで選択でき、1回の検索で選択区分だけを楽天へ指定する
- [x] `FreeText` / `ExcludedText` を対応しているように見せず、通信前に非対応として拒否する
- [x] Limit 1〜30とNextCursorで次ページを取得できる
- [x] ISBN入力を1件に制限し、1件の `ISBNLookupResult.Items` と1回のHTTPレスポンスを対応付ける
- [x] 楽天Books固有レスポンス型が公開APIへ露出していない
- [x] 通常商品URLとAffiliate URLを混同せず、`BookSource.URL` と `BookSource.AffiliateURL` に分離している
- [x] Application ID / Access Key等の認証値が公開エラー、Cursor、ドキュメント例へ漏れていない。Raw responseと `BookSource.AffiliateURL` には楽天Booksが返すAffiliate URLを保持する
- [x] `itemPrice` は取得時点価格として共通化し、在庫・レビュー等の未整理な変動販売情報は共通モデルへ混在させていない
- [x] 楽天Booksの結果をライブラリ内部で独自に並べ替えていない
- [x] README、共通spec、楽天Books spec / guide、ARCHITECTURE、examples READMEが実装と一致している
- [x] 既存プロバイダを含む通常テスト、コンパイル確認、`go vet`相当の検証が成功する

## リスク・懸念

- 一般・BL・TLが別ジャンルIDのため、既定の一般コミックだけではBL/TL専用ジャンルの商品を検索しない
- Client Optionでジャンルを固定するため、同じClientで検索ごとに区分を変更する用途には新しいClientが必要になる
- 楽天Booksは販売商品APIでもあるため、セット商品、特装版、関連商品等の後段判定が必要になる場合がある
- `author` の複数人物区切りが公式に保証されていないため、初期実装では表示文字列を分割しない
- `salesDate` は不完全な日付表現を含み得るため原文精度を保持する
- 楽天由来データには保存・更新・クレジット表示等の利用条件があり、Rawを返せることは無期限保存・再配布を許す意味ではない
- ライブラリが内部レート制御をしないため、並行利用時も呼び出し側が楽天のリクエスト制限を守る必要がある

## 未決事項

- TODO031の実装開始を妨げる未決事項はない
- 在庫、送料、レビューなど未共通化の販売情報は楽天Kobo、DMM等の比較後に判断する
- 複数漫画区分の横断検索が必要になった場合は共通検索設計と合わせて別TODOで検討する

## AIへの入力メモ（任意）

- 会話で決まった前提:
  - `.env` に `RAKUTEN_APP_ID`、`RAKUTEN_ACCESS_KEY`、`RAKUTEN_AFFILIATE_ID` がある
  - Affiliate IDは検索の必須条件にしない
  - TODO030の調査結果から続けて楽天Books providerを実装する
- 優先度:
  - Google Booksに続く紙書籍プロバイダとして楽天Booksを追加する
- 先に決めた論点:
  - パッケージ名は `rakutenbooks`
  - 既定検索区分は一般コミック、BL/TLはClient Optionで明示する
  - `size=9` はBL/TLも取得できるが漫画文庫等を落とし得るため固定しない
  - Affiliate URLは `BookSource.AffiliateURL`、`itemPrice` は取得時点価格として共通化し、通常商品URLや未整理な販売情報と混同しない
  - 内部レートリミッターは追加せず、利用者へ1秒1回以下の制限を案内する
