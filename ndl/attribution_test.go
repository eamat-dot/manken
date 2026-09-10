package ndl

import (
	"reflect"
	"testing"
)

// TestAttributionsReturnsStableIndependentValues は、NDLの出典情報、順序、返却sliceの独立性を検証する
func TestAttributionsReturnsStableIndependentValues(t *testing.T) {
	want := []Attribution{
		{
			Source:          SourceNDL,
			Scope:           AttributionScopeService,
			Text:            "国立国会図書館サーチAPIを利用",
			URL:             "https://ndlsearch.ndl.go.jp/",
			RequirementsURL: "https://ndlsearch.ndl.go.jp/help/api",
		},
		{
			Source:          SourceNDL,
			Scope:           AttributionScopeData,
			Text:            "国立国会図書館全国書誌情報（国立国会図書館）をもとに、mankenのBookモデルに変換して作成",
			URL:             "https://ndlsearch.ndl.go.jp/",
			License:         "CC BY 4.0",
			LicenseURL:      "https://creativecommons.org/licenses/by/4.0/",
			RequirementsURL: "https://ndlsearch.ndl.go.jp/help/api/provider",
		},
	}

	got := Attributions()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Attributions() = %#v, want %#v", got, want)
	}
	got[0].Text = "changed"
	if next := Attributions(); !reflect.DeepEqual(next, want) {
		t.Fatalf("Attributions() after mutation = %#v, want %#v", next, want)
	}
}
