package openbd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// oneOrMany は、JSONの単一オブジェクトまたは配列を同じスライスで保持する
type oneOrMany[T any] []T

// UnmarshalJSON は、単一オブジェクトと配列の両方を読み込む
func (values *oneOrMany[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*values = nil
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return json.Unmarshal(trimmed, (*[]T)(values))
	}
	var value T
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return err
	}
	*values = []T{value}
	return nil
}

// responseBook は、初期変換に必要なopenBD書誌を保持する
type responseBook struct {
	Onix    responseOnix    `json:"onix"`
	Hanmoto responseHanmoto `json:"hanmoto"`
	Summary responseSummary `json:"summary"`
}

// responseOnix は、初期変換に必要なONIX項目を保持する
type responseOnix struct {
	RecordReference   string                    `json:"RecordReference"`
	ProductIdentifier responseProductIdentifier `json:"ProductIdentifier"`
	DescriptiveDetail responseDescriptiveDetail `json:"DescriptiveDetail"`
	PublishingDetail  responsePublishingDetail  `json:"PublishingDetail"`
}

// responseProductIdentifier は、ONIXの商品識別子を保持する
type responseProductIdentifier struct {
	ProductIDType string `json:"ProductIDType"`
	IDValue       string `json:"IDValue"`
}

// responseDescriptiveDetail は、ONIXのタイトル、シリーズ、寄与者を保持する
type responseDescriptiveDetail struct {
	TitleDetail responseTitleDetail            `json:"TitleDetail"`
	Collection  oneOrMany[responseCollection]  `json:"Collection"`
	Contributor oneOrMany[responseContributor] `json:"Contributor"`
}

// responseTitleDetail は、ONIXのタイトル種別と要素を保持する
type responseTitleDetail struct {
	TitleType    string                          `json:"TitleType"`
	TitleElement oneOrMany[responseTitleElement] `json:"TitleElement"`
}

// responseTitleElement は、ONIXのタイトル階層と表示を保持する
type responseTitleElement struct {
	TitleElementLevel string               `json:"TitleElementLevel"`
	TitleText         responseContentValue `json:"TitleText"`
	Subtitle          responseContentValue `json:"Subtitle"`
}

// responseContentValue は、ONIXの表示値と照合キーを保持する
type responseContentValue struct {
	Content      string `json:"content"`
	CollationKey string `json:"collationkey"`
}

// responseCollection は、ONIXのコレクションタイトルを保持する
type responseCollection struct {
	TitleDetail responseTitleDetail `json:"TitleDetail"`
}

// responseContributor は、ONIXの寄与者名、順序、役割を保持する
type responseContributor struct {
	SequenceNumber  string               `json:"SequenceNumber"`
	ContributorRole []string             `json:"ContributorRole"`
	PersonName      responseContentValue `json:"PersonName"`
}

// responsePublishingDetail は、ONIXの出版社と出版日を保持する
type responsePublishingDetail struct {
	Imprint        oneOrMany[responseImprint]        `json:"Imprint"`
	Publisher      oneOrMany[responsePublisher]      `json:"Publisher"`
	PublishingDate oneOrMany[responsePublishingDate] `json:"PublishingDate"`
}

// responseImprint は、ONIXの発行元名を保持する
type responseImprint struct {
	ImprintName string `json:"ImprintName"`
}

// responsePublisher は、ONIXの発売元名を保持する
type responsePublisher struct {
	PublisherName string `json:"PublisherName"`
}

// responsePublishingDate は、ONIXの日付役割と値を保持する
type responsePublishingDate struct {
	PublishingDateRole string `json:"PublishingDateRole"`
	Date               string `json:"Date"`
}

// responseHanmoto は、初期変換に必要な版元ドットコム項目を保持する
type responseHanmoto struct {
	DateShuppan string `json:"dateshuppan"`
}

