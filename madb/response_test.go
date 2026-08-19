package madb

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildSearchResult_AggregatesAndConverts は、複数bindingの集約と共通モデル変換を検証する
func TestBuildSearchResult_AggregatesAndConverts(t *testing.T) {
	response := newSPARQLResponse(
		testBinding("M1", map[string]string{
			"id":                "M1",
			"title":             "B",
			"subtitle":          "副題B",
			"seriesName":        "シリーズ",
			"relatedSeriesName": "参照シリーズ",
			"seriesID":          "C1",
			"volumeNumber":      "1",
			"version":           "新装版",
			"creator":           "[著]使わない著者",
			"agentName":         "著者B",
			"publisher":         "出版社B",
			"brand":             "レーベルB",
			"isbn":              "0-8044-2957-X",
			"publishedDate":     "2026-07",
		}),
		testBinding("M1", map[string]string{
			"title":             "A",
			"subtitle":          "副題A",
			"relatedSeriesName": "シリーズ",
			"seriesID":          "C1",
			"agentName":         "著者A",
			"publisher":         "出版社A",
			"version":           "[通常版]",
			"brand":             "レーベルA",
			"isbn":              "978-0-306-40615-7",
		}),
		testBinding("M1", map[string]string{
			"isbn": "0-306-40615-2",
		}),
		testBinding("M1", map[string]string{
			"isbn": "978-3-16-148410-0",
		}),
		testBinding("M1", map[string]string{
			"title":     "A",
			"agentName": "著者A",
			"isbn":      "invalid",
		}),
	)
	response.Results.Bindings[0]["seriesResource"] = sparqlValue{
		Type:  "uri",
		Value: seriesResourceURI("C1"),
	}
	response.Results.Bindings[1]["seriesResource"] = sparqlValue{
		Type:  "uri",
		Value: seriesResourceURI("C1"),
	}

	result, err := buildSearchResult(response, searchConditions{Title: "作品"}, 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("len(Books) = %d, want 1", len(result.Books))
	}

	book := result.Books[0]
	if book.Title != "A" || book.Subtitle != "副題A" {
		t.Fatalf("titles = %#v", book)
	}
	assertStrings(t, book.Editions, []string{"[通常版]", "新装版"})
	assertStrings(t, book.Authors, []string{"使わない著者"})
	assertContributors(t, book.Contributors, []Contributor{{
		Name:  "使わない著者",
		Roles: []string{"著者"},
	}})
	assertStrings(t, book.Publishers, []string{"出版社A", "出版社B"})
	assertStrings(t, book.PublicationSeries, []string{"レーベルA", "レーベルB"})
	assertStrings(t, book.ISBN10, []string{"0306406152", "080442957X"})
	assertStrings(t, book.ISBN13, []string{"9780306406157", "9783161484100"})
	if book.PublishedDate != "2026-07" {
		t.Fatalf("PublishedDate = %q, want 2026-07", book.PublishedDate)
	}
	if book.Volume.Number == nil || *book.Volume.Number != 1 || book.Volume.Label != "1" {
		t.Fatalf("Volume = %#v, want number 1", book.Volume)
	}
	if len(book.BookSeries) != 2 ||
		book.BookSeries[0].Name != "シリーズ" ||
		book.BookSeries[0].ID != "C1" ||
		book.BookSeries[0].URL != seriesResourceURI("C1") ||
		book.BookSeries[0].Source != SourceMADB ||
		book.BookSeries[1].Name != "参照シリーズ" ||
		book.BookSeries[1].ID != "C1" ||
		book.BookSeries[1].URL != seriesResourceURI("C1") ||
		book.BookSeries[1].Source != SourceMADB {
		t.Fatalf("BookSeries = %#v", book.BookSeries)
	}
	if len(book.Sources) != 1 ||
		book.Sources[0].Source != SourceMADB ||
		book.Sources[0].ID != "M1" ||
		book.Sources[0].URL != resourceURI("M1") {
		t.Fatalf("Sources = %#v", book.Sources)
	}
}

// TestConvertBook_PreservesTitleWithVolumeAndEdition は、巻数と版表示を抽出しても取得元タイトルを変更しないことを検証する
func TestConvertBook_PreservesTitleWithVolumeAndEdition(t *testing.T) {
	book := convertBook(sourceBook{
		Titles:       []string{"作品 第1巻 新装版"},
		VolumeNumber: "1",
		Versions:     []string{"新装版"},
	})

	if book.Title != "作品 第1巻 新装版" {
		t.Fatalf("Title = %q, want source title unchanged", book.Title)
	}
	if book.Volume.Number == nil || *book.Volume.Number != 1 {
		t.Fatalf("Volume = %#v, want 1", book.Volume)
	}
	assertStrings(t, book.Editions, []string{"新装版"})
}

