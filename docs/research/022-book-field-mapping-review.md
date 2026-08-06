# 書誌項目の変換漏れと共通モデルの再調査

## 1. 目的

MADBとopenBDについて、取得元に存在する書誌値がどの段階で失われているかを確認し、
`Sources` と `Normalized` の責務、著者名と読み、シリーズ、価格、識別子、画像の
扱いを再検討する。

調査開始日は2026年8月5日、方針再検討日は2026年8月6日。本書は調査結果と設計候補を
記録する文書であり、実装内容や公開APIの確定仕様ではない。実装範囲と受け入れ条件は、
本調査の完了後に作成する実装TODOで定める。

関連TODO:

- [`../todo/022-review-book-field-mapping.md`](../todo/022-review-book-field-mapping.md)

## 2. 調査方法

書誌値の流れを、次の4段階に分けて確認した。

1. 取得元に存在する値
2. SPARQLまたはAPIのRaw responseに含まれる値
3. 取得元固有の非公開型へ読み込む値
4. 現行 `BookSource.Values` と `NormalizedBook` へ変換する値

`BookSource.Values` は現行実装の欠落箇所を確認するために調査対象へ含めるが、
今後も維持することを前提としない。

欠落の状態は次の語で区別する。

| 状態 | 意味 |
| --- | --- |
| 未取得 | 取得元には存在するが、SPARQLなどの問い合わせ対象に含めていない |
| 未読込 | Raw responseには存在するが、取得元固有の非公開型で読み込んでいない |
| Source未保持 | 非公開型または変換処理では参照しているが、`BookSource.Values` に根拠を残していない |
| Normalized未変換 | 取得した値を共通の意味へ変換していない |
| 意図的な対象外 | 現行仕様で、意味を確定できないなどの理由から変換しないと定めている |

調査対象は、現行コード、保存済みRaw response、共通仕様、MADB・openBD固有仕様、
公式資料とした。

## 3. 現行共通モデルの確認結果

### 3.1 `SourceBookValues` は不完全な複製になっている

現行の `SourceBookValues` は、タイトル、著者、ISBN、シリーズ名などを主に
文字列または文字列スライスで保持する。

この構造では、次の対応関係を表せない。

- 人物名とその読み
- 人物名と取得元の役割コード
- 人物名と取得元が示した順序
- 複数人物へ分割する前の元表記
- 識別子の値と取得元の種別コード
- 同じ識別子がどのレスポンス項目から得られたか
- Collectionのタイトル、種別、刊行番号、読みの組
- 画像URLと画像用途、取得元、ONIXの種別コード
- シリーズ名と取得元内ID、URLの組

対応関係を保つために構造体を追加し続けると、共通 `api` パッケージへopenBDやMADBの
固有構造を再実装することになる。一方、共通化できる項目だけへ絞ると、元データの一部を
切り取った不完全な表現になる。

### 3.2 型変換した時点で「元値」ではなくなる項目がある

MADBの `schema:numberOfPages` は、保存済みRaw responseでは `76p` のような文字列で
返されている。現行実装はこれを整数へ変換した後の `*int` を
`SourceBookValues.PageCount` に設定するため、元の `76p` はRaw response以外に残らない。

価格も同じ問題を持つ。現行 `SourcePrice.Amount` は `int64` であり、取得元が返した
文字列表記、小数、桁区切り、不正値をそのまま保持できない。

`SourceBookValues` を完全な変換根拠として使うには、元文字列、取得経路、役割コード、
関連する別項目を追加し続ける必要がある。これはRaw responseの縮小版を別形式で
再構築することになり、保守対象を増やす。

### 3.3 現行コードは変換後の処理に `SourceBookValues` を使用していない

Goコードの利用箇所を確認したところ、`SourceBookValues` は主に次で参照されている。

- `Book` を組み立てる変換処理
- 公開JSONと型のテスト
- 取得元値を確認する単体テストと結合テスト
- 仕様書と調査文書

作成済みの `Book.Sources[].Values` を、検索、並べ替え、ISBN対応付けなどの後続処理が
入力として使用する箇所は確認できなかった。廃止時の影響は大きな公開API変更になるが、
検索・参照アルゴリズムの依存は小さい。

### 3.4 Raw responseは `BookSource` 全体の代替にはしない

完全な元データは、既存の `SearchBooksWithRawResponse` と
`LookupBooksByISBNWithRawResponse` がリクエスト単位で無加工の本文を返している。
この仕組みは `SourceBookValues` より完全であり、取得元の新項目にも追従できる。

ただし、Raw responseを各 `BookSource` へ埋め込む案は採用しない。

- openBDのRaw responseは複数ISBNを含む配列であり、入力重複の展開後結果と1対1ではない
- MADBのRaw responseは複数冊のbinding行と `head` を含み、1冊分の独立オブジェクトではない
- MADBで1冊分を切り出すと、無加工のRawではなく再構成したJSONになる
- 同じレスポンス全体を各Bookへ入れると、冊数に比例してデータを重複する
- Rawを通常の `Book` に含めると、利用者が必要としない取得元固有データまで常に保持する

`BookSource` はRaw本体ではなく、正規化済みBookを取得元へ結び付ける参照情報として残す。
現在のMADB ID・URL、openBDの検証済みISBN-13が、Raw response内の対応箇所を探す手掛かりになる。

### 3.5 方針案の比較

| 案 | 評価 | 理由 |
| --- | --- | --- |
| 現行 `SourceBookValues` を拡張する | 採用しない | 取得元固有構造を共通APIへ再実装し続けることになり、完全性も保証できない |
| 元タイトルなど少数項目だけへ縮小する | 第二候補 | 現状より単純になるが、残す項目の境界が恣意的で、Rawとの二重管理は残る |
| `SourceBookValues` を廃止し、Rawを別戻り値で返す | 採用 | 共通モデルと取得元固有データの境界が明確で、既存Rawメソッドを利用できる |
| Raw responseを各 `BookSource` へ埋め込む | 採用しない | MADBでは1冊単位のRawではなく、再構成またはレスポンス全体の重複になる |

元タイトルなど一部だけを通常利用したい要件が後から明確になった場合は、
`SourceBookValues` を復活させるのではなく、その値が本当に複数取得元で共有できる
`Normalized` の項目かを個別に検討する。

### 3.6 推奨する境界

次の構造を採用する。

```text
Book
  Normalized
  Sources
    Source
    ID
    URL

WithRawResponseの戻り値
  変換済みResult
  リクエスト単位の無加工レスポンス本文
```

- `SourceBookValues` は初回リリース前に削除し、互換用フィールドを残さない
- `BookSource` は `Source`、`ID`、`URL` を持つ取得元参照へ縮小する
- 共通の意味へ安全に変換できる値だけを `Normalized` へ設定する
- 安全に変換できない値は `Book` へ不完全な形で複製せず、Raw responseに残す
- Raw responseが必要な利用者だけが `WithRawResponse` を使用する
- 取得元固有の公開Raw型を共通 `api` パッケージへ追加しない

