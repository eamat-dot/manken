package ndl

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestBuildSearchQueryEscapesUserInput は、CQLの外側引用符と内部エスケープを確認する
func TestBuildSearchQueryEscapesUserInput(t *testing.T) {
	query, err := buildSearchQuery(SearchRequest{Title: `日本 AND "本"\test`, Author: "foo OR bar"}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `dpid = "iss-ndl-opac-national" AND mediatype = "books" AND title = "日本 AND \"本\"\\test" AND creator = "foo OR bar" AND ndc = "726.1" AND ndlc = "Y84" AND sortBy=issued_date/sort.ascending`
	if query != want {
		t.Fatalf("query = %q, want %q", query, want)
	}
}

// TestBuildSearchQueryAlwaysUsesFixedDefaultSort は、利用者入力にかかわらず固定の既定ソートを付与することを確認する
func TestBuildSearchQueryAlwaysUsesFixedDefaultSort(t *testing.T) {
	query, err := buildSearchQuery(SearchRequest{Title: `ndc = "999" AND sortBy=created_date/sort.descending`}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `dpid = "iss-ndl-opac-national" AND mediatype = "books" AND title = "ndc = \"999\" AND sortBy=created_date/sort.descending" AND ndc = "726.1" AND ndlc = "Y84" AND sortBy=issued_date/sort.ascending`
	if query != want {
		t.Fatalf("query = %q, want %q", query, want)
	}
}

// TestBuildSearchQueryWithOptions は、NDL固有検索条件を安定した順序でCQLへ追加することを確認する
func TestBuildSearchQueryWithOptions(t *testing.T) {
	query, err := buildSearchQueryWithOptions(SearchRequest{Title: "漫画", DateFrom: " 2024-01 ", DateTo: "2024-12"}, SearchOptions{
		Subject:     "動物",
		Description: "ハムテル",
	}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `dpid = "iss-ndl-opac-national" AND mediatype = "books" AND title = "漫画" AND from = "2024-01" AND until = "2024-12" AND subject = "動物" AND description = "ハムテル" AND ndc = "726.1" AND ndlc = "Y84" AND sortBy=issued_date/sort.ascending`
	if query != want {
		t.Fatalf("query = %q, want %q", query, want)
	}
}

// TestBuildSearchQueryWithOptionsAcceptsSingleNDLCondition は、NDL固有検索条件だけを検索条件として受け付けることを確認する
func TestBuildSearchQueryWithOptionsAcceptsSingleNDLCondition(t *testing.T) {
	for _, request := range []SearchRequest{{DateFrom: "2024"}, {DateTo: "2024-12"}, {}} {
		query, err := buildSearchQueryWithOptions(request, SearchOptions{Subject: "動物"}, true, true)
		if err != nil || !strings.Contains(query, defaultSortCQLTerm) {
			t.Fatalf("request = %#v, query = %q, err = %v", request, query, err)
		}
		if !strings.Contains(query, `subject = "動物"`) {
			t.Fatalf("request = %#v, query = %q, want subject condition", request, query)
		}
	}
}

// TestBuildSearchQueryWithOptionsRejectsInvalidDatesBeforeHTTP は、不正な日付と精度不一致を検索前に拒否することを確認する
func TestBuildSearchQueryWithOptionsRejectsInvalidDatesBeforeHTTP(t *testing.T) {
	for _, request := range []SearchRequest{{DateFrom: "2024-13"}, {DateTo: "2024-02-30"}, {DateFrom: "2024/01"}, {DateFrom: "2024", DateTo: "2024-01"}} {
		if _, err := buildSearchQueryWithOptions(request, SearchOptions{}, true, true); err == nil {
			t.Fatalf("request = %#v accepted", request)
		}
	}
}

// TestBuildSearchQueryWithOptionsEscapesUserInput は、NDL固有検索条件がCQL構文を注入できないことを確認する
func TestBuildSearchQueryWithOptionsEscapesUserInput(t *testing.T) {
	query, err := buildSearchQueryWithOptions(SearchRequest{}, SearchOptions{Subject: `動物" OR ndc = "999`, Description: `foo\\bar`}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, `subject = "動物\" OR ndc = \"999"`) || !strings.Contains(query, `description = "foo\\\\bar"`) {
		t.Fatalf("query = %q", query)
	}
}

// TestBuildSearchQueryMangaFilters は、漫画分類フィルタの有効状態ごとのCQLを確認する
func TestBuildSearchQueryMangaFilters(t *testing.T) {
	tests := []struct {
		name            string
		mangaNDCFilter  bool
		mangaNDLCFilter bool
		want            string
	}{
		{name: "both enabled", mangaNDCFilter: true, mangaNDLCFilter: true, want: `ndc = "726.1" AND ndlc = "Y84"`},
		{name: "NDC disabled", mangaNDLCFilter: true, want: `ndlc = "Y84"`},
		{name: "NDLC disabled", mangaNDCFilter: true, want: `ndc = "726.1"`},
		{name: "both disabled", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := buildSearchQuery(SearchRequest{Title: "漫画"}, test.mangaNDCFilter, test.mangaNDLCFilter)
			if err != nil {
				t.Fatal(err)
			}
			for _, filter := range []string{`ndc = "726.1"`, `ndlc = "Y84"`} {
				if strings.Contains(query, filter) != strings.Contains(test.want, filter) {
					t.Errorf("query = %q, filter %q presence differs from %q", query, filter, test.want)
				}
			}
			if !strings.HasSuffix(query, defaultSortCQLTerm) {
				t.Errorf("query = %q does not end with %q", query, defaultSortCQLTerm)
			}
		})
	}
}

// TestSearchBooksMangaFilterOptions は、Client Optionが通常検索の分類フィルタを切り替えることを確認する
func TestSearchBooksMangaFilterOptions(t *testing.T) {
	requestedQuery := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedQuery = request.URL.Query().Get("query")
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><searchRetrieveResponse><numberOfRecords>0</numberOfRecords></searchRetrieveResponse>`))
	}))
	defer server.Close()
	tests := []struct {
		name    string
		options []Option
		want    string
	}{
		{name: "defaults", want: `ndc = "726.1" AND ndlc = "Y84"`},
		{name: "NDC disabled", options: []Option{WithMangaNDCFilter(false)}, want: `ndlc = "Y84"`},
		{name: "NDLC disabled", options: []Option{WithMangaNDLCFilter(false)}, want: `ndc = "726.1"`},
		{name: "both disabled", options: []Option{WithMangaNDCFilter(false), WithMangaNDLCFilter(false)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := append([]Option{WithEndpoint(server.URL)}, test.options...)
			client, err := NewClient(nil, options...)
			if err != nil {
				t.Fatal(err)
			}
			query, err := buildSearchQuery(SearchRequest{Title: "漫画"}, client.mangaNDCFilter, client.mangaNDLCFilter)
			if err != nil {
				t.Fatal(err)
			}
			for _, filter := range []string{`ndc = "726.1"`, `ndlc = "Y84"`} {
				if strings.Contains(query, filter) != strings.Contains(test.want, filter) {
					t.Errorf("query = %q, filter %q presence differs from %q", query, filter, test.want)
				}
			}
			if _, err := client.SearchBooks(context.Background(), SearchRequest{Title: "漫画"}); err != nil {
				t.Fatal(err)
			}
			for _, filter := range []string{`ndc = "726.1"`, `ndlc = "Y84"`} {
				if strings.Contains(requestedQuery, filter) != strings.Contains(test.want, filter) {
					t.Errorf("request query = %q, filter %q presence differs from %q", requestedQuery, filter, test.want)
				}
			}
		})
	}
}

// TestSearchBooksSendsDecodedCQL は、HTTP要求のqueryがCQL文字列をそのまま保持することを確認する
func TestSearchBooksSendsDecodedCQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		values := request.URL.Query()
		for key, want := range map[string]string{"operation": "searchRetrieve", "version": "1.2", "recordSchema": "dcndl_v3", "recordPacking": "xml", "onlyBib": "true", "maximumRecords": "20", "startRecord": "1"} {
			if values.Get(key) != want {
				t.Errorf("%s = %q, want %q", key, values.Get(key), want)
			}
		}
		if got, want := values.Get("query"), `dpid = "iss-ndl-opac-national" AND mediatype = "books" AND title = "日本 AND \"本\"\\test" AND ndc = "726.1" AND ndlc = "Y84" AND sortBy=issued_date/sort.ascending`; got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><searchRetrieveResponse><numberOfRecords>0</numberOfRecords></searchRetrieveResponse>`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	result, raw, err := client.SearchBooksWithRawResponse(context.Background(), SearchRequest{Title: `日本 AND "本"\test`})
	if err != nil {
		t.Fatal(err)
	}
	if result.Books == nil || len(raw) == 0 {
		t.Fatal("expected non-nil books and raw response")
	}
}

// TestConvertRecordMapsDCNDLV3Fixture は、DC-NDL v3の名前空間と属性に基づく変換を確認する
func TestConvertRecordMapsDCNDLV3Fixture(t *testing.T) {
	body := readFixture(t, "testdata/dcndl-v3.xml")
	response, err := decodeResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.records) != 1 {
		t.Fatalf("records = %d", len(response.records))
	}
	book, err := convertRecord(response.records[0])
	if err != nil {
		t.Fatal(err)
	}
	if book.Sources[0].ID != "000000001" || book.Title != "サンプル作品" || book.TitleReading != "サンプル サクヒン" {
		t.Fatalf("book = %#v", book)
	}
	if len(book.PublicationSeries) != 1 || book.PublicationSeries[0] != "サンプルコミックス" {
		t.Fatalf("PublicationSeries = %#v", book.PublicationSeries)
	}
	if book.Volume.Label != "1" || book.Volume.Number == nil || *book.Volume.Number != 1 {
		t.Fatalf("volume = %#v", book.Volume)
	}
	if !containsString(book.Authors, "原作者A") || !containsString(book.Authors, "脚本家B") || !containsString(book.Authors, "作画家C") || !containsString(book.Authors, "構成者E") || containsString(book.Authors, "作家, 太郎") || !containsString(book.Publishers, "出版社A") || containsString(book.Publishers, "流通会社") {
		t.Fatalf("authors/publishers = %#v/%#v", book.Authors, book.Publishers)
	}
	if len(book.ISBN10) != 0 || len(book.ISBN13) != 1 || book.ISBN13[0] != "9780000000002" || book.PublishedDate != "2020.01" || book.PageCount == nil || *book.PageCount != 200 || book.Size != "18cm" || book.Medium != PublicationMediumPrint {
		t.Fatalf("ISBN/published date/page/size/medium = %#v/%q/%#v/%q/%q", book.ISBN13, book.PublishedDate, book.PageCount, book.Size, book.Medium)
	}
	if !hasSubject(book.Subjects, "NDC", "726.1", "") || !hasSubject(book.Subjects, "NDLC", "Y84", "") || !hasSubject(book.Subjects, "NDLSH", "00000001", "漫画") || !hasSubject(book.Subjects, "NDLGFT", "000000002", "漫画") {
		t.Fatalf("subjects = %#v", book.Subjects)
	}
	if book.ListPrice == nil || book.ListPrice.Amount != 700 || book.ListPrice.Currency != "JPY" || book.ListPrice.Source != SourceNDL || book.ListPrice.TaxIncluded != nil || book.CurrentPrice != nil || len(book.Contributors) != 5 || book.Contributors[0].Name != "原作者A" || book.Contributors[0].Roles[0] != "原作" || book.Contributors[3].Name != "監修者D" || book.Contributors[3].Roles[0] != "監修" || containsString(book.Authors, "監修者D") {
		t.Fatalf("contributors = %#v", book.Contributors)
	}
}

// TestConvertRecordPreservesTitleWithVolume は、巻次を抽出しても取得元タイトルを変更しないことを確認する
func TestConvertRecordPreservesTitleWithVolume(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{
		{Name: xml.Name{Space: nsDCTerms, Local: "identifier"}, Attr: []xml.Attr{{Name: xml.Name{Space: nsRDF, Local: "datatype"}, Value: ndlbibIDDatatype}}, Text: "123456789"},
		{Name: xml.Name{Space: nsDC, Local: "title"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: "作品 第1巻 新装版"}}}}},
		{Name: xml.Name{Space: nsDCNDL, Local: "volume"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: "1"}}}}},
	}}
	book, err := convertRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if book.Title != "作品 第1巻 新装版" || book.Volume.Label != "1" || book.Volume.Number == nil || *book.Volume.Number != 1 {
		t.Fatalf("book = %#v", book)
	}
}

