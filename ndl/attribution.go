package ndl

// Attributions は、NDLサーチAPI利用と取得データに関する表示・保存用の出典情報を返す
func Attributions() []Attribution {
	return []Attribution{
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
}
