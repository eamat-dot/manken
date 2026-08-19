package openbd

import "testing"

// TestConvertBook_SummaryFallback は、ONIX項目がない代替書誌をsummaryから変換する
func TestConvertBook_SummaryFallback(t *testing.T) {
	book := convertBook(responseBook{
		Summary: responseSummary{
			ISBN:      "9784088466361",
			Title:     "欠落項目を持つ書籍",
			Author:    "著者A,著者B",
			Publisher: "出版社",
			PubDate:   "2026-08",
			Cover:     "https://example.test/cover.jpg",
			Series:    "要約叢書",
		},
	}, "9784088466361")

	if book.Title != "欠落項目を持つ書籍" {
		t.Fatalf("Title = %q", book.Title)
	}
	assertStrings(t, book.Authors, []string{"著者A,著者B"})
	assertStrings(t, book.Publishers, []string{"出版社"})
	if len(book.PublicationSeries) != 1 || book.PublicationSeries[0] != "要約叢書" {
		t.Fatalf("PublicationSeries = %#v", book.PublicationSeries)
	}
	if book.PublishedDate != "2026-08" {
		t.Fatalf("PublishedDate = %q", book.PublishedDate)
	}
	if book.CoverURL != "https://example.test/cover.jpg" {
		t.Fatalf("CoverURL = %q", book.CoverURL)
	}
	if len(book.ISBN13) != 1 || book.ISBN13[0] != "9784088466361" {
		t.Fatalf("ISBN13 = %#v", book.ISBN13)
	}
}

// TestApplyTitleMetadata_GuardsParallelTrailingNumber は、並列タイトルの曖昧な末尾数値を巻数にしないことを確認する
func TestApplyTitleMetadata_GuardsParallelTrailingNumber(t *testing.T) {
	book := Book{Title: "作品 = WORK. 1", TitleReading: "サクヒン = ワーク. 1"}
	applyTitleMetadata(&book)
	if book.Volume.Number != nil || book.Volume.Label != "" || book.Title != "作品 = WORK. 1" || book.TitleReading != "サクヒン = ワーク. 1" {
		t.Fatalf("book = %#v", book)
	}

	book = Book{Title: "作品 = WORK 第1巻"}
	applyTitleMetadata(&book)
	if book.Volume.Number == nil || *book.Volume.Number != 1 {
		t.Fatalf("book = %#v", book)
	}
}

// TestConvertBook_PreservesTitleElementPair は、同じ商品階層タイトル要素の本文と読みを変更せずに設定する
func TestConvertBook_PreservesTitleElementPair(t *testing.T) {
	response, err := decodeResponse([]byte(`[
		{
			"onix": {
				"DescriptiveDetail": {
					"TitleDetail": {
						"TitleType": "01",
						"TitleElement": [
							{"TitleElementLevel": "01", "TitleText": {"content": "", "collationkey": "対応しないヨミ"}},
							{"TitleElementLevel": "01", "TitleText": {"content": "主題 = MAIN TITLE. 4", "collationkey": "シュダイ = メインタイトル. 4"}, "Subtitle": {"content": "副題"}}
						]
					},
					"Collection": {"TitleDetail": {"TitleElement": [
						{"TitleElementLevel": "02", "TitleText": {"content": "出版社コレクション", "collationkey": "シュッパンシャコレクション"}, "PartNumber": "999"},
						{"TitleElementLevel": "03", "TitleText": {"content": "商品階層タイトル"}}
					]}}
				}
			},
			"summary": {"title": "代替題", "series": "要約レーベル", "volume": "999"}
		}
	]`))
	if err != nil {
		t.Fatalf("decodeResponse() error = %v", err)
	}

	book := convertBook(*response[0], "9784088466361")
	if book.Title != "主題 = MAIN TITLE. 4" || book.TitleReading != "シュダイ = メインタイトル. 4" ||
		book.Subtitle != "副題" {
		t.Fatalf("Book = %#v", book)
	}
	if book.Volume.Number != nil || book.Volume.Label != "" ||
		len(book.PublicationSeries) != 2 || book.PublicationSeries[0] != "出版社コレクション" || book.PublicationSeries[1] != "要約レーベル" {
		t.Fatalf("unexpected inferred fields: %#v", book)
	}
}

