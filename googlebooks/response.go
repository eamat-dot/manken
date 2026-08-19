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
	"github.com/eamat-dot/manken/internal/titlemeta"
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
	book := Book{
		Title:        info.Title,
		Subtitle:     info.Subtitle,
		Authors:      nonEmptyStrings(info.Authors),
		Contributors: contributors(info.Authors),
		Description:  info.Description,
		Languages:    nonEmptyStrings([]string{info.Language}),
		Subjects:     subjects(info.Categories),
		PageCount:    positivePageCount(info.PageCount),
		CoverURL:     coverURL(info.ImageLinks),
		Sources:      []BookSource{{Source: SourceGoogleBooks, ID: value.ID, URL: sourceURL(info)}},
	}
	if info.Publisher != "" {
		book.Publishers = []string{info.Publisher}
	}
	if info.PublishedDate != "" {
		book.PublishedDate = info.PublishedDate
	}
	book.ISBN10, book.ISBN13 = identifiers(info.IndustryIdentifiers)
	if value.SaleInfo.IsEbook {
		book.Medium = PublicationMediumDigital
	}
	book.ListPrice, book.CurrentPrice = prices(value.SaleInfo, observedAt)
	applyTitleMetadata(&book)
	return book, nil
}

// applyTitleMetadata は、タイトルから安全に抽出できた付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	titlemeta.Apply(book)
}

// prices は、条件を満たす日本向けの日本円販売価格を用途別に変換する
func prices(value saleInfo, observedAt string) (*Price, *Price) {
	if !strings.EqualFold(value.Country, "JP") {
		return nil, nil
	}

	var listPrice, currentPrice *Price
	if amount, ok := priceAmount(value.ListPrice); ok {
		listPrice = &Price{
			Amount:   amount,
			Currency: "JPY",
			Source:   SourceGoogleBooks,
		}
	}
	if amount, ok := priceAmount(value.RetailPrice); ok && observedAt != "" {
		currentPrice = &Price{
			Amount:     amount,
			Currency:   "JPY",
			Source:     SourceGoogleBooks,
			ObservedAt: observedAt,
		}
	}
	return listPrice, currentPrice
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

// identifiers は、検証済みのISBNを種類ごとのスライスへ変換する
func identifiers(values []industryIdentifier) ([]string, []string) {
	isbn10 := make([]string, 0, len(values))
	isbn13 := make([]string, 0, len(values))
	for _, value := range values {
		switch value.Type {
		case "ISBN_10":
			if !internalisbn.IsValidISBN10(value.Identifier) {
				continue
			}
			isbn10 = append(isbn10, value.Identifier)
		case "ISBN_13":
			if !internalisbn.IsValidISBN13(value.Identifier) {
				continue
			}
			isbn13 = append(isbn13, value.Identifier)
		default:
			continue
		}
	}
	return isbn10, isbn13
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

// coverURL は、Google Booksの利用可能な最大サイズの画像URLを返す
func coverURL(links imageLinks) string {
	for _, value := range []string{links.ExtraLarge, links.Large, links.Medium, links.Small, links.Thumbnail, links.SmallThumbnail} {
		if value != "" {
			return value
		}
	}
	return ""
}
