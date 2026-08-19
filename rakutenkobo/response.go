package rakutenkobo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/eamat-dot/manken/internal/titlemeta"
)

// koboResponse は、楽天Kobo検索APIの初期変換に必要な項目を保持する
type koboResponse struct {
	Count     int        `json:"count"`
	Page      int        `json:"page"`
	PageCount int        `json:"pageCount"`
	Hits      int        `json:"hits"`
	Items     []koboItem `json:"Items"`
}

// koboItem は、共通書籍モデルへ対応できる楽天Kobo商品情報を保持する
type koboItem struct {
	Title          string `json:"title"`
	TitleKana      string `json:"titleKana"`
	SubTitle       string `json:"subTitle"`
	SeriesName     string `json:"seriesName"`
	Author         string `json:"author"`
	AuthorKana     string `json:"authorKana"`
	PublisherName  string `json:"publisherName"`
	ItemNumber     string `json:"itemNumber"`
	SalesDate      string `json:"salesDate"`
	ItemCaption    string `json:"itemCaption"`
	KoboGenreID    string `json:"koboGenreId"`
	ItemURL        string `json:"itemUrl"`
	AffiliateURL   string `json:"affiliateUrl"`
	ItemPrice      *int64 `json:"itemPrice"`
	SmallImageURL  string `json:"smallImageUrl"`
	MediumImageURL string `json:"mediumImageUrl"`
	LargeImageURL  string `json:"largeImageUrl"`
}

// decodeKoboResponse は、楽天Kobo検索本文を非公開レスポンス型へ変換する
func decodeKoboResponse(body []byte) (koboResponse, error) {
	var response koboResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil {
		return koboResponse{}, fmt.Errorf("decode Rakuten Kobo response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return koboResponse{}, err
	}
	if response.Count < 0 || response.PageCount < 0 || response.Hits < 0 || response.Page < 0 {
		return koboResponse{}, errors.New("response pagination values must not be negative")
	}
	return response, nil
}

// convertItem は、楽天Kobo商品を共通のBookへ変換する
func convertItem(value koboItem, observedAt string) Book {
	book := Book{
		Title:        value.Title,
		TitleReading: value.TitleKana,
		Subtitle:     value.SubTitle,
		Description:  value.ItemCaption,
		Medium:       PublicationMediumDigital,
		Subjects:     subjects(value.KoboGenreID),
		CoverURL:     coverURL(value),
		Sources:      []BookSource{{Source: SourceRakutenKobo, ID: value.ItemNumber, URL: sourceURL(value.ItemURL), AffiliateURL: sourceURL(value.AffiliateURL)}},
	}
	if value.SeriesName != "" {
		book.PublicationSeries = []string{value.SeriesName}
	}
	book.Authors, book.Contributors = contributors(value.Author, value.AuthorKana)
	if value.PublisherName != "" {
		book.Publishers = []string{value.PublisherName}
	}
	if value.SalesDate != "" {
		book.ReleaseDate = value.SalesDate
	}
	if value.ItemPrice != nil {
		taxIncluded := true
		book.CurrentPrice = &Price{Amount: *value.ItemPrice, Currency: "JPY", TaxIncluded: &taxIncluded, Source: SourceRakutenKobo, ObservedAt: observedAt}
	}
	applyTitleMetadata(&book)
	return book
}

// applyTitleMetadata は、タイトルから安全に抽出できた付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	titlemeta.Apply(book)
}

// contributors は、楽天Koboのスラッシュ区切り著者と検証済みの読みを人物単位へ変換する
func contributors(author string, authorKana string) ([]string, []Contributor) {
	parts := strings.Split(author, "/")
	readings := contributorReadings(parts, authorKana)
	authors := make([]string, 0, len(parts))
	contributors := make([]Contributor, 0, len(parts))
	for index, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		authors = append(authors, name)
		contributor := Contributor{Name: name}
		if readings != nil {
			contributor.Reading = readings[index]
		}
		contributors = append(contributors, contributor)
	}
	if len(authors) == 0 {
		return nil, nil
	}
	return authors, contributors
}

// contributorReadings は、楽天Koboで実測したauthorKana形式だけを著者順の読みへ変換する
func contributorReadings(authorParts []string, authorKana string) []string {
	if len(authorParts) == 1 {
		if strings.TrimSpace(authorParts[0]) == "" {
			return nil
		}
		reading := strings.TrimSpace(authorKana)
		if reading == "" || strings.ContainsAny(reading, "/,") {
			return nil
		}
		return []string{reading}
	}
	if len(authorParts) < 2 {
		return nil
	}
	for _, authorPart := range authorParts {
		if strings.TrimSpace(authorPart) == "" {
			return nil
		}
	}
	kanaParts := strings.Split(authorKana, "/")
	if len(kanaParts) != len(authorParts) {
		return nil
	}
	common := strings.TrimSpace(kanaParts[0])
	if common == "" {
		return nil
	}
	for _, kanaPart := range kanaParts[1:] {
		if strings.TrimSpace(kanaPart) != common {
			return nil
		}
	}
	readings := strings.Split(common, ",")
	if len(readings) != len(authorParts) {
		return nil
	}
	for index, reading := range readings {
		readings[index] = strings.TrimSpace(reading)
		if readings[index] == "" {
			return nil
		}
	}
	return readings
}

// sourceURL は、通常商品URLとして利用できるHTTP(S) URLだけを返す
func sourceURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return value
}

// subjects は、楽天Koboのスラッシュ区切りジャンルIDを取得元分類へ変換する
func subjects(value string) []Subject {
	parts := strings.Split(value, "/")
	result := make([]Subject, 0, len(parts))
	for _, part := range parts {
		code := strings.TrimSpace(part)
		if code != "" {
			result = append(result, Subject{Scheme: "rakuten_kobo", Code: code})
		}
	}
	return result
}

// coverURL は、楽天Koboの利用可能な最大サイズ画像URLを返す
func coverURL(value koboItem) string {
	for _, value := range []string{value.LargeImageURL, value.MediumImageURL, value.SmallImageURL} {
		if value != "" {
			return value
		}
	}
	return ""
}
