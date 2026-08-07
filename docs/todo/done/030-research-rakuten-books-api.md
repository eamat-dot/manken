# 楽天ブックス書籍検索APIの現行仕様とmankenでの利用範囲を再調査する

## 背景

- 現状:
  - `manken` は複数の書籍・電子書籍・販売系プロバイダを用途に応じて利用するハブを目指している
  - RESEARCH027で楽天ブックス書籍検索APIは、紙書籍の商品検索と書誌情報取得の有力候補として整理した
  - RESEARCH012系では2026-07-31の実API結果を含む検索・ジャンル・項目比較を行っている
  - Google Booksプロバイダ実装後の次候補として、楽天Booksプロバイダの調査・実装を進める段階にある
- 課題:
  - 既存調査には現行の認証方式、全検索パラメータ、ページング、エラー、利用条件の整理が不足している
  - 楽天Booksは書誌情報と価格・在庫・商品URL等を同時に返すため、書誌プロバイダと販売情報プロバイダのどちらの責務を持たせるか整理する必要がある
  - `affiliateId` は検索そのものには必須ではないが、アフィリエイトURL生成に影響するため、通常の商品URLと分けて扱う必要がある
- 変更が必要な理由:
  - 後続の楽天Booksプロバイダ実装TODOで、認証・検索条件・正規化範囲・Raw response・アフィリエイト情報を推測せず実装できる調査根拠を残すため

## 目的

- このTODOで達成すること:
  - 楽天ブックス書籍検索APIの2026年8月時点の公式仕様を確認する
  - 認証、検索条件、ページング、主要レスポンス項目、エラー、利用制限・表示・保存条件を整理する
  - `RAKUTEN_APP_ID`、`RAKUTEN_ACCESS_KEY`、任意の `RAKUTEN_AFFILIATE_ID` の役割を区別する
  - アフィリエイトIDを指定した場合と指定しない場合のレスポンス差を確認する
  - 既存のRESEARCH012系とRESEARCH027の前提が現在も妥当か確認する
  - mankenで採用する検索・取得capabilityと、後続判断に回す販売情報の範囲を整理する
  - 調査結果を `docs/research/030-rakuten-books-api.md` にまとめる
- 期待する利用者視点の結果:
  - 楽天Booksを使うために必要な認証情報と、任意のアフィリエイト設定が分かる
  - 何で検索でき、どの情報を共通化でき、何をRaw responseまたは販売情報として扱うべきかの判断根拠が分かる
  - 後続の楽天Books実装TODOを現在のAPI仕様に基づいて作成できる

## 非目的

- 今回は扱わないこと:
  - 楽天BooksプロバイダのGo実装
  - 楽天Kobo電子書籍検索APIの再調査・実装
  - 公開API、共通型、`NormalizedBook` の変更
  - 横断検索の実装
  - 販売情報・アフィリエイト情報の共通モデル確定
  - MCPツールの実装・Schema確定
- 別Issueや別TODOで扱うこと:
  - 楽天Booksプロバイダの実装、テスト、guide/spec/examples
  - 楽天Kobo APIの現行仕様再調査と実装
  - 複数販売系プロバイダを比較した販売情報・アフィリエイト情報の共通モデル設計

## 前提・制約

- 変更してよい範囲:
  - `docs/research/030-rakuten-books-api.md`
  - `docs/backlog.md`
  - `docs/research/027-provider-capabilities.md` の軽微な同期
  - TODO030本文
  - 実API確認に必要なコミット対象外の `__sandbox/` 調査補助物
- 変更してはいけない範囲:
  - Goコードと公開API
  - 現行機能を表す `docs/spec.md` とパッケージ仕様を、楽天Booksが実装済みであるように変更しない
  - `.env` の認証値を文書、ログ、保存レスポンスへ記録しない
  - 過去の調査記録を現在の仕様に合わせて改変しない
- 互換性要件:
  - 現行のMADB、openBD、Google Books、共通APIの仕様を変更しない
- パフォーマンス要件:
  - 実API確認では公式のリクエスト制限を守り、同一URLへの短時間の反復アクセスを避ける

## 対象範囲

### 対象