将来、複数取得元のBookを統合する場合のフィールド単位の出典管理は、
`SourceBookValues` では解決できない。必要になった時点で、正規化項目自体の出典または
統合処理専用のprovenanceを別に設計する。

## 4. MADBの項目対応

### 4.1 現行対応表

| 取得元の値 | Raw response | 非公開型 | `Sources` | `Normalized` | 状態・補足 |
| --- | --- | --- | --- | --- | --- |
| リソースURI | 取得 | `ResourceURI` | `BookSource.URL` | なし | 対応済み |
| `schema:identifier` | 取得 | `ID` | `BookSource.ID` | なし | 取得元内IDとして対応済み |
| 言語タグなし `schema:name` | 取得 | `Titles` | `Titles` | `Title` | 対応済み。複数候補はSourceに残る |
| `ja-hrkt` の `schema:name` | 取得 | `TitleKana` | `TitleKana` | `TitleKana` | 対応済み |
| `schema:alternativeHeadline` | 取得 | `Subtitles` | `Subtitles` | `Subtitle` | 対応済み |
| `ma:seriesName` | 取得 | `SeriesNames` | `SeriesNames`へ統合 | `Series` | 参照先シリーズ名との区別がSourceで失われる |
| MangaBookSeriesの `schema:name` | 取得 | `RelatedSeriesNames` | `SeriesNames`へ統合 | `Series` | 直接値との区別がSourceで失われる |
| MangaBookSeriesのID・URI | 取得 | `SeriesID`、`SeriesResourceURI` | 格納先なし | `Series.ID`、`Series.URL` | Source未保持 |
| `schema:volumeNumber` | 取得 | `VolumeNumber` | `Volume` | `Volume` | 対応済み |
| `schema:version` | 取得 | `Versions` | `Editions` | `EditionStatements` | 対応済み |
| 言語タグなし `schema:creator` | 取得 | `Creators` | `Authors` | `Authors`、`Contributors` | 対応済み。ただしAgent名との併存時にSource欠落あり |
| creator Agentの `rdfs:label` | 取得 | `AgentNames` | creatorがない場合だけ `Authors` | creatorがない場合だけ `Authors` | creator併存時はSource未保持 |
| `schema:publisher` | 取得 | `Publishers` | `Publishers` | `Publishers` | 元値と整形後を分けて保持できている |
| `schema:brand` | 取得 | `Brands` | `Imprints` | `Imprints` | 対応済み |
| `schema:isbn` | 取得 | `ISBNs` | `ISBNs` | `Identifiers` | 対応済み。種別は検証結果から決定 |
| `schema:datePublished` | 取得 | `PublishedDate` | `PublishedDate` | `Dates` | 対応済み |
| `schema:numberOfPages` | 取得 | 元文字列 | 解析後の `*int` | `PageCount` | Sourceで元文字列を失う |
| `schema:size` | 取得 | `Size` | `Size` | `PhysicalSize`、`Medium` | 対応済み |
| `schema:provider` 内の `schema:price` | 未取得 | なし | なし | なし | 所蔵・提供元との組を保った実例調査が必要 |
| creatorの読み | 未取得 | なし | なし | なし | 人物名との対応可能性の調査が必要 |
| 説明、言語、キーワード、画像 | 未取得 | なし | なし | なし | 現行検索では一部を検索条件に使うが結果へ返していない |

### 4.2 creatorとAgent名

現行 `convertCreators` は、言語タグなしの `schema:creator` が1件以上ある場合、
`Sources[0].Values.Authors` にcreator文字列だけを残す。Agent名もRaw responseと
非公開型には存在するが、creatorと併存するとSourceから失われる。

creator文字列とAgent参照はRDF上で1対1に対応していないため、名前や配列位置で
結び付けるべきではない。対応を確定できないAgent名は `Normalized` へ入れず、
取得元固有の値としてRaw responseから確認する。

これらは取得元固有の非公開型で変換中に区別し、完全な値はRaw responseから確認する。
共通 `BookSource` へcreator文字列、Agent名、Agent URIを複製しない。

- `schema:creator` の文字列
- `dcterms:creator` が参照するAgentの表示名
- Agent URI
- Agent側で取得できる読み候補

安全に対応付けられた人物情報だけを `Normalized` へ設定し、対応できない値は
Raw responseに残す。

2026年8月3日に保存したRaw responseでは、次を確認した。

- `佐々木倫子`、`小林有吾` はcreator文字列とAgent名が実質的に一致する
- `藤子・F・不二雄` に対してAgent名が `藤子不二雄` となる例がある
- 同じ書誌に `藤子・F・不二雄` と `[著]藤子不二雄` の複数creator表記がある
- 現行クエリはAgent URIをSELECTしていないため、結果行からAgent単位に集約できない

この例から、文字列一致、配列位置、行順ではcreatorとAgentを対応付けられないことを
確認できる。少なくとも `?agent` を結果へ含め、同じAgent URIに属するラベルだけを
同一人物の候補として扱う必要がある。

### 4.3 MADBの著者読み

MADBでは、同じプロパティに言語タグの異なる値が存在し得る。現行の公式スキーマ解説図では、
マンガ単行本の `schema:creator` に通常表記と `ja-Hrkt` の読みが別リテラルとして
記録される例が示されている。過去データには `ja-Hrkt-Hrkt` の表記例もあるため、
実装では固定文字列を増やす前に現行SPARQLの言語タグを確認する。

現行クエリは `FILTER (LANG(?creator) = "")` により言語タグなしのcreatorだけを取得する。
そのため、MADBが保持するcreator読みは現在のRaw responseへ入らない。

ただし、現在のMangaBook実データについて、複数の通常表記と読みをどのように対応させるか、
Agent URI側にも読みが存在するかは未確認である。

人物名と読みが独立した複数リテラルとして返る場合、次のような平行配列を作っても
対応は保証されない。

```text
人物名: [佐々木倫子, 藤原新也]
読み:   [ササキノリコ, フジワラシンヤ]
```

SPARQLの行順や文字列ソート順だけで人物と読みを結び付けない。

安全に `Normalized` へ設定する条件は、次に限定する。

- 取得元の同一Agent URIに通常表記と読みが付いている
- 取得元が人物と読みの明示的な関連を提供する

`schema:creator` の通常表記と読みが1件ずつでも、同じ人物を表す関係はRDF上で明示されない。
件数の一致だけでは対応付けない。それ以外の読みはRaw responseに残し、人物へ設定しない。

人物項目は、他の複数値項目との直積を避けるため、次のような縦長の別クエリ候補とする。

