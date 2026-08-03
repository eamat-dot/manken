# openBD API 調査メモ

## 1. 目的

ISBN参照のみ対応のopenBDをデータ取得元として追加する前に、取得できる
レスポンス構造と、既存の共通モデルにない項目を確認する。

この文書は 2026-07-30 時点の調査結果であり、確定仕様ではない。
実装時に確定した契約と代替APIへの移行状況は
[`../pkg/openbd/spec.md`](../pkg/openbd/spec.md)を参照する。

## 2. API の基本情報

- 公式サイト: https://openbd.jp/
- 仕様ページ: https://openbd.jp/spec/
- スキーマ: https://api.openbd.jp/v1/schema?pretty
- 取得エンドポイント: https://api.openbd.jp/v1/get?isbn=ISBN
- 複数 ISBN の取得: `isbn=ISBN1,ISBN2` 形式

### 重要な制約

- 取得対象はISBNによる書誌参照
- 検索語によるタイトル検索は想定されない
- 1 リクエストで複数 ISBN をまとめて問い合わせ可能
- GETでは最大1,000件をカンマ区切りで指定できる
- レスポンスは配列形式で返る
- レスポンスは入力ISBNと同じ順序で、未収録の位置を `null` として返す

## 3. レスポンス構造の概要

openBD のレスポンスは、主に次の 3 つのセクションで構成される。

- `onix`: JPRO-ONIX 準拠の書誌情報
- `hanmoto`: 版元ドットコム独自・会員社拡張情報
- `summary`: 主要項目の簡易要約

## 4. 取得可能な項目一覧

### 4.1 `summary`

簡易要約として返される項目。

- `isbn`
- `title`
- `volume`
- `series`
- `author`
- `publisher`
- `pubdate`
- `cover`

### 4.2 `onix`

#### 識別情報

- `RecordReference`
- `NotificationType`
- `ProductIdentifier.ProductIDType`
- `ProductIdentifier.IDValue`

#### 書誌情報

- `DescriptiveDetail.ProductComposition`
- `DescriptiveDetail.ProductForm`
- `DescriptiveDetail.ProductFormDetail`
- `DescriptiveDetail.EditionType`
- `DescriptiveDetail.EditionStatement`

#### 言語情報

- `DescriptiveDetail.Language[]`
  - `LanguageCode`
  - `LanguageRole`
  - `CountryCode`

#### タイトル情報

- `DescriptiveDetail.TitleDetail.TitleType`
- `DescriptiveDetail.TitleDetail.TitleElement`
  - `TitleElementLevel`
  - `TitleText.collationkey`
  - `TitleText.content`
  - `Subtitle.collationkey`
  - `Subtitle.content`
  - `PartNumber`

#### シリーズ情報

- `DescriptiveDetail.Collection`
  - `CollectionType`
  - `CollectionSequence`
  - `CollectionSequenceArray[]`
  - `TitleDetail.TitleElement[]`

#### 物理情報

- `DescriptiveDetail.Extent[]`
  - `ExtentType`
  - `ExtentUnit`
  - `ExtentValue`
- `DescriptiveDetail.Measure[]`
  - `MeasureType`
  - `MeasureUnitCode`
  - `Measurement`

#### 著者情報

- `DescriptiveDetail.Contributor[]`
  - `SequenceNumber`
  - `ContributorRole[]`
  - `PersonName.collationkey`
  - `PersonName.content`
  - `BiographicalNote`

#### 主題・読者情報

- `DescriptiveDetail.Subject[]`
  - `SubjectSchemeIdentifier`
  - `SubjectCode`
  - `MainSubject`
  - `SubjectHeadingText`
  - `SubjectSchemeVersion`
  - `sourcename`
- `DescriptiveDetail.Audience[]`
  - `AudienceCodeType`
  - `AudienceCodeValue`

#### 付録情報

- `DescriptiveDetail.ProductPart[]`
  - `NumberOfItemsOfThisForm`
  - `ProductForm`
  - `ProductFormDescription`

#### 旧版情報

- `RelatedMaterial[]`
  - `RelatedProduct.ProductRelationCode`
  - `RelatedProduct.ProductIdentifier.ProductIDType`
  - `RelatedProduct.ProductIdentifier.IDValue`

#### 付帯情報

- `CollateralDetail.TextContent[]`
  - `TextType`
  - `ContentAudience`
  - `Text`
- `CollateralDetail.SupportingResource[]`
  - `ResourceContentType`
  - `ResourceMode`
  - `ContentAudience`
  - `ResourceVersion[]`
    - `ResourceLink`
    - `ResourceForm`
    - `ResourceVersionFeature[]`
      - `ResourceVersionFeatureType`
      - `FeatureValue`

#### 出版情報

- `PublishingDetail.Imprint.ImprintName`
- `PublishingDetail.Imprint.ImprintIdentifier[]`
  - `ImprintIDType`
  - `IDValue`
- `PublishingDetail.Publisher.PublisherName`
- `PublishingDetail.Publisher.PublisherIdentifier[]`
  - `PublisherIDType`
  - `IDValue`
