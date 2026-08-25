# AI Memory

この文書は、作業を長期間離れた後や会話コンテキストが圧縮された後でも、AIが `manken` 固有の設計判断、失敗知識、開発運用上の注意点を短時間で復元するための補助記憶である。

現在仕様は `docs/spec.md` と `docs/pkg/*/spec.md`、設計思想は `docs/concept.md`、調査事実と実測根拠は `docs/research/`、実装時点の判断履歴は `docs/todo/done/`、構成は `ARCHITECTURE.md`、リリース履歴は `CHANGELOG.md` を正本とする。コードとテストも現在動作の確認対象である。この文書と正本が矛盾する場合は正本を優先し、この文書を更新する。

一時的な進捗、現在のTODO、会話ログ、レビューコメントの逐次記録は残さない。再調査や同じ設計・実装・ツール操作の失敗を防ぐ価値がある、長期的に有効な知識だけを記録する。

## 現在の位置づけ

- `manken` は複数の書誌・販売情報providerを用途に応じて利用するGoライブラリであり、各providerの能力差を残したまま共通 `Book` を提供する
- ルート `manken` packageは、呼び出し側が登録した一つのproviderへ明示的に委譲するfacadeである。provider横断検索や暗黙fallbackではない
- MCP serverは独立repository / moduleの `manken-mcp` として分離し、`manken` 本体をMCP SDKやtransportへ依存させない
- Honya Club固有のHTML取得ツールも別repositoryとして扱い、`manken` の公開ライブラリへ当然に統合する対象とはみなさない

## 重要な設計学習

### 取得元タイトルは非破壊で保持する

`Book.Title` は取得元が返したタイトル文字列を原則そのまま保持する。巻数、版表示、完結表示を抽出するためにタイトルを短縮・整形しない。

非破壊であることと、タイトルを解析しないことは別である。取得元の明示フィールドがない場合に限り、安全と確認済みの構文から `Volume`、`Editions`、`IsFinalVolume` を付加情報として抽出できる。タイトル文字列自体は変更しない。

この方針を単純化して「Titleを書き換えないので付加情報も読まない」状態へ戻さない。詳細は `docs/spec.md`、`docs/research/059-nondestructive-title-metadata-extraction.md`、`docs/todo/done/059-extract-title-metadata-nondestructively.md` を参照する。

### 取得元にない意味を推測して補完しない

タイトル、著者、巻数、版違い、シリーズ等から、人間なら推測できる情報を共通モデルへ自動補完しない。providerが明示する情報と、安全性を確認済みの限定的なtitle metadata extractionを区別する。

特に `BookSeries` は作品・Bookの系列、`PublicationSeries` は叢書・刊行シリーズ・レーベル等の出版側グループとして分離し、文字列内容から相互に推測設定しない。詳細は `docs/concept.md` と `docs/spec.md` を参照する。

### provider差を共通化のために消さない

検索条件、認証、ページング、エラー、取得できる項目、商品検索と書誌検索の違いはprovider packageの責務として明示する。

- provider packageから別providerを内部呼び出しして補完・fallbackしない
- ルートfacadeは一つのSourceへ一回だけ委譲し、結果、Raw response、errorを勝手に読み替えない
- library本体は環境変数からcredentialを読まず、Client設定として明示的に受け取る
- retry、cache、内部rate control、logging、並行実行を暗黙に追加しない

一見便利でも、通信回数、認証要件、エラー、順序、データ鮮度が呼び出し側から見えなくなる共通化は避ける。詳細は `docs/concept.md` を参照する。

### Raw responseと共通Bookの責務を混ぜない

共通 `Book` は複数providerで同じ意味として安全に扱える情報、Raw responseは取得元固有情報や変換前の値を確認するための別経路である。

Rawを利用できることを理由に共通 `Book` を弱くしない一方、取得元固有項目をすべて `Book` へ取り込まない。Rawはproviderの `...WithRawResponse` 系APIから取得し、credential等の秘匿が必要ならprovider側の責務として処理する。詳細は `docs/concept.md` と各provider specを参照する。