// TestConvertRecord_MapsSafePriceAndExtent は、安全なNDL価格とextentのページ数・大きさを変換する
func TestConvertRecord_MapsSafePriceAndExtent(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{
		{Name: xml.Name{Space: nsDCTerms, Local: "identifier"}, Attr: []xml.Attr{{Name: xml.Name{Space: nsRDF, Local: "datatype"}, Value: ndlbibIDDatatype}}, Text: "123456789"},
		{Name: xml.Name{Space: nsDCNDL, Local: "price"}, Text: " 463 円 "},
		{Name: xml.Name{Space: nsDCTerms, Local: "extent"}, Text: "190 p ; 19 cm"},
	}}
	book, err := convertRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if book.PageCount == nil || *book.PageCount != 190 || book.Size != "19 cm" {
		t.Fatalf("PageCount/Size = %#v/%q", book.PageCount, book.Size)
	}
	if book.ListPrice == nil || book.ListPrice.Amount != 463 || book.ListPrice.Currency != "JPY" ||
		book.ListPrice.Source != SourceNDL || book.ListPrice.TaxIncluded != nil || book.ListPrice.ObservedAt != "" || book.CurrentPrice != nil {
		t.Fatalf("prices = %#v/%#v", book.ListPrice, book.CurrentPrice)
	}
}