// TestConvertBook_IgnoresOutOfScopeONIXFields は、対象外のONIX項目を共通書籍情報へ設定しないことを検証する
func TestConvertBook_IgnoresOutOfScopeONIXFields(t *testing.T) {
	response, err := decodeResponse([]byte(`[
		{
			"onix": {
				"DescriptiveDetail": {
					"TitleDetail": {
						"TitleType": "01",
						"TitleElement": {"TitleElementLevel": "01", "TitleText": {"content": "対象外項目を持つ本 4"}, "PartNumber": "4"}
					},
					"Collection": {"CollectionSequence": {"CollectionSequenceNumber": "999"}, "TitleDetail": {"TitleElement": {"TitleElementLevel": "02", "TitleText": {"content": "出版社コレクション"}, "PartNumber": "999"}}}
				},
				"CollateralDetail": {"SupportingResource": [{"ResourceContentType": "01", "ResourceVersion": [{"ResourceLink": "https://example.test/resource.jpg"}]}]},
				"ProductSupply": {"SupplyDetail": {"Price": [{"PriceType": "01", "PriceAmount": "1000", "CurrencyCode": "JPY"}]}}
			},
			"summary": {"isbn": "9784088466361", "series": "要約レーベル", "volume": "999"}
		}
	]`))
	if err != nil {
		t.Fatalf("decodeResponse() error = %v", err)
	}

	book := convertBook(*response[0], "9784088466361")
	if book.Title != "対象外項目を持つ本 4" {
		t.Fatalf("Title = %q", book.Title)
	}
	if book.Volume.Number == nil || *book.Volume.Number != 4 || book.Volume.Label != "4" ||
		len(book.PublicationSeries) != 2 ||
		book.ListPrice == nil || book.ListPrice.Amount != 1000 || book.ListPrice.TaxIncluded == nil || *book.ListPrice.TaxIncluded || book.CurrentPrice != nil || book.CoverURL != "" {
		t.Fatalf("unexpected out-of-scope values: %#v", book)
	}
}

// TestListPrice は、対応するONIX価格だけを一意の定価として変換する
func TestListPrice(t *testing.T) {
	tests := []struct {
		name   string
		prices []responsePrice
		want   *Price
	}{
		{name: "excluding tax", prices: []responsePrice{{PriceType: "01", CurrencyCode: "JPY", PriceAmount: "630"}}, want: price(630, false)},
		{name: "including tax", prices: []responsePrice{{PriceType: "02", CurrencyCode: "JPY", PriceAmount: "693"}}, want: price(693, true)},
		{name: "deduplicates", prices: []responsePrice{{PriceType: "01", CurrencyCode: "JPY", PriceAmount: "630"}, {PriceType: "01", CurrencyCode: "JPY", PriceAmount: "630"}}, want: price(630, false)},
		{name: "different candidates", prices: []responsePrice{{PriceType: "01", CurrencyCode: "JPY", PriceAmount: "630"}, {PriceType: "02", CurrencyCode: "JPY", PriceAmount: "693"}}},
		{name: "skips unsupported", prices: []responsePrice{{PriceType: "03", CurrencyCode: "JPY", PriceAmount: "630"}}},
		{name: "skips invalid", prices: []responsePrice{{PriceType: "01", CurrencyCode: "USD", PriceAmount: "630"}, {PriceType: "02", CurrencyCode: "JPY", PriceAmount: "630.5"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := listPrice(responseProductSupply{SupplyDetail: []responseSupplyDetail{{Price: test.prices}}})
			if !samePrice(got, test.want) {
				t.Fatalf("listPrice() = %#v, want %#v", got, test.want)
			}
		})
	}
}

// price は、openBD価格テスト用のPriceを返す
func price(amount int64, taxIncluded bool) *Price {
	return &Price{Amount: amount, Currency: "JPY", TaxIncluded: &taxIncluded, Source: SourceOpenBD}
}

// samePrice は、価格と税込情報が一致するか判定する
func samePrice(left, right *Price) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.Amount != right.Amount || left.Currency != right.Currency || left.Source != right.Source || left.ObservedAt != right.ObservedAt {
		return false
	}
	if left.TaxIncluded == nil || right.TaxIncluded == nil {
		return left.TaxIncluded == right.TaxIncluded
	}
	return *left.TaxIncluded == *right.TaxIncluded
}