- 楽天ブックス書籍検索API `BooksBook/Search/20170404`
- 必須認証の `applicationId` と `accessKey`
- 任意の `affiliateId`
- `formatVersion` とJSONレスポンス構造
- `title`、`author`、`publisherName`、`size`、`isbn`、`booksGenreId`
- `hits`、`page`、`availability`、`outOfStockFlag`、`chirayomiFlag`、`sort`、`limitedFlag`、`genreInformationFlag`
- タイトル・ISBN・著者・漫画ジャンル検索の適性
- ISBN、タイトル、サブタイトル、シリーズ、著者、出版社、発売日、説明、画像、ジャンル等の書誌候補
- 価格、在庫、商品URL、試し読みURL、レビュー、送料等の商品・販売情報
- `itemUrl` と `affiliateUrl` の役割分担
- HTTPエラーと共通 `ErrorKind` への対応候補
- レート制限、データ保存・更新、クレジット表示など利用判断に影響する条件

### 対象外

- 楽天ブックス総合検索API、CD/DVD/ゲーム等の他カテゴリAPI
- 楽天Kobo API
- 楽天市場商品検索API
- 楽天会員固有情報を扱うAPI
- 検索品質の大規模ベンチマーク

## 想定ユースケース

- 入力:
  - タイトル、ISBN、著者名、必要に応じて漫画ジャンルなどの検索条件
  - 必須のApplication IDとAccess Key
  - アフィリエイトURLが必要な場合だけAffiliate ID
- 実行:
  - 楽天ブックス書籍検索APIへ問い合わせ、紙書籍の商品情報と書誌候補を取得する
- 出力:
  - 安全に共通化できる項目は将来 `NormalizedBook` として利用し、販売情報と楽天固有情報はRaw responseまたは別モデルから利用できる候補とする
  - Affiliate IDを指定した場合だけアフィリエイトURLを利用できる候補とする
- 失敗時の扱い:
  - 公式仕様または実データで意味を確認できない項目は推測で共通化しない
  - 認証情報をエラー文や保存Raw responseへ含めない実装方針を後続TODOへ引き継ぐ

## 処理方針

1. 現行公式ドキュメントからAPIバージョン、認証、検索条件、ページング、レスポンス、エラー、利用制限を確認する
2. RESEARCH012系、RESEARCH027、既存 `__sample/book-api` の楽天実装と照合し、古い前提や不足を洗い出す
3. `.env` の `RAKUTEN_APP_ID` と `RAKUTEN_ACCESS_KEY` を使い、必要最小限の代表リクエストを実行して現行APIと既存調査を照合する
4. `RAKUTEN_AFFILIATE_ID` の有無で代表リクエストを比較し、検索結果と `affiliateUrl` の差を確認する。認証値そのものは保存しない
5. mankenで採用するcapability、正規化候補、Rawに残す情報、販売・アフィリエイト情報、実装時の再確認事項を整理する
6. RESEARCH030を作成し、backlogとRESEARCH027を必要な範囲だけ同期する

## 実施項目

### 調査

- [x] 楽天ブックス書籍検索APIの現行公式リファレンス、バージョン、エンドポイントを確認する
- [x] `applicationId`、`accessKey`、任意の `affiliateId` の要件を確認する
- [x] 検索条件、絞り込み、並び順、ページング、`formatVersion` を確認する
- [x] 主要レスポンス項目、0件時、HTTPエラーを確認する
- [x] レート制限、クレジット表示、データ保存・更新、アフィリエイト利用条件を確認する
- [x] 既存RESEARCH012系、RESEARCH027、`__sample/book-api` の前提を現行仕様と照合する
- [x] 代表実リクエストでタイトル・ISBN・著者・漫画ジャンル検索を確認する
- [x] Affiliate IDあり・なしの代表実リクエストを比較する
- [x] RESEARCH027の標準チェックリストに沿って不足項目を確認する

### 実装