// TestListPrice_RejectsWhitespaceInsideDigits は、数字途中に空白がある価格を推測変換しないことを確認する
func TestListPrice_RejectsWhitespaceInsideDigits(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{{Name: xml.Name{Space: nsDCNDL, Local: "price"}, Text: "4 63円"}}}
	if got := listPrice(record); got != nil {
		t.Fatalf("listPrice() = %#v", got)
	}
}

// TestConvertRecord_SkipsAmbiguousPriceAndExtent は、曖昧な価格と複雑な大きさを変換しない
func TestConvertRecord_SkipsAmbiguousPriceAndExtent(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{
		{Name: xml.Name{Space: nsDCTerms, Local: "identifier"}, Attr: []xml.Attr{{Name: xml.Name{Space: nsRDF, Local: "datatype"}, Value: ndlbibIDDatatype}}, Text: "123456789"},
		{Name: xml.Name{Space: nsDCNDL, Local: "price"}, Text: "463円"},
		{Name: xml.Name{Space: nsDCNDL, Local: "price"}, Text: "500円"},
		{Name: xml.Name{Space: nsDCNDL, Local: "price"}, Text: "650円(税込)"},
		{Name: xml.Name{Space: nsDCTerms, Local: "extent"}, Text: "188p ; 19cm + アクリルキーホルダー 2個"},
	}}
	book, err := convertRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if book.ListPrice != nil || book.Size != "19cm" || book.PageCount == nil || *book.PageCount != 188 {
		t.Fatalf("book = %#v", book)
	}
}

