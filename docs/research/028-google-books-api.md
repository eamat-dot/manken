# Google Books API現行仕様とmankenでの利用範囲調査

## 1. 目的

Google Books API v1の現行仕様を確認し、`manken` でタイトル・ISBN・著者検索、
書誌情報取得、Raw responseを提供するプロバイダとして利用する範囲を整理する。

この文書は2026年8月7日時点の調査結果であり、確定仕様や実装指示ではない。
TODO029で確定した現在仕様は [Google Booksパッケージ仕様](../pkg/googlebooks/spec.md) を正とし、
本書の後半には実装時に確定した判断と実サービス確認結果を追記している。

## 2. 調査方法と確認水準

次の情報を区別して確認した。

1. 2026年8月7日に確認したGoogle公式ドキュメント
2. 2026年7月31日にRESEARCH012系で実APIへ問い合わせた結果
3. `__sample/book-api` にある既存のGoogle Booksクライアント
4. 2026年8月7日のChatGPT調査環境からの公開API再実行

主な公式資料:

- Using the API: https://developers.google.com/books/docs/v1/using
- Volume resource: https://developers.google.com/books/docs/v1/reference/volumes
- Volumes list: https://developers.google.com/books/docs/v1/reference/volumes/list
- Volumes get: https://developers.google.com/books/docs/v1/reference/volumes/get
- Books API Terms of Service: https://developers.google.com/books/terms
- Branding Guidelines: https://developers.google.com/books/branding
- Google APIs Terms of Service: https://developers.google.com/terms

2026年8月7日の外部調査環境からAPIキーなしで公開APIへ直接問い合わせたところHTTP 429となり、
有効なAPIキーを使った代表検索の再実測はこの環境では完了できなかった。
検索結果の実例は7月31日の実測を使用し、認証・検索構文・パラメータは現在の公式仕様で再確認した。

## 3. APIと認証

書籍検索にはVolume collectionを使用する。

```text
GET https://www.googleapis.com/books/v1/volumes?q=<search terms>
```

Google Books内のVolume IDが既知の場合は1件を取得できる。

```text
GET https://www.googleapis.com/books/v1/volumes/<volumeId>
```

公開Volumeの検索・取得は利用者のOAuth認可を必要としない。ただし公式ドキュメントでは、
公開データを取得するリクエストもアプリケーションを識別するためAPIキーまたはOAuth 2.0
アクセストークンのいずれかを付ける必要があるとしている。

`manken` の通常利用では利用者個人のMy Libraryを扱わないため、Google Books APIキーを
設定して公開Volumeだけを取得する構成が適している。OAuth 2.0は初期対象に含めない。

APIキーは `key` クエリパラメータへ設定する。Raw response本文そのものへAPIキーが
含まれることは公式のVolumeレスポンス仕様からは確認できないが、完全なリクエストURLを
ログやエラーへ出すとAPIキーが漏れるため、URLをそのまま記録しない。

数値の固定クォータは今回確認したBooks APIドキュメントには記載されていない。
APIキーはプロジェクトのAPIアクセス、クォータ、レポートに使用されるため、利用可能量は
Google Cloud側のプロジェクト設定を確認する。ライブラリ独自のクォータ値を固定しない。

## 4. 検索構文

`q` は必須の全文検索文字列である。通常の検索語に加え、次のフィールド指定を公式に利用できる。

| 検索 | 構文 | mankenでの候補 |
| --- | --- | --- |
| タイトル | `intitle:<term>` | `SearchBooksRequest.Title` |
| 著者 | `inauthor:<term>` | `SearchBooksRequest.Author` |
| 出版社 | `inpublisher:<term>` | 現行共通検索条件には公開しない |
| 主題 | `subject:<term>` | 漫画判定の補助候補。固定条件にはしない |
| ISBN | `isbn:<value>` | ISBN参照候補 |
| LCCN | `lccn:<value>` | Rawまたは将来用途 |
| OCLC | `oclc:<value>` | Rawまたは将来用途 |

