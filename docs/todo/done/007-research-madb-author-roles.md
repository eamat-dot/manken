# MADBのcreator役割と寄与者変換規則を調査

## 状態

完了。

2026年8月1日に公式SPARQL Query Serviceで調査した結果、現在の変換規則を
そのまま拡張できない実例が見つかった。調査結果、推奨案、代替案は
[`../../research/007-madb-author-roles.md`](../../research/007-madb-author-roles.md) に記録した。

2026年8月1日に推奨案が承認されたため、共通仕様とMADB仕様を確定し、実装作業を
[`013-implement-madb-author-roles.md`](013-implement-madb-author-roles.md) に分離した。

今回の共通モデル変更で `NormalizedBook.Contributors` と
`SourceBookValues.Authors` は追加されたが、MADBのcreator役割と人物を対応付ける
規則は確定していない。現状は `Contributors` を空にし、Agent名が存在する場合は
全Agent名を `Normalized.Authors` と取得元値へ設定している。

## 背景

- MADBの `schema:creator` には `[著]`、`[原作]`、`[作画]`、`[解説]` などの
  役割付き文字列が含まれる
- `dcterms:creator` はAgentリソースを参照するが、Agentと
  `schema:creator` の各文字列にはRDF上の1対1対応がない
- Agent名が1件でもあれば `schema:creator` ではなく全Agent名を優先するため、
  解説者なども `Normalized.Authors` に含まれる場合がある
- `M292132` では `[著]佐々木倫子` と `[解説]藤原新也` があり、現状は両名を
  `Normalized.Authors` に含める
- Raw responseでは取得元の全bindingを確認できるが、通常の `Book` では
  Agent名を採用した場合に役割付き文字列を保持しない

## 目的

- MADBのcreator役割を収集し、`Normalized.Authors` に含める範囲を決める
- 役割と人物を対応付けられる場合だけ `Normalized.Contributors` を設定する規則を
  決める
- `Sources[0].Values.Authors` に保持する取得元値と表示順を決める
- 未知の役割、Agent欠落、役割とAgentを対応付けられない場合の扱いを決める

## 非目的

- Goコードの実装
- 人名の異体字統合や外部典拠による同一人物判定
- 著者名検索の実装
- 対応関係がないAgentとcreatorを推測で結び付けること
- MADB固有の役割名をそのまま共通の役割語彙として確定すること

## 前提・制約

- 取得元の情報を失わないことと、正規化値へ推測を混入させないことを分けて扱う
- `[解説]` などを元データから削除しない
- 未知の役割を推測で著者または既知の寄与者役割へ分類しない
- `Authors` と `Contributors` の順序を意味のない文字列ソートで変更しない
- 役割語彙は他の取得元でも意味が通じる一般名にする
- 確定事項は `docs/spec.md` と `docs/pkg/madb/spec.md` に分けて記載する

## 実施項目

### 実データ調査

- [x] `schema:creator` の先頭役割、件数、代表MADB IDを収集する
- [x] 複数の角括弧、役割なし、不正な角括弧表記の実例を確認する
- [x] Agent参照だけ、creator文字列だけ、両方がある実例を確認する
- [x] 複数Agentと複数creator文字列の対応可否を確認する
- [x] `M292132` を著者と解説者が混在する基準ケースとして記録する

### 変換規則

- [x] `Normalized.Authors` に含める役割と除外する役割を決める
- [x] 共通の `ContributorRole` 語彙とMADB役割からの対応を決める
- [x] 対応を確定できる場合だけ `Normalized.Contributors` を作る条件を決める
- [x] 対応不能なAgentと未知の役割を推測せず保持する推奨案を整理する
- [x] `Sources[0].Values.Authors` に役割付きcreatorを優先して保持する推奨案を
  整理する
- [x] 取得元の表示順と重複除去規則を決める

### ドキュメント

- [x] 調査日、確認クエリ、役割一覧、代表IDを調査文書へ記録する
- [x] 共通の `Authors`、`Contributors` 契約を `docs/spec.md` へ反映する
- [x] MADB固有の取得・変換規則を `docs/pkg/madb/spec.md` へ反映する
- [x] 確定仕様の実装とテストを別TODOへ分離する

## 検証

- [x] `M292132` で佐々木倫子と藤原新也をcreator文字列から区別できる
- [x] Agent参照の有無が異なる代表ケースを記録する
- [x] 未知または対応不能な値を推測で `Contributors` に入れない案を整理する
- [x] `Sources[0].Values.Authors` またはRaw responseへ調査根拠を残す案を整理する
- [x] `Authors` と `Contributors` の順序が共通仕様と一致する
- [x] ドキュメント内のリンクと `git diff --check` を確認する

## 受け入れ条件

- [x] creator役割と代表実例が記録されている
- [x] `Authors`、`Contributors`、取得元値の責務が区別されている
- [x] 役割と人物を対応付けられない場合の扱いが決まっている
- [x] 未知の役割、Agent欠落、表示順の扱いが決まっている
- [x] 共通仕様とMADB仕様に矛盾がない
- [x] 実装作業が別TODOとして整理されている

## リスク・懸念

- creator文字列とAgent URIを完全には対応付けられない可能性がある
- `[監修]` などは利用目的によって著者扱いが変わる可能性がある
- 役割名の表記揺れに固定一覧だけでは対応できない可能性がある
- 通常の `Book` に取得元固有の根拠を増やすと共通モデルの責務が広がる
- `[作画][田辺節雄]` のように、2つ目の角括弧が役割ではなく人名の例がある
- MADBのRDFはcreatorの順序を持たず、取得元の表示順を復元できない

## 確定事項

- `ContributorRole` はONIXコードではなく一般名を公開する
- `Authors` には主要な創作者を含め、その他の既知役割は `Contributors` だけに含める
- 未知または不正な役割は推測で正規化しない
- MADBのcreator順序は復元できないため、Goの文字列昇順で安定化する