- [x] `docs/research/030-rakuten-books-api.md` を作成する
- [x] mankenで採用候補とする検索capabilityを整理する
- [x] `NormalizedBook` へ安全に変換できる候補とRaw responseへ残す情報を分ける
- [x] 書誌参照URL、商品URL、アフィリエイトURLの役割を整理する
- [x] 価格・在庫等を初期の書誌プロバイダ実装へ含める範囲を整理する
- [x] 後続の楽天Books実装TODOで必要な入力条件と受け入れ条件を整理する

### テスト

- [x] Goコードと公開APIを変更していないことを確認する
- [x] 認証情報がTODO、RESEARCH、保存ファイル、差分へ含まれていないことを確認する
- [x] 公式仕様、実測結果、既存資料の情報源と確認水準を混同していないことを確認する
- [x] 未確認事項を推測で「対応可能」としていないことを確認する

### ドキュメント

- [x] `docs/backlog.md` の楽天Books項目を調査結果に同期する
- [x] `docs/research/027-provider-capabilities.md` の楽天Books評価を必要に応じて更新する
- [x] `docs/concept.md`、`docs/idea.md`、現行specの修正要否を確認する

## 検証

- [x] `go test -v ./...` を実行する
- [x] `go test -run TestDoesNotExist ./...` を実行して全パッケージのコンパイルを確認する
- [x] CodexProの `show_changes` でTODO030由来の変更範囲と文書間の整合を確認する

## 受け入れ条件

- [x] 楽天ブックス書籍検索APIの現在の認証、検索条件、ページング、エラーが整理されている
- [x] `RAKUTEN_APP_ID` と `RAKUTEN_ACCESS_KEY` は必須、`RAKUTEN_AFFILIATE_ID` は任意という役割が根拠とともに整理されている
- [x] Affiliate IDの有無による `affiliateUrl` の扱いが明確になっている
- [x] タイトル、ISBN、著者、漫画ジャンル検索をmankenで利用する際の意味と制約が分かる
- [x] `NormalizedBook`、Raw response、商品・販売情報、アフィリエイト情報の役割分担が整理されている
- [x] 楽天APIの表示・保存・更新・レート制限など、利用者へ案内すべき主要条件が整理されている
- [x] 既存楽天サンプルからそのまま流用できる点と、変更・再設計が必要な点が区別されている
- [x] 後続の楽天Booksプロバイダ実装TODOを作成できるだけの調査結果がある
- [x] Goコード、公開API、現行仕様の動作を変更していない

## リスク・懸念

- 楽天Booksは書誌データベースではなく販売商品APIでもあるため、価格・在庫・URL等を共通書誌へ混在させると責務が曖昧になる
- `salesDate` は不完全な日付表現を含み得るため、機械的な日付型への変換はできない
- 商品タイトルから巻数、版、特装版等を推測して分解すると誤変換の可能性がある
- 価格・在庫等は変動情報であり、楽天の保存期間・更新頻度・表示条件に従う必要がある
- 楽天APIの利用規約は、取得情報の利用目的や収益化方法に制約を設けているため、ライブラリが返すデータと利用者側の表示・利用責任を区別して案内する必要がある
- API仕様や認証要件は変更され得るため、後続実装でも現行公式仕様を再確認する

## 未決事項

- TODO030の調査結果を踏まえて、楽天Books初期実装に価格・在庫・通常商品URLをどこまで含めるか決める
- `affiliateUrl` を楽天Booksパッケージ固有Raw responseだけで提供するか、将来の販売情報モデルへ含めるかは、楽天Kobo・DMM等との比較後に決める
- Raw responseを利用者へ返す場合の楽天APIデータ保存条件への対応方法は、実装TODOで公開APIの形と合わせて決める

## AIへの入力メモ（任意）

- 会話で決まった前提:
  - TODO030は楽天Books APIの調査TODOとし、Go実装は後続TODOへ分離する
  - `.env` には `RAKUTEN_APP_ID`、`RAKUTEN_ACCESS_KEY`、`RAKUTEN_AFFILIATE_ID` が設定されている
  - アフィリエイト利用は検索の必須要件にしない
- 優先度:
  - Google Books実装完了後の次プロバイダとして楽天Booksを調査する
- 先に決めたい論点:
  - 楽天Booksを書誌検索と商品検索のどこまでに使うか
  - 通常商品URLとアフィリエイトURLをどう分離するか
