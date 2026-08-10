package googlebooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

// volumesResponse は、Google Books Volumes listの初期変換に必要な項目を保持する
type volumesResponse struct {
	TotalItems int      `json:"totalItems"`
	Items      []volume `json:"items"`
}

// volume は、Google Books Volumeの初期変換に必要な項目を保持する
type volume struct {
	ID         string     `json:"id"`
	VolumeInfo volumeInfo `json:"volumeInfo"`
	SaleInfo   saleInfo   `json:"saleInfo"`
}

// saleInfo は、Google Booksが返す国別の販売情報を保持する
type saleInfo struct {
	Country     string `json:"country"`
	IsEbook     bool   `json:"isEbook"`
	ListPrice   price  `json:"listPrice"`
	RetailPrice price  `json:"retailPrice"`
}

// price は、Google Booksが返す金額と通貨を保持する
type price struct {
	Amount       *json.Number `json:"amount"`
	CurrencyCode string       `json:"currencyCode"`
}

// volumeInfo は、共通書籍モデルへ対応できるVolume情報を保持する
type volumeInfo struct {
	Title               string               `json:"title"`
	Subtitle            string               `json:"subtitle"`
	Authors             []string             `json:"authors"`
	Publisher           string               `json:"publisher"`
	PublishedDate       string               `json:"publishedDate"`
	IndustryIdentifiers []industryIdentifier `json:"industryIdentifiers"`
	Description         string               `json:"description"`
	Language            string               `json:"language"`
	Categories          []string             `json:"categories"`
	PageCount           *int                 `json:"pageCount"`
	ImageLinks          imageLinks           `json:"imageLinks"`
	CanonicalVolumeLink string               `json:"canonicalVolumeLink"`
	InfoLink            string               `json:"infoLink"`
}

// industryIdentifier は、Google Booksが返す業界標準識別子を保持する
type industryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

// imageLinks は、Google Booksが返す画像URLを保持する
type imageLinks struct {
	SmallThumbnail string `json:"smallThumbnail"`
	Thumbnail      string `json:"thumbnail"`
	Small          string `json:"small"`
	Medium         string `json:"medium"`
	Large          string `json:"large"`
	ExtraLarge     string `json:"extraLarge"`
}

// decodeVolumesResponse は、Google Books検索本文を非公開レスポンス型へ変換する
func decodeVolumesResponse(body []byte) (volumesResponse, error) {
	var response volumesResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return volumesResponse{}, fmt.Errorf("decode Google Books response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return volumesResponse{}, err
	}
	if response.TotalItems < 0 {
		return volumesResponse{}, errors.New("response totalItems must not be negative")
	}
	return response, nil
}

// convertVolume は、Google Books Volumeを共通のBookへ変換する
func convertVolume(value volume, observedAt string) (Book, error) {
	if value.ID == "" {
		return Book{}, errors.New("response volume is missing id")
	}
	info := value.VolumeInfo
	book := Book{Normalized: NormalizedBook{
		Title:        info.Title,
		Subtitle:     info.Subtitle,
		Authors:      nonEmptyStrings(info.Authors),
		Contributors: contributors(info.Authors),
		Description:  info.Description,
		Languages:    nonEmptyStrings([]string{info.Language}),
		Subjects:     subjects(info.Categories),
		PageCount:    positivePageCount(info.PageCount),
		Images:       images(info.ImageLinks),
	}, Sources: []BookSource{{Source: SourceGoogleBooks, ID: value.ID, URL: sourceURL(info)}}}
	if info.Publisher != "" {
		book.Normalized.Publishers = []string{info.Publisher}
	}
	if info.PublishedDate != "" {
		book.Normalized.Dates = []BookDate{{Type: BookDateTypePublished, Value: info.PublishedDate}}
	}
	book.Normalized.Identifiers = identifiers(info.IndustryIdentifiers)
	if value.SaleInfo.IsEbook {
		book.Normalized.Medium = PublicationMediumDigital
	}
	book.Normalized.Prices = prices(value.SaleInfo, observedAt)
	return book, nil
}

