# MADBのcreator役割と寄与者変換規則

## 1. 調査目的

MADBのマンガ単行本に記録された `schema:creator` と `dcterms:creator` を調べ、
`Normalized.Authors`、`Normalized.Contributors`、
`Sources[0].Values.Authors` へ安全に変換できる範囲を確認する。

調査日は2026年8月1日。件数は同日時点の公式SPARQL Query Serviceの応答であり、
データ更新によって変わる可能性がある。

## 2. 確認方法

マンガ単行本は `class:MangaBook` に限定した。creator役割の集計では、
言語タグなしの `schema:creator` の先頭にある `[役割]` を抽出し、役割、件数、
代表リソースを取得した。creatorとAgentの有無は、各リソースに対する
`schema:creator` と `dcterms:creator` の存在を別々に集計した。

```sparql
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX schema: <https://schema.org/>
PREFIX class: <https://mediaarts-db.artmuseums.go.jp/data/class#>

SELECT ?role (COUNT(?creator) AS ?count) (SAMPLE(?resource) AS ?example)
WHERE {
  ?resource rdf:type class:MangaBook ; schema:creator ?creator .
  FILTER(LANG(?creator) = "")
  BIND(
    IF(
      REGEX(STR(?creator), "^\\[[^\\]]+\\]"),
      REPLACE(STR(?creator), "^\\[([^\\]]+)\\].*$", "$1"),
      "(役割なし)"
    ) AS ?role
  )
}
GROUP BY ?role
ORDER BY DESC(?count) ?role
```

```sparql
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX schema: <https://schema.org/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX class: <https://mediaarts-db.artmuseums.go.jp/data/class#>

SELECT ?hasCreator ?hasAgent (COUNT(?resource) AS ?count)
       (SAMPLE(?resource) AS ?example)
WHERE {
  ?resource rdf:type class:MangaBook .
  BIND(EXISTS {
    ?resource schema:creator ?creator .
    FILTER(LANG(?creator) = "")
  } AS ?hasCreator)
  BIND(EXISTS { ?resource dcterms:creator ?agent } AS ?hasAgent)
}
GROUP BY ?hasCreator ?hasAgent
ORDER BY ?hasCreator ?hasAgent
```

## 3. 調査結果

### 3.1 creatorとAgentの有無

| creator文字列 | Agent参照 | 単行本数 | 代表MADB ID |
| --- | --- | ---: | --- |
| なし | なし | 13,575 | `M521412` |
| なし | あり | 182 | `M830542` |
| あり | なし | 10,843 | `M521385` |
| あり | あり | 376,711 | `M526224` |

`M830542` はcreator文字列がなく、Agent名として `KotzDean`、`ZubJim`、
`皆川由美` を持つ。`M521385` はAgent参照がなく、creator文字列
`[著]前田悠` だけを持つ。どちらか一方だけを前提にできない。

### 3.2 上位の先頭役割

次の件数は単行本数ではなく、言語タグなしcreator文字列の件数である。

| 先頭役割 | 件数 | 代表MADB ID |
| --- | ---: | --- |
| `著` | 278,499 | `M521661` |
| 役割なし | 65,430 | `M850396` |
| `原作` | 60,746 | `M521662` |
| `漫画` | 31,623 | `M521639` |
| `作画` | 16,393 | `M521662` |
| `作` | 10,811 | `M521631` |
| `画` | 10,763 | `M521631` |
| `キャラクター原案` | 8,931 | `M521620` |
| `編` | 7,340 | `M521452` |
| `監修` | 4,252 | `M521369` |
| `原案` | 2,928 | `M521639` |
| `訳` | 2,808 | `M521616` |
| `劇画` | 2,246 | `M521379` |
| `まんが` | 2,155 | `M521611` |
| `脚本` | 2,130 | `M521725` |
| `[著` | 1,626 | `M809985` |
| `カバーデザイン` | 1,435 | `M202021` |
| `絵` | 1,386 | `M817916` |
| `ほか著` | 1,316 | `M521752` |
| `装丁` | 1,153 | `M198096` |
| `シナリオ` | 1,115 | `M521611` |
| `キャラクターデザイン` | 1,070 | `M521666` |
| `構成` | 1,067 | `M521655` |
| `脚色` | 878 | `M521699` |
| `装幀` | 714 | `M198095` |
| `原作・監修` | 701 | `M521623` |
| `著者` | 662 | `M200372` |
| `comic` | 658 | `M226676` |
| `編集` | 584 | `M810371` |
| `解説` | 522 | `M531437` |

上位以外にも複合役割、英字表記、旧字体、具体的な作業名、人名と見られる値がある。
固定一覧だけで全creatorを正確に分類することはできない。

### 3.3 creator文字列とAgentの対応

`M292132` は、次のcreator文字列とAgent名を持つ。

```text
[著]佐々木倫子
[解説]藤原新也
佐々木倫子
藤原新也
```