// TestConvertBook_PrefersExplicitMetadata は、MADBの明示巻数と版表示をタイトル候補より優先することを検証する
func TestConvertBook_PrefersExplicitMetadata(t *testing.T) {
	book := convertBook(sourceBook{Titles: []string{"作品 第2巻 完全版"}, VolumeNumber: "7", Versions: []string{"特装版"}})
	if book.Volume.Number == nil || *book.Volume.Number != 7 || book.Volume.Label != "7" {
		t.Fatalf("Volume = %#v", book.Volume)
	}
	assertStrings(t, book.Editions, []string{"特装版"})
}

// TestBuildSearchResult_MADBAdditionalFields は、タイトル読み、ページ数、大きさ、出版社の変換を検証する
func TestBuildSearchResult_MADBAdditionalFields(t *testing.T) {
	response := newSPARQLResponse(
		testBinding("M1", map[string]string{
			"title":     "動物のお医者さん",
			"titleKana": "ドウブツ ノ オイシャサン",
			"publisher": "白泉社",
			"pageCount": "197p",
			"size":      "17.3cm　×　10.6cm",
		}),
		testBinding("M1", map[string]string{
			"titleKana": "ドウブツノオイシャサン",
			"publisher": "白泉社　∥　ハクセンシャ",
		}),
		testBinding("M1", map[string]string{
			"titleKana": "ドウブツ ノ オイシヤサン",
		}),
	)

	result, err := buildSearchResult(response, searchConditions{Title: "動物のお医者さん"}, 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	book := result.Books[0]
	if book.TitleReading != "ドウブツノオイシャサン" {
		t.Fatalf("TitleReading = %q", book.TitleReading)
	}
	assertStrings(t, book.Publishers, []string{"白泉社"})
	if book.PageCount == nil || *book.PageCount != 197 {
		t.Fatalf("PageCount = %#v", book.PageCount)
	}
	if book.Medium != PublicationMediumPrint || book.Size != "17.3cm　×　10.6cm" {
		t.Fatalf("physical fields = medium %q, size %q", book.Medium, book.Size)
	}

}

// TestHasStructuredPhysicalSize_AcceptsMultipleDecimalPlaces は、cm寸法の小数部を1桁以上受け付けることを検証する
func TestHasStructuredPhysicalSize_AcceptsMultipleDecimalPlaces(t *testing.T) {
	for _, value := range []string{"17.3cm", "17.35cm", "17.35cm×10.625cm", "１７．３５cm＊１０．６２５cm"} {
		if !hasStructuredPhysicalSize(value) {
			t.Fatalf("hasStructuredPhysicalSize(%q) = false", value)
		}
	}
}

// TestConvertBook_PreservesUnstructuredSize は、判型名をSizeへ保持して紙書籍媒体を推測しないことを検証する
func TestConvertBook_PreservesUnstructuredSize(t *testing.T) {
	book := convertBook(sourceBook{Size: "四六判"})
	if book.Size != "四六判" {
		t.Fatalf("Size = %q, want raw MADB size", book.Size)
	}
	if book.Medium != PublicationMediumUnknown {
		t.Fatalf("Medium = %q, want unknown", book.Medium)
	}
}

// TestNormalizePublishers は、カナ読みだけを除去し任意の併記は維持することを検証する
func TestNormalizePublishers(t *testing.T) {
	got := normalizePublishers([]string{
		"白泉社",
		"白泉社　∥　ハクセンシャ",
		"発行元 ∥ 発売元",
	})
	assertStrings(t, got, []string{"発行元 ∥ 発売元", "白泉社"})
}

// TestNormalizeTitleKana_Ambiguous は、最多候補を決められない読みを空にすることを検証する
func TestNormalizeTitleKana_Ambiguous(t *testing.T) {
	if got := normalizeTitleKana([]string{"サクヒン", "サクヒーン"}); got != "" {
		t.Fatalf("normalizeTitleKana() = %q, want empty", got)
	}
}

// TestNormalizePageCount は、ページ数として安全に解釈できる表記だけを変換する
func TestNormalizePageCount(t *testing.T) {
	for _, test := range []struct {
		value string
		want  *int
	}{
		{value: "197p", want: intPointer(197)},
		{value: "１９７Ｐ", want: intPointer(197)},
		{value: "197", want: intPointer(197)},
		{value: "xii, 197p"},
	} {
		if got := normalizePageCount(test.value); !equalOptionalInts(got, test.want) {
			t.Fatalf("normalizePageCount(%q) = %#v, want %#v", test.value, got, test.want)
		}
	}
}

// TestBuildSearchResult_CreatorRolesAndMissingValues は、creator役割変換と欠落値を検証する
func TestBuildSearchResult_CreatorRolesAndMissingValues(t *testing.T) {
	response := newSPARQLResponse(
		testBinding("M1", map[string]string{
			"creator":   "[著]佐々木倫子",
			"agentName": "佐々木倫子",
		}),
		testBinding("M1", map[string]string{
			"creator":   "[解説]藤原新也",
			"agentName": "藤原新也",
		}),
	)

	result, err := buildSearchResult(response, searchConditions{Title: "作品"}, 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	book := result.Books[0]
	assertStrings(t, book.Authors, []string{"佐々木倫子"})
	assertContributors(t, book.Contributors, []Contributor{
		{Name: "佐々木倫子", Roles: []string{"著者"}},
		{Name: "藤原新也", Roles: []string{"解説"}},
	})
	for _, contributor := range book.Contributors {
		if contributor.Reading != "" {
			t.Fatalf("Contributor.Reading = %q, want empty", contributor.Reading)
		}
	}
	if book.Title != "" || book.Subtitle != "" || len(book.BookSeries) != 0 || len(book.ISBN10) != 0 || len(book.ISBN13) != 0 {
		t.Fatalf("Book = %#v, want omitted optional values", book)
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, field := range []string{"title", "subtitle", "series", "editions"} {
		if strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("JSON contains missing field %s: %s", field, encoded)
		}
	}
	if !strings.Contains(string(encoded), `"sources":[`) {
		t.Fatalf("JSON omits sources: %s", encoded)
	}
}

// TestParseCreator は、既知役割、角括弧付き人名、未知または不正な値を検証する
func TestParseCreator(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantName  string
		wantRoles []string
		hasRole   bool
		wantOK    bool
	}{
		{name: "author", value: "[著]佐藤花子", wantName: "佐藤花子", wantRoles: []string{"著者"}, hasRole: true, wantOK: true},
		{name: "compound", value: "[原作・監修]山田太郎", wantName: "山田太郎", wantRoles: []string{"原作", "監修"}, hasRole: true, wantOK: true},
		{name: "author and artist", value: "[作・画]鈴木一郎", wantName: "鈴木一郎", wantRoles: []string{"著者", "作画"}, hasRole: true, wantOK: true},
		{name: "bracketed full name", value: "[作画][田辺節雄]", wantName: "田辺節雄", wantRoles: []string{"作画"}, hasRole: true, wantOK: true},
		{name: "bracketed given part", value: "[画][葛飾]北斎", wantName: "葛飾北斎", wantRoles: []string{"作画"}, hasRole: true, wantOK: true},
		{name: "roleless", value: "Arinco", wantName: "Arinco", wantOK: true},
		{name: "unknown role", value: "[協力]佐藤花子", wantName: "佐藤花子", hasRole: true, wantOK: true},
		{name: "unknown compound part", value: "[監修・協力]佐藤花子", wantName: "佐藤花子", hasRole: true, wantOK: true},
		{name: "malformed brackets", value: "[[著]]近江のこ"},
		{name: "missing role end", value: "[著佐藤花子"},
		{name: "missing name end", value: "[著][佐藤花子"},
		{name: "empty name", value: "[著]"},
		{name: "empty", value: "　"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotName, gotRoles, gotHasRole, gotOK := parseCreator(test.value)
			if gotName != test.wantName || gotHasRole != test.hasRole || gotOK != test.wantOK {
				t.Fatalf("parseCreator(%q) = name %q, roles %#v, hasRole %t, ok %t", test.value, gotName, gotRoles, gotHasRole, gotOK)
			}
			assertContributorRoles(t, gotRoles, test.wantRoles)
		})
	}
}