// prices は、条件を満たす日本向けの日本円販売価格だけを共通価格へ変換する
func prices(value saleInfo, observedAt string) []Price {
	if !strings.EqualFold(value.Country, "JP") {
		return nil
	}

	prices := make([]Price, 0, 2)
	if amount, ok := priceAmount(value.ListPrice); ok {
		prices = append(prices, Price{
			Type:     PriceTypeList,
			Amount:   amount,
			Currency: "JPY",
			Source:   SourceGoogleBooks,
		})
	}
	if amount, ok := priceAmount(value.RetailPrice); ok && observedAt != "" {
		prices = append(prices, Price{
			Type:       PriceTypeCurrent,
			Amount:     amount,
			Currency:   "JPY",
			Source:     SourceGoogleBooks,
			ObservedAt: observedAt,
		})
	}
	if len(prices) == 0 {
		return nil
	}
	return prices
}

// priceAmount は、Google Booksの価格を正確に表現できる日本円の整数へ変換する
func priceAmount(value price) (int64, bool) {
	if value.Amount == nil || !strings.EqualFold(value.CurrencyCode, "JPY") {
		return 0, false
	}
	text := value.Amount.String()
	if text != strings.TrimSpace(text) || !json.Valid([]byte(text)) {
		return 0, false
	}
	amount, ok := new(big.Rat).SetString(text)
	if !ok || amount.Sign() < 0 || !amount.IsInt() || !amount.Num().IsInt64() {
		return 0, false
	}
	return amount.Num().Int64(), true
}

// positivePageCount は、Google Booksの0以下のpageCountを不明値として扱う
func positivePageCount(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}

// sourceURL は、Google Booksの書籍参照URLを優先順位に従って選ぶ
func sourceURL(info volumeInfo) string {
	for _, value := range []string{info.CanonicalVolumeLink, info.InfoLink} {
		parsed, err := url.Parse(value)
		if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" {
			return value
		}
	}
	return ""
}

// nonEmptyStrings は、空文字列を除き応答順を維持する
func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

// contributors は、Google Booksのauthorsを役割なしの寄与者として変換する
func contributors(authors []string) []Contributor {
	result := make([]Contributor, 0, len(authors))
	for _, author := range authors {
		if author != "" {
			result = append(result, Contributor{Name: author})
		}
	}
	return result
}

// identifiers は、検証済みのISBNだけを共通識別子へ変換する
func identifiers(values []industryIdentifier) []Identifier {
	result := make([]Identifier, 0, len(values))
	for _, value := range values {
		var kind IdentifierType
		switch value.Type {
		case "ISBN_10":
			if !internalisbn.IsValidISBN10(value.Identifier) {
				continue
			}
			kind = IdentifierTypeISBN10
		case "ISBN_13":
			if !internalisbn.IsValidISBN13(value.Identifier) {
				continue
			}
			kind = IdentifierTypeISBN13
		default:
			continue
		}
		result = append(result, Identifier{Type: kind, Value: value.Identifier})
	}
	return result
}

// subjects は、Google Booksカテゴリーを取得元の分類として変換する
func subjects(values []string) []Subject {
	result := make([]Subject, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, Subject{Scheme: "google_books", Name: value})
		}
	}
	return result
}

// images は、Google Booksの画像URLを安定したフィールド順で変換する
func images(links imageLinks) []Image {
	values := []struct{ purpose, url string }{
		{"smallThumbnail", links.SmallThumbnail}, {"thumbnail", links.Thumbnail}, {"small", links.Small},
		{"medium", links.Medium}, {"large", links.Large}, {"extraLarge", links.ExtraLarge},
	}
	result := make([]Image, 0, len(values))
	for _, value := range values {
		if value.url != "" {
			result = append(result, Image{URL: value.url, Purpose: value.purpose})
		}
	}
	return result
}