```sparql
SELECT ?resource ?kind ?agent ?value ?lang
WHERE {
  VALUES ?resource { ... }
  {
    ?resource schema:creator ?value .
    BIND("creator" AS ?kind)
  }
  UNION
  {
    ?resource dcterms:creator ?agent .
    ?agent rdfs:label ?value .
    BIND("agentLabel" AS ?kind)
  }
  BIND(LANG(?value) AS ?lang)
}
```

同じ `?agent` の `agentLabel` だけを名前・読み候補として集約する。直接creatorリテラルは
役割付きの表示文字列として利用できるが、Agentラベルと結び付けない。

保存済みRaw responseと現行クエリでは、creator読みとAgent読みの現在の実データまでは
確認できなかった。上記試験クエリの実行が必要である。

### 4.4 現行SPARQLサービスでの人物実例

2026-08-06T01:59:42+09:00 に、公開SPARQLサービスへ既知4リソースを指定した
縦長クエリを2回送った。1回目はHTTP 200、本文14,860 bytes、HTTP本文取得まで197 ms
だったが、調査側の出力変換の不具合により行を記録できなかった。2回目は同じ対象を
より小さい出力形式で確認し、HTTP 200、本文4,430 bytes、HTTP本文取得まで177 ms、
10 bindingsだった。これは人物データの確認用であり、方式A・B・Cの性能比較ではない。

実行したクエリは次のとおり。`VALUES` の4 URIは、保存済みISBN参照応答に含まれる
佐々木倫子、小林有吾、藤子・F・不二雄／藤子不二雄の例である。

```sparql
PREFIX schema: <https://schema.org/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

SELECT ?resource ?kind ?agent ?value ?lang
WHERE {
  VALUES ?resource {
    <https://mediaarts-db.artmuseums.go.jp/id/M292127>
    <https://mediaarts-db.artmuseums.go.jp/id/M879037>
    <https://mediaarts-db.artmuseums.go.jp/id/M235839>
    <https://mediaarts-db.artmuseums.go.jp/id/M236091>
  }
  { ?resource schema:creator ?value . BIND("creator" AS ?kind) }
  UNION
  { ?resource dcterms:creator ?agent . ?agent rdfs:label ?value . BIND("agentLabel" AS ?kind) }
  BIND(LANG(?value) AS ?lang)
}
ORDER BY ?resource ?kind ?agent ?lang ?value
```

| resource | 値の種別 | 人物名または値 | 読み | 言語タグ | Agent URI | 対応可否 |
| --- | --- | --- | --- | --- | --- | --- |
| `M292127` | creator | `[著]佐々木倫子` | なし | なし | なし | 不可 |
| `M292127` | Agent label | `佐々木倫子` | なし | なし | `C55929` | 読みなし |
| `M879037` | creator | `[著]小林有吾` | なし | なし | なし | 不可 |
| `M879037` | Agent label | `小林有吾` | なし | なし | `C66657` | 読みなし |
| `M235839` | creator | `[著]藤子・F・不二雄` | `フジコエフフジオ` | `ja-hrkt` | なし | 不可 |
| `M236091` | creator | `藤子・F・不二雄`、`[著]藤子不二雄` | `フジコフジオ　／　フジコエフフジオ` | `ja-hrkt` | なし | 不可 |
| `M236091` | Agent label | `藤子不二雄` | なし | なし | `C47471` | 読みなし |

この試料では `schema:creator` に通常表記と読みがあり、読みのタグはすべて
小文字の `ja-hrkt` だった。`ja-Hrkt` と `ja-Hrkt-Hrkt` は観測されなかった。ただし、
4リソースだけの観測であり、現行データ全体でこれらの表記が存在しないことを示すものではない。

Agentラベルは各Agent URIごとに分離できるため、同じAgent URI内に通常表記と読みの
ラベルがあれば組にできる。しかし今回の3 Agentはいずれも通常表記だけで、直接creatorの
読みをAgentへ結び付けるプロパティも、このクエリで確認した範囲では得られなかった。
`M236091` のように同じresourceに複数creator表記があるため、creatorとAgentを
文字列、件数、位置、行順で対応付けることもできない。Agent URIを持たないcreator読みは
`M235839` と `M236091` で実在した。

従って、現時点で安全に `Contributor{Name, Reading}` を設定できるのは、同じAgent URIの
`rdfs:label` に、通常表記と読みがともにあり、読みの言語タグを個別に確認できる場合だけである。
未知の言語タグは読み候補へ自動採用せず、Raw responseに残す。直接creatorは、既存方針どおり
`Authors` の表示・簡易一覧には使えても、Agent由来の `Contributor` の読みとは結び付けない。

### 4.5 今回の人物クエリ方式の結論

方式Bは横長クエリにcreator読み、Agentラベル読み、言語タグを追加するため、既存の複数値
`OPTIONAL` との直積を増やす。一方、今回の実例では追加したcreator読みを人物へ安全に
結び付けられない。方式Cは縦長の人物クエリによって通常書誌項目との直積を避けられるが、
同じ理由で直接creatorの読みを変換できない。Agentラベルに読みを持つ実例も得られなかった。

MADBの人物読みは現時点では実装しない。方式Aを現行の人物取得方式として維持する。
これは性能上の優位を実測で確定した結論ではない。方式BとCが今回の共通モデルへ安全に
追加できる人物情報を返さなかったため、性能比較の対象ではなく今回の対象から外す。

将来、同一Agent URIに通常表記と読みを持つ実例、またはcreatorリテラルとAgentを明示的に
結び付ける取得元プロパティが確認された場合だけ再調査する。その場合の候補は方式Cである。
通常の書誌項目と人物項目の直積を避けられる一方、後段だけの失敗、ページング、Limit、
Raw responseを2本文として返すかの公開API仕様を別途検討する必要がある。再調査条件は
[`../backlog.md`](../backlog.md) で管理する。

### 4.6 MADBの価格

MADBの所蔵情報は、アイテム直下の `schema:provider` に入れ子で記録される。
2022年版MADBデータを使用した公開例では、マンガ単行本「鬼滅の刃 1」に
2件のproviderがあり、それぞれの空白ノードにprovider名と `schema:price "400円"` が
記録されている。

```text
item
  schema:provider
    schema:name  国立国会図書館
    schema:price 400円

item
  schema:provider
    schema:name  大阪府立中央図書館国際児童文学館
    schema:price 400円
```

この例から、価格を単純な次の形では取得できない可能性が高い。

```sparql
?resource schema:price ?price .
```

少なくともproviderを経由し、価格と所蔵・提供元の対応を保つ必要がある。

```sparql
?resource schema:provider ?provider .
?provider schema:name ?providerName .
?provider schema:price ?price .
```

この価格はproviderごとの所蔵情報に含まれるため、書誌レコード全体に対する単一価格と
決め付けない。複数providerが同じ値を返しても、重複除去によってproviderとの対応を
失わない。

