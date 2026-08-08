package rakutenbooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

// booksResponse は、楽天ブックス書籍検索APIの初期変換に必要な項目を保持する
type booksResponse struct {
	Count     int         `json:"count"`
	Page      int         `json:"page"`
	PageCount int         `json:"pageCount"`
	Hits      int         `json:"hits"`
	Items     []booksItem `json:"Items"`
}

// booksItem は、共通書籍モデルへ対応できる楽天Books商品情報を保持する
type booksItem struct {
	Title          string `json:"title"`
	TitleKana      string `json:"titleKana"`
	SubTitle       string `json:"subTitle"`
	SeriesName     string `json:"seriesName"`
	Author         string `json:"author"`
	AuthorKana     string `json:"authorKana"`
	PublisherName  string `json:"publisherName"`
	ISBN           string `json:"isbn"`
	SalesDate      string `json:"salesDate"`
	ItemCaption    string `json:"itemCaption"`
	BooksGenreID   string `json:"booksGenreId"`
	Size           string `json:"size"`
	ItemURL        string `json:"itemUrl"`
	AffiliateURL   string `json:"affiliateUrl"`
	SmallImageURL  string `json:"smallImageUrl"`
	MediumImageURL string `json:"mediumImageUrl"`
	LargeImageURL  string `json:"largeImageUrl"`
}

// decodeBooksResponse は、楽天Books検索本文を非公開レスポンス型へ変換する
func decodeBooksResponse(body []byte) (booksResponse, error) {
	var response booksResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil {
		return booksResponse{}, fmt.Errorf("decode Rakuten Books response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return booksResponse{}, err
	}
	if response.Count < 0 || response.PageCount < 0 || response.Hits < 0 || response.Page < 0 {
		return booksResponse{}, errors.New("response pagination values must not be negative")
	}
	return response, nil
}

// convertItem は、楽天Books商品を共通のBookへ変換する
func convertItem(value booksItem) Book {
	book := Book{
		Normalized: NormalizedBook{
			Title:        value.Title,
			TitleReading: value.TitleKana,
			Subtitle:     value.SubTitle,
			Description:  value.ItemCaption,
			Medium:       PublicationMediumPrint,
			Subjects:     subjects(value.BooksGenreID),
			Images:       images(value),
		},
		Sources: []BookSource{{Source: SourceRakutenBooks, URL: sourceURL(value.ItemURL)}},
	}
	if value.SeriesName != "" {
		book.Normalized.Series = []Series{{Name: value.SeriesName}}
	}
	if value.Author != "" {
		book.Normalized.Authors = []string{value.Author}
		book.Normalized.Contributors = []Contributor{{Name: value.Author, Reading: value.AuthorKana}}
	}
	if value.PublisherName != "" {
		book.Normalized.Publishers = []string{value.PublisherName}
	}
	if identifier, ok := identifier(value.ISBN); ok {
		book.Normalized.Identifiers = []Identifier{identifier}
	}
	if value.SalesDate != "" {
		book.Normalized.Dates = []BookDate{{Type: BookDateTypeReleased, Value: value.SalesDate}}
	}
	if value.Size != "" {
		book.Normalized.PhysicalSize = &PhysicalSize{Name: value.Size}
	}
	return book
}

// sourceURL は、通常商品URLとして利用できるHTTP(S) URLだけを返す
func sourceURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return value
}

// identifier は、検証済みISBNを共通識別子へ変換する
func identifier(value string) (Identifier, bool) {
	switch {
	case internalisbn.IsValidISBN10(value):
		return Identifier{Type: IdentifierTypeISBN10, Value: value}, true
	case internalisbn.IsValidISBN13(value):
		return Identifier{Type: IdentifierTypeISBN13, Value: value}, true
	default:
		return Identifier{}, false
	}
}

// subjects は、楽天Booksのスラッシュ区切りジャンルIDを取得元分類へ変換する
func subjects(value string) []Subject {
	parts := strings.Split(value, "/")
	result := make([]Subject, 0, len(parts))
	for _, part := range parts {
		code := strings.TrimSpace(part)
		if code != "" {
			result = append(result, Subject{Scheme: "rakuten_books", Code: code})
		}
	}
	return result
}

// images は、楽天Booksの3サイズ画像URLを安定したフィールド順で変換する
func images(value booksItem) []Image {
	values := []struct{ purpose, url string }{
		{"smallImageUrl", value.SmallImageURL},
		{"mediumImageUrl", value.MediumImageURL},
		{"largeImageUrl", value.LargeImageURL},
	}
	result := make([]Image, 0, len(values))
	for _, image := range values {
		if image.url != "" {
			result = append(result, Image{URL: image.url, Purpose: image.purpose})
		}
	}
	return result
}
