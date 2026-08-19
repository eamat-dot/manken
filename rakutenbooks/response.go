package rakutenbooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/eamat-dot/manken/internal/titlemeta"

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
	SeriesNameKana string `json:"seriesNameKana"`
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
	ItemPrice      *int64 `json:"itemPrice"`
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
func convertItem(value booksItem, observedAt string) Book {
	book := Book{
		Title:        value.Title,
		TitleReading: value.TitleKana,
		Subtitle:     value.SubTitle,
		Description:  value.ItemCaption,
		Medium:       PublicationMediumPrint,
		Subjects:     subjects(value.BooksGenreID),
		Size:         value.Size,
		CoverURL:     coverURL(value),
		Sources: []BookSource{{
			Source:       SourceRakutenBooks,
			URL:          sourceURL(value.ItemURL),
			AffiliateURL: sourceURL(value.AffiliateURL),
		}},
	}
	if value.SeriesName != "" {
		book.PublicationSeries = []string{value.SeriesName}
	}
	book.Authors, book.Contributors = contributors(value.Author, value.AuthorKana)
	if value.PublisherName != "" {
		book.Publishers = []string{value.PublisherName}
	}
	book.ISBN10, book.ISBN13 = isbn(value.ISBN)
	if value.SalesDate != "" {
		book.ReleaseDate = value.SalesDate
	}
	if value.ItemPrice != nil {
		taxIncluded := true
		book.CurrentPrice = &Price{
			Amount:      *value.ItemPrice,
			Currency:    "JPY",
			TaxIncluded: &taxIncluded,
			Source:      SourceRakutenBooks,
			ObservedAt:  observedAt,
		}
	}
	applyTitleMetadata(&book)
	return book
}

// applyTitleMetadata は、タイトルから安全に抽出できた付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	titlemeta.Apply(book)
}

// contributors は、楽天Booksのスラッシュ区切り著者と読みを人物単位へ変換する
func contributors(author string, authorKana string) ([]string, []Contributor) {
	originalAuthors := strings.Split(author, "/")
	originalReadings := strings.Split(authorKana, "/")
	readingsMatch := len(originalAuthors) == len(originalReadings)
	authors := make([]string, 0, len(originalAuthors))
	contributors := make([]Contributor, 0, len(originalAuthors))
	for index, originalAuthor := range originalAuthors {
		name := strings.TrimSpace(originalAuthor)
		if name == "" {
			continue
		}
		contributor := Contributor{Name: name}
		if readingsMatch {
			contributor.Reading = strings.TrimSpace(originalReadings[index])
		}
		authors = append(authors, name)
		contributors = append(contributors, contributor)
	}
	if len(authors) == 0 {
		return nil, nil
	}
	return authors, contributors
}

// sourceURL は、通常商品URLとして利用できるHTTP(S) URLだけを返す
func sourceURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return value
}

// isbn は、検証済みISBNを種類ごとのスライスへ変換する
func isbn(value string) ([]string, []string) {
	switch {
	case internalisbn.IsValidISBN10(value):
		return []string{value}, nil
	case internalisbn.IsValidISBN13(value):
		return nil, []string{value}
	default:
		return nil, nil
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

// coverURL は、楽天Booksの利用可能な最大サイズ画像URLを返す
func coverURL(value booksItem) string {
	for _, candidate := range []string{value.LargeImageURL, value.MediumImageURL, value.SmallImageURL} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}