// TestMapCreatorRole は、確定したMADB役割を11種類の共通役割へ変換する
func TestMapCreatorRole(t *testing.T) {
	tests := []struct {
		role   string
		values []string
	}{
		{role: "著者", values: []string{"著", "著者", "作", "共著", "ほか著", "他著"}},
		{role: "原作", values: []string{"原作", "原案", "共原作"}},
		{role: "脚本", values: []string{"脚本", "シナリオ", "構成", "脚色", "文", "ストーリー", "ライター"}},
		{role: "作画", values: []string{
			"漫画", "作画", "画", "劇画", "まんが", "絵",
			"comic", "Comic", "COMIC", "comics", "コミック", "マンガ", "アーティスト",
		}},
		{role: "キャラクター原案", values: []string{"キャラクター原案"}},
		{role: "キャラクターデザイン", values: []string{"キャラクターデザイン"}},
		{role: "編集", values: []string{"編", "編集"}},
		{role: "翻訳", values: []string{"訳"}},
		{role: "監修", values: []string{"監修"}},
		{role: "解説", values: []string{"解説"}},
		{role: "デザイン", values: []string{"カバーデザイン", "装丁", "装幀", "デザイン"}},
	}
	for _, test := range tests {
		for _, value := range test.values {
			got, ok := mapCreatorRole(value)
			if !ok || got != test.role {
				t.Fatalf("mapCreatorRole(%q) = %q, %t, want %q, true", value, got, ok, test.role)
			}
		}
	}
	if got, ok := mapCreatorRole("協力"); ok || got != "" {
		t.Fatalf("mapCreatorRole(unknown) = %q, %t, want empty, false", got, ok)
	}
}

