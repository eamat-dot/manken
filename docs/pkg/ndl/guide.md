# NDLガイド

NDLサーチSRUの利用にAPIキーやアクセストークンは不要である。

```go
client, err := ndl.NewClient(nil)
result, err := client.SearchBooks(ctx, ndl.SearchRequest{Title: "動物のお医者さん"})
```

通常検索は既定で`NDC 726.1`と`NDLC Y84`の両方へ絞り込み、漫画候補を優先する。古い漫画など分類が付与されていない書誌も含めて探したい場合は、必要なフィルタだけ無効化する。

```go
client, err := ndl.NewClient(nil,
    ndl.WithMangaNDLCFilter(false),
)
```

`WithMangaNDCFilter(false)`と`WithMangaNDLCFilter(false)`を両方指定すると分類絞り込みを行わない。分類フィルタは通常検索だけに適用し、ISBN参照には影響しない。

出版時期は共通 `SearchRequest` の `DateFrom` / `DateTo`、件名と内容記述は`SearchOptions`で絞り込む。

```go
result, err := client.SearchBooksWithOptions(ctx,
    ndl.SearchRequest{Title: "動物のお医者さん", DateFrom: "2024", DateTo: "2024"},
    ndl.SearchOptions{
        Description: "ハムテル",
    },
)
```

`DateFrom` / `DateTo`は`YYYY`、`YYYY-MM`、`YYYY-MM-DD`で指定し、両方使う場合は精度をそろえる。`Subject`は人物・地域・題材等の件名検索、`Description`は要約等を含む内容記述系インデックスの検索に使う。`Description`で検索できた語が変換前SRU XMLに含まれるとは限らず、現在のNDL実装は「要約等」の本文を`Description`へ格納しない。

NDLサーチAPIを利用するサイトやアプリケーションでは、NDLサーチAPIを使用していることを表示する。全国書誌情報を二次利用する場合は、適用されるメタデータの利用条件・表示要件を確認する。完成済み全国書誌の二次利用条件はCC BY 4.0互換として案内されている。

同時・継続的な大量アクセスは制限または遮断される場合がある。公開された固定の数値上限は前提にせず、利用量が大きい用途ではNDLサーチの利用案内を確認する。クライアントはリトライ、キャッシュ、待機を自動では行わない。
