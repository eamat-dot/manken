# 楽天Booksの検索順・サイズ絞り込み・著者分割を改善する

## 背景

- 現状:
  - `rakutenbooks.SearchBooks` は楽天Booksの `sort=standard` を固定している。
  - `author` / `authorKana` は楽天Booksレスポンスの1文字列をそのまま1人分として `Authors` / `Contributors` へ変換している。
  - 漫画区分は `WithComicGenre` で指定できるが、楽天Books固有の `size` 検索条件は公開していない。
  - `seriesName` は現行どおり `Normalized.Series` へ変換している。
- 課題:
  - `standard` は漫画一覧として利用者から見た並びの規則性が乏しく、巻を追って確認する用途で扱いにくい。
  - 複数著者が `采　和輝/八月　八/大橋キッカ` のように `/` 区切りで1つの `author` に返るため、現行変換では人物単位で扱えない。
  - 文庫など特定の商品形態を絞り込みたい場合に、楽天Booksの `size` 条件を利用できない。
- 変更が必要な理由:
  - 楽天Booksの検索結果を漫画一覧として読みやすくし、取得元が明示する複数人物を安全に人物単位へ分離する。
  - `size` は共通検索条件ではなく楽天Books固有の分類だが、文庫版等を切り分ける用途に有用なため任意フィルターとして公開する。

## 目的

- このTODOで達成すること:
  - 楽天Books検索の固定既定sortを `+releaseDate`（発売日の古い順）へ変更する。
  - `author` / `authorKana` の `/` 区切りを人物単位へ分割して共通モデルへ変換する。
  - 楽天Books固有の `size` を型付きClient Optionで指定できるようにする。
- 期待する利用者視点の結果:
  - 同一作品を検索した際、`standard` より発売日の流れを追いやすい順で取得できる。
  - 複数著者を `Authors` と `Contributors` の複数要素として利用できる。
  - 文庫、コミック等の楽天Books商品形態を必要な場合だけ絞り込める。

## 非目的

- 今回は扱わないこと:
  - 共通Sort型やsort指定APIを追加すること。楽天Books検索では `+releaseDate` を固定既定値として使用する。
  - `SearchBooksRequest.Publisher` を追加すること。
  - MADB、Google Books等へPublisher検索を追加すること。
  - `seriesName` の意味や `Normalized.Series` への変換を変更すること。
  - openBD Collectionを `Normalized.Series` へ戻すこと。
  - 氏名内部の半角空白、全角空白、カンマ等を除去・統一する本格的な人名正規化。
  - 著者名から原作者・作画者等の役割を推測すること。
  - 在庫、販売状態、レビュー等の追加共通化。
- 別Issueや別TODOで扱うこと:
  - 共通 `SearchBooksRequest.Publisher` と、楽天Books `publisherName` / MADB `schema:publisher` / Google Books `inpublisher:` の対応。
  - 未実装プロバイダの同種項目を比較した後の `Series`、出版コレクション、レーベル、Imprintの共通モデル再検討。

## 前提・制約

- 変更してよい範囲:
  - `rakutenbooks/`
  - `examples/rakutenbooks/`
  - 楽天Booksの現行仕様を説明する `docs/pkg/rakutenbooks/`、`examples/README.md`、README、CHANGELOG等の必要箇所
  - TODO032の進捗更新
- 変更してはいけない範囲:
  - `api.SearchBooksRequest` を含む共通検索Requestのフィールド追加
  - `madb/`、`googlebooks/`、`openbd/` の動作変更
  - 現行 `seriesName -> Normalized.Series` の変換
  - ユーザーが別途変更している `Taskfile.yml` やbacklogの無関係な項目
- 互換性要件:
  - 既存のTitle / Author検索、ComicGenre、Limit、Cursor、Raw response、ISBN参照、Affiliate URL、価格変換を維持する。
  - `size` 未指定時は従来と同じくsizeで絞り込まず、漫画ジャンルだけを使用する。
  - ISBN参照にはClientのsize設定を送らない。
  - 著者の内部表記は保持し、`/` による複数値の分離以外の正規化を行わない。
- パフォーマンス要件:
  - `size` 対応や著者分割のために追加HTTPリクエストを発生させない。
  - 自動リトライ、内部レート制御、バックグラウンド処理を追加しない。

## 対象範囲

### 対象

- 楽天Books検索時の固定 `sort=+releaseDate`
- 楽天Books公式 `size=0..10` に対応する公開 `BookSize` 型とClient Option
- `size` を検索パラメーターとCursor条件へ反映する処理
- `author` / `authorKana` の `/` 分割と `Authors` / `Contributors` 変換
- examples、テスト、楽天Books現行仕様文書の同期

### 対象外

- sort値を呼び出し側から切り替える公開API
- Publisher共通検索
- Series共通モデル変更
- 著者同一人物判定用canonical key
- 楽天Books以外のproviderコード変更

## 想定ユースケース

- 入力:
  - `SearchBooksRequest{Title: "動物のお医者さん"}` と通常の楽天Books Client
  - 文庫を絞り込む場合は `WithBookSize(BookSizeBunko)` を追加したClient
