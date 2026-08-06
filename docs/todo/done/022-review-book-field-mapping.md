# 書誌項目の変換漏れと共通モデルを再検討する

状態: 完了（2026年8月6日）

## 背景

- `NormalizedBook` に項目がある一方で、MADBとopenBDの取得元データから値を設定していない項目がある
- Raw responseに存在する値を取得元固有の非公開型へ取り込んでいない場合と、非公開型には存在するが `Sources` または `Normalized` へ反映していない場合が混在している
- 著者名と読み、作品シリーズ、出版コレクション、レーベルは、現在の共通モデルでは取得元ごとの差を十分に表現できない可能性がある
- openBDの著者名には、複数人物、生年などの注記、姓名を区切るコンマが含まれる実例がある
- MADBは検索条件や時間帯によって応答が遅くなることがあり、取得項目の追加が性能へ与える影響を確認する必要がある

確認済みの主な例は次のとおり。

- MADB
  - 価格を取得・変換していない
  - 著者読みを取得していない
  - creatorとAgent名の両方がある場合、現行 `SourceBookValues` には一方しか残らない
- openBD
  - タイトル読みが取得できていても、現在の分解条件によって `Normalized.TitleKana` が空になる場合がある
  - Contributorの読み、元の役割コード、順序はRaw responseにあるが、現行 `SourceBookValues` では表現できない
  - ONIXの価格を取得・変換していない
  - 識別子の取得経路と種別はRaw responseにあるが、現行 `SourceBookValues` では表現できない
  - `summary.cover` は `Normalized.Images` へ設定済みで、元URLはRaw responseから確認できる
  - ONIXのCollectionに、作品シリーズではなくレーベルに相当する値が入る場合がある

## 目的

- MADBとopenBDについて、書誌値がどの段階で失われているかを項目ごとに明確にする
- `SourceBookValues` を維持する必要があるか、Raw responseで代替できるかを判断する
- `BookSource` と `Normalized` の責務を、取得元固有データを不完全に複製しない形で決める
- 著者名、著者読み、シリーズ、価格、識別子、画像の安全な変換方針を決める
- 調査結果を、独立して実施できる実装TODOへ分割できる状態にする

## 調査結果

静的なコード・仕様・保存済みRaw responseの確認結果と設計候補は、
[書誌項目の変換漏れと共通モデルの再調査](../research/022-book-field-mapping-review.md)にまとめる。

現時点では、項目対応、openBDで安全に行える変換の境界、共通モデル候補の整理まで
完了した。openBDは主要な正規化用データ取得元ではなく、ISBN参照用の補助取得元として
維持する。新しいONIX項目を網羅的に調査・実装せず、既存の明らかな誤変換だけを
後続TODOで修正する。

`SourceBookValues` は完全な元データにも変換後の共通データにもならず、不完全な複製に
なっている。今後はこれを拡張せず、廃止して `BookSource` を取得元参照へ縮小し、
完全な元データは既存の `WithRawResponse` メソッドで返す。
Raw responseを各Bookへ埋め込む案は採用しない。

MADBのcreator読み・Agent読みは現行実サービスで確認した。今回の既知3人物の試料では、
読みはAgent URIを持たない直接creatorリテラルにだけ存在し、Agentラベルは通常表記だけだった。
そのためMADBの人物読みは実装しない。人物読みを安全に追加できる値がないため、方式B・Cは
性能比較の対象ではなく今回の対象外とする。
MADB provider価格とopenBDの高度な項目調査は022の完了条件から外す。実装TODOは、
残るMADB調査後に作成する。

## 非目的

- このTODOではGoコードを変更しない
- このTODOでは公開型やJSON形式の変更を確定しない
- 取得元の値から、根拠のない人物の対応、役割、シリーズ種別を推測しない
- MADBやopenBDで取得できない書誌値を、タイトルなど別項目から補完しない

## 前提・制約

- Raw response、取得元固有の非公開型、`BookSource`、`NormalizedBook` を別の段階として扱う
- `BookSource` は取得元、取得元内ID、参照URLを示す参照情報とし、取得元固有データの縮小版にしない
- `SourceBookValues` へ新しい構造や項目を追加することを前提にしない
- 完全なレスポンス本文はRaw response用メソッドでリクエスト単位に扱う
- Raw response全体を各Bookへ重複して埋め込まない
- MADBのbindingを切り出し、1冊分のRaw responseとして再構成しない
- 著者名と著者読みを別々のスライスへ格納し、配列位置だけで対応付けない
- 取得元が人物名と読みの対応を示さない場合は、`Normalized` で対応を推測しない
- MADBの取得項目を増やす場合は、応答時間、レスポンス量、binding件数への影響を測定する

## 対象範囲

### 対象