// TestConvertRecord_MapsSizeWithoutPageCount は、ページ数がない単純なcm表記も大きさとして変換する
func TestConvertRecord_MapsSizeWithoutPageCount(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{
		{Name: xml.Name{Space: nsDCTerms, Local: "identifier"}, Attr: []xml.Attr{{Name: xml.Name{Space: nsRDF, Local: "datatype"}, Value: ndlbibIDDatatype}}, Text: "123456789"},
		{Name: xml.Name{Space: nsDCTerms, Local: "extent"}, Text: "1冊 ; 19cm"},
	}}
	book, err := convertRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if book.PageCount != nil || book.Size != "19cm" {
		t.Fatalf("PageCount/Size = %#v/%q", book.PageCount, book.Size)
	}
}

// TestApplyTitleMetadata_PrefersExplicitMetadata は、NDLの明示巻次と版表示をタイトル候補より優先することを確認する
func TestApplyTitleMetadata_PrefersExplicitMetadata(t *testing.T) {
	seven := 7
	book := Book{Title: "作品 第2巻 完全版", Volume: Volume{Number: &seven, Label: "7"}, Editions: []string{"特装版"}}
	applyTitleMetadata(&book)
	if book.Volume.Number == nil || *book.Volume.Number != 7 || book.Volume.Label != "7" {
		t.Fatalf("Volume = %#v", book.Volume)
	}
	if len(book.Editions) != 1 || book.Editions[0] != "特装版" {
		t.Fatalf("Editions = %#v", book.Editions)
	}
}