- 実行:
  - 検索では漫画ジャンル、任意size、`sort=+releaseDate` を1回の楽天Books APIリクエストへ送る。
  - 返却された `author` / `authorKana` を `/` 単位で人物へ分割する。
- 出力:
  - 検索結果の順序は楽天Booksが `+releaseDate` で返した順序をそのまま維持する。
  - 複数著者は `Authors` と人物単位の `Contributors` として返す。
- 失敗時の扱い:
  - 未対応の `BookSize` はClient生成時に `invalid_argument` とする。
  - Cursorを別sizeのClientで再利用した場合は通信前に `invalid_argument` とする。
  - `author` と `authorKana` の分割要素数が一致しない場合はReadingを推測で対応付けず、人物名だけを保持する。

## 処理方針

1. 楽天Books固有の `BookSize` 型を追加し、公式の0〜10を表す定数を公開する。0は全てを意味し、既定値とする。
2. `WithBookSize(BookSize)` をClient Optionとして追加する。0では検索queryに `size` を付加せず、1〜10では対応する数値を送る。ISBN参照では送らない。
3. SearchBooksのCursor用検索キーへBookSizeを含め、同じTitle / Author / Genre / Limitでもsizeが異なるCursor再利用を拒否する。
4. 検索queryの固定sortを `standard` から `+releaseDate` へ変更する。返却後に独自並べ替えは行わない。
5. `author` を `/` で分割し、各要素の前後空白だけを除去する。空要素は著者として追加しない。氏名内部の空白・カンマ等は保持する。
6. `authorKana` も `/` で位置を保って分割する。著者側と分割要素数が一致する場合だけ同じ位置のContributorへReadingを設定し、一致しない場合はReadingを設定しない。
7. ContributorのRolesは現行どおり推測しない。単一著者の場合も同じ分割ロジックを通す。
8. `seriesName` は現行どおり `Normalized.Series` に保持し、このTODOでは変換規則を変更しない。
9. 実装後に楽天Books spec / guide、README、examples README、CHANGELOG等を現在仕様へ同期する。計画段階の内容をspecへ残さない。

### BookSize公開型の案

`BookSize` は楽天Books固有型として `rakutenbooks` パッケージに置き、公式値を型付き定数で表す。定数名は既存Go命名規則に合わせて最終確認してよいが、値と意味は次を維持する。

| 値 | 楽天Booksの意味 |
| ---: | --- |
| 0 | 全て |
| 1 | 単行本 |
| 2 | 文庫 |
| 3 | 新書 |
| 4 | 全集・双書 |
| 5 | 事・辞典 |
| 6 | 図鑑 |
| 7 | 絵本 |
| 8 | カセット、CDなど |
| 9 | コミック |
| 10 | ムックその他 |

少なくとも利用頻度の高い `BookSizeAll`、`BookSizeBunko`、`BookSizeComic` を含め、全公式値を名前付き定数として公開する。整数を直接渡すだけのAPIにはしない。

## 実施項目

### 調査

- [ ] 現行 `rakutenbooks` のClient Option、検索query、Cursor、レスポンス変換、examplesを確認する
- [ ] RESEARCH030に記録した `+releaseDate`、`/` 区切り、size 0〜10の前提と現行実装の差分を確認する
- [ ] `seriesName`、Publisher、openBDを今回変更しないことを影響範囲として確認する

### 実装

- [ ] 楽天Books固有の `BookSize` 型、0〜10の公開定数、`WithBookSize` を追加する
- [ ] size未指定時はqueryへsizeを送らず、指定時だけSearchBooksへ送る
- [ ] ISBN参照ではsizeを送らない
- [ ] Cursorの検索条件へBookSizeを含め、別sizeでの再利用を拒否する
- [ ] SearchBooksの固定sortを `+releaseDate` へ変更する
- [ ] `author` を `/` で分割して `Authors` と人物単位の `Contributors` へ変換する
- [ ] `authorKana` は分割数が一致した場合だけ各ContributorのReadingへ対応付ける
- [ ] 著者名内部の空白・カンマ、Contributor Roles、`seriesName` の現行変換を変更しない
- [ ] examples/rakutenbooksからsizeを指定して動作確認できるようにする。CLIでは公式0〜10を検証してClient Optionへ変換する

### テスト

- [ ] `WithBookSize` の0〜10正常系と範囲外エラーをテストする
- [ ] size=0/未指定でqueryにsizeがなく、1〜10指定時だけ正しい値が入ることをテストする
- [ ] ISBN参照queryにsizeが入らないことをテストする
- [ ] Cursorが同じsizeでは継続でき、異なるsizeでは通信前に拒否されることをテストする
- [ ] 検索queryのsortが `+releaseDate` であることをテストする
- [ ] 複数 `author` を `/` で分割し、前後空白だけを除去することをテストする
- [ ] `authorKana` の件数一致時にReadingを対応付け、不一致時はReadingを推測しないことをテストする
- [ ] `采　和輝` の全角空白や `苗字, 名前` のカンマ等、氏名内部表記を保持する退行防止テストを追加する
- [ ] 単一著者、空著者、ISBN参照結果でも共通変換が成立することをテストする
- [ ] `seriesName` が従来どおり `Normalized.Series` に残ることを退行防止テストで確認する
- [ ] examplesのsize引数正常系・範囲外をテストする

