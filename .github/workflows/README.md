# Workflows

Goライブラリの継続的インテグレーションに使用する。

## ci.yml

次を実行する。

1. `go mod tidy` による差分確認
2. `go test -v ./...`
3. `go build -v ./...`

## golangci-lint.yml

golangci-lint v2で `.golangci.yml` のlintとformatterを実行する。

## Goモジュール作成前

`go.mod` が存在しない間は、各workflowのGo関連ステップをスキップする。
Goモジュール作成後は自動的にすべての検証を実行する。

## ローカル確認

```text
task all
```

`task all` は `go.mod` 作成後に使用する。

リリースworkflowは、公開方法を決めるまで作成しない。