公式解説でも `schema:provider` は所蔵情報として説明されている。したがって、その配下の
価格は書誌レコード全体に対する共通価格ではなく、所蔵・提供元に付随する値として扱う。
現行の共通 `Price` は書籍の定価または取得時点価格を表すため、意味が一致しない。

このため、MADB provider価格は今回取得・変換しない。共通 `BookSource` へ複製せず、
価格が必要になった場合はprovider、所蔵ID、注記、価格を組にした所蔵情報モデルとして
別途調査する。現行の横長クエリへproviderを追加する性能比較も行わない。

## 5. openBDの項目対応

### 5.1 現行対応表

| 取得元の値 | Raw response | 非公開型 | `Sources` | `Normalized` | 状態・補足 |
| --- | --- | --- | --- | --- | --- |
| `RecordReference` | 取得 | 読込 | ISBN文字列へ統合 | 問い合わせISBNをISBN-13として設定 | 元の取得経路が失われる |
| `ProductIdentifier.ProductIDType` | 取得 | 読込 | 格納先なし | `15`の場合の値を応答検証に使用 | Source未保持 |
| `ProductIdentifier.IDValue` | 取得 | 読込 | ISBN文字列へ統合 | 問い合わせISBNをISBN-13として設定 | 元の種別・経路が失われる |
| `summary.isbn` | 取得 | 読込 | ISBN文字列へ統合 | 応答検証に使用 | 元の取得経路が失われる |
| 商品階層タイトル | 取得 | 読込 | `Titles` | `Title`、推測した `ParallelTitles`・`Volume` | 区切り記号と末尾表記による推測的分解を行っており、見直し対象 |
| タイトル `collationkey` | 取得 | 読込 | `TitleKana` | `TitleKana` | 現行は空になる場合がある。修正後は全体を `TitleReading` へ設定する |
| Subtitle | 取得 | 読込 | `Subtitles` | `Subtitle` | 明示項目の対応として維持可能 |
| Collection | 取得 | タイトルだけ読込 | `SeriesNames` | `Series` | シリーズ名とレーベル名を区別できない項目を作品Seriesへ固定しているため見直し対象 |
| Contributor名 | 取得 | 読込 | `Authors` | `Authors`、一部を `Contributors` | 名前自体は変更していない。読みとの組を共通モデルで表現できない |
| Contributor読み | 取得 | 読込 | 格納先なし | なし | 同じPersonName要素の名前と読みを組にして共通モデルへ移す候補 |
| ContributorRole | 取得 | 読込・変換に使用 | 格納先なし | 既知役割だけ `Contributors` | `A38` の現行対応は公式コードの意味と不一致。ほかの統合コードも詳細消失を確認する |
| SequenceNumber | 取得 | 読込・並べ替えに使用 | 格納先なし | 順序へ反映 | 明示された順序の保持として維持可能 |
| Imprint | 取得 | 読込 | `Publishers`へ統合 | `Publishers` | openBDでは発行元出版社。Publisher候補として維持可能 |
| Publisher | 取得 | 読込 | `Publishers`へ統合 | `Publishers` | openBDでは発売元出版社。Publisher候補として維持可能 |
| PublishingDate | 取得 | 読込 | 選択値だけ | `Dates` | `01` と `11` を同じpublishedへ丸めており見直し対象 |
| `hanmoto.dateshuppan` | 取得 | 読込 | 選択値だけ | `Dates` | 当該商品の出版日候補。ONIX `01` がない場合の優先順位を確認する |
| `summary.pubdate` | 取得 | 読込 | 選択値だけ | `Dates` | 当該商品の出版年月候補。ONIX `01` がない場合の優先順位を確認する |
| `ProductSupply.SupplyDetail.Price` | 取得 | 未読込 | なし | なし | 現行仕様では意図的な対象外 |
| `summary.cover` | 取得 | 読込 | 格納先なし | `Images` | Source未保持。保存済みマンガ例では空、現行API例では値あり |
| SupportingResource | 取得され得る | 未読込 | なし | なし | 現行API例で書影を確認。現行仕様では意図的な対象外 |
| 説明、言語、主題、ページ数、寸法、版表示 | 取得され得る | 多くを未読込 | なし | なし | 現行仕様では意図的な対象外 |

### 5.2 openBDの位置付けと変換の境界

openBDは出版社・流通向けのONIX構造を中心とし、マンガ利用者向けの作品名、巻数、
レーベル、著者表示へ直接対応するデータ取得元ではない。さらに、利用ガイドラインは
APIから得たデータを任意に改変しないよう求めている。

このため、openBDを主要な検索・正規化用データ取得元として発展させず、ISBN参照で
不足情報を補う補助取得元として維持する。新しいONIX項目を網羅的に調査・実装することは
行わず、現行実装の明らかな誤変換を除去し、取得元が明示した値を最小限共通化する。

API仕様はONIXの項目とコードを利用側が読み取ることを前提としている。この調査では、
規約の法的解釈を確定するのではなく、誤変換と不要な書換えを避ける設計基準として
次を採用する。

#### 行う変換

- 現在利用しているONIXの明示項目を、対応する共通項目へ移す
- `ProductIDType=15` の値を検証済みISBN-13として扱う
- `SequenceNumber` に従ってContributorの表示順を保つ
- `PersonName.content` と同じ要素の `collationkey` を名前と読みの組として扱う
- 公式コードの意味と共通役割が一致する場合だけContributorRoleを変換する
- `PublishingDateRole=01` だけを当該商品の出版日として扱う
- 空文字列を除き、完全に同一の値だけを重複除去する

#### 行わない変換

- タイトル文字列の区切り記号から並列タイトルを推測する
- タイトル末尾の表記から巻数を抽出する
- 抽出した巻数に合わせてタイトル読みの一部を削る
- 人物名を書き換える、注記を削る、1要素を複数人物へ分割する
- Collection名から作品シリーズ、出版コレクション、レーベルのいずれかを推測する
- 未知または意味の異なるONIXコードを、近い共通enumへ丸める
- 具体的な利用要件がない価格、CollectionSequence、商品階層PartNumber、SupportingResourceを追加実装する

ここでいう `Normalized` は、読みやすい表示へ編集した値ではなく、取得元が明示した意味を
取得元非依存の共通型へ移した値とする。表示上の整形が必要になった場合は、書誌変換とは
別のヘルパーまたはアプリケーション層で検討する。

### 5.3 現行のタイトル分解

ISBN `9784063765786` の保存済みRaw responseは次の値を持つ。

```text
TitleText.content:
転生したらスライムだった件 = Regarding Reincarnated to Slime. 1

TitleText.collationkey:
テンセイ シタラ スライム ダッタ ケン
```

現行 `parseSourceTitle` は ` = ` と末尾数字を解釈し、日本語タイトル、英語並列タイトル、
巻数へ分ける。さらに `normalizeTitleKana` は、読み側から同じ巻数を分離できない場合に
読みを空にする。

