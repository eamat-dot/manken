# Google Booksガイド

`googlebooks` はGoogle Books Volumes APIから、タイトル、著者、出版社、フリーワード、除外条件、ISBNで書誌候補を
取得するパッケージである。漫画専用のデータベースではないため、結果を漫画単行本とみなす判定は
利用側で行う。

## APIキーの準備

Google CloudでBooks APIを有効にし、利用するアプリケーション用のAPIキーを作成する。キーは
利用側の環境変数または秘密情報管理機能で保管し、ソースコード、CLI引数、ログへ書かない。

```powershell
$env:GOOGLE_BOOKS_API_KEY = "your-key"
```

`manken` はAPIキーの作成、保存、環境変数の自動読込を行わない。Clientへ明示的に渡す。

```go
client, err := googlebooks.NewClient(nil,
    googlebooks.WithAPIKey(os.Getenv("GOOGLE_BOOKS_API_KEY")),
)
```

検索条件、ページング、ISBN参照、変換項目、エラーの完全な仕様は
[Google Booksパッケージ仕様](spec.md)を参照する。

## CLIデモ

リポジトリのルートで、環境変数を設定してから実行する。

```text
go run ./examples/googlebooks -title "動物のお医者さん" -limit 5
go run ./examples/googlebooks -author "佐々木倫子"
go run ./examples/googlebooks -publisher "白泉社"
go run ./examples/googlebooks 4088466365
```

全オプションとraw responseの保存は [CLIデモ](../../../examples/README.md#google-books)を参照する。

## 利用条件と表示

Google Booksの結果、画像、プレビュー、販売・閲覧情報を利用・表示するアプリケーションは、
[Books API Terms of Service](https://developers.google.com/books/terms)、
[Branding Guidelines](https://developers.google.com/books/branding)、
[Google APIs Terms of Service](https://developers.google.com/terms)を確認する。

- 結果やプレビューを表示する場合は、必要なGoogleへのattributionとGoogle Booksへのリンクを設ける
- API結果を長期保存・キャッシュする場合は、利用規約とレスポンスのcache headerを確認する
- 利用者への課金を伴う形態はBooks API固有のTermsを確認する
- Google Booksの結果順を変更しない。将来の横断検索では、他取得元との混在・順位変更が
  Branding Guidelinesに適合するかを実装前に確認する

このパッケージは規約適合性を保証しない。利用側が表示、保存、課金の形態に応じて現行の公式条件を確認する。