// responseSummary は、openBDの要約項目を保持する
type responseSummary struct {
	ISBN      string `json:"isbn"`
	Title     string `json:"title"`
	Series    string `json:"series"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	PubDate   string `json:"pubdate"`
	Cover     string `json:"cover"`
}

// sourceBook は、openBD固有の主要値を共通モデルへ変換する前に保持する
type sourceBook struct {
	ISBNs         []string
	Titles        []string
	TitleKana     []string
	Subtitles     []string
	SeriesNames   []string
	Authors       []string
	Contributors  []Contributor
	Publishers    []string
	PublishedDate string
	Cover         string
}

// decodeResponse は、openBDのJSON配列を非公開レスポンス型へ変換する
func decodeResponse(body []byte) ([]*responseBook, error) {
	var response []*responseBook
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode openBD response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, err
	}
	if response == nil {
		return nil, errors.New("openBD response must be a JSON array")
	}
	return response, nil
}

// ensureJSONEnd は、JSON値の後ろに別の値がないことを検証する
func ensureJSONEnd(decoder *json.Decoder) error {
	var extra json.RawMessage
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("decode trailing JSON data: %w", err)
	}
	return errors.New("openBD response contains trailing data")
}

// buildISBNLookupResult は、openBDレスポンスを入力ISBNごとの参照結果へ変換する
func buildISBNLookupResult(
	response []*responseBook,
	inputs []isbnLookupInput,
	queries []string,
) (ISBNLookupResult, error) {
	if len(response) != len(queries) {
		return ISBNLookupResult{}, newError(
			operationISBNLookup,
			ErrorKindInvalidResponse,
			fmt.Errorf("response item count is %d, want %d", len(response), len(queries)),
		)
	}

	booksByISBN := make(map[string][]Book, len(queries))
	for index, value := range response {
		queryISBN := queries[index]
		booksByISBN[queryISBN] = make([]Book, 0)
		if value == nil {
			continue
		}
		if err := validateResponseISBN(*value, queryISBN); err != nil {
			return ISBNLookupResult{}, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
		}
		booksByISBN[queryISBN] = append(booksByISBN[queryISBN], convertBook(*value, queryISBN))
	}

	items := make([]ISBNLookupItem, len(inputs))
	for index, input := range inputs {
		sourceBooks := booksByISBN[input.canonical]
		books := make([]Book, len(sourceBooks))
		copy(books, sourceBooks)
		items[index] = ISBNLookupItem{
			RequestedISBN: input.requested,
			Books:         books,
		}
	}
	return ISBNLookupResult{Items: items}, nil
}

// validateResponseISBN は、応答内のISBNが問い合わせISBNと矛盾しないことを検証する
func validateResponseISBN(book responseBook, expected string) error {
	found := false
	for _, value := range responseISBNValues(book) {
		if value == "" {
			continue
		}
		canonical, err := canonicalISBN13(value)
		if err != nil {
			return fmt.Errorf("response contains invalid ISBN %q", value)
		}
		if canonical != expected {
			return fmt.Errorf("response ISBN %q does not match requested ISBN %q", value, expected)
		}
		found = true
	}
	if !found {
		return errors.New("response does not contain a verifiable ISBN")
	}
	return nil
}

// responseISBNValues は、openBD応答に含まれるISBNを取得順に重複なく返す
func responseISBNValues(book responseBook) []string {
	values := make([]string, 0, 3)
	values = appendUnique(values, book.Onix.RecordReference)
	if book.Onix.ProductIdentifier.ProductIDType == "15" {
		values = appendUnique(values, book.Onix.ProductIdentifier.IDValue)
	}
	return appendUnique(values, book.Summary.ISBN)
}

// convertBook は、openBDレスポンスを共通のBookへ変換する
func convertBook(response responseBook, isbn string) Book {
	source := collectSourceBook(response)
	title, parallelTitles, volume := parseSourceTitle(firstValue(source.Titles), len(source.SeriesNames) > 0)
	titleKana := normalizeTitleKana(firstValue(source.TitleKana), firstValue(source.Titles), title, volume)

	series := make([]Series, 0, len(source.SeriesNames))
	for _, name := range source.SeriesNames {
		series = append(series, Series{Name: name, Source: SourceOpenBD})
	}

	dates := make([]BookDate, 0, 1)
	if source.PublishedDate != "" {
		dates = append(dates, BookDate{Type: BookDateTypePublished, Value: source.PublishedDate})
	}

	images := make([]Image, 0, 1)
	if source.Cover != "" {
		images = append(images, Image{URL: source.Cover, Purpose: "cover"})
	}

	return Book{
		Normalized: NormalizedBook{
			Title:          title,
			ParallelTitles: parallelTitles,
			TitleKana:      titleKana,
			Subtitle:       firstValue(source.Subtitles),
			Series:         series,
			Volume:         volume,
			Authors:        source.Authors,
			Contributors:   source.Contributors,
			Publishers:     source.Publishers,
			Identifiers: []Identifier{{
				Type:  IdentifierTypeISBN13,
				Value: isbn,
			}},
			Dates:  dates,
			Images: images,
		},
		Sources: []BookSource{{
			Source: SourceOpenBD,
			ID:     isbn,
			Values: SourceBookValues{
				Titles:        source.Titles,
				TitleKana:     source.TitleKana,
				Subtitles:     source.Subtitles,
				SeriesNames:   source.SeriesNames,
				Authors:       source.Authors,
				Publishers:    source.Publishers,
				ISBNs:         source.ISBNs,
				PublishedDate: source.PublishedDate,
			},
		}},
	}
}

// collectSourceBook は、優先順位に従ってopenBDの主要値を収集する
func collectSourceBook(response responseBook) sourceBook {
	result := sourceBook{ISBNs: responseISBNValues(response)}
	for _, element := range response.Onix.DescriptiveDetail.TitleDetail.TitleElement {
		if response.Onix.DescriptiveDetail.TitleDetail.TitleType != "01" ||
			element.TitleElementLevel != "01" {
			continue
		}
		result.Titles = appendUnique(result.Titles, element.TitleText.Content)
		result.TitleKana = appendUnique(result.TitleKana, element.TitleText.CollationKey)
		result.Subtitles = appendUnique(result.Subtitles, element.Subtitle.Content)
	}
	result.Titles = appendUnique(result.Titles, response.Summary.Title)

	for _, collection := range response.Onix.DescriptiveDetail.Collection {
		for _, element := range collection.TitleDetail.TitleElement {
			result.SeriesNames = appendUnique(result.SeriesNames, element.TitleText.Content)
		}
	}
	if len(result.SeriesNames) == 0 {
		result.SeriesNames = appendUnique(result.SeriesNames, response.Summary.Series)
	}

	result.Authors, result.Contributors = convertContributors(response.Onix.DescriptiveDetail.Contributor)
	if len(result.Authors) == 0 && len(response.Onix.DescriptiveDetail.Contributor) == 0 {
		result.Authors = appendUnique(result.Authors, response.Summary.Author)
	}

	for _, imprint := range response.Onix.PublishingDetail.Imprint {
		result.Publishers = appendUnique(result.Publishers, imprint.ImprintName)
	}
	for _, publisher := range response.Onix.PublishingDetail.Publisher {
		result.Publishers = appendUnique(result.Publishers, publisher.PublisherName)
	}
	if len(result.Publishers) == 0 {
		result.Publishers = appendUnique(result.Publishers, response.Summary.Publisher)
	}

	result.PublishedDate = selectPublishedDate(response)
	result.Cover = response.Summary.Cover
	return result
}

// convertContributors は、ONIX寄与者をAuthorsと既知役割へ変換する
func convertContributors(values []responseContributor) ([]string, []Contributor) {
	ordered := append([]responseContributor(nil), values...)
	sort.SliceStable(ordered, func(left, right int) bool {
		leftNumber, leftOK := parseSequenceNumber(ordered[left].SequenceNumber)
		rightNumber, rightOK := parseSequenceNumber(ordered[right].SequenceNumber)
		if leftOK != rightOK {
			return leftOK
		}
		return leftOK && leftNumber < rightNumber
	})

	var authors []string
	var contributors []Contributor
	for _, value := range ordered {
		name := value.PersonName.Content
		if name == "" {
			continue
		}
		if len(value.ContributorRole) == 0 {
			authors = appendUnique(authors, name)
			continue
		}

		roles := mapContributorRoles(value.ContributorRole)
		if len(roles) == 0 {
			continue
		}
		contributors = append(contributors, Contributor{Name: name, Roles: roles})
		if containsAuthorRole(roles) {
			authors = appendUnique(authors, name)
		}
	}
	return authors, contributors
}

// parseSequenceNumber は、正のONIX寄与者順序を整数へ変換する
func parseSequenceNumber(value string) (int, bool) {
	number, err := strconv.Atoi(value)
	return number, err == nil && number > 0
}

// mapContributorRoles は、既知のONIX寄与者役割を重複なく共通役割へ変換する
func mapContributorRoles(values []string) []ContributorRole {
	roles := make([]ContributorRole, 0, len(values))
	seen := make(map[ContributorRole]struct{}, len(values))
	for _, value := range values {
		role, ok := mapContributorRole(value)
		if !ok {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles
}

// mapContributorRole は、対応済みのONIX寄与者役割を共通役割へ変換する
func mapContributorRole(value string) (ContributorRole, bool) {
	switch value {
	case "A01":
		return ContributorRoleAuthor, true
	case "A03", "A14", "A45":
		return ContributorRoleWriter, true
	case "A07", "A12", "A35", "A46", "A47":
		return ContributorRoleArtist, true
	case "A38":
		return ContributorRoleOriginalCreator, true
	case "A36":
		return ContributorRoleDesigner, true
	case "B01":
		return ContributorRoleEditor, true
	case "B06":
		return ContributorRoleTranslator, true
	default:
		return "", false
	}
}

// containsAuthorRole は、Authorsへ含める主要な創作者役割があるか判定する
func containsAuthorRole(roles []ContributorRole) bool {
	for _, role := range roles {
		switch role {
		case ContributorRoleAuthor, ContributorRoleOriginalCreator,
			ContributorRoleWriter, ContributorRoleArtist:
			return true
		}
	}
	return false
}

// selectPublishedDate は、対応可能な出版日を採用順序に従って選ぶ
func selectPublishedDate(response responseBook) string {
	for _, date := range response.Onix.PublishingDetail.PublishingDate {
		if (date.PublishingDateRole == "01" || date.PublishingDateRole == "11") && date.Date != "" {
			return date.Date
		}
	}
	if response.Hanmoto.DateShuppan != "" {
		return response.Hanmoto.DateShuppan
	}
	return response.Summary.PubDate
}

// appendUnique は、空でなく未追加の文字列だけを順序を変えずに追加する
func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// firstValue は、文字列スライスの先頭または空文字列を返す
func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