// TestConvertCreators_MergesRolesAndKeepsStableOrder は、著者順と同名寄与者の役割統合を検証する
func TestConvertCreators_MergesRolesAndKeepsStableOrder(t *testing.T) {
	authors, contributors := convertCreators(sourceBook{
		Creators: []string{
			"[作画][田辺節雄]",
			"[原作]山田太郎",
			"[監修]山田太郎",
			"[解説]藤原新也",
			"役割なし",
			"[協力]佐藤花子",
			"[監修・協力]鈴木一郎",
		},
		AgentNames: []string{"使用しないAgent"},
	})

	assertStrings(t, authors, []string{"田辺節雄", "山田太郎", "役割なし"})
	assertContributors(t, contributors, []Contributor{
		{Name: "田辺節雄", Roles: []string{"作画"}},
		{Name: "山田太郎", Roles: []string{"原作", "監修"}},
		{Name: "藤原新也", Roles: []string{"解説"}},
		{Name: "役割なし"},
		{Name: "佐藤花子"},
		{Name: "鈴木一郎"},
	})
}

// TestConvertCreators_AgentFallback は、creatorがない場合だけAgent名を著者に使用する
func TestConvertCreators_AgentFallback(t *testing.T) {
	authors, contributors := convertCreators(sourceBook{
		AgentNames: []string{"KotzDean", "ZubJim", "皆川由美"},
	})
	assertStrings(t, authors, []string{"KotzDean", "ZubJim", "皆川由美"})
	assertContributors(t, contributors, []Contributor{{Name: "KotzDean"}, {Name: "ZubJim"}, {Name: "皆川由美"}})
}

// TestNormalizeVolume は、許可した巻数表記と解析対象外の表記を検証する
func TestNormalizeVolume(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		number *int
		label  string
	}{
		{name: "ASCII integer", value: "16", number: intPointer(16), label: "16"},
		{name: "full-width integer", value: "１６", number: intPointer(16), label: "16"},
		{name: "numbered volume", value: "第8巻", number: intPointer(8), label: "8"},
		{name: "volume prefix", value: "Vol. 04", number: intPointer(4), label: "4"},
		{name: "hash prefix", value: "＃2", number: intPointer(2), label: "2"},
		{name: "parentheses", value: "（3）", number: intPointer(3), label: "3"},
		{name: "repeated same volume", value: "7 ／ 第7巻", number: intPointer(7), label: "7"},
		{name: "zero volume", value: "0巻", number: intPointer(0), label: "0"},
		{name: "upper", value: "上巻", label: "上"},
		{name: "middle", value: "（中巻）", label: "中"},
		{name: "last part", value: "後編", label: "後編"},
		{name: "decimal", value: "１．５巻", label: "1.5"},
		{name: "extra", value: "別巻", label: "別巻"},
		{name: "side story", value: "外伝", label: "外伝"},
		{name: "bonus", value: "番外編", label: "番外編"},
		{name: "range", value: "1-2"},
		{name: "different repeated volumes", value: "1／2"},
		{name: "missing value after separator", value: "1/"},
		{name: "missing value before separator", value: "/1"},
		{name: "repeated separators", value: "1//1"},
		{name: "month range", value: "7-12月"},
		{name: "four digits", value: "2024"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeVolume(test.value)
			if got.Label != test.label || !equalOptionalInts(got.Number, test.number) {
				t.Fatalf("normalizeVolume(%q) = %#v, want number %#v label %q", test.value, got, test.number, test.label)
			}
		})
	}
}