- MADBとopenBDの項目対応表
- `SourceBookValues` の必要性、廃止時の影響、Raw responseによる代替範囲
- `BookSource.ID`・`URL` とRaw response内の書誌を対応付ける方法
- 著者名、寄与者、人物名の読み
- シリーズ、出版コレクション、レーベル
- 価格
- ISBNその他の識別子
- 書影と画像の元値
- MADBの著者読み・Agent URIを取得するSPARQL
- MADBの検索・ISBN参照の性能測定方法
- openBDの既存変換に含まれる明らかな意味の不一致

### 対象外

- Google Books、楽天Books、楽天Kobo、Yahoo!ショッピングの実装
- 複数取得元の検索結果統合
- 著者の同一人物判定
- 書影のダウンロード、保存、キャッシュ
- 自動リトライ、レート制限、永続キャッシュ
- openBDのONIX項目を網羅する追加調査
- openBDの価格、CollectionSequence、商品階層PartNumber、SupportingResourceの新規対応
- openBDを主要データ取得元として利用するための高度な正規化

## 暫定方針

### 項目対応の確認

各プロバイダについて、次の4段階を分けて一覧化する。

1. 取得元に存在する値
2. SPARQLまたはAPIのRaw responseに含まれる値
3. 取得元固有の非公開型へ読み込む値
4. 現行 `SourceBookValues` と `Normalized` へ変換する値

「取得していない」「非公開型へ取り込んでいない」「現行Sourceにだけある」
「Normalizedへ変換していない」を区別して記録する。ただし、現行Sourceの欠落を
すべて新しい公開型で補うことは前提にしない。

### Raw responseとBookSource

- `SourceBookValues` はRaw responseの代替になっているかを確認し、拡張より廃止を優先して比較する
- `BookSource` は `Source`、`ID`、`URL` の取得元参照へ縮小する
- openBDでは検証済みISBN-13、MADBではIDまたはリソースURIをRaw内の対応箇所を探す手掛かりにする
- 完全な取得元値が必要な利用者は `WithRawResponse` を使用する
- 将来の複数取得元統合に必要なフィールド単位出典は、SourceBookValuesとは別に検討する

### 著者名と読み

- タイトル読みと人物読みは、検索や表示順の補助情報として共通モデルに残す
- `Authors` と `AuthorReading` のような対応を保証できない平行配列は追加しない
- `Contributor` を役割の有無にかかわらず人物情報として使い、人物名と読みを同じ要素に保持する
- `Authors []string` は利用者向けの簡易一覧として残す
- `TitleKana` は `TitleReading` へ変更し、人物の読みは `Contributor.Reading` に保持する

### openBDのタイトルと人物情報

- `TitleText.content` と同じ要素の `collationkey` は変更せず共通項目へ移す
- 区切り記号から並列タイトルを推測せず、タイトル末尾から巻数を抽出しない
- 商品階層の `PartNumber` は、具体的な利用要件が生じるまで対応しない
- Collection階層の `PartNumber` とCollectionSequenceは作品巻数へ使用しない
- `PersonName.content` と同じ要素の `collationkey` を、名前と読みの組として扱う
- 人物名の分割、注記削除、コンマ・空白・中黒などの書換えは行わない
- `ContributorRole` は公式コードと共通役割の意味が一致する場合だけ変換する
- 現行の `A38` から `original_creator` への変換は意味が一致しないため見直す
- 元のONIXコードと表記はRaw responseから確認し、共通 `BookSource` へ複製しない
- 表示向けの人名・タイトル整形が必要な場合は、書誌変換とは別の処理として検討する

### シリーズ

- MADBの作品・関連巻のまとまりと、openBDのONIX Collectionを同じ意味として扱わない
- 共通モデルで、作品シリーズ、出版コレクション、レーベルを区別する必要があるか検討する
- openBDのCollectionは、意味を確定できるまで無条件に `Normalized.Series` へ入れない案を比較する
- 名称だけの辞書や末尾文字列だけで、シリーズとレーベルを断定しない

### 出版社と刊行日

- openBDの `Imprint` は発行元出版社、`Publisher` は発行元と異なる場合の発売元出版社として扱う
- openBDの `Imprint` をコミックレーベルとみなし、共通 `Imprints` へ変換しない
- 発行元と発売元はどちらも `Normalized.Publishers` の候補とする
- `PublishingDateRole=01` は当該商品の出版日として共通 `published` へ変換する
- `PublishingDateRole=11` は収録作品の初版刊行日であり、当該商品の `published` へ変換しない
- 日付はONIX `01`、`hanmoto.dateshuppan`、`summary.pubdate` の順に採用し、`11` は使用しない

### 価格、識別子、画像

