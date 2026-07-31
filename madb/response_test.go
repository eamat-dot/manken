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

	result, err := buildSearchResult(response, "作品", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("len(Books) = %d, want 1", len(result.Books))
	}

	book := result.Books[0]
	if book.Normalized.Title != "A" || book.Normalized.Subtitle != "副題A" {
		t.Fatalf("normalized titles = %#v", book.Normalized)
	}
	assertStrings(t, book.Normalized.EditionStatements, []string{"[通常版]", "新装版"})
	assertStrings(t, book.Normalized.Authors, []string{"使わない著者"})
	assertContributors(t, book.Normalized.Contributors, []Contributor{{
		Name:  "使わない著者",
		Roles: []ContributorRole{ContributorRoleAuthor},
	}})
	assertStrings(t, book.Normalized.Publishers, []string{"出版社A", "出版社B"})
	assertStrings(t, book.Normalized.Imprints, []string{"レーベルA", "レーベルB"})
	assertIdentifiers(t, book.Normalized.Identifiers, []Identifier{
		{Type: IdentifierTypeISBN10, Value: "0306406152"},
		{Type: IdentifierTypeISBN10, Value: "080442957X"},
		{Type: IdentifierTypeISBN13, Value: "9780306406157"},
		{Type: IdentifierTypeISBN13, Value: "9783161484100"},
	})
	if book.Normalized.Volume.Number == nil || *book.Normalized.Volume.Number != 1 ||
		book.Normalized.Volume.Label != "1" {
		t.Fatalf("Volume = %#v, want number 1", book.Normalized.Volume)
	}
	if len(book.Normalized.Series) != 2 ||
		book.Normalized.Series[0].Name != "シリーズ" ||
		book.Normalized.Series[0].ID != "C1" ||
		book.Normalized.Series[0].URL != seriesResourceURI("C1") ||
		book.Normalized.Series[1].Name != "参照シリーズ" {
		t.Fatalf("Series = %#v", book.Normalized.Series)
	}
	if len(book.Sources) != 1 ||
		book.Sources[0].Source != SourceMADB ||
		book.Sources[0].ID != "M1" ||
		book.Sources[0].URL != resourceURI("M1") {
		t.Fatalf("Sources = %#v", book.Sources)
	}
	assertStrings(t, book.Sources[0].Values.Titles, []string{"A", "B"})
	assertStrings(t, book.Sources[0].Values.Subtitles, []string{"副題A", "副題B"})
	assertStrings(t, book.Sources[0].Values.SeriesNames, []string{"シリーズ", "参照シリーズ"})
	assertStrings(t, book.Sources[0].Values.Authors, []string{"[著]使わない著者"})
	assertStrings(t, book.Sources[0].Values.ISBNs, []string{
		"0-306-40615-2",
		"0-8044-2957-X",
		"978-0-306-40615-7",
		"978-3-16-148410-0",
		"invalid",
	})
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

	result, err := buildSearchResult(response, "動物のお医者さん", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	book := result.Books[0]
	if book.Normalized.TitleKana != "ドウブツノオイシャサン" {
		t.Fatalf("TitleKana = %q", book.Normalized.TitleKana)
	}
	assertStrings(t, book.Normalized.Publishers, []string{"白泉社"})
	if book.Normalized.PageCount == nil || *book.Normalized.PageCount != 197 {
		t.Fatalf("PageCount = %#v", book.Normalized.PageCount)
	}
	if book.Normalized.Medium != PublicationMediumPrint ||
		book.Normalized.PhysicalSize == nil ||
		book.Normalized.PhysicalSize.HeightMM == nil ||
		*book.Normalized.PhysicalSize.HeightMM != 173 ||
		book.Normalized.PhysicalSize.WidthMM == nil ||
		*book.Normalized.PhysicalSize.WidthMM != 106 {
		t.Fatalf("physical fields = medium %q, size %#v", book.Normalized.Medium, book.Normalized.PhysicalSize)
	}

	source := book.Sources[0].Values
	assertStrings(t, source.TitleKana, []string{
		"ドウブツ ノ オイシャサン",
		"ドウブツ ノ オイシヤサン",
		"ドウブツノオイシャサン",
	})
	assertStrings(t, source.Publishers, []string{"白泉社", "白泉社　∥　ハクセンシャ"})
	if source.PageCount == nil || *source.PageCount != 197 || source.Size != "17.3cm　×　10.6cm" {
		t.Fatalf("source fields = %#v", source)
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

// TestNormalizePhysicalSize は、MADBのセンチメートル表記をミリメートルへ変換する
func TestNormalizePhysicalSize(t *testing.T) {
	got := normalizePhysicalSize("17.3cm　×　10.6cm")
	if got == nil || got.HeightMM == nil || *got.HeightMM != 173 ||
		got.WidthMM == nil || *got.WidthMM != 106 {
		t.Fatalf("normalizePhysicalSize() = %#v", got)
	}
	if got := normalizePhysicalSize("四六判"); got != nil {
		t.Fatalf("normalizePhysicalSize(unknown) = %#v, want nil", got)
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

	result, err := buildSearchResult(response, "作品", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	book := result.Books[0]
	assertStrings(t, book.Normalized.Authors, []string{"佐々木倫子"})
	assertContributors(t, book.Normalized.Contributors, []Contributor{
		{Name: "佐々木倫子", Roles: []ContributorRole{ContributorRoleAuthor}},
		{Name: "藤原新也", Roles: []ContributorRole{ContributorRoleCommentator}},
	})
	assertStrings(t, book.Sources[0].Values.Authors, []string{
		"[著]佐々木倫子",
		"[解説]藤原新也",
	})
	if book.Normalized.Title != "" || book.Normalized.Subtitle != "" ||
		len(book.Normalized.Series) != 0 || len(book.Normalized.Identifiers) != 0 {
		t.Fatalf("Normalized = %#v, want omitted optional values", book.Normalized)
	}

	encoded, err := json.Marshal(book)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, field := range []string{"title", "subtitle", "series", "edition_statements", "imprints"} {
		if strings.Contains(string(encoded), `"`+field+`"`) {
			t.Fatalf("JSON contains missing field %s: %s", field, encoded)
		}
	}
	if !strings.Contains(string(encoded), `"normalized":{`) ||
		!strings.Contains(string(encoded), `"sources":[`) {
		t.Fatalf("JSON omits required containers: %s", encoded)
	}
}

// TestParseCreator は、既知役割、角括弧付き人名、未知または不正な値を検証する
func TestParseCreator(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantName  string
		wantRoles []ContributorRole
		wantOK    bool
	}{
		{name: "author", value: "[著]佐藤花子", wantName: "佐藤花子", wantRoles: []ContributorRole{ContributorRoleAuthor}, wantOK: true},
		{name: "compound", value: "[原作・監修]山田太郎", wantName: "山田太郎", wantRoles: []ContributorRole{ContributorRoleOriginalCreator, ContributorRoleSupervisor}, wantOK: true},
		{name: "author and artist", value: "[作・画]鈴木一郎", wantName: "鈴木一郎", wantRoles: []ContributorRole{ContributorRoleAuthor, ContributorRoleArtist}, wantOK: true},
		{name: "bracketed full name", value: "[作画][田辺節雄]", wantName: "田辺節雄", wantRoles: []ContributorRole{ContributorRoleArtist}, wantOK: true},
		{name: "bracketed given part", value: "[画][葛飾]北斎", wantName: "葛飾北斎", wantRoles: []ContributorRole{ContributorRoleArtist}, wantOK: true},
		{name: "roleless", value: "Arinco", wantName: "Arinco", wantOK: true},
		{name: "unknown role", value: "[協力]佐藤花子"},
		{name: "unknown compound part", value: "[監修・協力]佐藤花子"},
		{name: "malformed brackets", value: "[[著]]近江のこ"},
		{name: "missing role end", value: "[著佐藤花子"},
		{name: "missing name end", value: "[著][佐藤花子"},
		{name: "empty name", value: "[著]"},
		{name: "empty", value: "　"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotName, gotRoles, gotOK := parseCreator(test.value)
			if gotName != test.wantName || gotOK != test.wantOK {
				t.Fatalf("parseCreator(%q) = name %q, roles %#v, ok %t", test.value, gotName, gotRoles, gotOK)
			}
			assertContributorRoles(t, gotRoles, test.wantRoles)
		})
	}
}

// TestMapCreatorRole は、確定したMADB役割を11種類の共通役割へ変換する
func TestMapCreatorRole(t *testing.T) {
	tests := []struct {
		role   ContributorRole
		values []string
	}{
		{role: ContributorRoleAuthor, values: []string{"著", "著者", "作", "共著", "ほか著", "他著"}},
		{role: ContributorRoleOriginalCreator, values: []string{"原作", "原案", "共原作"}},
		{role: ContributorRoleWriter, values: []string{"脚本", "シナリオ", "構成", "脚色", "文", "ストーリー", "ライター"}},
		{role: ContributorRoleArtist, values: []string{
			"漫画", "作画", "画", "劇画", "まんが", "絵",
			"comic", "Comic", "COMIC", "comics", "コミック", "マンガ", "アーティスト",
		}},
		{role: ContributorRoleCharacterCreator, values: []string{"キャラクター原案"}},
		{role: ContributorRoleCharacterDesigner, values: []string{"キャラクターデザイン"}},
		{role: ContributorRoleEditor, values: []string{"編", "編集"}},
		{role: ContributorRoleTranslator, values: []string{"訳"}},
		{role: ContributorRoleSupervisor, values: []string{"監修"}},
		{role: ContributorRoleCommentator, values: []string{"解説"}},
		{role: ContributorRoleDesigner, values: []string{"カバーデザイン", "装丁", "装幀", "デザイン"}},
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
	sourceAuthors, authors, contributors := convertCreators(sourceBook{
		Creators: []string{
			"[作画][田辺節雄]",
			"[原作]山田太郎",
			"[監修]山田太郎",
			"[解説]藤原新也",
			"役割なし",
		},
		AgentNames: []string{"使用しないAgent"},
	})

	assertStrings(t, sourceAuthors, []string{
		"[作画][田辺節雄]",
		"[原作]山田太郎",
		"[監修]山田太郎",
		"[解説]藤原新也",
		"役割なし",
	})
	assertStrings(t, authors, []string{"田辺節雄", "山田太郎", "役割なし"})
	assertContributors(t, contributors, []Contributor{
		{Name: "田辺節雄", Roles: []ContributorRole{ContributorRoleArtist}},
		{Name: "山田太郎", Roles: []ContributorRole{ContributorRoleOriginalCreator, ContributorRoleSupervisor}},
		{Name: "藤原新也", Roles: []ContributorRole{ContributorRoleCommentator}},
	})
}

// TestConvertCreators_AgentFallback は、creatorがない場合だけAgent名を著者に使用する
func TestConvertCreators_AgentFallback(t *testing.T) {
	sourceAuthors, authors, contributors := convertCreators(sourceBook{
		AgentNames: []string{"KotzDean", "ZubJim", "皆川由美"},
	})
	assertStrings(t, sourceAuthors, []string{"KotzDean", "ZubJim", "皆川由美"})
	assertStrings(t, authors, []string{"KotzDean", "ZubJim", "皆川由美"})
	if len(contributors) != 0 {
		t.Fatalf("Contributors = %#v, want empty", contributors)
	}
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
			_, err := buildSearchResult(response, "作品", 20)
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
	result, err := buildSearchResult(newSPARQLResponse(binding), "作品", 20)
	if err != nil {
		t.Fatalf("buildSearchResult() error = %v", err)
	}
	if len(result.Books) != 1 {
		t.Fatalf("len(Books) = %d, want 1", len(result.Books))
	}
}

// TestISBNValidation は、ISBN-10とISBN-13のチェックディジットを検証する
func TestISBNValidation(t *testing.T) {
	tests := []struct {
		value  string
		isbn10 bool
		isbn13 bool
	}{
		{value: "080442957X", isbn10: true},
		{value: "080442957x"},
		{value: "0804429570"},
		{value: "9780306406157", isbn13: true},
		{value: "9780306406150"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if got := isValidISBN10(test.value); got != test.isbn10 {
				t.Fatalf("isValidISBN10() = %t, want %t", got, test.isbn10)
			}
			if got := isValidISBN13(test.value); got != test.isbn13 {
				t.Fatalf("isValidISBN13() = %t, want %t", got, test.isbn13)
			}
		})
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

// assertIdentifiers は、識別子スライスの内容と順序を検証する
func assertIdentifiers(t *testing.T, got, want []Identifier) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("identifiers = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("identifiers = %#v, want %#v", got, want)
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
func assertContributorRoles(t *testing.T, got, want []ContributorRole) {
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