通常の検索語はAND相当で組み合わされ、完全なフレーズは引用符で囲める。
`-term` で一致する語を除外する検索構文も公式に記載されている。
検索語自体は大文字小文字を区別しないが、`intitle` などの特殊キーワードは
大文字小文字を区別する。

したがってGoogle Booksは、既存サンプルで使用しているTitle・Author・ISBNに加え、
`FreeText` と除外条件を実装できる。公式ドキュメントには除外語を指定する検索例があり、
TODO029では `ExcludedText` の各語を利用者入力の演算子注入を避けた除外条件へ変換する方式を採用した。

### 4.1 タイトル検索は完全一致ではない

`intitle:` は指定した文字列がタイトルに見つかるVolumeを検索するものであり、
タイトル全体が完全一致することを保証しない。

2026年7月31日の実測では、`とんがり帽子のアトリエ` の検索結果へ
`小説　とんがり帽子のアトリエ（１）` が含まれた。`上巻` の検索でも漫画以外の書籍や、
タイトルではなく副題側に検索語を持つ候補が確認されている。

`intitle:` を使うことだけで漫画単行本と判断せず、取得後のタイトル、カテゴリ、識別子、
その他の項目を使って候補を確認する必要がある。

## 5. 検索制御とページング

Volumes検索で確認した主なパラメータは次のとおり。

| パラメータ | 現行公式仕様 | mankenでの扱い候補 |
| --- | --- | --- |
| `startIndex` | 0始まり | カーソル内部の位置情報候補 |
| `maxResults` | 既定10、最大40 | `Limit` の上限候補は40 |
| `orderBy` | `relevance` / `newest` | 初期実装は `relevance` 固定候補 |
| `printType` | `all` / `books` / `magazines` | 漫画単行本検索では `books` 候補 |
| `langRestrict` | ISO 639-1の2文字コード | 日本語検索の補助候補。固定可否は実装時判断 |
| `filter` | `partial` / `full` / `free-ebooks` / `paid-ebooks` / `ebooks` | 電子書籍用途の能力。通常書誌検索では固定しない |
| `download` | 現在は `epub` | 初期書誌検索では使用しない |
| `projection` | `full` / `lite` | 既定の `full` を使用する候補 |
| `showPreorders` | 予約商品を含める | 初期実装で公開する必要性は低い |

`printType=books` は雑誌を除外できるが、漫画と小説を区別する値ではない。
`langRestrict=ja` も言語タグによる絞り込みであり、漫画単行本を保証しない。
`subject:` や `volumeInfo.categories` も候補判定には使えるが、カテゴリ欠落や分類差を考慮し、
漫画判定の必須条件として固定する前に実例を増やす必要がある。

検索レスポンスは `totalItems` と `items[]` を返す。共通 `SearchBooksResult` は合計件数を
公開しないため、Google側の位置情報を不透明な `NextCursor` へ変換する構成が候補になる。
カーソルへ少なくとも検索条件、Limit、次の `startIndex` を拘束し、別条件へ再利用できないようにする。

## 6. Volumeリソース

### 6.1 取得元参照

- `id`: Google Books内のVolume ID
- `selfLink`: APIリソースURL
- `volumeInfo.infoLink`: Google Books上の情報ページ
- `volumeInfo.canonicalVolumeLink`: Volumeのcanonical URL
- `volumeInfo.previewLink`: Google Books上のプレビューURL

`BookSource.ID` には `id` が適している。
`BookSource.URL` は書籍参照URLを表すため、Google Books上のcanonical URLを優先し、
欠落時に `infoLink` を使用する案が自然である。`selfLink` と `previewLink` はRaw responseへ残す。

### 6.2 書誌情報

現行Volume resourceでは少なくとも次を取得できる。

- `volumeInfo.title`
- `volumeInfo.subtitle`
- `volumeInfo.authors[]`
- `volumeInfo.publisher`
- `volumeInfo.publishedDate`
- `volumeInfo.description`
- `volumeInfo.industryIdentifiers[]`
  - `ISBN_10`
  - `ISBN_13`
  - `ISSN`
  - `OTHER`