- openBDの価格は今回新規対応せず、必要になった場合に別途調査する
- 識別子は対応済みの種別だけを `Normalized.Identifiers` へ変換し、元の種別コードと取得経路はRaw responseから確認する
- 書影は現在使用している `summary.cover` の範囲に限定し、SupportingResourceは今回対応しない
- 画像用途コードなどの取得元固有情報を共通 `BookSource` へ複製しない

### MADBの性能

- タイトル検索、著者検索、フリーワード検索、ISBN参照を分けて測定する
- 検索条件、Limit、処理時間、レスポンスバイト数、binding件数、取得冊数、エラー種別を記録する
- 取得項目追加前後を同じ条件で比較する
- 多数の `OPTIONAL` による複数値の組み合わせ増加を確認する
- 必要に応じて、resource取得と詳細取得の二段階化、または縦長形式の結果取得を比較する
- ライブラリ内での自動リトライは、この調査だけを理由に追加しない

## 実施項目

### SourceとRaw responseの境界

- [x] `SourceBookValues` のGoコード上の利用箇所を確認する
- [x] `SourceBookValues` の拡張、最小化、廃止を比較する
- [x] openBDとMADBについて、Raw responseを各Bookへ埋め込む案の問題を整理する
- [x] `BookSource` を `Source`、`ID`、`URL` の取得元参照へ縮小する案を整理する
- [x] 現行の `WithRawResponse` メソッドをリクエスト単位のRaw返却として維持する案を整理する
- [x] 初回リリース前のため、互換用フィールドを残さず `SourceBookValues` を削除する方針を確定する

### 項目対応表

- [x] MADBのRaw response、非公開型、`Sources`、`Normalized` の対応表を作る
- [x] openBDのRaw response、非公開型、`Sources`、`Normalized` の対応表を作る
- [x] 各欠落を、未取得、未読込、Source未保持、Normalized未変換に分類する
- [x] 現在の仕様で意図的に変換対象外としている項目を区別する

### タイトル、著者名、読み、役割

- [x] MADBのcreator読みとAgent読みの現行実データを試験SPARQLで確認する
- [x] MADBで人物名と読みを安全に対応付ける条件を、同一Agent URIに属する値だけに限定する
- [x] creator文字列とAgent名を文字列一致、件数、配列位置、行順で対応付けないと判断する
- [x] Agent URIと値種別・言語タグを返す縦長の試験クエリ案を作る
- [x] openBDの `PersonName.content` と `collationkey` の実例を追加収集する
- [x] openBDの人名を分割・書換え・注記削除しない方針を整理する
- [x] openBDの推測的なタイトル・並列タイトル・巻数分解を見直す方針を整理する
- [x] openBDの現行ContributorRole対応を公式コード表と照合する
- [x] `A38` から `original_creator` への変換が意味不一致であることを確認する
- [x] openBDで初期対応するContributorRoleの推奨範囲を整理する
- [x] `A36`、`A38`、`A46`、`A47` は当面未対応とし、共通役割を追加しない方針を整理する
- [x] 人物名と読みを保持する共通型の候補を整理する

### シリーズ

- [x] MADBの `ma:seriesName` と参照先MangaBookSeriesの意味を整理する
- [x] openBDのCollectionType、TitleElementLevel、CollectionSequence、PartNumberの公式定義を確認する
- [x] Collectionの `PartNumber` と `summary.volume` が作品巻数と一致しない実例を確認する
- [x] openBDの商品階層 `PartNumber` とCollectionSequenceは今回対応しないと判断する
- [x] 作品シリーズ、出版コレクション、レーベルの境界と公開型候補を整理する

### 出版社と刊行日

- [x] openBDのImprintとPublisherの公式な意味を確認する
- [x] openBDのImprintを共通 `Imprints` へ変換しない方針を整理する
- [x] `PublishingDateRole=01` と `11` の公式な意味を確認する
- [x] `11` を当該商品の `published` へ丸めない方針を整理する
- [x] ONIX `01`、`hanmoto.dateshuppan`、`summary.pubdate` の順に採用する方針を整理する
- [x] openBD用に発行元・発売元を区別する共通出版社型は追加しないと判断する
- [x] openBD用に作品の初版刊行日を表す共通日付種別は追加しないと判断する

### 価格、識別子、画像

- [x] MADBのprovider配下の価格は所蔵情報に属し、共通 `Price` へ変換しないと判断する
- [x] MADB provider価格は今回取得せず、必要時は所蔵情報モデルとして別途調査すると整理する
- [x] openBDの価格は今回新規対応しないと判断する
- [x] openBDのProductIdentifierによるISBN応答検証を維持すると判断する
- [x] openBDの書影は `summary.cover` の既存対応に限定し、SupportingResourceは今回対応しないと判断する

### MADBの性能