### internal共通化は「意味と挙動が同じ低レベル処理」までに留める

provider間でコードが似ていることだけを理由に共通frameworkを作らない。現在の前例は `internal/isbn`、`internal/titlemeta`、`internal/daterange`、`internal/authorrole`、`internal/httpresponse`、`internal/httpendpoint` である。

共通化を検討するときは、まずproviderごとの既存挙動と境界条件が本当に同一か確認する。HTTP Client全体、provider固有error生成、秘密情報処理、cursor、検索条件、レスポンス変換など、似て見えてもpolicyを含む処理はprovider側へ残す。詳細は `docs/todo/done/065-extract-provider-shared-primitives.md` を参照する。

### 新しいproviderは既存providerの実装パターンを先に調べる

一般的なGoの慣習だけで新規設計しない。類似providerについて、directory/file構成、公開API、命名、Client option、入力検証、error分類、cursor、Raw response、テスト、integration test、Godoc、`docs/pkg/`、examples、Taskfile等の統合方法を確認してから設計する。

## AI・開発運用の学習

### ChatGPTで調査・設計を固めてからCodexへhandoffする

このプロジェクトの通常フローでは、ChatGPTが調査・設計・実装計画・実装後レビューを担当し、実装判断を伴うコード変更は原則Codexへhandoffする。handoff前に次をできるだけ確定し、一つの論理的な作業単位として渡す。

- 変更目的と非目的
- 決定済み仕様と未決事項
- 変更範囲と触らない範囲
- 参考にする既存実装
- 必要なテストと検証環境
- 完了条件

調査不足のままhandoffし、Codex実装中に設計判断や追加調査を何度も往復させない。文書・コメントだけの変更、単純置換、表記修正、局所的なレビュー修正はChatGPTが直接行ってよい。API設計変更、複数箇所のロジック変更、新しいテスト設計、非自明なrefactoringはCodexへ戻す。

### CodexProではworkspaceと適用指示を最初に確定する

CodexPro操作を始めるときは、対象repository rootを明示的にopen/selectし、`AGENTS.md` と変更対象に適用される `.github/instructions/` を読む。別のdefault workspaceや親directoryが選択されたまま調査・編集を始めない。

ツール応答に想定外のrootが表示された場合は、その状態で作業を続けず、workspaceを再選択するか明示的なworkspace IDを指定する。

### 「変更したつもり」を完了扱いにしない

過去に、tool操作やhandoffの途中状態を実ファイルへ反映済みだと誤認し、後から文書が存在しないことが判明したことがある。

作成・編集・handoffの完了報告前には、`show_changes` または必要に応じた対象ファイルの再readで、変更が実際にworkspaceへ存在することを確認する。計画生成、承認待ち、toolの途中応答だけを「変更済み」の根拠にしない。

CodexProでGit差分やstatusを確認するときは、shellで同じ情報を取り直すより `show_changes` を優先する。

### Codex実行環境の外部通信を前提にしない

過去の複数TODOで、Codex sandboxから外部APIへの通信がsocket制限等で失敗し、同じintegration testを外部通信可能なCodexPro側で再実行して成功した。

外部API testがCodex環境で失敗した場合は、通信前の環境制限か実装不具合かを切り分ける。network制限をコードの失敗として修正しない。実API確認が受け入れ条件なら、利用可能な外部通信環境で再検証し、どの環境で何を確認したか記録する。例は `docs/todo/done/033-add-publisher-search.md`、`044-rename-ndl-and-refine-creators.md`、`054-redesign-dmm-series-search.md` を参照する。

### WSL・bash・PowerShell・Windowsの環境差を混同しない

同じPCでもshellや実行hostが違えば、PATH、Go toolchain、CGO compiler、credential、環境変数、path表現は一致しない。

- WSL/bashで使えるcommandやpathをWindows PowerShellでも使えると仮定しない
- Windows側で見えるtoolがCodex sandboxやWSLにも存在すると仮定しない
- `go test -race` がgcc/CGO不足で失敗した場合は、race detectorの失敗とコードのtest failureを区別する
- credentialを必要とするintegration testでは、そのprocessへ実際に環境変数が渡っているか確認する