- `volumeInfo.pageCount`
- `volumeInfo.dimensions`
- `volumeInfo.printType`
- `volumeInfo.categories[]`
- `volumeInfo.mainCategory`
- `volumeInfo.language`
- `volumeInfo.imageLinks`

`description` は単純なHTML要素を含む場合がある。
`authors[]` は公式仕様上、著者と編集者を同じ配列で表し、役割を区別できない。
取得元にない役割を推測して `ContributorRole` を設定しない。

Google Booksには作品シリーズや巻数を安定して表す専用項目がない。
既存実測では巻数や版表示が `title` に含まれる例もあるが、TODO029では取得元にない構造を
推測しない方針を採り、タイトルからシリーズ、巻数、版表示を分解しない。

`contentVersion` はVolume本文・画像のコンテンツ版識別子であり、書誌上の版表示ではない。
`EditionStatements` へ変換しない。

## 7. `NormalizedBook` への変換候補

現時点で比較的安全に変換できる候補は次のとおり。

| Google Books | 共通モデル候補 | 注意点 |
| --- | --- | --- |
| `id` | `BookSource.ID` | Google Books内ID |
| `canonicalVolumeLink` / `infoLink` | `BookSource.URL` | canonicalを優先する案 |
| `title` | `Normalized.Title` | 巻数・版を含む場合の分解は別途検討 |
| `subtitle` | `Normalized.Subtitle` | 欠落を正常として扱う |
| `authors[]` | `Normalized.Authors`、役割なしContributor候補 | 著者と編集者を区別できない |
| `publisher` | `Normalized.Publishers` | 単一値 |
| `publishedDate` | `Normalized.Dates` の出版日候補 | 年だけなど精度差を保持する |
| `industryIdentifiers` | `Normalized.Identifiers` | ISBNは共通ISBN検証を通す |
| `description` | `Normalized.Description` | HTMLを含む可能性も含め、取得元文字列をそのまま保持する |
| `language` | `Normalized.Languages` | ISO 639-1 |
| `categories[]` | `Normalized.Subjects` 候補 | Google分類であり漫画判定の保証にはしない |
| `pageCount` | `Normalized.PageCount` | TODO029後の実測を踏まえ、0以下は不明値として未設定にする |
| `dimensions` | `Normalized.PhysicalSize` 候補 | 単位変換規則を確認する |
| `imageLinks` | `Normalized.Images` | 利用時はGoogleの規約・brandingに従う |

次は初期の安全な共通化対象にしない。

- `contentVersion` を版表示として扱うこと
- `averageRating`、`ratingsCount`
- `searchInfo.textSnippet`
- `userInfo`
- `accessInfo.downloadAccess`
- タイトルから推測だけで作るシリーズ、巻数、版表示

## 8. ISBN検索

`q=isbn:<ISBN>` は公式にサポートされる検索構文である。
既存サンプルのISBN検索方針は現在の公式仕様と一致している。

一方、Google BooksにはopenBDのような複数ISBNを1リクエストで参照する専用APIは確認できない。
後続のTODO029では、Raw responseとHTTP通信の対応を明確にするため、Google Booksの
`LookupBooksByISBN` は1回につき1 ISBNだけ受け付け、1回のVolumes検索で完結させる方針を採用した。

同じISBNに複数のVolumeが返る可能性を前提に、共通仕様どおり該当候補を勝手に1冊へ統合しない。

## 9. 販売・閲覧情報

### 9.1 `saleInfo`

Google Booksは次の販売情報を返せる。

- 国
- `saleability`
- `onSaleDate`
- `isEbook`
- 定価候補 `listPrice`
- 実売価格候補 `retailPrice`
- `buyLink`

販売情報はリクエスト元の国によって変わり得る。通常の書誌情報と同じ安定性を前提にしない。
Google Books初期実装では、これらを新しい共通販売モデルへ先行して取り込まず、Raw responseから
利用できる状態を維持する案を優先する。既存 `NormalizedBook.Prices` へ取り込むかは、
楽天Books、Kobo、DMM等との販売情報設計を行う際にまとめて判断する。