- [x] 現行クエリの測定条件と記録形式を決める
- [x] 保存済みRaw responseからbinding件数とレスポンスサイズの静的基準を整理する
- [x] 既知実例で同一Agent URI内の通常表記・読み、またはcreatorとAgentの明示的な対応がないことを確認する
- [x] 安全に追加できる人物読みがないため、A・B・Cの性能比較を今回の対象外とする
- [x] 現行の試料では方式Aを維持し、方式B・Cを実装候補にしないと判断する

### 次のTODO

- [x] [`023-remove-source-book-values-and-revise-common-model.md`](023-remove-source-book-values-and-revise-common-model.md) を作成する
- [x] [`024-remove-openbd-incorrect-mappings.md`](024-remove-openbd-incorrect-mappings.md) を作成する
- [x] MADBの人物読み・Agent情報用の実装TODOは作成しない（安全に追加できる値がない）
- [x] 確定した動作だけを後続TODOの変更対象として整理する

## 受け入れ条件

- [x] MADBとopenBDの各対象項目について、値が失われる段階を説明できる
- [x] `SourceBookValues` が不完全な元データ複製になる問題を説明できる
- [x] Raw responseを各Bookへ埋め込まず、リクエスト単位で返す理由が整理されている
- [x] `BookSource` を取得元参照へ縮小し、安全に変換できる値だけを `Normalized` へ入れる案がある
- [x] openBDで行う明示的な構造変換と、行わない推測的な書換えの境界が整理されている
- [x] 著者名と読みを、対応を推測せず保持できる型の候補が比較されている
- [x] openBDの現行ContributorRole変換に意味不一致があることを説明できる
- [x] シリーズ、出版コレクション、レーベルの扱いに推奨案と代替案がある
- [x] 価格、識別子、画像について、Raw responseと共通値の境界が整理されている
- [x] MADBの人物読みは安全な値がないため実装しないと判断できる
- [x] 実装作業を、範囲と受け入れ条件を持つ2件のTODOへ分割している

## リスク・懸念

- 共通公開型の変更は、JSON形式と既存利用コードへ影響する可能性がある
- 著者読みと人物名の対応を推測すると、別人物の読みを誤って結び付ける可能性がある
- openBDのCollectionをレーベルまたは作品シリーズへ誤分類する可能性がある
- ONIXコードの一部だけを理解した状態で価格や役割を変換すると、意味を誤る可能性がある
- MADBの取得項目追加によってbinding数と応答時間が大きく増える可能性がある

## 結論

- MADBの人物読みは現時点では実装しない
- 直接 `schema:creator` の読みはAgent URIを持たず、人物との対応を確定できない
- Agentラベルには通常表記しか確認できず、同一Agent URI内で名前と読みを組にできない
- creatorとAgentを文字列、件数、配列位置、行順で対応付けない
- 現行クエリ方式を維持する
- 方式B・Cは性能上の比較対象として不採用にしたのではなく、安全に追加できる値が存在しないため今回の対象から外す
- 再調査条件は [`../backlog.md`](../backlog.md) で管理する

次は方針確定済みとする。

- `BookSource.Source` は必須とし、`ID` と `URL` は取得元で利用できる場合だけ設定する
- `Contributor` は役割不明の人物にも使用し、人物名、読み、既知役割を同じ要素に保持する
- `Authors []string` は利用者向けの簡易一覧として残す
- `TitleKana` は初回リリース前に `TitleReading` へ変更する
- 人物の読みは `Contributor.Reading` に保持し、平行配列を追加しない
- MADBの作品系列は既存 `Series` を使う
- openBDのCollection用に共通の出版Collection型を追加しない
- 共通 `Imprints` は取得元がレーベルまたはブランドと明示した値だけに使う

- `SourceBookValues` は初回リリース前に削除する
- openBDの `parseSourceTitle` と `normalizeTitleKana` は廃止し、タイトル全体と読み全体を採用する
- openBDの商品階層 `PartNumber` は使用しない
- openBDの `A36`、`A38`、`A46`、`A47` は未対応とし、専用の共通役割を追加しない
- openBDのCollectionは `Normalized.Series` へ変換しない
- 将来の横断検索におけるRaw応答集合型は022の対象外とする

## AIへの入力メモ

- このTODOは完了した調査と設計の記録であり、実装は023または024に従う
- 各論点について、推奨案、代替案、利点、欠点、互換性への影響を整理する
- `SourceBookValues` の不足を新しい公開フィールド追加だけで解決しようとしない
- Raw responseに存在することと、現在のSPARQLまたは非公開型で取得していることを混同しない
- Raw responseをBook単位へ切り出す場合、それが無加工データではなく再構成データになるか確認する
- コード変更は、調査結果から作成した後続TODOで行う
