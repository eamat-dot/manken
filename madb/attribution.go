package madb

// Attributions は、MADBの取得データに関する表示・保存用の出典情報を返す
func Attributions() []Attribution {
	return []Attribution{{
		Source:          SourceMADB,
		Scope:           AttributionScopeData,
		Text:            `独立行政法人国立美術館国立アートリサーチセンター「メディア芸術データベース」のデータをもとに、mankenのBookモデルに変換して作成`,
		URL:             "https://mediaarts-db.artmuseums.go.jp/",
		RequirementsURL: "https://mediaarts-db.artmuseums.go.jp/user_terms",
	}}
}