SPARQLで両方をOPTIONAL取得すると2件ずつの直積になる。RDF上には、
`[著]佐々木倫子` と佐々木倫子のAgent、`[解説]藤原新也` と藤原新也のAgentを
結び付ける辺がない。名前の完全一致で対応できる例はあるが、表記差や異体字が
ある場合まで一般化できない。

### 3.4 角括弧は複数役割だけを表さない

角括弧が続く実データには、2つ目以降が人名の一部である例がある。

| MADB ID | creator文字列 |
| --- | --- |
| `M196958` | `[作画][田辺節雄]` |
| `M208098` | `[作][ジョン・ブッセマン]` |
| `M208454` | `[画][葛飾]北斎` |
| `M212780` | `[監修][指導]奥山修平` |
| `M213006` | `[原作][スタン・リー]` |
| `M809985` | `[[著]]近江のこ` |

現在の `removeCreatorRoles` は先頭が `[` である間、最初の `]` までを
繰り返し削除する。この規則では `[作画][田辺節雄]` などの名前全体を削除し、
`[画][葛飾]北斎` を `北斎` に変える。角括弧の個数だけでは役割と人名を
区別できない。

### 3.5 順序

`schema:creator` と `dcterms:creator` はRDF上の順序を持たない。SPARQLの応答順も
表示順を表さないため、MADBから原作者、作画担当などの意味のある表示順を
復元できない。安定化のための文字列ソートは可能だが、取得元の表示順ではない。

## 4. 実装前に必要な判断

### 4.1 推奨案: creator文字列を保守的に解析する

- 言語タグなしcreator文字列がある場合、`Sources[0].Values.Authors` には
  役割表記を含むcreator文字列を変更せず残す
- creator文字列がない場合だけ、Agent名を取得元の著者表示として残す
- 明示的に対応を決めた先頭役割だけを共通 `ContributorRole` へ変換する
- 役割を確定できたcreator文字列は、その文字列内の名前と役割から
  `Contributor` を作る。Agentとの対応を推測しない
- 未知の役割、壊れた角括弧、名前を確定できない値は正規化せず、取得元値だけに残す
- 役割なしcreatorとAgentだけのレコードは、名前を `Authors` に残すが、
  根拠のない役割を持つ `Contributor` は作らない
- MADBには意味のある順序がないことを仕様へ明記し、安定化方法と表示優先度を
  混同しない

この案は `[解説]` を著者から除外でき、creatorとAgentの対応を推測しない。
一方、明示的な対応一覧へ入っていない役割は `Contributors` に現れない。

### 4.2 代替案A: 全creator名をAuthorsへ残す

全creatorから役割表記だけを除き、現在と同様に全員を `Authors` へ入れる。
後方の情報量は保てるが、解説者、装丁者、監修者などが著者として残り、
TODO007の主目的を満たさない。

### 4.3 代替案B: Agent名との文字列一致を使う

creatorから抽出した名前とAgent名が完全一致する場合だけ対応付ける。
一致したケースではAgentの参照を利用できるが、空白、異体字、読み、別名などの
差で対応結果が変わる。Agent参照は公開モデルへ保持していないため、現時点では
複雑さに対する利点が小さい。

### 4.4 共通役割語彙

共通役割語彙はMADBの文字列をそのまま公開せず、他の取得元にも適用できる名称にする。
候補は `author`、`original_creator`、`writer`、`artist`、`editor`、
`translator`、`supervisor`、`commentator`、`designer` などである。

出版分野にはONIX Code List 17という確立された寄与者役割体系があり、著者、作画、
編集、翻訳、解説、カバーデザインなどを区別している。ただし、共通APIでONIXコードを
そのまま公開するか、読みやすい一般名へ対応付けるかは公開API仕様の判断になる。

- [ONIX Code List 17: Contributor role code](https://ns.editeur.org/onix/en/17)

## 5. 確定事項

2026年8月1日に推奨案を採用した。

- `ContributorRole` はONIXコードではなく、取得元に依存しない一般名を公開する
- 初期役割は `author`、`original_creator`、`writer`、`artist`、
  `character_creator`、`character_designer`、`editor`、`translator`、
  `supervisor`、`commentator`、`designer` とする
- creator文字列をAgent名より優先して取得元値と変換根拠にする
- 明示的な対応一覧にある役割だけを正規化する
- 著者、原作者、構成または脚本担当、作画担当、キャラクター原案、
  キャラクターデザインを `Authors` に含める
- その他の既知役割は `Contributors` だけに含める
- creatorの順序を復元せず、Goの文字列昇順は結果の安定化だけに使用する

確定した共通仕様は [`../spec.md`](../spec.md)、MADB固有の変換規則は
[`../pkg/madb/spec.md`](../pkg/madb/spec.md) に記載した。

## 6. 結論

creator文字列を使えば、`M292132` の佐々木倫子と藤原新也を役割に基づいて
区別できる。ただし、角括弧の反復除去、Agent優先、取得元順序の維持という
現在の前提は実データに適用できない。

公開役割と順序の扱いを確定したため、実装作業は
[`../todo/done/013-implement-madb-author-roles.md`](../todo/done/013-implement-madb-author-roles.md)
に分離する。
