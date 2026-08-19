# 楽天Koboガイド

## 利用開始に必要なもの

楽天Kobo電子書籍検索APIを使うには、楽天ウェブサービスで発行したApplication IDと
Access Keyが必要である。Affiliate IDは任意で、アフィリエイトURLが必要な場合だけ設定する。

Application IDとAccess Keyはアプリ単位で発行・アクセス制御される。2026年8月10日に確認した
楽天ウェブサービスの登録画面では、楽天Books APIと楽天Kobo APIのアクセススコープは連動して
選択された。このUI上の挙動は将来の登録仕様を保証しないため、利用時は登録アプリがKobo APIを
利用できる状態か最新の画面で確認する。

楽天の案内ではAffiliate IDはデベロッパー単位の共通IDとして扱われ、登録したアプリ間で共用する。

## Clientの作成

ライブラリ本体は環境変数を読み込まない。Application ID、Access Key、任意のAffiliate IDを
利用側で安全に読み込み、Client Optionへ渡す。

```go
options := []rakutenkobo.Option{
    rakutenkobo.WithApplicationID(os.Getenv("RAKUTEN_APP_ID")),
    rakutenkobo.WithAccessKey(os.Getenv("RAKUTEN_ACCESS_KEY")),
}
if affiliateID := os.Getenv("RAKUTEN_AFFILIATE_ID"); affiliateID != "" {
    options = append(options, rakutenkobo.WithAffiliateID(affiliateID))
}
client, err := rakutenkobo.NewClient(nil, options...)
```

リポジトリ内のデモも次の共通環境変数を使用する。

```text
RAKUTEN_APP_ID
RAKUTEN_ACCESS_KEY
RAKUTEN_AFFILIATE_ID  # 任意
```

楽天Books用、Kobo用に別の環境変数は設けていない。別アプリの資格情報を使い分ける必要がある場合も、
ライブラリのClientには呼び出し側が任意の値を直接渡せる。

## 検索

タイトル検索の最小例は次のとおりである。

```go
result, err := client.SearchBooks(
    context.Background(),
    rakutenkobo.SearchRequest{Title: "ふつつかな悪女ではございますが"},
)
```

タイトル、著者、出版社、商品キーワードを検索条件にできる。除外語はこれらのいずれとも併用できるが、
除外語だけでは検索できない。楽天Kobo APIが `NGKeyword` 利用時に要求する `keyword` は、
ライブラリが既存の正条件から補完する。補完順序は
[楽天Koboパッケージ仕様](spec.md#41-対応する共通検索条件)を参照する。

既定の漫画区分は一般コミックである。BLまたはTLを検索する場合は、Client作成時に
`WithComicGenre`を指定する。

## デモ

リポジトリルートから次のように実行できる。

```text
go run ./examples/rakutenkobo -title "ふつつかな悪女ではございますが" -limit 5
```

変換前の楽天Koboレスポンスも確認する場合は、未作成のファイルを`-raw-output`へ指定する。
既存ファイルは上書きしない。

```text
go run ./examples/rakutenkobo \
  -title "ふつつかな悪女ではございますが" \
  -raw-output ./kobo-raw.json
```

利用できる全オプションは[CLIデモ](../../../examples/README.md#楽天kobo)を参照する。

## 利用条件

楽天ウェブサービスを利用するアプリでは、楽天の現行案内に従ったクレジット表示が必要である。
取得した商品情報、画像、価格、アフィリエイトURL等の表示・保存・更新についても、組み込み先の
利用形態に応じて最新の公式条件を確認する。

- [楽天ウェブサービス利用ガイド](https://webservice.rakuten.co.jp/guide)
- [クレジット表示](https://webservice.rakuten.co.jp/guide/credit)
- [楽天ウェブサービス利用規約](https://webservice.rakuten.co.jp/guide/rule)

`manken` は利用規約の適用可否を独自に判断して保証しない。アプリケーションの公開方法、
データの保存方法、アフィリエイト利用の有無に応じて利用者が最新条件を確認する。
