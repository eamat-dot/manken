# MADBに著者名検索を追加

## 状態

2026年8月1日完了。

## 背景

- MADBの著者情報は `schema:creator` 文字列とAgent参照の2経路に存在する
- Agent参照だけを持つ漫画単行本があり、片方だけの検索では取りこぼす
- 公開SPARQLで両経路を `UNION` し、`M830542` を `KotzDean` で取得できた

## 目的

- creator文字列とAgent名の両方を対象に著者名検索できるようにする
- 著者名内の空白区切り語と他の正条件をAND結合する

## 根拠

- [`../../research/014-madb-search-conditions.md`](../../research/014-madb-search-conditions.md)
- [`../../research/007-madb-author-roles.md`](../../research/007-madb-author-roles.md)

## 非目的

- 著者名の異体字、別名、読み、姓名順の同一人物判定
- creator文字列とAgentの1対1対応付け
- 著者役割による検索

## 前提・制約

- `SearchBooksRequest` へ `Author string` を追加する
- 014と015の検索条件・カーソル拡張後に実施する
- 正条件を1つ以上必須とし、複数の正条件はAND結合する
- Agent検索は実測で直接creator検索より遅いため、60秒制限を考慮する

## 処理方針

1. 著者名をUnicode空白で分割し、引用した各語を `AND` で結ぶ
2. `schema:creator` を対象とする全文検索で単行本URIを得る
3. `rdfs:label` を対象とする全文検索でAgent URIを得て、
   `?resource dcterms:creator ?agent` を逆引きする
4. 2経路を `UNION` し、同じ単行本を `DISTINCT` で1件にする
5. タイトル、ISBNなどの正条件とAND結合する

## 実施項目

### 公開契約と実装

- [x] `api.SearchBooksRequest` に `Author` を追加する
- [x] 著者名の全文検索式を安全に生成する
- [x] creator文字列とAgent参照の `UNION` を生成する
- [x] 他の正条件とのAND結合へ組み込む
- [x] カーソル条件ハッシュへ正規化済み著者名を含める

### テスト

- [x] creator文字列だけ、Agent参照だけ、両方、どちらもないケースを確認する
- [x] 空白区切りの複数語をAND検索する
- [x] 著者名だけ、タイトルと著者名、ISBNと著者名を確認する
- [x] `UNION` による重複を1冊へまとめる
- [x] 利用者入力の全文検索演算子を構文として解釈しない
- [x] 実サービスで `KotzDean` から `M830542` を取得する
- [x] 実サービスで一般的なcreator文字列の著者を取得する
- [x] 代表条件の応答時間が60秒以内であることを確認する

### ドキュメント

- [x] `docs/spec.md` と `docs/pkg/madb/spec.md` に著者名検索を追加する
- [x] creator文字列とAgent参照の両方を対象にすることを記載する
- [x] デモCLIへ `-author` を追加する

## 検証

- [x] `go test -v ./...` を実行する
- [x] `go build -v ./...` を実行する
- [x] `golangci-lint run` を実行する
- [x] `go test -tags=integration ./madb` を実行する
- [x] `git diff --check` を実行する

## 受け入れ条件

- [x] creator文字列とAgent参照のどちらにだけ存在する著者も検索できる
- [x] 他の正条件とAND結合できる
- [x] 同じ単行本を重複して返さない
- [x] 既存の著者・寄与者変換結果を変更しない

## リスク・懸念

- Agent参照の逆引きにより検索時間が増える
- 著者役割を検索条件に含めないため、解説者なども名前では一致し得る

## 未決事項

- なし

## 実装結果

- タイトルと著者名で共通の全文検索式生成を使用し、各語を引用したAND検索にした
- creator文字列とAgent参照を `UNION` し、ページ対象の `SELECT DISTINCT` で重複を除いた
- 著者名だけ、タイトルと著者名、ISBNと著者名の実サービス検索を確認した
- 初回実測ではcreator文字列の著者検索が約25.5秒、Agent参照の著者検索が約17.7秒、
  タイトルとのAND検索が約0.7秒で完了した
- 著者検索語数にはタイトルと同様に独自上限を設けず、ContextまたはHTTPクライアントの
  タイムアウトで制御する
- デモCLIの `-author KotzDean` から `M830542` を取得した