// TestPublicationSeries_PreservesNamesAndSkipsEmptyNames は、系列名を応答順で保持し空の名称を除外する
func TestPublicationSeries_PreservesNamesAndSkipsEmptyNames(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{
		{Name: xml.Name{Space: nsDCNDL, Local: "seriesTitle"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: "叢書A"}, {Name: xml.Name{Space: nsDCNDL, Local: "transcription"}, Text: "ソウショエー"}}}}},
		{Name: xml.Name{Space: nsDCNDL, Local: "seriesTitle"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: "叢書B"}}}}},
		{Name: xml.Name{Space: nsDCNDL, Local: "seriesTitle"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: "叢書A"}}}}},
		{Name: xml.Name{Space: nsDCNDL, Local: "seriesTitle"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: ""}, {Name: xml.Name{Space: nsDCNDL, Local: "transcription"}, Text: "ヨミダケ"}}}}},
		{Name: xml.Name{Space: nsDCNDL, Local: "seriesTitle"}, Children: []*xmlNode{
			{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: "叢書C"}}},
			{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsDCNDL, Local: "transcription"}, Text: "ソウショシー"}}},
		}},
	}}
	got := publicationSeries(record)
	if len(got) != 3 || got[0] != "叢書A" || got[1] != "叢書B" || got[2] != "叢書C" {
		t.Fatalf("publicationSeries() = %#v", got)
	}
}

// TestConvertRecordHandlesVolumeNumberOverflow は、整数化できない巻号でもラベルを保持することを確認する
func TestConvertRecordHandlesVolumeNumberOverflow(t *testing.T) {
	tests := []struct {
		name      string
		volume    string
		wantValue int
		hasNumber bool
	}{
		{name: "ordinary integer", volume: "11", wantValue: 11, hasNumber: true},
		{name: "overflow", volume: "999999999999999999999999"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := &xmlNode{Children: []*xmlNode{
				{Name: xml.Name{Space: nsDCTerms, Local: "identifier"}, Attr: []xml.Attr{{Name: xml.Name{Space: nsRDF, Local: "datatype"}, Value: ndlbibIDDatatype}}, Text: "123456789"},
				{Name: xml.Name{Space: nsDCNDL, Local: "volume"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "Description"}, Children: []*xmlNode{{Name: xml.Name{Space: nsRDF, Local: "value"}, Text: test.volume}}}}},
			}}
			book, err := convertRecord(record)
			if err != nil {
				t.Fatal(err)
			}
			if book.Volume.Label != test.volume {
				t.Errorf("Label = %q, want %q", book.Volume.Label, test.volume)
			}
			if !test.hasNumber {
				if book.Volume.Number != nil {
					t.Errorf("Number = %d, want nil", *book.Volume.Number)
				}
				return
			}
			if book.Volume.Number == nil || *book.Volume.Number != test.wantValue {
				t.Errorf("Number = %v, want %d", book.Volume.Number, test.wantValue)
			}
		})
	}
}

