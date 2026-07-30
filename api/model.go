// Package api は、漫画本の書誌情報を検索するための共通契約を提供する
package api

// Source は、書誌情報の取得元を表す
type Source string

const (
	// SourceMADB は、メディア芸術データベースを表す
	SourceMADB Source = "madb"
)

// Book は、検索で取得した1冊の漫画本を表す
type Book struct {
	ID                string   `json:"id"`
	Titles            []string `json:"titles"`
	Subtitles         []string `json:"subtitles"`
	SeriesNames       []string `json:"series_names"`
	SeriesID          string   `json:"series_id"`
	SeriesURL         string   `json:"series_url"`
	VolumeNumber      string   `json:"volume_number"`
	EditionStatements []string `json:"edition_statements"`
	Authors           []string `json:"authors"`
	Publishers        []string `json:"publishers"`
	Imprints          []string `json:"imprints"`
	ISBN10s           []string `json:"isbn10s"`
	ISBN13s           []string `json:"isbn13s"`
	PublishedDate     string   `json:"published_date"`
	Source            Source   `json:"source"`
	SourceURL         string   `json:"source_url"`
}

// SearchBooksRequest は、漫画本の検索条件を表す
type SearchBooksRequest struct {
	Title  string `json:"title"`
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult struct {
	Books      []Book `json:"books"`
	NextCursor string `json:"next_cursor"`
}