- `PublishingDetail.PublishingDate[]`
  - `PublishingDateRole`
  - `Date`

#### 供給情報

- `ProductSupply.MarketPublishingDetail.MarketPublishingStatus`
- `ProductSupply.MarketPublishingDetail.MarketPublishingStatusNote`
- `ProductSupply.MarketPublishingDetail.PublisherRepresentative[]`
  - `AgentRole`
  - `AgentName`
  - `AgentIdentifier[]`
- `ProductSupply.SupplyDetail.ReturnsConditions`
- `ProductSupply.SupplyDetail.ProductAvailability`
- `ProductSupply.SupplyDetail.Price[]`
  - `PriceType`
  - `PriceAmount`
  - `CurrencyCode`
  - `PriceDate[]`

### 4.3 `hanmoto`

版元ドットコム独自・会員社拡張情報として、かなり多くの項目が入る。

#### 基本日付・更新情報

- `datekoukai`
- `datemodified`
- `datecreated`
- `dateshuppan`

#### 在庫・販売情報

- `kubunhanbai`
- `toji`
- `zaiko`
- `han`
- `hatsubai`
- `hatsubaiyomi`

#### 分類・コード

- `genrecodetrc`
- `genrecodetrcjidou`
- `ndccode`
- `kankoukeitai`
- `furoku`
- `zasshicode`
- `gatsugougousuu`

#### テキスト・説明

- `genshomei`
- `maegakinado`
- `hanmotokarahitokoto`
- `kaisetsu105w`
- `tsuiki`
- `obinaiyou`
- `kanrensho`
- `ruishokyougousho`
- `bessoushiryou`
- `sonotatokkijikou`
- `jushoujouhou`
- `dokushakakikomi`
- `dokushakakikomipagesuu`
- `bikoutrc`
- `bikoujpo`
- `furokusonota`
- `kanrenshoisbn`

#### フラグ・状態

- `lanove`
- `hastameshiyomi`
- `rubynoumu`
- `hankeidokuji`
- `hankeisonota`

#### 日付系の追加情報

- `datezeppan`
- `datejuuhanyotei`

#### 追加の配列情報

- `jyuhan[]`
  - `date`
  - `suri`
  - `comment`
  - `ctime`
- `author[]`
  - `listseq`
  - `dokujikubun`
- `reviews[]`
  - `appearance`
  - `reviewer`
  - `source_id`
  - `kubun_id`
  - `source`
  - `choyukan`
  - `han`
  - `link`
  - `post_user`
  - `gou`

#### 版元情報

- `hanmotoinfo.name`
- `hanmotoinfo.yomi`
- `hanmotoinfo.url`
- `hanmotoinfo.twitter`
- `hanmotoinfo.facebook`
- `hanmotoinfo.eigyoudaihyousha`
- `hanmotoinfo.toritsugisonota`
- `hanmotoinfo.toritsugitorikyo`
- `hanmotoinfo.phoneshoten`
- `hanmotoinfo.facsimileshoten`
- `hanmotoinfo.ordersitejisha`
- `hanmotoinfo.emailshoten`
- `hanmotoinfo.ordersitesonota`
- `hanmotoinfo.ordersite`
- `hanmotoinfo.henpin`
- `hanmotoinfo.chokutori`
- `hanmotoinfo.jiyuukinyuu`
- `hanmotoinfo.shiiresite`

#### その他

- `storelink`

## 5. 既存の共通モデル [api/model.go](../../api/model.go) との比較

現状の共通モデルは、主に次の項目を持つ。

- `ID`
- `Titles`
- `Subtitles`
- `SeriesNames`
- `SeriesID`
- `SeriesURL`
- `VolumeNumber`
- `EditionStatements`
- `Authors`
- `Publishers`
- `Imprints`
- `ISBN10s`
- `ISBN13s`
- `PublishedDate`
- `Source`
- `SourceURL`

openBD で追加しやすい項目は、次のようなものがある。

- `subtitle` や `title` の別表現
- `description` / `summary` / `contents`
- `cover` 画像 URL
- `publisher` / `imprint` の詳細情報
- `publication date` の複数バージョン
- `price` 情報
- `stock` / `availability`
- `review` 情報
- `series` / `collection` 情報
- `subject` / `genre` / `audience` 情報

## 6. 実装時の注意点

- いくつかの項目は空文字や空配列、または欠落することがある
- `onix` と `hanmoto` には同じ概念でも別名の項目がある
- 既存の共通モデルにそのまま落とし込むには、項目ごとの正規化が必要
- ISBN参照限定なので、タイトル検索や作者検索は別途設計が必要
- `summary.volume` はCollection階層の `PartNumber` を返す場合があり、
  書籍の巻数として使用できない
- `summary.volume` の調査結果は
  [`012-06-title-volume-edition-examples.md`](012-06-title-volume-edition-examples.md#12-openbdのcollection-partnumberとsummaryvolume)
  に記録する
- 正規化に使用した主要な元値は文字列を変更せず `Sources` に残し、
  完全な成功応答本文はRaw response用メソッドで返す