この例は人間には分解できそうに見えるが、区切り記号と末尾表記から意味を導出する処理であり、
openBDが独立項目として明示した並列タイトルまたは巻数ではない。次の方針を確定する。

- `TitleText.content` 全体を `Normalized.Title` に変更せず設定する
- `TitleText.collationkey` 全体をタイトルの読みとして設定する
- 区切り記号から `ParallelTitles` を作らない
- タイトル末尾から `Volume` を作らない
- 商品階層の `PartNumber` は今回対応しない
- Collection階層の `PartNumber` とCollectionSequenceは作品巻数へ使用しない
- `summary.volume` もCollection由来の刊行番号を返す場合があるため、作品巻数へ使用しない

保存済みISBN `9784063765786` では、タイトル末尾が作品巻数の `1` である一方、
Collectionの `PartNumber` と `summary.volume` はどちらも `578` だった。これは
`シリウスKC` 内の刊行番号と考えられ、作品巻数として使用できない。

```text
TitleText.content: 転生したらスライムだった件 = Regarding Reincarnated to Slime. 1
Collection.TitleElement.PartNumber: 578
summary.volume: 578
```

現在のopenBD JSON Schemaは商品階層の `TitleElement` にも `PartNumber` を定義している。
しかし、保存済みマンガ例では商品階層の実値を確認できず、補助取得元へ縮小する方針では
追加調査の費用に見合わない。具体的な利用要件が生じるまで対応しない。

これにより、現在の読み欠落問題は読みだけを救済するのではなく、原因となる推測的な
タイトル分解自体を行わないことで解消する。

### 5.4 openBDのContributor実例と変換方針

保存済みRaw responseで次を確認した。

| ISBN | `PersonName.content` | `collationkey` | 論点 |
| --- | --- | --- | --- |
| `9784758088732` | `オノ・マサユキ／阪元裕吾` | `オノマサユキ　サカモトユウゴ` | 1要素に複数人物 |
| `9784063765786` | `伏瀬, 1975-` | `フセ, 1975-` | 人物名に生年注記 |
| `9784063765786` | `川上, 泰樹` | `カワカミ, タイキ` | 日本語姓名をコンマで区切る典拠形 |
| `9784091400017` | `藤子, F. 不二雄` | `フジコ, F. フジオ` | 日本語とラテン文字が混在する典拠形 |
| `9784865546156` | `長頼` | `ナガヨリ` | 単純な1人物と1読み |

これらはopenBDが返した著者表示として、そのまま保持する。

- `／` を含んでいても複数人物へ分割しない
- `1975-` などの生没年注記を削除しない
- `川上, 泰樹` を `川上泰樹` へ書き換えない
- 空白、コンマ、中黒などの表記を統一しない
- `PersonName.content` と同じ要素の `collationkey` は、分割せず同じContributorに設定する

表示用に整形した名前が必要になった場合も、それをopenBD由来の書誌値として上書きしない。

#### ContributorRoleの確認結果

現行の役割変換には、単なる情報圧縮とは別に意味の不一致がある。

- `A01` はauthorであり、共通 `author` へ対応できる
- `A14` はコミックを含む本文執筆であり、共通 `writer` へ対応できる
- `A45` はコミックの脚本であり、共通 `writer` へ対応できる
- `A07`、`A12`、`A35` は美術・作画系であり、広い共通 `artist` へ対応できる
- `A46` はinker、`A47` はcoloristであり、`artist` への統合は詳細を失うため明記が必要である
- `A36` は表紙デザインまたは表紙アートであり、一般的なdesignerへの統合は表紙役割を失う
- `A38` は初版のoriginal authorを表し、マンガの原作者・原案者を意味しない

現行実装は `A38` を `original_creator` へ変換しているが、これは公式コードの意味と一致しない。
名前や日本語で期待される役割から `original_creator` と推測せず、当面未対応とする。

初期対応範囲は次のように確定する。

| ONIXコード | 初期方針 | 理由 |
| --- | --- | --- |
| `A01` | `author` へ変換 | 意味が直接対応する |
| `A03`、`A14`、`A45` | `writer` へ変換 | 脚本・本文・コミック脚本を広い執筆役割として扱える |
| `A07`、`A12`、`A35` | `artist` へ変換 | 視覚作品・挿絵・作画を広い美術役割として扱える |
| `B01` | `editor` へ変換 | 意味が直接対応する |
| `B06` | `translator` へ変換 | 意味が直接対応する |
| `A36` | 当面未対応 | 表紙デザイン・表紙アートを一般的なdesignerへ丸めると対象範囲を失う |
| `A38` | 当面未対応 | 初版著者であり、共通 `original_creator` と意味が異なる |
| `A46` | 当面未対応 | inkerを一般的なartistへ丸めると役割を失う |
| `A47` | 当面未対応 | coloristを一般的なartistへ丸めると役割を失う |

`A36`、`A38`、`A46`、`A47` 用の共通役割は今回追加しない。未対応コードを持つ人物自体を
失わないため、役割不明でも `Contributor` に人物を残せる構造を共通モデル候補とする。

### 5.5 ContributorとAuthorsの構造候補

#### 案A: `Contributor` を全人物情報として使用する

- `Contributor` に `Reading` を追加する
- 役割不明の人物も、空の `Roles` で `Contributors` に含める
- `Authors []string` は利用者向けの簡易一覧として残す

利点:

- 人物名、読み、役割を同じ要素で保持できる
- 既存の `Contributor` を拡張できる
- 役割を返さない現在のopenBD代替データでも人物の読みを保持できる
- 将来、別取得元から役割を得た場合に同じ構造を利用できる

欠点:

- 現在の「役割を確定できた人物だけをContributorsへ入れる」という意味が変わる
- `Authors` と `Contributors` に人物名が重複する
- 既存利用者が `Contributors` の各要素に役割があると仮定している可能性がある

#### 案B: `Authors` を人物構造体へ変更する

- `Author{Name, Reading}` のような型を作る
- `Contributors` は役割付き人物のまま維持する

利点:

- 著者名と読みの対応が明確になる
- 役割不明人物と役割付き寄与者の現行区分を維持できる

欠点:

- `Authors []string` を破壊的に変更する
- 同一人物をAuthorとContributorの両方で表現する
- 著者、原作者、作画者を `Authors` に含める現在の広い意味と型名が合いにくい

#### 案C: 詳細人物情報を別項目へ追加する

- `Authors []string` と `Contributors` を維持する
- `People` または `AuthorDetails` のような別項目を追加する

利点:

- 既存項目の意味を変えず追加できる

欠点:

- 3種類の人物一覧が生まれる
- どの項目を一次情報として使うべきか分かりにくい
- 同じ人物の重複と同期規則が増える

