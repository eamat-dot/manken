// Package model は、漫画本の書誌情報を検索するための共通の書籍モデルと検索条件を提供する
package model

// Source は、書誌情報の取得元を表す
type Source string

const (
	// SourceMADB は、メディア芸術データベースを表す
	SourceMADB Source = "madb"
	// SourceOpenBD は、openBDを表す
	SourceOpenBD Source = "openbd"
	// SourceGoogleBooks は、Google Booksを表す
	SourceGoogleBooks Source = "googlebooks"
	// SourceRakutenBooks は、楽天ブックス書籍検索APIを表す
	SourceRakutenBooks Source = "rakutenbooks"
	// SourceRakutenKobo は、楽天Kobo電子書籍検索APIを表す
	SourceRakutenKobo Source = "rakutenkobo"
	// SourceNDL は、国立国会図書館サーチを表す
	SourceNDL Source = "ndl"
	// SourceYahooShopping は、Yahoo!ショッピング商品検索APIを表す
	SourceYahooShopping Source = "yahooshopping"
	// SourceDMM は、DMM.com Webサービスの商品検索APIを表す
	SourceDMM Source = "dmm"
)

// AttributionScope は、クレジット情報がサービス利用または取得データのどちらに関するものかを表す
type AttributionScope string

const (
	// AttributionScopeService は、APIやサービスを利用していること自体に関するクレジットを表す
	AttributionScopeService AttributionScope = "service"
	// AttributionScopeData は、取得したデータの出典や利用条件に関するクレジットを表す
	AttributionScopeData AttributionScope = "data"
)

// Attribution は、取得元に関する表示・保存用の出典情報を表し、利用条件への適合を保証しない
type Attribution struct {
	Source          Source           `json:"source"`
	Scope           AttributionScope `json:"scope"`
	Text            string           `json:"text,omitempty"`
	URL             string           `json:"url,omitempty"`
	License         string           `json:"license,omitempty"`
	LicenseURL      string           `json:"license_url,omitempty"`
	RequirementsURL string           `json:"requirements_url,omitempty"`
}

// Book は、取得した1冊の漫画本を表す
type Book struct {
	Title              string            `json:"title,omitempty"`
	TitleReading       string            `json:"title_reading,omitempty"`
	Subtitle           string            `json:"subtitle,omitempty"`
	BookSeries         []BookSeries      `json:"book_series,omitempty"`
	PublicationSeries  []string          `json:"publication_series,omitempty"`
	Volume             Volume            `json:"volume,omitempty,omitzero"`
	Editions           []string          `json:"editions,omitempty"`
	IsFinalVolume      bool              `json:"is_final_volume,omitempty"`
	Authors            []string          `json:"authors,omitempty"`
	Contributors       []Contributor     `json:"contributors,omitempty"`
	Publishers         []string          `json:"publishers,omitempty"`
	ISBN10             []string          `json:"isbn10,omitempty"`
	ISBN13             []string          `json:"isbn13,omitempty"`
	JAN                []string          `json:"jan,omitempty"`
	PublishedDate      string            `json:"published_date,omitempty"`
	ReleaseDate        string            `json:"release_date,omitempty"`
	DigitalReleaseDate string            `json:"digital_release_date,omitempty"`
	Description        string            `json:"description,omitempty"`
	Languages          []string          `json:"languages,omitempty"`
	Subjects           []Subject         `json:"subjects,omitempty"`
	PageCount          *int              `json:"page_count,omitempty"`
	Medium             PublicationMedium `json:"medium,omitempty"`
	Size               string            `json:"size,omitempty"`
	ListPrice          *Price            `json:"list_price,omitempty"`
	CurrentPrice       *Price            `json:"current_price,omitempty"`
	CoverURL           string            `json:"cover_url,omitempty"`
	Sources            []BookSource      `json:"sources"`
}

// BookSource は、Bookの情報を取得した取得元と参照先を表す
type BookSource struct {
	Source       Source `json:"source"`
	ID           string `json:"id,omitempty"`
	URL          string `json:"url,omitempty"`
	AffiliateURL string `json:"affiliate_url,omitempty"`
}

// Volume は、整数化できる巻数と正規化済みの巻表示を表す
type Volume struct {
	Number *int   `json:"number,omitempty"`
	Label  string `json:"label,omitempty"`
}

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor struct {
	Name    string   `json:"name"`
	Reading string   `json:"reading,omitempty"`
	Roles   []string `json:"roles,omitempty"`
}

// BookSeries は、書籍が属する作品または個別Bookの系列と取得元内の参照情報を表す
type BookSeries struct {
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
	URL    string `json:"url,omitempty"`
	Source Source `json:"source,omitempty"`
}

// PublicationMedium は、出版物が紙または電子のどちらかを表す
type PublicationMedium string

const (
	// PublicationMediumUnknown は、紙または電子を判定できない状態を表す
	PublicationMediumUnknown PublicationMedium = ""
	// PublicationMediumPrint は、紙書籍を表す
	PublicationMediumPrint PublicationMedium = "print"
	// PublicationMediumDigital は、電子書籍を表す
	PublicationMediumDigital PublicationMedium = "digital"
)

// Price は、金額、通貨、税込情報、取得元、観測時刻を表す
type Price struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	TaxIncluded *bool  `json:"tax_included,omitempty"`
	Source      Source `json:"source"`
	ObservedAt  string `json:"observed_at,omitempty"`
}

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject struct {
	Scheme string `json:"scheme,omitempty"`
	Code   string `json:"code,omitempty"`
	Name   string `json:"name,omitempty"`
}

// SearchRequest は、漫画本の検索条件を表す
type SearchRequest struct {
	Title     string `json:"title"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	Query     string `json:"query"`
	Exclude   string `json:"exclude"`
	DateFrom  string `json:"date_from"`
	DateTo    string `json:"date_to"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor"`
}

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult struct {
	Books        []Book        `json:"books"`
	NextCursor   string        `json:"next_cursor,omitempty"`
	Attributions []Attribution `json:"attributions,omitempty"`
}

// ISBNLookupResult は、入力ISBNごとの書籍参照結果を表す
type ISBNLookupResult struct {
	Items        []ISBNLookupItem `json:"items"`
	Attributions []Attribution    `json:"attributions,omitempty"`
}

// ISBNLookupItem は、指定された1つのISBNと対応する書籍を表す
type ISBNLookupItem struct {
	RequestedISBN string `json:"requested_isbn"`
	Books         []Book `json:"books"`
}
