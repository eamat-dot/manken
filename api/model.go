// Package api は、漫画本の書誌情報を検索するための共通の書籍モデルと検索条件を提供する
package api

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
)

// Book は、取得した1冊の漫画本を表す
type Book struct {
	Normalized NormalizedBook `json:"normalized"`
	Sources    []BookSource   `json:"sources"`
}

// NormalizedBook は、取得元に依存せず利用できる共通書籍情報を表す
type NormalizedBook struct {
	Title             string            `json:"title,omitempty"`
	ParallelTitles    []string          `json:"parallel_titles,omitempty"`
	TitleReading      string            `json:"title_reading,omitempty"`
	Subtitle          string            `json:"subtitle,omitempty"`
	Series            []Series          `json:"series,omitempty"`
	Volume            Volume            `json:"volume,omitzero"`
	EditionStatements []string          `json:"edition_statements,omitempty"`
	IsFinalVolume     bool              `json:"is_final_volume,omitempty"`
	Authors           []string          `json:"authors,omitempty"`
	Contributors      []Contributor     `json:"contributors,omitempty"`
	Publishers        []string          `json:"publishers,omitempty"`
	Imprints          []string          `json:"imprints,omitempty"`
	Identifiers       []Identifier      `json:"identifiers,omitempty"`
	Dates             []BookDate        `json:"dates,omitempty"`
	Description       string            `json:"description,omitempty"`
	Languages         []string          `json:"languages,omitempty"`
	Subjects          []Subject         `json:"subjects,omitempty"`
	PageCount         *int              `json:"page_count,omitempty"`
	Medium            PublicationMedium `json:"medium,omitempty"`
	PhysicalSize      *PhysicalSize     `json:"physical_size,omitempty"`
	Prices            []Price           `json:"prices,omitempty"`
	Images            []Image           `json:"images,omitempty"`
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

// IdentifierType は、書誌識別子の種類を表す
type IdentifierType string

const (
	// IdentifierTypeISBN10 は、ISBN-10を表す
	IdentifierTypeISBN10 IdentifierType = "isbn10"
	// IdentifierTypeISBN13 は、ISBN-13を表す
	IdentifierTypeISBN13 IdentifierType = "isbn13"
	// IdentifierTypeJAN は、JANコードを表す
	IdentifierTypeJAN IdentifierType = "jan"
)

// Identifier は、種類を明示した書誌識別子を表す
type Identifier struct {
	Type  IdentifierType `json:"type"`
	Value string         `json:"value"`
}

// ContributorRole は、制作への寄与者の役割を表す
type ContributorRole string

const (
	// ContributorRoleAuthor は、著者を表す
	ContributorRoleAuthor ContributorRole = "author"
	// ContributorRoleOriginalCreator は、原作者または原案者を表す
	ContributorRoleOriginalCreator ContributorRole = "original_creator"
	// ContributorRoleWriter は、構成または脚本の執筆者を表す
	ContributorRoleWriter ContributorRole = "writer"
	// ContributorRoleArtist は、漫画または作画の担当者を表す
	ContributorRoleArtist ContributorRole = "artist"
	// ContributorRoleCharacterCreator は、キャラクター原案者を表す
	ContributorRoleCharacterCreator ContributorRole = "character_creator"
	// ContributorRoleCharacterDesigner は、キャラクターデザイン担当者を表す
	ContributorRoleCharacterDesigner ContributorRole = "character_designer"
	// ContributorRoleEditor は、編集者を表す
	ContributorRoleEditor ContributorRole = "editor"
	// ContributorRoleTranslator は、翻訳者を表す
	ContributorRoleTranslator ContributorRole = "translator"
	// ContributorRoleSupervisor は、監修者を表す
	ContributorRoleSupervisor ContributorRole = "supervisor"
	// ContributorRoleCommentator は、解説者を表す
	ContributorRoleCommentator ContributorRole = "commentator"
	// ContributorRoleDesigner は、装丁またはデザインの担当者を表す
	ContributorRoleDesigner ContributorRole = "designer"
)

// Contributor は、制作への寄与者と複数の役割を表す
type Contributor struct {
	Name    string            `json:"name"`
	Reading string            `json:"reading,omitempty"`
	Roles   []ContributorRole `json:"roles,omitempty"`
}

// Series は、シリーズ名と取得元内の参照情報を表す
type Series struct {
	Name   string `json:"name"`
	ID     string `json:"id,omitempty"`
	URL    string `json:"url,omitempty"`
	Source Source `json:"source,omitempty"`
}

// BookDateType は、書誌に関係する日付の種類を表す
type BookDateType string

const (
	// BookDateTypePublished は、出版日を表す
	BookDateTypePublished BookDateType = "published"
	// BookDateTypeReleased は、発売日を表す
	BookDateTypeReleased BookDateType = "released"
	// BookDateTypeDigitalReleased は、電子版の配信開始日を表す
	BookDateTypeDigitalReleased BookDateType = "digital_released"
)

// BookDate は、種類と精度を維持した日付文字列を表す
type BookDate struct {
	Type  BookDateType `json:"type"`
	Value string       `json:"value"`
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

// PhysicalSize は、紙書籍の判型名と寸法をミリメートル単位で表す
type PhysicalSize struct {
	Name        string `json:"name,omitempty"`
	HeightMM    *int   `json:"height_mm,omitempty"`
	WidthMM     *int   `json:"width_mm,omitempty"`
	ThicknessMM *int   `json:"thickness_mm,omitempty"`
}

// PriceType は、価格が定価または取得時点価格のどちらかを表す
type PriceType string

const (
	// PriceTypeList は、定価を表す
	PriceTypeList PriceType = "list"
	// PriceTypeCurrent は、API取得時点の販売価格を表す
	PriceTypeCurrent PriceType = "current"
)

// Price は、種類と出典を明示した価格を表す
type Price struct {
	Type        PriceType `json:"type"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	TaxIncluded *bool     `json:"tax_included,omitempty"`
	Source      Source    `json:"source"`
	ObservedAt  string    `json:"observed_at,omitempty"`
}

// Subject は、取得元の分類体系に基づく主題またはジャンルを表す
type Subject struct {
	Scheme string `json:"scheme,omitempty"`
	Code   string `json:"code,omitempty"`
	Name   string `json:"name,omitempty"`
}

// Image は、表紙など書籍に関係する画像を表す
type Image struct {
	URL     string `json:"url"`
	Purpose string `json:"purpose,omitempty"`
	Width   *int   `json:"width,omitempty"`
	Height  *int   `json:"height,omitempty"`
}

// SearchBooksRequest は、漫画本の検索条件を表す
type SearchBooksRequest struct {
	Title        string `json:"title"`
	Author       string `json:"author"`
	Publisher    string `json:"publisher"`
	FreeText     string `json:"free_text"`
	ExcludedText string `json:"excluded_text"`
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
}

// SearchBooksResult は、漫画本の検索結果と続きの取得に使うカーソルを表す
type SearchBooksResult struct {
	Books      []Book `json:"books"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ISBNLookupResult は、入力ISBNごとの書籍参照結果を表す
type ISBNLookupResult struct {
	Items []ISBNLookupItem `json:"items"`
}

// ISBNLookupItem は、指定された1つのISBNと対応する書籍を表す
type ISBNLookupItem struct {
	RequestedISBN string `json:"requested_isbn"`
	Books         []Book `json:"books"`
}