// TestParseCreatorLiteral は、既知の末尾roleだけを表示名から分離することを検証する
func TestParseCreatorLiteral(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantName  string
		wantRoles []string
		wantOK    bool
	}{
		{name: "original creator with middle dot", value: "藤子・F・不二雄 原作", wantName: "藤子・F・不二雄", wantRoles: []string{"原作"}, wantOK: true},
		{name: "writer", value: "小野寺紳 脚本", wantName: "小野寺紳", wantRoles: []string{"脚本"}, wantOK: true},
		{name: "artist", value: "三谷幸広 作画", wantName: "三谷幸広", wantRoles: []string{"作画"}, wantOK: true},
		{name: "character designer", value: "山田太郎 キャラクターデザイン", wantName: "山田太郎", wantRoles: []string{"キャラクターデザイン"}, wantOK: true},
		{name: "supervisor", value: "今泉忠明 監修", wantName: "今泉忠明", wantRoles: []string{"監修"}, wantOK: true},
		{name: "compound", value: "加藤晃 構成・絵", wantName: "加藤晃", wantRoles: []string{"脚本", "作画"}, wantOK: true},
		{name: "bracketed author", value: "横山都美子 [著]", wantName: "横山都美子", wantRoles: []string{"著者"}, wantOK: true},
		{name: "bracketed compound", value: "加藤晃 [構成・絵]", wantName: "加藤晃", wantRoles: []string{"脚本", "作画"}, wantOK: true},
		{name: "bracketed original creator with middle dot", value: "藤子・F・不二雄 [原作]", wantName: "藤子・F・不二雄", wantRoles: []string{"原作"}, wantOK: true},
		{name: "unbalanced opening bracket", value: "横山都美子 [著", wantOK: false},
		{name: "unbalanced closing bracket", value: "横山都美子 著]", wantOK: false},
		{name: "nested bracket", value: "横山都美子 [[著]]", wantOK: false},
		{name: "unknown", value: "不明太郎 協力", wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, roles, ok := parseCreatorLiteral(test.value)
			if name != test.wantName || ok != test.wantOK || !slicesEqual(roles, test.wantRoles) {
				t.Fatalf("parseCreatorLiteral(%q) = %q, %#v, %t", test.value, name, roles, ok)
			}
		})
	}
}

// TestMapCreatorRoleMapsCharacterDesigner は、キャラクターデザインを共通役割へ変換することを確認する
func TestMapCreatorRoleMapsCharacterDesigner(t *testing.T) {
	role, ok := mapCreatorRole("キャラクターデザイン")
	if !ok || role != "キャラクターデザイン" {
		t.Fatalf("mapCreatorRole() = %q, %t", role, ok)
	}
}

// TestCreatorsAddsCharacterDesignerToAuthorsAndContributors は、キャラクターデザイン担当者を著者と寄与者へ追加することを確認する
func TestCreatorsAddsCharacterDesignerToAuthorsAndContributors(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{{Name: xml.Name{Space: nsDC, Local: "creator"}, Text: "山田太郎 キャラクターデザイン"}}}
	authors, contributors := creators(record)
	if !slicesEqual(authors, []string{"山田太郎"}) {
		t.Fatalf("authors = %#v", authors)
	}
	if len(contributors) != 1 || contributors[0].Name != "山田太郎" || !slicesEqual(contributors[0].Roles, []string{"キャラクターデザイン"}) {
		t.Fatalf("contributors = %#v", contributors)
	}
}

// TestCreatorsFallsBackToAuthorityNames は、利用可能なdc:creatorがない場合だけ典拠形を著者に使うことを検証する
func TestCreatorsFallsBackToAuthorityNames(t *testing.T) {
	record := &xmlNode{Children: []*xmlNode{
		{Name: xml.Name{Space: nsDC, Local: "creator"}, Text: "藤子・F・不二雄 協力"},
		{Name: xml.Name{Space: nsDCTerms, Local: "creator"}, Children: []*xmlNode{{Name: xml.Name{Space: nsFOAF, Local: "Agent"}, Children: []*xmlNode{{Name: xml.Name{Space: nsFOAF, Local: "name"}, Text: "藤子, 不二雄F, 1933-1996"}}}}},
	}}
	authors, contributors := creators(record)
	if !slicesEqual(authors, []string{"藤子, 不二雄F, 1933-1996"}) || contributors != nil {
		t.Fatalf("creators() = %#v, %#v", authors, contributors)
	}
}

// slicesEqual は、順序を含めて同じ要素列か判定する
func slicesEqual[T comparable](got, want []T) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