### 9.2 `accessInfo`

`accessInfo` は国ごとの閲覧可能性、EPUB/PDFの利用可否、Google Books上のReader URL、
public domain、埋め込み可否などを表す。

これは書誌情報ではなくGoogle Books上の閲覧・アクセス状態であり、初期の
`NormalizedBook` へ共通化しない。MCP / LLMや高度な利用者はRaw responseから参照できる。

## 10. 地域依存

Google Booksは著作権、契約、その他の法的制約に応じて、リクエスト元のIPアドレスを基準に
結果やプレビュー可否を制限する。`saleInfo.country` と `accessInfo.country` も地域に依存する。

同じ検索条件でも実行場所によって販売・閲覧情報が変わる可能性があるため、
`manken` はGoogle Booksの価格、購入可否、プレビュー可否を普遍的な書誌属性として扱わない。

## 11. 利用規約・branding上の注意

Google Books APIにはGoogle APIs Terms of Serviceに加えてBooks API固有のTerms of Serviceと
Branding Guidelinesがある。利用者向けguideを作成する場合は、現在の文面を再確認して案内する。

特に確認した事項:

- Books API固有のTerms of Serviceでは、Googleとの別契約または書面による許可がない限り、アプリケーションの利用に対して利用者へ料金を課すことを認めていない
- API経由のコンテンツから恒久的なデータベースを作ることや、cache headerで許可された期間を超えてキャッシュすることは、権利者または法令による許可がない限りGoogle APIs Termsで制限されている
- Google Booksの検索結果やプレビュー等をアプリ上で表示する場合、GoogleへのattributionとGoogle Booksへの目立つリンクが必要になる
- Branding GuidelinesはGoogle Books検索結果の並び替えや改変を行わないよう求めている
- Google Books APIは商用サービスの代替を意図していないと公式overviewで説明されている

`manken` 自体は課金機能を持たないGoライブラリだが、Google Booksを利用するアプリケーション側の
課金形態、保存方法、表示方法には上記条件が影響する。アフィリエイトや商用利用を一律に禁止するという
意味には広げず、実際の利用形態が該当する場合は利用者が現行Termsを確認する必要がある。

`manken` はライブラリとして利用者の表示UIを制御できないため、Google Booksパッケージのguideでは
少なくとも公式Terms / Brandingへの導線と主要な表示上の注意を案内する必要がある。

また、将来の横断検索でGoogle Books結果と他プロバイダの結果を混在させて順位を変更する設計は、
Branding Guidelinesの検索結果を並べ替え・改変しないという条件に抵触し得る。Google Booksの結果順を
保持したまま別枠で表示する方法を含め、実装前に現行条件との整合を改めて確認する。ライブラリ内部で
技術的に可能であることと、利用条件上許容される表示方法は分けて判断する。

## 12. 既存サンプルの評価

`__sample/book-api/internal/apis/google.go` で現在の公式仕様と一致している点:

- `https://www.googleapis.com/books/v1/volumes` を使用する
- `q` に `intitle:`、`isbn:`、`inauthor:` を使用する
- APIキーを `key` クエリパラメータへ設定する
- `title`、`authors`、`publisher`、`publishedDate`、`description`、画像、`previewLink`、`categories` を読む

そのまま新パッケージへ移植しない点:

- Google Books固有の簡易 `CommonBook` を使用しており、現在の `api.Book` / `NormalizedBook` と責務が異なる
- `ISBN`、`subtitle`、`language`、`pageCount`、`dimensions` など利用可能な書誌項目を捨てている
- `maxResults`、`startIndex` と共通カーソルを実装していない
- `FreeText` と除外条件を扱っていない
- 漫画判定をタイトルのノイズ語除外だけに依存しており、十分ではない
- Raw responseを返す経路がない
- 共通 `ErrorKind` への分類がない
- `previewLink` を一般の商品・参照URLと同じ用途に扱っている
- Google Booksのbranding・保存条件を利用者向けに案内していない

