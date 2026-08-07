# Google Books APIの現行仕様とmankenでの利用範囲を再調査する

## 背景

- 現状:
  - `manken` は複数の書籍・電子書籍・販売系プロバイダを用途に応じて利用するハブを目指している
  - RESEARCH027でGoogle Books APIは、タイトル・ISBN・著者検索と書誌情報取得の有力候補として整理した
  - 2026-07-31の既存調査では実レスポンスを確認しているが、検索パラメータ、ページング、認証条件などは既存サンプルに依存する部分が残っている
- 課題:
  - Google Books APIを実装する前に、現在の公式仕様と既存サンプルの前提が一致しているか再確認する必要がある
  - タイトル検索結果へ原作・派生小説などが混在するため、漫画単行本の検索元としてどこまで安全に利用できるかを決める必要がある
  - 書誌情報、販売情報、Raw responseのどこまでをmankenで扱うかを、現在の設計方針に合わせて整理する必要がある
- 変更が必要な理由:
  - 後続のGoogle Booksプロバイダ実装TODOで、検索条件や変換範囲を推測せず実装できる調査根拠を残すため

## 目的

- このTODOで達成すること:
  - Google Books APIの2026年8月時点の公式仕様を確認する
  - 検索条件、ページング、認証、主要なレスポンス項目、販売・閲覧情報、利用上の制約を整理する
  - 既存のRESEARCH012系と `__sample/book-api` の前提が現在も妥当か確認する
  - mankenで採用する検索・取得capabilityと、採用しないまたは後続判断とする範囲を整理する
  - 調査結果を `docs/research/028-google-books-api.md` にまとめる
- 期待する利用者視点の結果:
  - Google Booksを使う場合に、何で検索でき、どの情報を共通化でき、何をRaw responseへ残すかの判断根拠が分かる
  - 後続の実装TODOを、現在のAPI仕様に基づいて作成できる

## 非目的

- 今回は扱わないこと:
  - Google BooksプロバイダのGo実装
  - 公開API、共通型、`NormalizedBook` の変更
  - 横断検索の実装
  - MCPツールの実装・Schema確定
  - Google Booksの検索結果だけを根拠に、漫画単行本の完全な判定規則を確定すること
- 別Issueや別TODOで扱うこと:
  - Google Booksプロバイダの実装、テスト、guide/spec/examples
  - 必要に応じた販売情報共通モデルの設計
  - 複数プロバイダをまたぐ重複統合と横断検索

## 前提・制約

- 変更してよい範囲:
  - `docs/research/028-google-books-api.md`
  - `docs/backlog.md`
  - `docs/research/027-provider-capabilities.md` の軽微な同期
  - TODO028本文
- 変更してはいけない範囲:
  - Goコードと公開API
  - 現行機能を表す `docs/spec.md` とパッケージ仕様を、Google Booksが実装済みであるように変更しない
  - 過去の調査記録を現在の仕様に合わせて改変しない
- 互換性要件:
  - 現行のMADB、openBD、共通APIの仕様を変更しない
- パフォーマンス要件:
  - 文書調査のみのため該当なし

## 対象範囲

### 対象

- Volumes APIの検索と個別取得
- `q` と検索語の特殊キーワード
- `startIndex`、`maxResults`、`orderBy`、`filter`、`printType`、`projection` など検索制御
- APIキー、OAuth、公開データ取得時の認証要否
- `volumeInfo`、`saleInfo`、`accessInfo`、`searchInfo` の主要項目
- ISBN、Google Books内ID、URL、画像、価格、電子書籍情報
- Raw responseを返す場合の注意点
- 漫画単行本検索への適性と後段判定の必要性

### 対象外

- My Libraryなど利用者固有データを扱うBooks API
- OAuthを必要とする利用者ライブラリ操作の実装方針
- Google Books以外のGoogle API
- 検索品質の大規模ベンチマーク

## 想定ユースケース

- 入力:
  - タイトル、ISBN、著者名などの書籍検索条件
- 実行:
  - Google Books APIへ問い合わせ、書誌情報と必要に応じて販売・閲覧情報を取得する
- 出力:
  - 安全に共通化できる項目は将来 `NormalizedBook` として利用し、Google固有情報はRaw responseで参照できる候補とする
- 失敗時の扱い:
  - 公式仕様または実データで意味を確認できない項目は推測で共通化せず、後続実装の未決事項として残す

## 処理方針