// TestSubjectSchemeAndCodeRecognizesNDCEditionURIs は、NDC版別URIを共通Schemeへ変換することを確認する
func TestSubjectSchemeAndCodeRecognizesNDCEditionURIs(t *testing.T) {
	for _, uri := range []string{
		"http://id.ndl.go.jp/class/ndc8/726.1",
		"http://id.ndl.go.jp/class/ndc9/726.1",
		"http://id.ndl.go.jp/class/ndc10/726.1",
		"http://id.ndl.go.jp/class/ndc/726.1",
	} {
		if scheme, code := subjectSchemeAndCode(uri); scheme != "NDC" || code != "726.1" {
			t.Errorf("subjectSchemeAndCode(%q) = %q, %q", uri, scheme, code)
		}
	}
}

// TestDiagnosticsClassifyNotFoundAndUpstream は、0件Diagnosticsとその他Diagnosticsの分類を確認する
func TestDiagnosticsClassifyNotFoundAndUpstream(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		server := diagnosticServer(t, "Record does not exist")
		defer server.Close()
		result, _, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(context.Background(), []string{"9784088466361"})
		if err != nil || len(result.Items) != 1 || result.Items[0].Books == nil || len(result.Items[0].Books) != 0 {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})
	t.Run("upstream", func(t *testing.T) {
		server := diagnosticServer(t, "illegal mediaType value")
		defer server.Close()
		_, err := newTestClient(t, server.URL).SearchBooks(context.Background(), SearchRequest{Title: "日本"})
		assertErrorKind(t, err, ErrorKindUpstream)
	})
}

// TestLookupBooksByISBNMatchesSourceISBN は、ハイフン付き取得元ISBNを入力ISBNと照合する
func TestLookupBooksByISBNMatchesSourceISBN(t *testing.T) {
	body := readFixture(t, "testdata/dcndl-v3.xml")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query().Get("query")
		if strings.Contains(query, "ndc") || strings.Contains(query, "ndlc") {
			t.Errorf("ISBN lookup query contains manga filters: %q", query)
		}
		_, _ = writer.Write(body)
	}))
	defer server.Close()
	result, _, err := newTestClient(t, server.URL).LookupBooksByISBNWithRawResponse(context.Background(), []string{"9780000000002"})
	if err != nil || len(result.Items) != 1 || len(result.Items[0].Books) != 1 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if result.Items[0].RequestedISBN != "9780000000002" {
		t.Fatalf("RequestedISBN = %q", result.Items[0].RequestedISBN)
	}
}

// TestCursorBindsQueryLimitAndRange は、Cursorの検索条件、Limit、取得範囲の検証を確認する
func TestCursorBindsQueryLimitAndRange(t *testing.T) {
	cursor, err := encodeCursor(21, "query", 20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCursor(cursor, "other", 20); err == nil {
		t.Fatal("different query accepted")
	}
	if _, err := decodeCursor(cursor, "query", 10); err == nil {
		t.Fatal("different limit accepted")
	}
	if _, err := encodeCursor(501, "query", 20); err == nil {
		t.Fatal("record 501 cursor accepted")
	}
}

// TestSearchBooksRejectsCursorFromDifferentMangaFilterState は、分類フィルタ状態が異なるCursorを拒否することを確認する
func TestSearchBooksRejectsCursorFromDifferentMangaFilterState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><searchRetrieveResponse><numberOfRecords>1</numberOfRecords><nextRecordPosition>2</nextRecordPosition></searchRetrieveResponse>`))
	}))
	defer server.Close()
	filtered := newTestClient(t, server.URL)
	first, err := filtered.SearchBooks(context.Background(), SearchRequest{Title: "漫画", Limit: 1})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first result = %#v, error = %v", first, err)
	}
	unfiltered, err := NewClient(nil, WithEndpoint(server.URL), WithMangaNDCFilter(false))
	if err != nil {
		t.Fatal(err)
	}
	_, err = unfiltered.SearchBooks(context.Background(), SearchRequest{Title: "漫画", Limit: 1, Cursor: first.NextCursor})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooksWithOptionsRejectsCursorFromDifferentOptions は、異なるNDL固有検索条件のCursorを拒否することを確認する
func TestSearchBooksWithOptionsRejectsCursorFromDifferentOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><searchRetrieveResponse><numberOfRecords>1</numberOfRecords><nextRecordPosition>2</nextRecordPosition></searchRetrieveResponse>`))
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	first, err := client.SearchBooksWithOptions(context.Background(), SearchRequest{Title: "漫画", Limit: 1}, SearchOptions{Description: "ハムテル"})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first result = %#v, error = %v", first, err)
	}
	_, err = client.SearchBooksWithOptions(context.Background(), SearchRequest{Title: "漫画", Limit: 1, Cursor: first.NextCursor}, SearchOptions{Description: "チョビ"})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
}