初回リリース前で互換性維持が不要なため、案Aを採用する。`Contributor` を人物情報の
一次構造とし、役割不明でも保持する。`Authors []string` は表示・簡易利用向けの一覧として
残し、同じ人物が両方に現れることを許容する。

### 5.6 `Kana` と `Reading`

タイトル読みと人物読みは、検索、並べ替え、表記揺れの確認に利用できるため、
`Normalized` に保持する価値がある。

一方、openBDの `collationkey` は読み・照合キーであり、必ずしもカタカナだけとは限らない。
`F.`、数字、空白、記号などを含み得る。MADBの言語タグも、値が常にカタカナだけで
構成されることを保証する名称ではない。

意味としては `Kana` より `Reading` が広く正確である。

初回リリース前の共通モデル変更で、`TitleKana` を `TitleReading` へ変更する。
`Contributor` には `Reading` を追加し、タイトルと人物の用語を統一する。
旧フィールドは互換用に併存させない。

### 5.7 出版社と刊行日

openBDの現行JSON Schemaでは、次の意味が明記されている。

- `Imprint`: 発行元出版社
- `Publisher`: 発行元と異なる場合の発売元出版社

したがって、openBDの `Imprint` を共通 `Imprints` のコミックレーベルへ移す案は採用しない。
現行のように、Imprint、Publisher、`summary.publisher` を出版社候補として扱う。
補助取得元のためだけに発行元と発売元を区別する共通出版社型は追加しない。

PublishingDateRoleも、公式コードの意味が一致する場合だけ変換する。

| ONIXコード | 公式の意味 | 初期方針 |
| --- | --- | --- |
| `01` | 当該商品の出版日 | 共通 `published` へ変換する |
| `11` | 収録作品が最初に刊行された日 | 当該商品の `published` には使用しない |

現行実装は `01` と `11` の最初の値をどちらも `published` として採用するため、作品の
初版刊行日を当該商品の出版日として返す可能性がある。修正後はONIX `01`、
`hanmoto.dateshuppan`、`summary.pubdate` の順に採用し、`11` は使用しない。
openBDのためだけに作品初版刊行日の共通日付種別は追加しない。

保存済みマンガ例では次を確認した。

| ISBN | ONIX日付 | `hanmoto.dateshuppan` | `summary.pubdate` |
| --- | --- | --- | --- |
| `9784063765786` | `11: 201510` | なし | `201510` |
| `9784865546156` | `11: 202002` | なし | `202002` |
| `9784758088732` | なし | なし | なし |
| `9784091400017` | なし | `1979-03` | `1979-03` |

`11` と `summary.pubdate` が一致する例はあるが、同じ値であることだけでは意味が同じとは
判断しない。`11` を無視し、当該商品の出版日として明示された候補だけを採用する。

## 6. シリーズ、出版コレクション、レーベル

### 6.1 MADB

MADBのMangaBookSeriesは、同一タイトルで連続するマンガ単行本のまとまりを表す。
`動物のお医者さん` など、作品と巻の系列として `Normalized.Series` へ設定する意味が
通る。

現行実装は、単行本へ直接記録された `ma:seriesName` と、参照先MangaBookSeriesの
名前をどちらも `Normalized.Series` へ入れる。この扱いは維持できる可能性が高い。

直接値と参照先シリーズの違いは変換中の非公開型で扱い、完全な組はRaw responseから
確認する。共通 `BookSource` へシリーズ固有構造を複製しない。

### 6.2 openBD

openBD仕様はCollection階層を「シリーズ名／レーベル名」と説明している。
ONIXの `CollectionType=10` はPublisher collectionであり、作品シリーズだけでなく、
出版社が共通の主題、デザイン、著者などでまとめたコレクションも含み得る。
`TitleElementLevel=02` はCollection階層であることを示すだけで、作品シリーズか
レーベルかを確定しない。

保存済みRaw responseでは、次がCollectionとして返された。

- `シリウスKC`
- `HOWLコミックス`
- `ガルドコミックス`

過去に公開されたopenBD応答例では、Collection内に `CollectionSequence`、
`TitleElementLevel=03`、`PartNumber=200` など、作品の巻数ではなく出版上の管理情報と
考えられる値も確認できる。これは現行APIの実例ではないが、Collectionの
`PartNumber` やSequenceをそのまま作品巻数へ使用しない現行方針を補強する。

これらは作品シリーズより、コミックレーベルまたは出版社コレクションとして扱う方が
自然である。現行実装はすべて `Normalized.Series` へ設定するため、意味が異なる値を
作品シリーズと同じ型へ入れている。

### 6.3 確定する境界

- `Series`: 取得元が作品と巻の系列として明示した値
- `Imprints`: 取得元が発行レーベルまたはブランドとして明示した値
- openBD Collection: 共通モデルへ変換せずRaw responseにだけ残す

openBD Collectionは、取得元がシリーズとレーベルを区別しないため、名称辞書で
`Series` または `Imprints` に振り分けない。補助取得元のためだけにPublisher collectionを
表す共通型も追加しない。

openBDのImprintは発行元出版社、Publisherは発売元出版社として、どちらも
`Normalized.Publishers` の候補にする。共通 `Imprints` は、MADBの `schema:brand` など、
取得元がレーベルまたはブランドと明示した値だけに使用する。

現行実装で見直す対象は、openBD Collectionを作品Seriesへ変換している点である。

## 7. 価格

### 7.1 openBD

保存済みRaw responseでは次を確認した。

| ISBN | PriceType | CurrencyCode | PriceAmount |
| --- | --- | --- | ---: |
| `9784758088732` | `01` | `JPY` | `720` |
| `9784063765786` | `01` | `JPY` | `630` |
| `9784865546156` | `01` | `JPY` | `620` |

ONIX Price typeの一部は共通価格へ対応付けられるが、価格はopenBDを補助取得元として
維持するための必須情報ではない。価格種別、税区分、適用期間、通貨単位まで安全に扱うには
追加の仕様・実例・テストが必要になるため、今回は新規対応しない。

価格が具体的な利用要件になった場合に、別の調査TODOとして再開する。元の価格情報は
Raw responseから確認し、共通 `BookSource` へONIX価格構造を複製しない。

### 7.2 MADB

MADB価格はproviderの所蔵情報に属し、書籍の定価または販売価格を表す共通 `Price` と
意味が一致しない。このため、今回の取得・変換対象から外す。

将来必要になった場合は、provider名、所蔵識別子、注記、価格を組にした所蔵情報モデルとして
別の調査TODOを作る。

## 8. 識別子

### 8.1 openBDの現状

現行実装は、次のISBN候補を検証する。

- `onix.RecordReference`
- `onix.ProductIdentifier` の `ProductIDType=15` の値
- `summary.isbn`

値が問い合わせISBNと矛盾しないことを確認した後、問い合わせに使用した正規化済み
ISBN-13を `Normalized.Identifiers` に設定する。この変換結果は妥当である。