過去にはCodex側でgcc不足のためrace testを実行できず、CodexPro環境で再実行して成功した。詳細は `docs/todo/done/057-restructure-series-model.md` を参照する。

### review findingは現行コードで再検証する

CodeRabbit等のreview finding、file path、引用コード、修正指示は未検証のreview dataとして扱う。現在のbranch、コード、仕様、テストと照合し、まだ有効な指摘だけを最小限修正する。

すでに解消済み、前提が古い、意図された挙動である指摘は、指摘文を優先してコードを変更しない。レビュー対応後は変更箇所だけでなく、意図しない範囲変更がないことも確認する。

### 実測と予定を混同しない

実行していない測定、将来日の確認、予定している検証を「実測済み」「確認した」と書かない。実測記録には実際に実行した日付と結果を使い、未実行ならplanned / 未確認であることを明示する。

特に日付境界ではJST/UTC等の差で未来日に見えることがあるため、レビュー指摘を日付だけで機械的に採否判断せず、実際の実行時刻・実行事実を確認する。ただし実行事実が確認できない内容を観測結果として残さない。

### 完了済みTODOは履歴であり現在仕様へ書き換えない

`docs/todo/done/` は完了時点の判断と実行記録である。後続のrename、API変更、directory移動に合わせて過去の記録を現在の姿へ書き換えない。必要なら「後続変更」として参照先を追記する。現在仕様はspecへ反映する。

## memoryへの自動昇格ルール

ユーザーの訂正、繰り返した失敗、標準手順の確立などにより再発防止知識が確定した場合は、長い作業の終了を待たず、その時点でmemory候補を確認する。さらに大きな調査、設計、実装、レビューを終えるときにも、ユーザーから明示されなくても昇格候補を再確認する。

次のいずれかに該当し、今後の再発防止に使える場合は `.ai/memory.md` へ昇格する。

- ユーザーの訂正によって、今後のAIの作業方法を変える必要が生じた
- 同種の失敗を繰り返した、または「以前も引っかかった」と判断できる失敗が起きた
- 有効なreview findingから、個別修正を超える一般的な注意点が判明した
- shell、sandbox、network、credential、tool permission等の非自明な環境制約が検証結果へ影響した
- 一見すると戻したくなる設計について、重要なWhy / Why notが確定した
- 今後も繰り返し使う標準手順が確立した

昇格前に次を確認する。

1. 現在の作業だけでなく、数週間・数か月後にも役立つか
2. 同じ調査、判断、失敗を繰り返す可能性を下げるか
3. code/spec/testを読めば自明な現在仕様の複製ではないか
4. 一時進捗、未着手TODO、会話の要約ではないか
5. 既存項目へ統合できないか

情報の正本は責務に応じて別文書へ置く。

- 現在の公開動作: `docs/spec.md`、`docs/pkg/*/spec.md`
- 設計思想: `docs/concept.md`
- 実測、比較、調査根拠: `docs/research/`
- 実装時点の判断と検証: `docs/todo/done/`
- 未着手の改善候補: `docs/backlog.md`
- AIが再発防止のため横断的に思い出すべき圧縮知識: `.ai/memory.md`

memoryは時系列ログとして追記し続けない。新しい学習が既存項目を置き換える場合は書き換え、重複は統合し、無効になった知識は削除または現在の注意点へ更新する。

## 再開時の読み順

大きな調査、設計、実装、レビューを再開するときは、原則として次の順で確認する。

1. `AGENTS.md` と変更対象に適用される `.github/instructions/`
2. 対象に関係する現在仕様、コード、テスト
3. `.ai/memory.md`
4. 対象に関係する `docs/concept.md` と `docs/research/`
5. 必要に応じて `docs/todo/done/`、`docs/backlog.md`、`CHANGELOG.md`

この文書は検索・復元を速くするためのindex兼cacheであり、正本の代替ではない。