// intPointer は、テスト用のintポインターを返す
func intPointer(value int) *int {
	return &value
}

// equalOptionalInts は、nilを含むintポインターの値が同じか判定する
func equalOptionalInts(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// TestBuildSearchResult_RejectsInvalidBindings は、不正なresource、型、競合単一値を拒否する
func TestBuildSearchResult_RejectsInvalidBindings(t *testing.T) {
	tests := []struct {
		name     string
		bindings []map[string]sparqlValue
	}{
		{name: "missing resource", bindings: []map[string]sparqlValue{{"title": {Type: "literal", Value: "作品"}}}},
		{name: "invalid resource type", bindings: []map[string]sparqlValue{{
			"resource": {Type: "literal", Value: resourceURI("M1")},
		}}},
		{name: "invalid known type", bindings: []map[string]sparqlValue{{
			"resource": {Type: "uri", Value: resourceURI("M1")},
			"title":    {Type: "uri", Value: "https://example.test"},
		}}},
		{name: "conflicting scalar", bindings: []map[string]sparqlValue{
			testBinding("M1", map[string]string{"volumeNumber": "1"}),
			testBinding("M1", map[string]string{"volumeNumber": "2"}),
		}},
		{name: "invalid series resource type", bindings: []map[string]sparqlValue{{
			"resource":       {Type: "uri", Value: resourceURI("M1")},
			"seriesResource": {Type: "literal", Value: seriesResourceURI("C1")},
		}}},
		{name: "invalid series resource URI", bindings: []map[string]sparqlValue{{
			"resource":       {Type: "uri", Value: resourceURI("M1")},
			"seriesResource": {Type: "uri", Value: resourceURI("M2")},
		}}},
		{name: "conflicting series references", bindings: []map[string]sparqlValue{
			seriesBinding("M1", "C1", nil),
			seriesBinding("M1", "C2", nil),
		}},
		{name: "conflicting series IDs", bindings: []map[string]sparqlValue{
			seriesBinding("M1", "C1", map[string]string{"seriesID": "C1"}),
			seriesBinding("M1", "C1", map[string]string{"seriesID": "C2"}),
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := newSPARQLResponse(test.bindings...)
			_, err := buildSearchResult(response, searchConditions{Title: "作品"}, 20)
			assertErrorKind(t, err, ErrorKindInvalidResponse)
		})
	}
}

// seriesBinding は、参照先シリーズを含むテスト用bindingを生成する
func seriesBinding(bookID, seriesID string, fields map[string]string) map[string]sparqlValue {
	binding := testBinding(bookID, fields)
	binding["seriesResource"] = sparqlValue{
		Type:  "uri",
		Value: seriesResourceURI(seriesID),
	}
	return binding
}

// seriesResourceURI は、シリーズIDからテスト用リソースURIを生成する
func seriesResourceURI(id string) string {
	return "https://mediaarts-db.artmuseums.go.jp/id/" + id
}

// TestBuildSearchResult_IgnoresUnknownVariables は、未知変数を無視することを検証する
func TestBuildSearchResult_IgnoresUnknownVariables(t *testing.T) {
	binding := testBinding("M1", map[string]string{"title": "作品"})
	binding["futureField"] = sparqlValue{Type: "uri", Value: "https://example.test/value"}
	result, err := buildSearchResult(newSPARQLResponse(binding), searchConditions{Title: "作品"}, 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("len(Books) = %d, want 1", len(result.Books))
	}
}

// assertStrings は、文字列スライスの内容と順序を検証する
func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("values = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("values = %#v, want %#v", got, want)
		}
	}
}

// assertContributors は、寄与者の名前、役割、順序を検証する
func assertContributors(t *testing.T, got, want []Contributor) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("contributors = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index].Name != want[index].Name {
			t.Fatalf("contributors = %#v, want %#v", got, want)
		}
		assertContributorRoles(t, got[index].Roles, want[index].Roles)
	}
}

// assertContributorRoles は、寄与者役割の内容と順序を検証する
func assertContributorRoles(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("roles = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("roles = %#v, want %#v", got, want)
		}
	}
}