`RecordReference` は常にISBNとは限らないため、値の見た目だけでISBN種別へ変換しない。
`ProductIDType=15` はISBN-13として検証できる。その他の種別は公式コード表に従い、
対応済みの種別だけを `Normalized.Identifiers` へ変換する。

### 8.2 元の識別子情報

次の取得元固有情報は、共通 `BookSource` へ構造化して複製しない。

- 取得元の種別コード
- 元の値
- 取得元内の項目名または取得経路

これらはRaw responseから確認する。`BookSource.ID` は取得元レコードの参照キー、
`Normalized.Identifiers` は検証済みの共通識別子という境界にする。

## 9. 書影と画像

### 9.1 現行実装

openBDは `summary.cover` が空でない場合、`Normalized.Images` に
`Purpose=cover` として設定する。書影取得処理自体は実装済みである。

今回確認した保存済みマンガRaw responseでは、`summary.cover` はすべて空だった。
これは保存済みマンガ例に限った結果であり、openBD全体で書影が取得できないことを
意味しない。

2026年8月5日に公式サイトから案内されているAPI例 `9784780802047` を確認したところ、
次の両方に同じ書影URLが含まれていた。

```text
CollateralDetail.SupportingResource
  ResourceContentType: 01
  ResourceMode:        03
  ResourceForm:        02
  ResourceLink:        https://cover.openbd.jp/9784780802047.jpg

summary.cover:
  https://cover.openbd.jp/9784780802047.jpg
```

少なくとも1件について、現行APIでもSupportingResourceと `summary.cover` が併存し、
同じURLを返すことを確認できた。どちらか一方だけを返す書誌、異なるURLを返す書誌、
複数画像を返す書誌があるかは未確認である。

openBDは2023年の代替書誌提供開始時に、従来と比べて項目欠落と書影収録範囲の縮小が
生じると案内している。

### 9.2 SupportingResource

ONIXの `CollateralDetail.SupportingResource` は、画像などの付帯リソースを表す。
表紙画像として採用する場合は、少なくとも次を確認する。

- ResourceContentTypeが表紙、表紙サムネイルなどの対応済みコードか
- ResourceModeが静止画像か
- ResourceVersionのURLが空でないか
- ResourceFormによる利用・保存条件
- 複数画像がある場合の優先順位

単に `ResourceLink` が存在するだけで表紙として採用しない。

保存済みマンガRaw responseにはSupportingResourceの実例がなく、現行APIで確認できた
実例も1件だけである。補助取得元としては既存の `summary.cover` で十分と判断し、
SupportingResourceの収録率・優先順位調査と新規対応は行わない。

### 9.3 共通モデル

openBDでは `summary.cover` が空でない場合だけ、既存どおり表紙画像として設定する。
SupportingResourceの構造は共通 `BookSource` へ複製せず、今回の変換対象にも追加しない。

複数取得元をまとめる将来に `Image.Source` が必要かは、openBD対応とは分けて検討する。

## 10. MADBの性能

### 10.1 現行クエリの特徴

現行検索は、対象resourceを絞るサブクエリと、多数の `OPTIONAL` 項目取得を
1回のSPARQLに含める。

タイトル、creator、Agent、出版社、ISBNなどがそれぞれ複数値を持つと、
OPTIONALの結合により1冊あたりのbinding行数が増える可能性がある。著者読みとAgent URIを
同じ横長クエリへ追加すると、値の組み合わせがさらに増える可能性がある。

MADB provider価格は共通価格へ変換しない方針としたため、性能比較の対象から外す。
人物情報は、現行横長クエリへの追加と、resource確定後に縦長の別クエリで取得する方式を
比較する。

MADBは、推定実行時間が60秒を超えるクエリを拒否する場合があり、同一IPから短時間に
連続して呼び出すと一時的に利用できなくなる場合があると案内している。
性能測定自体が過剰な連続アクセスにならないようにする。

### 10.2 保存済みRaw responseの静的基準

2026年8月3日に保存したISBN参照応答から、現行クエリの構造的な基準を確認した。

| 問い合わせ例 | Rawバイト数 | binding件数 | distinct book件数 | 最大binding数/book |
| --- | ---: | ---: | ---: | ---: |
| `4592730933` | 1,852 | 1 | 1 | 1 |
| `9784098515189` | 3,234 | 2 | 2 | 1 |
| `4091400019` | 6,916 | 4 | 2 | 2 |

`4091400019` の例では、同じ書誌に複数creator表記とAgent名があり、1冊が複数bindingへ
展開されている。Agent URIと言語タグを横長に追加すると、さらに行数が増える可能性がある。

保存済みファイルには通信時間が記録されていないため、これはレスポンス構造とサイズだけの
基準であり、実サービスの性能基準値にはしない。

### 10.3 測定項目

次を1回ごとに記録する。

| 項目 | 内容 |
| --- | --- |
| 操作 | タイトル検索、著者検索、フリーワード検索、ISBN参照 |
| 条件 | 検索語、除外語、Limit、カーソルの有無 |
| 実行日時 | サービス側の時間帯差を確認できる時刻 |
| 試行番号 | 同条件を複数回行う場合の番号 |
| 経過時間 | リクエスト開始から本文読込・変換完了まで |
| HTTP結果 | ステータスまたは通信エラー |
| `ErrorKind` | ライブラリが返した分類 |
| Rawバイト数 | 受信本文の大きさ |
| binding件数 | SPARQLの行数 |
| distinct book件数 | 集約後の冊数 |
| 最大binding数 | 1resourceあたりの最大行数 |

### 10.4 比較するクエリ

1. 現行クエリ
2. 現行横長クエリへAgent URI、creator言語タグ、Agentラベル言語タグを追加する方式
3. resource取得後に人物情報だけを別クエリで取得する二段階方式
4. 人物情報を `kind`、`agent`、`value`、`lang` の縦長で返すUNION方式

同じ条件で比較し、少なくともタイトル、著者、フリーワード、ISBN参照を分ける。
呼び出し間隔を空け、少数回の測定から始める。

### 10.5 性能比較を行わない理由

今回の調査では、既知4リソースの縦長人物クエリを2回だけ実行した。前節のとおり、
記録できた2回目はHTTP 200、177 ms、4,430 bytes、10 bindingsだった。現行横長クエリ
（方式A）、人物詳細を追加した横長クエリ（方式B）、二段階取得全体（方式C）を、ISBN参照、
タイトル検索、著者検索、フリーワード検索の各条件で同一基準により測定してはいない。

方式BとCは今回の試料で安全な人物読みを生成できなかったため、性能だけを目的に追加の
連続アクセスを行わなかった。応答時間、Raw responseサイズ、binding増加の比較は、
値を安全に利用できることが確認された場合だけ行う。再調査時は、ISBN参照、タイトル検索、
著者検索、フリーワード検索ごとにLimit 3〜5のA・B・C測定を行う。