既存サンプルはHTTPリクエスト生成と実データ確認の参考には使えるが、現行パッケージ構造へ
直接移植する実装とはしない。

## 13. mankenでの採用判断

Google Books APIは、`manken` の書誌検索プロバイダとして採用し、TODO029で実装した。
ただし、漫画単行本だけを保証する主要データベースとして扱うのではなく、広い書籍データから
漫画候補を取得する補助的な書誌検索元として位置づけるのが安全である。

TODO029での採用結果:

- タイトル検索: 実装済み
- 著者検索: 実装済み
- フリーワード検索: 実装済み
- ISBN参照: 1回につき1 ISBN、1 HTTPリクエストとして実装済み
- 除外条件: 公式ドキュメントの除外検索例と実サービス確認を根拠に、共通 `ExcludedText` として実装する
- Volume IDによる直接取得: 初期実装には含めない
- `NormalizedBook`: 安全に意味を対応できる書誌項目だけを変換する
- Raw response: 検索とISBN参照の両方で提供する
- `saleInfo` / `accessInfo`: Raw responseに残し、初期共通化はしない

漫画単行本検索では `printType=books` を指定するが、それだけでは小説・関連本を除外できない。
カテゴリやタイトルによる後段判定は必要だが、Google Booksだけを根拠に厳密な漫画分類規則を
作ることは避ける。

## 14. TODO029で確定した実装方針

- APIキーは `WithAPIKey` でClientへ渡し、エラー、カーソル、Raw responseへ露出させない
- `Title`、`Author`、`FreeText`、`ExcludedText` を安全なGoogle Books検索式へ変換する
- 検索では `printType=books`、`orderBy=relevance`、`projection=full` を固定し、`langRestrict` は固定しない
- `Limit` は0で20件、1〜40件を許可し、`startIndex` は不透明カーソルへ隠蔽する
- ISBN参照は1回につき1 ISBNだけ受け付け、`maxResults=40`、`startIndex=0` の1 HTTPリクエストとする
- HTTPエラーは共通 `ErrorKind` へ分類し、成功本文は16 MiB、エラー本文は64 KiBを上限とする
- `description` は取得元文字列を保持し、タイトルから巻数・版表示を推測しない
- `BookSource.URL` は有効なcanonical URLを優先し、利用できない場合はinfoLinkへフォールバックする
- `imageLinks` は共通 `Images` へ変換し、利用条件はguideで案内する
- Google BooksのTerms / Branding上の注意を `docs/pkg/googlebooks/guide.md` に分離して記載する

## 15. 実サービス確認結果

2026年8月7日にローカルの有効なAPIキーを使い、`integration` build tagのテストで次を確認した。

- `intitle:` によるタイトル検索
- `inauthor:` による著者検索
- `isbn:` によるISBN参照
- 通常語を使うフリーワード検索
- `maxResults=40` と `startIndex` によるページング
- `printType=books` を指定した検索

Google公式ドキュメントの除外検索例を確認し、実サービスでも除外語が検索結果へ作用することを確認した。
このため、共通 `ExcludedText` は各語を安全な除外条件へ変換してGoogle Books検索へ適用する。
ただし除外はGoogle Booksのfull-text query全体に作用する。2026-08-08の実測では著者 `佐々木倫子` に
`ExcludedText=動物` を加えると0件となり、タイトルに「動物」を含まない「月館の殺人」も説明文中の
『動物のお医者さん』などが検索対象となって除外され得ることを確認した。

ISBN実レスポンスでは、`isbn:` 検索結果に要求ISBNを `industryIdentifiers` に持たない関連Volumeが
混在する例を確認した。このため実装では要求ISBNとの完全一致を後段で検証する。また、ISBN一致Volumeの
`pageCount` が0となる例があったため、Google Booksでは0以下の `pageCount` を不明値として扱う。
