package yahooshopping

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eamat-dot/manken/model"
)

// TestSearchBooks_FixedParametersAndConversion は、固定検索パラメータと商品変換を確認する
func TestSearchBooks_FixedParametersAndConversion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("appid") != "secret" || q.Get("seller_id") != "tower" || q.Get("genre_category_id") != "10251" || q.Get("image_size") != "600" || q.Get("query") != "作品 作者 出版社 語" {
			t.Errorf("query = %v", q)
		}
		_, _ = w.Write([]byte(`{"totalResultsAvailable":1,"totalResultsReturned":1,"firstResultsPosition":1,"hits":[{"code":"x","name":"作品 COMIC","description":"発売日:2026年01月02日 / レーベル:出版社 / アーティスト:著者 / アーティストカナ:チョシャ / タイトル:作品 2 / タイトルカナ:サクヒン2","janCode":"9784088466361","price":1000,"priceLabel":{"taxable":true},"url":"https://example.test/item","image":{"small":"https://example.test/s"},"seller":{"sellerId":"tower"},"genreCategory":{"id":10251}}]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithClientID("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "作品", Author: "作者", Publisher: "出版社", Query: "語"})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 || len(result.Books) != 1 {
		t.Fatalf("result=%+v", result)
	}
	book := result.Books[0]
	if book.Title != "作品 2" || book.TitleReading != "サクヒン2" || len(book.Authors) != 1 || book.Contributors[0].Reading != "チョシャ" || book.Volume.Number == nil || *book.Volume.Number != 2 || book.Medium != model.PublicationMediumPrint {
		t.Fatalf("book=%+v", book)
	}
	if len(book.ISBN13) != 1 || book.ISBN13[0] != "9784088466361" || book.ReleaseDate != "2026年01月02日" || book.CurrentPrice == nil || book.CurrentPrice.Amount != 1000 || book.CurrentPrice.TaxIncluded == nil || !*book.CurrentPrice.TaxIncluded {
		t.Fatalf("book=%+v", book)
	}
}

// TestConvertItem_TowerMeasuredTitlePatterns は、Towerで実測した付加情報の抽出と元タイトルの保持を確認する
func TestConvertItem_TowerMeasuredTitlePatterns(t *testing.T) {
	for _, test := range []struct {
		title    string
		volume   int
		editions []string
		label    string
	}{
		{"メダリスト(15)", 15, nil, "15"},
		{"SPY×FAMILY 17", 17, nil, "17"},
		{"新装版 動物のお医者さん (11)", 11, []string{"新装版"}, "11"},
		{"悲劇の元凶となる最強外道ラスボス女王は民の為に尽くします。 The Savior's Pride 1巻 (1)", 1, nil, "1"},
		{"作品 #1", 1, nil, "1"},
	} {
		t.Run(test.title, func(t *testing.T) {
			book := convertItem(item{Description: "タイトル:" + test.title}, "")
			if book.Title != test.title || book.Volume.Number == nil || *book.Volume.Number != test.volume || book.Volume.Label != test.label || strings.Join(book.Editions, ",") != strings.Join(test.editions, ",") {
				t.Fatalf("book=%+v", book)
			}
		})
	}
}

// TestConvertItem_TowerTitleReadingUsesIndependentLabeledValue は、明示タイトルに対応する読みだけを保持することを確認する
func TestConvertItem_TowerTitleReadingUsesIndependentLabeledValue(t *testing.T) {
	for _, test := range []struct {
		name, description, fallback, wantTitle, wantTitleReading string
	}{
		{"unchanged labeled title", "タイトル:作品 / タイトルカナ:サクヒン", "", "作品", "サクヒン"},
		{"volume metadata", "タイトル:異世界の沙汰は社畜次第 7 / タイトルカナ:イセカイノサタハシャチクシダイ ナナ", "", "異世界の沙汰は社畜次第 7", "イセカイノサタハシャチクシダイ ナナ"},
		{"edition metadata", "タイトル:愛蔵版 動物のお医者さん 4 / タイトルカナ:アイゾウバン・ドウブツノオイシャサン", "", "愛蔵版 動物のお医者さん 4", "アイゾウバン・ドウブツノオイシャサン"},
		{"fallback title", "タイトルカナ:サクヒン", "作品", "作品", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			book := convertItem(item{Description: test.description, Name: test.fallback}, "")
			if book.Title != test.wantTitle || book.TitleReading != test.wantTitleReading {
				t.Fatalf("book=%+v, want title=%q TitleReading=%q", book, test.wantTitle, test.wantTitleReading)
			}
		})
	}
}

// TestConvertItem_TowerPreservesAmbiguousTitleAndContributorSafety は、安全に分離できない情報を推測しないことを確認する
func TestConvertItem_TowerPreservesAmbiguousTitleAndContributorSafety(t *testing.T) {
	book := convertItem(item{Description: "アーティスト:日向夏、他 / アーティストカナ:ヒュウガ・ナツ / タイトル:ふつつかな悪女ではございますが 1 〜雛宮蝶鼠とりかえ伝〜 IDコミックス / タイトルカナ:フツツカナアクジョ"}, "")
	if book.Title != "ふつつかな悪女ではございますが 1 〜雛宮蝶鼠とりかえ伝〜 IDコミックス" || book.Volume.Number != nil || len(book.Authors) != 1 || book.Authors[0] != "日向夏" {
		t.Fatalf("book=%+v", book)
	}
	if contributor := book.Contributors[0]; contributor.Reading != "ヒュウガ・ナツ" || len(contributor.Roles) != 0 {
		t.Fatalf("contributor=%+v", contributor)
	}
}

// TestConvertItem_TowerMapsLabelsWithoutGuessing は、Towerの明示ラベルと読みの対応条件を確認する
func TestConvertItem_TowerMapsLabelsWithoutGuessing(t *testing.T) {
	book := convertItem(item{Description: "発売日:2026年01月02日 / レーベル:講談社 / アーティスト:甲、乙 / アーティストカナ:コウ / タイトル:作品 / タイトルカナ:サクヒン"}, "")
	if book.TitleReading != "サクヒン" || len(book.Publishers) != 1 || book.Publishers[0] != "講談社" || book.ReleaseDate != "2026年01月02日" {
		t.Fatalf("book=%+v", book)
	}
	if len(book.Contributors) != 2 || book.Contributors[0].Reading != "" || book.Contributors[1].Reading != "" {
		t.Fatalf("contributors=%+v", book.Contributors)
	}
}

// TestDecodeResponse_ActualYahooFieldShapes は、Yahoo!ショッピングの実際のフィールド形式を確認する
func TestDecodeResponse_ActualYahooFieldShapes(t *testing.T) {
	response, err := decodeResponse([]byte(`{"totalResultsAvailable":1,"totalResultsReturned":1,"firstResultsPosition":1,"hits":[{"releaseDate":null,"priceLabel":{"taxable":false},"exImage":{"url":"https://example.test/image","width":320,"height":480},"seller":{"sellerId":"tower"},"genreCategory":{"id":10251},"parentGenreCategories":[{"id":10002}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	book := convertItem(response.Hits[0], "")
	if book.CoverURL != "https://example.test/image" {
		t.Fatalf("CoverURL=%q", book.CoverURL)
	}
	if response.Hits[0].PriceLabel == nil || response.Hits[0].PriceLabel.Taxable == nil || *response.Hits[0].PriceLabel.Taxable {
		t.Fatalf("priceLabel=%+v", response.Hits[0].PriceLabel)
	}
	if _, err := decodeResponse([]byte(`{"totalResultsAvailable":1,"totalResultsReturned":1,"firstResultsPosition":1,"hits":[{"releaseDate":1780000000,"genreCategory":{"id":10251}}]}`)); err != nil {
		t.Fatalf("numeric releaseDate: %v", err)
	}
}

// TestConvertItem_SelectsLargestAvailableCover は、600px指定のexImageとその欠落時の候補順を確認する
func TestConvertItem_SelectsLargestAvailableCover(t *testing.T) {
	for _, test := range []struct {
		name, want string
		item       item
	}{
		{"exImage", "https://example.test/ex", item{Image: itemImage{Small: "https://example.test/small", Medium: "https://example.test/medium"}, ExImage: exImage{URL: "https://example.test/ex", Width: intPointer(600), Height: intPointer(600)}}},
		{"medium fallback", "https://example.test/medium", item{Image: itemImage{Small: "https://example.test/small", Medium: "https://example.test/medium"}}},
		{"small fallback", "https://example.test/small", item{Image: itemImage{Small: "https://example.test/small"}}},
		{"no image", "", item{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := convertItem(test.item, "").CoverURL; got != test.want {
				t.Fatalf("CoverURL=%q, want %q", got, test.want)
			}
		})
	}
}

// TestConvertItem_PriceTaxIncludedIsOptional は、税込情報の欠落を未設定として保持することを確認する
func TestConvertItem_PriceTaxIncludedIsOptional(t *testing.T) {
	for _, taxable := range []*bool{nil, boolPointer(true), boolPointer(false)} {
		book := convertItem(item{Price: int64Pointer(100), PriceLabel: &priceLabel{Taxable: taxable}}, "2026-08-18T00:00:00Z")
		if book.CurrentPrice == nil || book.CurrentPrice.Currency != "JPY" || book.CurrentPrice.Source != SourceYahooShopping || book.CurrentPrice.ObservedAt != "2026-08-18T00:00:00Z" {
			t.Fatalf("CurrentPrice=%+v", book.CurrentPrice)
		}
		if book.CurrentPrice.TaxIncluded == nil || taxable == nil {
			if book.CurrentPrice.TaxIncluded != taxable {
				t.Fatalf("TaxIncluded=%v, want %v", book.CurrentPrice.TaxIncluded, taxable)
			}
		} else if *book.CurrentPrice.TaxIncluded != *taxable {
			t.Fatalf("TaxIncluded=%t, want %t", *book.CurrentPrice.TaxIncluded, *taxable)
		}
	}
	if book := convertItem(item{Price: int64Pointer(0)}, ""); book.CurrentPrice == nil || book.CurrentPrice.Amount != 0 {
		t.Fatalf("CurrentPrice=%+v", book.CurrentPrice)
	}
}

// TestCanonicalJAN は、有効なJANだけをそのまま返すことを確認する
func TestCanonicalJAN(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "4901234567894", want: "4901234567894"},
		{value: "96385074", want: "96385074"},
		{value: "4901234567895"},
		{value: "96385075"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if got := canonicalJAN(test.value); got != test.want {
				t.Fatalf("canonicalJAN(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

// TestCursor_RejectsRequestsPastThousandResults は、APIページング上限を超えるカーソルを拒否することを確認する
func TestCursor_RejectsRequestsPastThousandResults(t *testing.T) {
	for _, test := range []struct {
		start, limit int
		valid        bool
	}{{800, 100, true}, {801, 100, true}, {900, 100, true}, {901, 100, false}} {
		cursor, err := encodeCursor(test.start, "x", test.limit)
		if test.valid && err != nil {
			t.Fatalf("encodeCursor(%d, %d): %v", test.start, test.limit, err)
		}
		if !test.valid && err == nil {
			t.Fatalf("encodeCursor(%d, %d) accepted", test.start, test.limit)
		}
		if test.valid {
			if got, err := decodeCursor(cursor, "x", test.limit); err != nil || got != test.start {
				t.Fatalf("decodeCursor() = %d, %v", got, err)
			}
		}
	}
}

// TestBuildSearchResult_DoesNotCreateUnusableCursor は、使用不能な次ページカーソルを作らないことを確認する
func TestBuildSearchResult_DoesNotCreateUnusableCursor(t *testing.T) {
	response := searchResponse{TotalResultsAvailable: 1000, TotalResultsReturned: 100, FirstResultsPosition: 801, Hits: make([]item, 100)}
	if got := buildSearchResult(response, "x", 100, "").NextCursor; got != "" {
		t.Fatalf("NextCursor=%q", got)
	}
}

// TestParseRetryAfter_RejectsOverflow は、durationを超えるRetry-After秒数を拒否することを確認する
func TestParseRetryAfter_RejectsOverflow(t *testing.T) {
	now := time.Now()
	if got := parseRetryAfter(strconv.FormatInt(maxRetryAfterSeconds+1, 10), now); got != 0 {
		t.Fatalf("RetryAfter=%s", got)
	}
}

// boolPointer は、bool値のポインタを返す
func boolPointer(value bool) *bool { return &value }

// intPointer は、int値のポインタを返す
func intPointer(value int) *int { return &value }

// int64Pointer は、int64値のポインタを返す
func int64Pointer(value int64) *int64 { return &value }

// TestSearchBooks_ValidationAndCursor は、検索入力とカーソルの検証を確認する
func TestSearchBooks_ValidationAndCursor(t *testing.T) {
	client, _ := NewClient(nil, WithClientID("secret"))
	for _, request := range []SearchRequest{{Title: "x", Limit: 101}, {Title: "x", Exclude: "x"}, {}} {
		_, err := client.SearchBooks(context.Background(), request)
		var typed *model.Error
		if !errors.As(err, &typed) || typed.Kind != model.ErrorKindInvalidArgument {
			t.Fatalf("err=%v", err)
		}
	}
	cursor, err := encodeCursor(900, "x", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCursor(cursor, "other", 100); err == nil {
		t.Fatal("cursor conditions accepted")
	}
}

// TestLookupISBN_NormalizesAndDoesNotForceGenre は、ISBNの正規化とISBN参照のカテゴリ未指定を確認する
func TestLookupISBN_NormalizesAndDoesNotForceGenre(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("jan_code") != "9784088466361" || q.Get("genre_category_id") != "" || q.Get("seller_id") != "tower" || q.Get("image_size") != "600" {
			t.Errorf("query=%v", q)
		}
		if err := json.NewEncoder(w).Encode(searchResponse{TotalResultsReturned: 1, FirstResultsPosition: 1, Hits: []item{{JanCode: "9784088466361", Seller: seller{SellerID: "tower"}}}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, _ := NewClient(nil, WithClientID("secret"), WithEndpoint(server.URL))
	result, _, err := client.LookupBooksByISBNWithRawResponse(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items[0].Books) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

// TestSearchBooks_ExcludesWholeSetsAndJAN は、全巻セットの除外とJANフィールドへの変換を確認する
func TestSearchBooks_ExcludesWholeSetsAndJAN(t *testing.T) {
	if isTowerComic(item{Name: "作品 1-10巻セット", Seller: seller{SellerID: "tower"}, GenreCategory: genre{ID: comicGenreCategoryID}}) {
		t.Fatal("set accepted")
	}
	book := convertItem(item{JanCode: "4901234567894"}, "")
	if len(book.JAN) != 1 || book.JAN[0] != "4901234567894" {
		t.Fatalf("JAN=%+v", book.JAN)
	}
}

// TestHTTPErrorKinds は、取得元のHTTPエラー分類を確認する
func TestHTTPErrorKinds(t *testing.T) {
	for _, status := range []int{429, 503, 400} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		client, _ := NewClient(nil, WithClientID("secret"), WithEndpoint(server.URL))
		_, err := client.SearchBooks(context.Background(), SearchRequest{Title: "x"})
		server.Close()
		var typed *model.Error
		if !errors.As(err, &typed) {
			t.Fatal(err)
		}
		if (status == 429 || status == 503) && typed.Kind != model.ErrorKindUnavailable {
			t.Fatalf("status=%d kind=%s", status, typed.Kind)
		}
	}
}

// TestEndpointRejectsUnsafe は、安全でないエンドポイント設定を拒否することを確認する
func TestEndpointRejectsUnsafe(t *testing.T) {
	if _, err := NewClient(nil, WithClientID("secret"), WithEndpoint("http://example.test")); err == nil {
		t.Fatal("public HTTP endpoint accepted")
	}
	if _, err := NewClient(nil, WithClientID("secret"), nil); err == nil {
		t.Fatal("nil option accepted")
	}
	_ = url.Values{}
}

// TestClient_RejectsInvalidCredentialsAndDoesNotExposeThem は、不正な認証情報の拒否とエラーの秘匿を確認する
func TestClient_RejectsInvalidCredentialsAndDoesNotExposeThem(t *testing.T) {
	for _, option := range []Option{WithClientID(""), WithClientID("bad\nvalue"), WithEndpoint("https://example.test/?secret")} {
		if _, err := NewClient(nil, option); err == nil {
			t.Fatalf("NewClient(%T) succeeded", option)
		}
	}
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("transport failure") })
	client, err := NewClient(&http.Client{Transport: transport}, WithClientID("secret"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "x"})
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error exposed client ID: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、関数形式のラウンドトリッパーを実行する
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// TestEffectiveLimit_Boundaries は、Limitの既定値とAPI境界値を確認する
func TestEffectiveLimit_Boundaries(t *testing.T) {
	for _, test := range []struct {
		limit int
		want  int
		valid bool
	}{{0, 20, true}, {1, 1, true}, {100, 100, true}, {101, 0, false}} {
		got, err := effectiveLimit(test.limit)
		if (err == nil) != test.valid || got != test.want {
			t.Fatalf("effectiveLimit(%d) = %d, %v", test.limit, got, err)
		}
	}
}

// TestDecodeCursor_RejectsInvalidPayloads は、不正なCursor形式と検索条件の不一致を拒否する
func TestDecodeCursor_RejectsInvalidPayloads(t *testing.T) {
	valid, err := encodeCursor(900, "conditions secret", 100)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(valid, "secret") {
		t.Fatalf("cursor exposed client ID: %q", valid)
	}
	invalidJSON := base64.RawURLEncoding.EncodeToString([]byte(`{`))
	trailingJSON := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"start":2,"limit":100,"search_sha256":"` + hashSearchKey("conditions secret") + `"} null`))
	for _, test := range []struct {
		name, cursor, key string
		limit             int
	}{
		{"base64", "*", "conditions secret", 100},
		{"json", invalidJSON, "conditions secret", 100},
		{"trailing JSON", trailingJSON, "conditions secret", 100},
		{"limit mismatch", valid, "conditions secret", 1},
		{"condition mismatch", valid, "other", 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeCursor(test.cursor, test.key, test.limit); err == nil {
				t.Fatal("decodeCursor accepted invalid cursor")
			}
		})
	}
}

// TestLookupBooksByISBN_ValidatesAndFiltersResults は、ISBN入力を検証し一致する複数商品だけを保持する
func TestLookupBooksByISBN_ValidatesAndFiltersResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("jan_code"); got != "9784088466361" {
			t.Errorf("jan_code=%q", got)
		}
		_, _ = w.Write([]byte(`{"totalResultsAvailable":3,"totalResultsReturned":3,"firstResultsPosition":1,"hits":[{"code":"one","janCode":"9784088466361","seller":{"sellerId":"tower"}},{"code":"two","janCode":"9784088466361","seller":{"sellerId":"tower"}},{"code":"other","janCode":"9784090000000","seller":{"sellerId":"tower"}}]}`))
	}))
	defer server.Close()
	client, err := NewClient(nil, WithClientID("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, isbn := range []string{"invalid", "9784088466361"} {
		if _, err := client.LookupBooksByISBN(context.Background(), []string{isbn}); (err == nil) != (isbn == "9784088466361") {
			t.Fatalf("LookupBooksByISBN(%q) error=%v", isbn, err)
		}
	}
	result, err := client.LookupBooksByISBN(context.Background(), []string{"4088466365"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || len(result.Items[0].Books) != 2 {
		t.Fatalf("result=%+v", result)
	}
}

// TestTowerFilteringAndOptionalFields は、商品除外規則と通常の欠落値を確認する
func TestTowerFilteringAndOptionalFields(t *testing.T) {
	base := item{Seller: seller{SellerID: towerSellerID}, GenreCategory: genre{ID: comicGenreCategoryID}}
	for _, test := range []struct {
		name string
		item item
		want bool
	}{
		{"whole set genre", item{Seller: base.Seller, GenreCategory: genre{ID: wholeSetGenreCategoryID}}, false},
		{"whole set name", item{Name: "作品 全巻セット", Seller: base.Seller, GenreCategory: base.GenreCategory}, false},
		{"range set name", item{Name: "作品 1-10巻セット", Seller: base.Seller, GenreCategory: base.GenreCategory}, false},
		{"illustration collection", item{Name: "作品 イラスト集", Seller: base.Seller, GenreCategory: base.GenreCategory}, true},
		{"illustration collection special edition", item{Name: "草凪みずほ 暁のヨナ YONA MEMORIAL イラスト集付き特装版 47＜イラスト集付き特装版＞ COMIC", Seller: base.Seller, GenreCategory: genre{ID: 68224}, ParentGenreCategories: []genre{{ID: comicGenreCategoryID}}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isTowerComic(test.item); got != test.want {
				t.Fatalf("isTowerComic(%+v) = %t, want %t", test.item, got, test.want)
			}
		})
	}
	book := convertItem(item{Seller: base.Seller, GenreCategory: base.GenreCategory}, "")
	if !isTowerComic(base) || len(book.ISBN13) != 0 || len(book.JAN) != 0 || book.CoverURL != "" || len(book.Publishers) != 0 {
		t.Fatalf("book=%+v", book)
	}
	for _, test := range []struct {
		jan        string
		wantISBN13 string
		wantJAN    string
	}{{" 9784088466361 ", "9784088466361", ""}, {"49012345", "", ""}, {"4901234567894", "", "4901234567894"}, {"123456789012", "", ""}, {"12345678901234", "", ""}, {"4901234A", "", ""}, {"   ", "", ""}} {
		book := convertItem(item{JanCode: test.jan}, "")
		if got := strings.Join(book.ISBN13, ","); got != test.wantISBN13 {
			t.Fatalf("ISBN13 for %q = %q, want %q", test.jan, got, test.wantISBN13)
		}
		if got := strings.Join(book.JAN, ","); got != test.wantJAN {
			t.Fatalf("JAN for %q = %q, want %q", test.jan, got, test.wantJAN)
		}
	}
}

// TestSearchBooks_RawResponseAndInvalidJSON は、成功本文の保持と不正JSONの分類を確認する
func TestSearchBooks_RawResponseAndInvalidJSON(t *testing.T) {
	body := []byte(`{"totalResultsAvailable":0,"totalResultsReturned":0,"firstResultsPosition":0,"hits":[]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) }))
	defer server.Close()
	client, err := NewClient(nil, WithClientID("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "x"})
	if err != nil || len(result.Books) != 0 || !bytes.Equal(raw, body) || bytes.Contains(raw, []byte("secret")) {
		t.Fatalf("result=%+v raw=%q err=%v", result, raw, err)
	}
	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{`)) }))
	defer invalidServer.Close()
	invalidClient, err := NewClient(nil, WithClientID("secret"), WithEndpoint(invalidServer.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = invalidClient.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: "x"})
	var typed *model.Error
	if !errors.As(err, &typed) || typed.Kind != model.ErrorKindInvalidResponse || strings.Contains(err.Error(), "secret") {
		t.Fatalf("err=%v", err)
	}
}

// TestClient_DoesNotFollowRedirects は、redirect先へ認証情報付きの要求を送らない
func TestClient_DoesNotFollowRedirects(t *testing.T) {
	redirectTargetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirectTargetCalled = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer server.Close()
	client, err := NewClient(nil, WithClientID("secret"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "x"})
	if err == nil || redirectTargetCalled || strings.Contains(err.Error(), "secret") {
		t.Fatalf("err=%v redirectTargetCalled=%t", err, redirectTargetCalled)
	}
}

// TestParseRetryAfter_ParsesSecondsAndDate は、正常な秒数とHTTP日付を待機時間へ変換する
func TestParseRetryAfter_ParsesSecondsAndDate(t *testing.T) {
	now := time.Date(2026, time.August, 13, 0, 0, 0, 0, time.UTC)
	if got := parseRetryAfter("30", now); got != 30*time.Second {
		t.Fatalf("seconds RetryAfter=%s", got)
	}
	if got := parseRetryAfter(now.Add(2*time.Minute).Format(http.TimeFormat), now); got != 2*time.Minute {
		t.Fatalf("date RetryAfter=%s", got)
	}
}
