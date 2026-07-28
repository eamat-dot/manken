# Cobra CLIテンプレートをライブラリ用へ移行

## 背景

- 現在のワークスペースには、Cobra CLIを作成するためのテンプレートが配置されている
- `manken` はCLIではなく、複数のGoプロジェクトから利用するライブラリとして開始する
- README、Taskfile、GitHub Actions、補助ファイルにテンプレート固有の説明や設定が残っている

## 目的

- テンプレート固有の内容を取り除き、`manken` のライブラリ開発を開始できる構成にする
- 後続の調査と実装で使用するドキュメント、タスク、CIの置き場所を整える

## 非目的

- MADBの仕様調査
- 公開APIの確定
- Goコードと `go.mod` の作成
- MADBクライアントの実装
- MCPサーバーの実装
- 初回コミット、push、タグ、GitHub Releaseの作成

## 対象範囲

### 対象

- `README.md`
- `docs/concept.md`
- `docs/spec.md`
- `docs/pkg/madb/spec.md`
- `docs/idea.md`
- `docs/backlog.md`
- `docs/todo/`
- `Taskfile.yml`
- `.taskfiles/`
- `.github/workflows/`
- `.gitignore`
- `.golangci.yml`
- `CHANGELOG.md`
- `LICENSE`
- `AGENTS.md`
- `.github/instructions/`

### 対象外

- `.go` ファイル
- `go.mod`
- `go.sum`
- MADBへ接続する処理

## 実施項目

### ドキュメント

- [x] `README.md` を `manken` の目的と現在の開発状態が分かる内容へ書き換える
- [x] `docs/concept.md` を本プロジェクトの目的と責務境界へ書き換える
- [x] `docs/spec.md` にテンプレート固有の説明が残っていないことを確認する
- [x] `docs/pkg/madb/spec.md` にテンプレート固有の説明が残っていないことを確認する
- [x] `docs/idea.md` と `docs/backlog.md` の役割を本プロジェクト向けに明記する
- [x] TODOファイルの番号、名称、責務が重複していないことを確認する
- [x] READMEでは未実装のAPIを利用可能であるかのように説明しない

### Taskfile

- [x] `BINARY_NAME` などのCLIバイナリ用変数を削除する
- [x] `run:*`、`exe:*` などのCLI実行タスクを削除する
- [x] バイナリ生成とリリースビルド用タスクを削除する
- [x] ライブラリで使用するtest、lint、build、mod-depsタスクだけに整理する
- [x] `.taskfiles/app.yml` がCLI専用のため削除する
- [x] `.taskfiles/cheat-sheet.yml` をライブラリ開発用に整理する
- [x] `.taskfiles/dev.yml.sample` がCLI専用のため削除する

### GitHub Actionsと設定

- [x] CIを `go mod tidy`、`go test`、`go build` の検証用に整理する
- [x] golangci-lint workflowと `.golangci.yml` のバージョン互換性を確認する
- [x] CLI配布用のrelease workflow sampleを削除する
- [x] `.gitignore` から不要なCLI成果物の設定を取り除く
- [x] `.coderabbit.yaml` をGoライブラリと仕様書のレビュー用に維持する
- [x] `.editorconfig` を維持する

### プロジェクト情報

- [x] `LICENSE` がMIT Licenseであることを確認する
- [x] GitHubでpublicリポジトリが作成済みであることを確認する
- [x] `CHANGELOG.md` にKeep a ChangelogとSemantic Versioningを採用し、未実装の変更履歴を記載しない
- [x] `AGENTS.md` と `.github/instructions/` を維持する
- [x] `__REPLACE_MODULE_NAME__` などのプレースホルダーをすべて取り除く
- [x] Cobra、Viper、コマンド、バイナリを前提とする記述を取り除く

## 検証

- [x] `docs/todo/` を除外してテンプレート固有の名称とCLI用設定の残存箇所を確認する
- [x] Markdown内のファイル名と実際の構成が一致することを確認する
- [x] YAMLファイルを構文解析できることを確認する
- [x] `git diff --check` と未追跡ファイルの空白確認を実行する
- [x] Goモジュール作成前のためGoのビルドとテストを実行対象外として記録する

## 受け入れ条件

- [x] テンプレート固有の名称とプレースホルダーが残っていない
- [x] ドキュメントが `manken` のライブラリ開発を説明している
- [x] TaskfileとGitHub ActionsがCLIの存在を前提としていない
- [x] Goコード、`go.mod`、`go.sum` を変更していない
- [x] 調査TODOと実装TODOの内容を含んでいない

## 実施結果

- `go.mod` が存在しない間、GitHub ActionsはGo関連ステップを正常にスキップする
- golangci-lint v2.12.2で `.golangci.yml` の設定検証に成功した
- `task --list-all`、`task cheat:go`、`task cheat:docs` の実行に成功した
- ローカルMarkdownリンクの参照先がすべて存在することを確認した
- Goコード、`go.mod`、`go.sum` は作成していない
- GitHub上のライセンス表示は、`LICENSE` を初回pushした後に有効になる
