# DMMブックス利用ガイド

DMMブックス検索にはDMM会員、DMMアフィリエイト、Webサービス利用登録で発行されるAPI IDとAffiliate IDが必要である。利用側で安全に読み込み、Client Optionへ渡す。

```go
client, err := dmm.NewClient(nil,
    dmm.WithAPIID(os.Getenv("DMM_API_ID")),
    dmm.WithAffiliateID(os.Getenv("DMM_AFFILIATE_ID")),
)
```

リポジトリのデモとintegration testは `DMM_API_ID` と `DMM_AFFILIATE_ID` を使用する。認証値、完全なリクエストURL、無加工Raw responseを保存または公開しない。

DMMの現行アフィリエイト規約と画像利用条件は組み込み先にも適用される。商品URL、アフィリエイトURL、画像URLを返すことは、保存・加工・再配布を許可するものではない。公開前に最新の公式条件を確認する。

シリーズを探し、得られたIDで個別商品を取得する。

```go
series, err := client.SearchSeries(ctx, dmm.SearchSeriesRequest{
    FreeText:     "黄泉のツガイ",
    ExcludedText: "特装版",
	DateFrom:     "2024",
	DateTo:       "2024",
})
if err != nil {
    // エラー処理
}
if len(series.BookSeries) == 0 {
    // 候補なしとして処理
    return
}
books, err := client.SearchBooksBySeries(ctx, dmm.SearchBooksBySeriesRequest{
    SeriesID: series.BookSeries[0].ID,
})
if err != nil {
    // エラー処理
}
_ = books
```

`DateFrom` と `DateTo` はDMM商品 `date` の時期を絞り込む。`SearchSeries` はDMMがkeyword結果へ付与したシリーズと、同じ応答の代表商品由来のタイトル、著者、出版社、genre、画像、商品参照先を候補判別用に返す。補助情報はシリーズ自体の確定属性ではない。`SearchBooksBySeries` はそのシリーズ内の個別商品だけを `Book` として返し、空でないmanufacturer名を出版社として保持する。各Bookの `BookSeries` にはDMMが明示したシリーズIDと名称が設定される。検索条件と失敗時の動作は[DMMパッケージ仕様](spec.md)を参照する。

- [DMMアフィリエイト](https://affiliate.dmm.com/)
- [DMM Webサービス](https://affiliate.dmm.com/api/)