// TestAppendUnique_DeduplicatesPublicationSeriesNames は、同名系列を重複なく最初の出現順で保持する
func TestAppendUnique_DeduplicatesPublicationSeriesNames(t *testing.T) {
	values := appendUnique(nil, "叢書")
	values = appendUnique(values, "叢書")
	values = appendUnique(values, "別叢書")
	if len(values) != 2 || values[0] != "叢書" || values[1] != "別叢書" {
		t.Fatalf("PublicationSeries = %#v", values)
	}
}

// TestConvertContributors_UnknownAndRoleless は、未知役割を推測せず役割なしの寄与者を保持する
func TestConvertContributors_UnknownAndRoleless(t *testing.T) {
	authors, contributors := convertContributors([]responseContributor{
		{SequenceNumber: "2", ContributorRole: []string{"Z99"}, PersonName: responseContentValue{Content: "未知役割"}},
		{SequenceNumber: "1", PersonName: responseContentValue{Content: "役割なし", CollationKey: "ヤクワリナシ"}},
	})
	assertStrings(t, authors, []string{"役割なし"})
	if len(contributors) != 2 || contributors[0].Name != "役割なし" || contributors[0].Reading != "ヤクワリナシ" ||
		contributors[1].Name != "未知役割" || len(contributors[1].Roles) != 0 {
		t.Fatalf("Contributors = %#v", contributors)
	}
}

// TestConvertContributors_PreservesReadingWithKnownRole は、既知役割を持つ人物の読みを同じ寄与者へ設定することを検証する
func TestConvertContributors_PreservesReadingWithKnownRole(t *testing.T) {
	authors, contributors := convertContributors([]responseContributor{{
		ContributorRole: []string{"A01"},
		PersonName: responseContentValue{
			Content:      "長頼",
			CollationKey: "ナガヨリ",
		},
	}})
	assertStrings(t, authors, []string{"長頼"})
	if len(contributors) != 1 || contributors[0].Name != "長頼" ||
		contributors[0].Reading != "ナガヨリ" ||
		len(contributors[0].Roles) != 1 || contributors[0].Roles[0] != "著者" {
		t.Fatalf("Contributors = %#v", contributors)
	}
}

// TestConvertContributors_PreservesOriginalNames は、人物名と同じ要素の読みを変更や分割をせずに保持することを検証する
func TestConvertContributors_PreservesOriginalNames(t *testing.T) {
	tests := []struct {
		name    string
		reading string
	}{
		{name: "オノ・マサユキ／阪元裕吾", reading: "オノマサユキサカモトユウゴ"},
		{name: "伏瀬, 1975-", reading: "フセ"},
		{name: "川上, 泰樹", reading: "カワカミ, タイキ"},
		{name: "藤子, F. 不二雄", reading: "フジコ, エフ. フジオ"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, contributors := convertContributors([]responseContributor{{
				ContributorRole: []string{"A01"},
				PersonName: responseContentValue{
					Content:      test.name,
					CollationKey: test.reading,
				},
			}})
			if len(contributors) != 1 || contributors[0].Name != test.name || contributors[0].Reading != test.reading {
				t.Fatalf("Contributors = %#v", contributors)
			}
		})
	}
}

// TestConvertContributors_DoesNotMergeSeparateElements は、同名でも別のONIX要素を統合しないことを検証する
func TestConvertContributors_DoesNotMergeSeparateElements(t *testing.T) {
	_, contributors := convertContributors([]responseContributor{
		{SequenceNumber: "1", ContributorRole: []string{"A01"}, PersonName: responseContentValue{Content: "著者", CollationKey: "チョシャ"}},
		{SequenceNumber: "2", ContributorRole: []string{"A36"}, PersonName: responseContentValue{Content: "著者", CollationKey: "チョシャ"}},
	})
	if len(contributors) != 2 || contributors[0].Name != "著者" || contributors[0].Reading != "チョシャ" ||
		len(contributors[0].Roles) != 1 || contributors[0].Roles[0] != "著者" ||
		contributors[1].Name != "著者" || contributors[1].Reading != "チョシャ" || len(contributors[1].Roles) != 0 {
		t.Fatalf("Contributors = %#v", contributors)
	}
}