1. 現行公式ドキュメントからVolumes API、検索構文、認証、制限、レスポンス項目を確認する
2. RESEARCH012系、RESEARCH027、既存Googleサンプルと照合し、古い前提や不足を洗い出す
3. 2026-07-31の実API結果を再確認し、2026-08-07の公式仕様と照合する。同日の有効なAPIキーを使った再実行は実装TODOの結合確認へ回す
4. mankenで採用するcapability、正規化候補、Rawに残す項目、実装時の再確認事項を整理する
5. RESEARCH028を作成し、backlogとRESEARCH027を必要な範囲だけ同期する

## 実施項目

### 調査

- [x] Google Books APIの現行公式リファレンスと認証要件を確認する
- [x] Volumes検索の `q`、特殊キーワード、検索制御パラメータ、ページングを確認する
- [x] Volumesリソースの主要レスポンス項目を確認する
- [x] 既存RESEARCH012系とGoogleサンプルの前提を現行仕様と照合する
- [x] 2026-07-31の代表実リクエストでタイトル・ISBN・著者検索と漫画検索上の注意点を確認し、現行公式仕様と照合する
- [x] RESEARCH027の標準チェックリストに沿って不足項目を確認する

### 実装

- [x] `docs/research/028-google-books-api.md` を作成する
- [x] mankenで採用候補とする検索capabilityを整理する
- [x] `NormalizedBook` へ安全に変換できる候補とRaw responseへ残す情報を分ける
- [x] 販売・閲覧情報を今回の書誌プロバイダ実装へ含める範囲を整理する
- [x] 後続のGoogle Books実装TODOで必要な入力条件と受け入れ条件を整理する

### テスト

- [x] コード変更がないことを確認する
- [x] 公式仕様、実測結果、既存資料の情報源と確認水準を混同していないことを確認する
- [x] 未確認事項を推測で「対応可能」としていないことを確認する

### ドキュメント

- [x] `docs/backlog.md` のGoogle Books項目を調査結果に同期する
- [x] `docs/research/027-provider-capabilities.md` のGoogle Books評価を必要に応じて更新する
- [x] `docs/concept.md`、`docs/idea.md`、現行specの修正要否を確認する（調査結果を現在仕様へ追加する必要がないため変更しない）

## 検証

- [x] `go test -v ./...` を実行する
- [x] `go test -run TestDoesNotExist ./...` を実行して全パッケージのコンパイルを確認する
- [x] CodexProの `show_changes` でTODO028由来の変更範囲と文書間の整合を確認する

## 受け入れ条件

- [x] Google Books APIの現在の検索条件、ページング、認証条件が整理されている
- [x] タイトル、ISBN、著者検索をmankenで利用する際の意味と制約が分かる
- [x] 漫画単行本への絞り込みをGoogle Books APIだけで保証できるか、後段判定が必要かが明確になっている
- [x] `NormalizedBook`、Raw response、販売・閲覧情報の役割分担が整理されている
- [x] 既存Googleサンプルからそのまま流用できる点と、変更・再設計が必要な点が区別されている
- [x] 後続のGoogle Booksプロバイダ実装TODOを作成できるだけの調査結果がある
- [x] Goコード、公開API、現行仕様の動作を変更していない

## リスク・懸念

- Google Booksの検索結果は同じ検索語でも地域、蔵書、更新状況などにより変化し得る
- `intitle:` などの検索構文が利用できても、タイトル完全一致や漫画単行本だけを保証するとは限らない
- `saleInfo` と `accessInfo` は国や利用条件によって値が変わり得るため、書誌情報と同じ扱いにしない
- APIキーなしで取得できるケースがあっても、公開ライブラリの推奨利用方法は公式の認証指針を基準に判断する
- Google Booksの検索結果表示にはattributionとリンク要件があり、検索結果の並べ替え・改変にも制約があるため、横断検索の表示設計と分離して考えない
- Books API固有Termsの料金条項とGoogle APIs Termsの保存・キャッシュ制約は、Google Booksを利用するアプリケーション側の利用形態に影響する

## 未決事項

- TODO028の完了を妨げる未決事項はない
- Google Booksは漫画専用の主要データベースとはせず、広い書誌検索を補うプロバイダとして実装候補にする
- `saleInfo` と `accessInfo` は初期実装で新しい共通販売モデルへ広げず、まずRaw responseから利用できる状態を優先する
- 2026-08-07時点の有効なAPIキーを使った代表検索の再実行は、後続の実装TODOで結合確認として行う

## AIへの入力メモ（任意）

- 会話で決まった前提:
  - TODO028は調査TODOとし、Google BooksのGo実装は後続TODOへ分離する
  - mankenは検索プロバイダのハブを目指し、`NormalizedBook` とRaw responseの双方を維持する
- 優先度:
  - Google Books実装へ進む前に、2026年8月時点の現行仕様を確認する
- 先に決めたい論点:
  - Google Booksを何の検索に使うか、漫画単行本の判定をどこまでAPI側に任せられるか
