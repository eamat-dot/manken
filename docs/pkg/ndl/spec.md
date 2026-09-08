# NDLパッケージ仕様

`github.com/eamat-dot/manken/ndl` は、国立国会図書館サーチのSRU APIから完成済み全国書誌の図書を書誌検索・ISBN参照する。

`NewClient(nil)` は認証情報なしで利用できる。検索は SRU 1.2、`dcndl_v3`、`recordPacking=xml`、`onlyBib=true` を使用し、`dpid="iss-ndl-opac-national"` と `mediatype="books"` を固定する。

通常検索は漫画候補へ寄せるため、既定で`ndc="726.1"`と`ndlc="Y84"`をAND条件として追加する。両条件は分類の付与差により古い漫画等を取りこぼすため、`WithMangaNDCFilter(false)`と`WithMangaNDLCFilter(false)`で個別に無効化できる。Option未指定時は両方とも有効である。これらは漫画だけを完全保証する条件ではなく、`dcndl:genre=漫画`による取得後除外は行わない。

`SearchBooks` は `Title`、`Author`、`Publisher`、`Query` をそれぞれ `title`、`creator`、`publisher`、`anywhere` のCQL条件へ変換してAND結合する。`DateFrom` と `DateTo` は出版年月日の `from` と `until` へ変換する。日付だけの検索と片側指定を受け付け、共通日付仕様に従う。少なくとも一つが必要で、`Exclude` は未対応の入力エラーである。値は常にCQL文字列として引用・エスケープする。

NDL固有条件を使う場合は`SearchOptions`を`SearchBooksWithOptions`または`SearchBooksWithOptionsAndRawResponse`へ渡す。`Subject` と `Description` はそれぞれSRUの`subject` と `description`へ変換する。固有条件だけでも検索できる。`Subject`は資料の主題・件名を検索する条件であり、漫画ジャンル判定には使用しない。`Description`は内容記述系の検索インデックスを対象とし、ヒット語を含む「要約等」の本文がSRU `dcndl_v3` Rawへ返ることは保証しない。

Limitの既定値は20、範囲は1から500である。Cursorは検索条件と実効Limitに結び付く不透明な値で、501件目以降を要求するCursorは返さない。通常検索では`sortBy=issued_date/sort.ascending`を固定で指定し、刊行日の古い順でNDLが返した順序をそのまま保持する。楽天Books / 楽天Koboの発売日の古い順という固定既定と検索結果の考え方を揃えている。巻数などのクライアント側ソートは行わず、利用者がソートキーや昇順・降順を指定するAPIは提供しない。

`LookupBooksByISBN` はISBNを一つだけ受け付ける。ISBN参照にはNDC / NDLC漫画フィルタを適用しない。該当なしの`Record does not exist` Diagnosticsは、非nilの空のBooksで返す。返却書誌は、取得元が実際に返したISBNが指定ISBNと同一書籍である場合だけ返す。空の結果は、選択したNDLデータセットに一致するISBNが索引化されていないことを表す。特に古い刊行物では、書誌が存在してもISBNが記録されていない場合があるため、書誌自体が存在しないことを意味しない。タイトル・著者検索への自動fallbackは行わない。

成功時の`SearchBooksWithRawResponse`、`SearchBooksWithOptionsAndRawResponse`、`LookupBooksByISBNWithRawResponse`は、受信したXMLを無加工で返す。XML本文はSRUラッパーの`recordSchema`値ではなく、DC-NDL v3の`BibResource`構造から解析する。

`Attributions()` は外部通信を行わず、`service`、`data` の順で2件のクレジット情報を返す。成功した検索・ISBN参照では、該当書籍が0件でも同じ値を結果の `Attributions` に設定する。

- `service`: `国立国会図書館サーチAPIを利用`、参照先 `https://ndlsearch.ndl.go.jp/`、条件確認先 `https://ndlsearch.ndl.go.jp/help/api`
- `data`: `国立国会図書館全国書誌情報（国立国会図書館）をもとに、mankenで共通書誌形式へ変換して作成`、参照先 `https://ndlsearch.ndl.go.jp/`、ライセンス `CC BY 4.0`、ライセンス参照先 `https://creativecommons.org/licenses/by/4.0/`、条件確認先 `https://ndlsearch.ndl.go.jp/help/api/provider`

`service` はNDLサーチAPIを利用していること自体の表示、`data` は固定の `dpid="iss-ndl-opac-national"` で取得する全国書誌情報の出典・利用条件を表す。返却するクレジット情報だけで利用条件への適合を保証せず、利用時点の公式条件を確認する。

変換する値は主タイトル・読み、巻、叢書・出版シリーズ、版、著者、寄与者、出版者、取得元ISBN、出版日、言語、安全に識別できる分類、単純なページ数・大きさ、書誌価格、図書の紙媒体、NDLBibIDと書誌URLである。`dcterms:extent`は先頭の`NNNp`または`NNN p`を`PageCount`へ変換し、セミコロン以降の単純な`NNNcm`または`NNN cm`を取得元表記のまま`Size`へ設定する。付属資料、複数寸法、複雑な寸法は設定しない。`dcndl:price`は前後空白を除いたうえでASCII数字の直後に任意空白と`円`が続く単純形式だけを`Source: ndl`、`Currency: JPY`の`ListPrice`へ変換する。数字途中の空白は許容しない。同じ価格の重複は許容するが異なる有効価格がある場合は設定しない。税込・税別と観測時刻は設定せず、`CurrentPrice`にも設定しない。`dcndl:seriesTitle/rdf:Description/rdf:value`は空文字列と完全一致の重複を除き、最初の出現順を保って`PublicationSeries`へ設定する。`dcndl:transcription`は共通Bookへ変換しない。`dc:creator`の直接の子要素は、末尾が既知の役割表記である場合だけ表示名と役割へ分ける。役割表記はそのまま、または全体をASCII角括弧`[...]`で囲んだ形を受け付ける。安全に分けられた値は`Contributors`へ応答順で入り、著者・原作・脚本・作画・キャラクター原作・キャラクターデザインの役割を持つ表示名だけが`Authors`へ入る。同じ表示名と役割は重複を除き、順序は変えない。監修、編集、翻訳、解説、デザインは`Contributors`だけへ入る。

`dcndl:volume` と edition はTitle解析より優先する。各値が欠落する場合だけ、Titleを非破壊で保持したまま、安全なタイトル構文からVolume、Editions、IsFinalVolumeを補う場合がある。

利用可能な`dc:creator`が一つもない場合だけ、`dcterms:creator/foaf:Agent/foaf:name`を`Authors`へ使う。典拠形、読み、典拠URI、未知の役割を推測して表示名や寄与者へ結び付けない。取得元にない値、JPNO、所蔵・個体情報、未対応の寄与者役割は共通モデルへ追加しない。