// TestMapContributorRole は、対応済みと未対応のONIX役割を共通役割へ誤りなく変換することを検証する
func TestMapContributorRole(t *testing.T) {
	tests := []struct {
		code string
		role string
		ok   bool
	}{
		{code: "A01", role: "著者", ok: true},
		{code: "A03", role: "脚本", ok: true},
		{code: "A14", role: "脚本", ok: true},
		{code: "A45", role: "脚本", ok: true},
		{code: "A07", role: "作画", ok: true},
		{code: "A12", role: "作画", ok: true},
		{code: "A35", role: "作画", ok: true},
		{code: "B01", role: "編集", ok: true},
		{code: "B06", role: "翻訳", ok: true},
		{code: "A36"},
		{code: "A38"},
		{code: "A46"},
		{code: "A47"},
	}

	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			role, ok := mapContributorRole(test.code)
			if role != test.role || ok != test.ok {
				t.Fatalf("mapContributorRole(%q) = %q, %t; want %q, %t", test.code, role, ok, test.role, test.ok)
			}
		})
	}
}

// TestConvertContributors_DoesNotMapUnsupportedRoles は、未対応のONIX役割を近い共通役割へ変換しないことを検証する
func TestConvertContributors_DoesNotMapUnsupportedRoles(t *testing.T) {
	authors, contributors := convertContributors([]responseContributor{
		{SequenceNumber: "1", ContributorRole: []string{"A01"}, PersonName: responseContentValue{Content: "著者", CollationKey: "チョシャ"}},
		{SequenceNumber: "2", ContributorRole: []string{"A36"}, PersonName: responseContentValue{Content: "表紙担当"}},
		{SequenceNumber: "3", ContributorRole: []string{"A38"}, PersonName: responseContentValue{Content: "初版著者"}},
		{SequenceNumber: "4", ContributorRole: []string{"A46"}, PersonName: responseContentValue{Content: "インカー"}},
		{SequenceNumber: "5", ContributorRole: []string{"A47"}, PersonName: responseContentValue{Content: "カラーリスト"}},
	})
	if len(contributors) != 5 || len(contributors[0].Roles) != 1 || contributors[0].Roles[0] != "著者" {
		t.Fatalf("Contributors = %#v", contributors)
	}
	assertStrings(t, authors, []string{"著者"})
	for _, contributor := range contributors[1:] {
		if len(contributor.Roles) != 0 {
			t.Fatalf("unsupported contributor = %#v", contributor)
		}
	}
}

// TestSelectPublishedDate は、商品の出版日だけをONIX候補として扱うことを検証する
func TestSelectPublishedDate(t *testing.T) {
	tests := []struct {
		name string
		book responseBook
		want string
	}{
		{
			name: "ONIX product publication date",
			book: responseBook{Onix: responseOnix{PublishingDetail: responsePublishingDetail{
				PublishingDate: []responsePublishingDate{{PublishingDateRole: "11", Date: "1999"}, {PublishingDateRole: "01", Date: "2026-08-06"}},
			}}},
			want: "2026-08-06",
		},
		{
			name: "hanmoto fallback after role 11",
			book: responseBook{
				Onix:    responseOnix{PublishingDetail: responsePublishingDetail{PublishingDate: []responsePublishingDate{{PublishingDateRole: "11", Date: "1999"}}}},
				Hanmoto: responseHanmoto{DateShuppan: "2026-08"},
				Summary: responseSummary{PubDate: "202608"},
			},
			want: "2026-08",
		},
		{
			name: "summary fallback",
			book: responseBook{
				Onix:    responseOnix{PublishingDetail: responsePublishingDetail{PublishingDate: []responsePublishingDate{{PublishingDateRole: "11", Date: "1999"}}}},
				Summary: responseSummary{PubDate: "202608"},
			},
			want: "202608",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := selectPublishedDate(test.book); got != test.want {
				t.Fatalf("selectPublishedDate() = %q, want %q", got, test.want)
			}
		})
	}
}