// TestSearchBooksWithOptionsRejectsInvalidDatesBeforeHTTP は、不正なNDL日付条件でHTTP要求を送信しないことを確認する
func TestSearchBooksWithOptionsRejectsInvalidDatesBeforeHTTP(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	_, err := newTestClient(t, server.URL).SearchBooksWithOptions(context.Background(), SearchRequest{DateFrom: "2024-02-30"}, SearchOptions{})
	assertErrorKind(t, err, ErrorKindInvalidArgument)
	if called {
		t.Fatal("HTTP request was sent")
	}
}

// TestHTTPAndTransportBehavior は、HTTP分類、Retry-After、redirect抑止、URL秘匿を確認する
func TestHTTPAndTransportBehavior(t *testing.T) {
	t.Run("retry after", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Retry-After", "10")
			writer.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()
		_, err := newTestClient(t, server.URL).SearchBooks(context.Background(), SearchRequest{Title: "日本"})
		var classified *Error
		if !errors.As(err, &classified) || classified.Kind != ErrorKindUnavailable || classified.RetryAfter != 10*time.Second {
			t.Fatalf("error = %#v", err)
		}
	})
	t.Run("redirect", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, "https://example.invalid/", http.StatusFound)
		}))
		defer server.Close()
		_, err := newTestClient(t, server.URL).SearchBooks(context.Background(), SearchRequest{Title: "日本"})
		assertErrorKind(t, err, ErrorKindUpstream)
	})
	t.Run("transport", func(t *testing.T) {
		client, err := NewClient(&http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, &url.Error{URL: "https://ndlsearch.ndl.go.jp/api/sru?query=秘密語", Err: errors.New("connection refused")}
		})})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.SearchBooks(context.Background(), SearchRequest{Title: "秘密語"})
		assertErrorKind(t, err, ErrorKindUnavailable)
		if strings.Contains(err.Error(), "秘密語") {
			t.Fatalf("transport error exposes query: %v", err)
		}
	})
}

// newTestClient は、テスト用loopback endpointのClientを生成する
func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	client, err := NewClient(nil, WithEndpoint(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// diagnosticServer は、指定されたSRU Diagnosticsを返すテストサーバーを生成する
func diagnosticServer(t *testing.T, message string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><srw:searchRetrieveResponse xmlns:srw="http://www.loc.gov/zing/srw/" xmlns:diag="http://www.loc.gov/zing/srw/diagnostic/"><srw:diagnostics><diag:diagnostic><diag:uri>info:srw/diagnostic/1/1</diag:uri><diag:details>An error occurred</diag:details><diag:message>` + message + `</diag:message></diag:diagnostic></srw:diagnostics></srw:searchRetrieveResponse>`))
	}))
}

// readFixture は、テストデータを読み込む
func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// containsString は、文字列配列に指定値が含まれるか判定する
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// hasSubject は、主題配列に指定値が含まれるか判定する
func hasSubject(values []Subject, scheme, code, name string) bool {
	for _, value := range values {
		if value.Scheme == scheme && value.Code == code && value.Name == name {
			return true
		}
	}
	return false
}

// assertErrorKind は、分類済みエラーが期待する種類を持つか確認する
func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var classified *Error
	if !errors.As(err, &classified) || classified.Kind != want {
		t.Fatalf("error = %#v, want kind %q", err, want)
	}
}

// roundTripperFunc は、関数をhttp.RoundTripperとして使えるようにする
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip は、関数が表す往復処理を実行する
func (roundTripper roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTripper(request)
}