### ドキュメント

- [ ] `docs/pkg/rakutenbooks/spec.md` を `+releaseDate`、BookSize、Cursor、著者分割の現在仕様へ同期する
- [ ] `docs/pkg/rakutenbooks/guide.md` にsizeの用途と公式0〜10の意味、`size=9` を既定固定しない理由を案内する
- [ ] `examples/README.md` に楽天Booksのsize指定方法と例を追加する
- [ ] README / CHANGELOGを利用者向け変更に必要な範囲で同期する
- [ ] `docs/backlog.md` のPublisher案とSeries保留事項を残し、完了扱いにしない

## 検証

- [ ] `go test -v ./rakutenbooks ./examples/rakutenbooks` を実行する
- [ ] `go test -v ./...` を実行する
- [ ] `go test -run TestDoesNotExist ./...` を実行して全パッケージのコンパイルを確認する
- [ ] `go test -vet=all ./...` を実行する
- [ ] リポジトリで利用可能なgofmt相当チェックを実行する
- [ ] 認証情報が利用可能なら `go test -v -tags=integration ./rakutenbooks` または既存gitignored検証でsize指定と通常検索を実API確認する。利用できない場合は未実行理由を記録する
- [ ] CodexProの差分確認で `api.SearchBooksRequest`、MADB、Google Books、openBD、Taskfileの意図しない変更がないことを確認する

## 受け入れ条件

- [ ] 楽天Booksの通常検索が `sort=+releaseDate` を送信し、返却順をライブラリ側で並べ替えない
- [ ] `WithBookSize` 未指定またはAllではsize絞り込みをせず、1〜10指定時だけ検索queryへsizeを送る
- [ ] BookSizeの範囲外値は通信前の `invalid_argument` になる
- [ ] CursorはBookSizeを検索条件として拘束し、異なるsizeでは再利用できない
- [ ] ISBN参照のリクエストと複数ISBN方針は変更されていない
- [ ] `/` 区切りの複数著者が人物単位の `Authors` / `Contributors` として返る
- [ ] `authorKana` は人物数が一致する場合だけReadingへ対応し、不一致時に推測しない
- [ ] 氏名内部の空白・カンマ等を独自正規化せず保持する
- [ ] Contributorの役割を推測しない
- [ ] `seriesName` のNormalized変換は変更されていない
- [ ] `SearchBooksRequest.Publisher`、MADB、Google Books、openBDには実装変更が入っていない
- [ ] 楽天Booksのspec / guide / examplesが実装後の現在仕様と一致する
- [ ] 既存providerを含む通常テスト、コンパイル確認、vet相当検証が成功する

## リスク・懸念

- `+releaseDate` は発売日の古い順であり、作品の巻数順を保証しない。新装版、特装版、再刊等が混在する場合は発売日順として扱う。
- `author` / `authorKana` の `/` 区切りは実データで確認済みだが公式出力表に区切り規則の明記はない。Raw responseは引き続き利用可能にし、分割以外の人物推測を行わない。
- `size` は楽天Books固有の商品形態分類であり、共通 `PhysicalSize` やPublicationMediumと同一概念として扱わない。
- Client Optionとしてsizeを持つため、同じClientで検索ごとにsizeを切り替える用途には新しいClient生成が必要になる。現行 `WithComicGenre` と同じ設定粒度を優先する。

## 未決事項

- TODO032の実装開始を妨げる未決事項はない。
- `SearchBooksRequest.Publisher` は別ブランチ・別TODOで実装判断する。
- `Series`、出版コレクション、レーベル、Imprintの共通モデルは未実装プロバイダの情報が出そろってから再検討する。
- 著者名の空白・カンマ等を同一人物照合用にどう正規化するかは別課題とする。

## AIへの入力メモ（任意）

- 会話で決まった前提:
  - 楽天Booksの固定既定sortは `standard` から `+releaseDate` へ変更する。
  - `author` / `authorKana` は `/` 区切りで人物単位へ分割する。
  - `size` は楽天Books固有フィルターとして追加し、共通 `SearchBooksRequest` へは入れない。
  - `seriesName` は違和感があっても情報価値を優先して現状維持し、共通Series設計を後で再検討する。
  - openBDで除外したCollectionも将来の再検討対象だが、TODO032ではコードを戻さない。
  - `SearchBooksRequest.Publisher` は有力な共通項目候補だが、既存provider修正を伴うため別ブランチに分離する。
- 優先度: 楽天Booksの現在ブランチで完結する改善だけを先に実装する。
- 先に決めたい論点: なし。公開BookSize定数名はGo命名規則と公式意味を損なわない範囲で実装時に整える。