実測前に、著者読みとAgent URIを本番クエリへ追加しない。MADB provider価格は追加しない。

## 11. 調査結果から推奨する判断順

実装TODOを作成する前に、次の順序で設計判断を行う。

1. `SourceBookValues` を削除し、`BookSource` を取得元参照へ縮小する方針を確定する
2. Raw responseの返却単位と、`BookSource.ID`・`URL` から対応箇所を確認する方法を整理する
3. openBDをISBN参照用の補助取得元へ縮小し、既存誤変換だけを修正対象にする
4. MADBの人物情報を確認し、共通の人物表現を決める
   - 役割不明人物を `Contributors` に含めるか
   - `Reading` の項目名
   - `Authors []string` の位置付け
5. Series、Publisher collection、Imprintの境界を決める
6. MADBの人物読みを安全に追加できるか確認し、できない場合は実装しない結論を記録する
7. 共通モデルとopenBDの誤変換修正を独立した実装TODOへ分ける

## 12. 現時点の推奨案

### 共通モデルとRaw response

- `SourceBookValues` は初回リリース前に削除し、互換用フィールドを残さない
- `BookSource` は `Source`、`ID`、`URL` を持つ取得元参照として残す
- 完全な元データは、既存の `WithRawResponse` メソッドでリクエスト単位に返す
- Raw responseを各Bookへ埋め込まず、MADBのbindingを1冊分のRawとして再構成しない
- 安全に共通化できない値は `Book` へ不完全な形で複製しない
- 変換規則の根拠はRaw response、仕様書、テストケースの組で確認できるようにする
- 読みはNormalizedに残す
- `TitleKana` を `TitleReading` へ変更し、人物には `Contributor.Reading` を追加する
- `Authors` と読みの平行配列は作らない
- `Contributor` を全人物情報として使い、役割不明人物も保持する
- `Authors []string` は簡易一覧として残す
- 複数取得元統合時のフィールド単位出典は、`SourceBookValues` と別の問題として扱う

### openBD

- ISBN参照用の補助取得元として維持し、主要な正規化用データ取得元にはしない
- `TitleText.content` と同じ要素の `collationkey` を変更せず共通項目へ移す
- 区切り記号や末尾表記から並列タイトルと巻数を推測しない
- Contributor名と読みを変更、分割、注記削除せず、同じ要素の組として扱う
- `A36`、`A38`、`A46`、`A47` は未対応とし、専用の共通役割を追加しない
- `PublishingDateRole=11` を当該商品の `published` へ変換しない
- Collectionを `Series` または `Imprints` へ変換しない
- Imprintは発行元、Publisherは発売元として、どちらも出版社候補にする
- openBDのImprintをコミックレーベルとして共通 `Imprints` へ設定しない
- 価格、商品階層PartNumber、CollectionSequence、SupportingResourceは今回追加対応しない
- 書影は既存の `summary.cover` の範囲に限定する

### MADB

- creator文字列とAgent情報は変換中の非公開型で区別し、Raw responseから確認する
- 人物名と読みは、同じAgent URIに属することを確認できる場合だけNormalizedへ設定する
- creator文字列とAgent名を文字列一致、件数、配列位置、行順で対応付けない
- provider価格は所蔵情報に属するため共通 `Price` へ変換しない
- 人物読みを安全に追加できる実例が得られるまで、著者読み・Agent URIを本番クエリへ追加しない

## 13. 将来の再調査条件

将来の再調査条件は [`../backlog.md`](../backlog.md) へ移した。
022では、MADBの人物読みを実装しない結論を確定する。

実装は次のTODOへ分ける。MADB人物読みの実装TODOは作成しない。

- [`../todo/done/023-remove-source-book-values-and-revise-common-model.md`](../todo/done/023-remove-source-book-values-and-revise-common-model.md)
- [`../todo/024-remove-openbd-incorrect-mappings.md`](../todo/024-remove-openbd-incorrect-mappings.md)

openBDの価格、商品階層PartNumber、CollectionSequence、SupportingResourceなどは、
未完了調査ではなく、補助取得元へ縮小したため今回の対象外とした項目である。具体的な
利用要件が生じた場合に、別の調査TODOとして再開する。

## 14. 参照資料

### プロジェクト内

- [`../spec.md`](../spec.md)
- [`../pkg/madb/spec.md`](../pkg/madb/spec.md)
- [`../pkg/openbd/spec.md`](../pkg/openbd/spec.md)
- [`012-01-openbd-api.md`](012-01-openbd-api.md)
- [`012-05-book-source-and-normalized-model.md`](012-05-book-source-and-normalized-model.md)
- [`012-06-title-volume-edition-examples.md`](012-06-title-volume-edition-examples.md)
- [`007-madb-author-roles.md`](007-madb-author-roles.md)
- [`014-madb-search-conditions.md`](014-madb-search-conditions.md)

### 公式資料

- [openBD 利用ガイドライン](https://openbd.jp/terms/)
- [openBD 書誌APIデータ仕様](https://openbd.jp/spec/)
- [openBD JSON Schema](https://api.openbd.jp/v1/schema)
- [openBD API応答例: 9784780802047](https://api.openbd.jp/v1/get?isbn=978-4-7808-0204-7&pretty=)
- [openBD API（バージョン1）の提供終了について](https://openbd.jp/news/20230725.html)
- [openBD 代替書誌情報の提供開始について](https://openbd.jp/news/20230829.html)
- [ONIX Code List 5: Product identifier type](https://ns.editeur.org/onix/en/5)
- [ONIX Code List 17: Contributor role](https://ns.editeur.org/onix/en/17)
- [ONIX Code List 58: Price type](https://ns.editeur.org/onix/en/58)
- [ONIX Code List 148: Collection type](https://ns.editeur.org/onix/en/148)
- [ONIX Code List 149: Title element level](https://ns.editeur.org/onix/en/149)
- [ONIX Code List 158: Resource content type](https://ns.editeur.org/onix/en/158)
- [ONIX Code List 159: Resource mode](https://ns.editeur.org/onix/en/159)
- [ONIX Code List 161: Resource form](https://ns.editeur.org/onix/en/161)
- [MADBデータセットおよびSPARQLクエリサービスの利用方法](https://mediag.bunka.go.jp/madb_lab/lod/howto/)
- [MADBデータ構成・メタデータスキーマ解説](https://mediag.bunka.go.jp/madb_lab/about-db/schema-guide/)
- [MADB SPARQLクエリサービス](https://mediaarts-db.artmuseums.go.jp/sparql)

### 補足資料

- [文化・芸術とLOD ハンズオン: 2022年版MADBデータ利用例](https://lodc2022-culture-art.metadata.moe/docs/mediaartsdb/)
- [過去のopenBD CollectionSequence応答例](https://qiita.com/kanary/items/5ec45bbc01efd4388fdb)
