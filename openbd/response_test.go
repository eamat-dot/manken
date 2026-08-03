package openbd

import "testing"

// TestConvertBook_SummaryFallback は、ONIX項目がない代替書誌をsummaryから変換する
func TestConvertBook_SummaryFallback(t *testing.T) {
	book := convertBook(responseBook{
		Summary: responseSummary{
			ISBN:      "9784088466361",
			Title:     "欠落項目を持つ書籍",
			Series:    "テストシリーズ",
			Author:    "著者A,著者B",
			Publisher: "出版社",
			PubDate:   "2026-08",
			Cover:     "https://example.test/cover.jpg",
		},
	}, "9784088466361")

	if book.Normalized.Title != "欠落項目を持つ書籍" {
		t.Fatalf("Title = %q", book.Normalized.Title)
	}
	assertStrings(t, book.Normalized.Authors, []string{"著者A,著者B"})
	assertStrings(t, book.Normalized.Publishers, []string{"出版社"})
	if len(book.Normalized.Series) != 1 || book.Normalized.Series[0].Name != "テストシリーズ" {
		t.Fatalf("Series = %#v", book.Normalized.Series)
	}
	if len(book.Normalized.Dates) != 1 || book.Normalized.Dates[0].Value != "2026-08" {
		t.Fatalf("Dates = %#v", book.Normalized.Dates)
	}
	if len(book.Normalized.Images) != 1 || book.Normalized.Images[0].Purpose != "cover" {
		t.Fatalf("Images = %#v", book.Normalized.Images)
	}
}

// TestConvertContributors_UnknownAndRoleless は、未知役割を推測せず役割なし著者を維持する
func TestConvertContributors_UnknownAndRoleless(t *testing.T) {
	authors, contributors := convertContributors([]responseContributor{
		{SequenceNumber: "2", ContributorRole: []string{"Z99"}, PersonName: responseContentValue{Content: "未知役割"}},
		{SequenceNumber: "1", PersonName: responseContentValue{Content: "役割なし"}},
	})
	assertStrings(t, authors, []string{"役割なし"})
	if len(contributors) != 0 {
		t.Fatalf("Contributors = %#v, want empty", contributors)
	}
}

// TestDecodeResponse_IgnoresSummaryVolume は、summary.volumeを巻数として読み取らない
func TestDecodeResponse_IgnoresSummaryVolume(t *testing.T) {
	response, err := decodeResponse([]byte(`[{"onix":{"RecordReference":"9784088466361"},"summary":{"isbn":"9784088466361","title":"巻数なし","series":"レーベル","volume":"9999"}}]`))
	if err != nil {
		t.Fatalf("decodeResponse() error = %v", err)
	}
	book := convertBook(*response[0], "9784088466361")
	if book.Normalized.Volume.Number != nil || book.Normalized.Volume.Label != "" {
		t.Fatalf("Volume = %#v, want empty", book.Normalized.Volume)
	}
}
