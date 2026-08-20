package openbd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
	"github.com/eamat-dot/manken/internal/titlemeta"
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
	ProductSupply     responseProductSupply     `json:"ProductSupply"`
}

// responseProductSupply は、ONIXの供給詳細を保持する
type responseProductSupply struct {
	SupplyDetail oneOrMany[responseSupplyDetail] `json:"SupplyDetail"`
}

// responseSupplyDetail は、ONIXの供給価格を保持する
type responseSupplyDetail struct {
	Price oneOrMany[responsePrice] `json:"Price"`
}

// responsePrice は、ONIXの価格種別、通貨、金額を保持する
type responsePrice struct {
	PriceType    string `json:"PriceType"`
	CurrencyCode string `json:"CurrencyCode"`
	PriceAmount  string `json:"PriceAmount"`
}

// responseProductIdentifier は、ONIXの商品識別子を保持する
type responseProductIdentifier struct {
	ProductIDType string `json:"ProductIDType"`
	IDValue       string `json:"IDValue"`
}

// responseDescriptiveDetail は、ONIXのタイトルと寄与者を保持する
type responseDescriptiveDetail struct {
	TitleDetail responseTitleDetail            `json:"TitleDetail"`
	Contributor oneOrMany[responseContributor] `json:"Contributor"`
	Collection  oneOrMany[responseCollection]  `json:"Collection"`
}

// responseCollection は、ONIXのコレクション階層にある出版系列表示を保持する
type responseCollection struct {
	TitleDetail responseTitleDetail `json:"TitleDetail"`
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
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	PubDate   string `json:"pubdate"`
	Cover     string `json:"cover"`
	Series    string `json:"series"`
}

// sourceBook は、openBD固有の主要値を共通モデルへ変換する前に保持する
type sourceBook struct {
	Title             string
	TitleReading      string
	Subtitle          string
	Authors           []string
	Contributors      []Contributor
	Publishers        []string
	PublishedDate     string
	Cover             string
	PublicationSeries []string
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
		canonical, err := internalisbn.Canonical13(value)
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

	book := Book{
		Title:             source.Title,
		TitleReading:      source.TitleReading,
		Subtitle:          source.Subtitle,
		PublicationSeries: source.PublicationSeries,
		Authors:           source.Authors,
		Contributors:      source.Contributors,
		Publishers:        source.Publishers,
		ISBN13:            []string{isbn},
		PublishedDate:     source.PublishedDate,
		CoverURL:          source.Cover,
		Sources: []BookSource{{
			Source: SourceOpenBD,
			ID:     isbn,
		}},
	}
	applyTitleMetadata(&book)
	book.ListPrice = listPrice(response.Onix.ProductSupply)
	return book
}

// listPrice は、意味を判定できる唯一のONIX推奨小売価格を返す
func listPrice(supply responseProductSupply) *Price {
	type candidate struct {
		amount      int64
		taxIncluded bool
	}
	values := map[candidate]struct{}{}
	for _, detail := range supply.SupplyDetail {
		for _, value := range detail.Price {
			taxIncluded, ok := priceTaxIncluded(value.PriceType)
			if !ok || value.CurrencyCode != "JPY" || !isASCIIInteger(value.PriceAmount) {
				continue
			}
			amount, err := strconv.ParseInt(value.PriceAmount, 10, 64)
			if err != nil || amount < 0 {
				continue
			}
			values[candidate{amount: amount, taxIncluded: taxIncluded}] = struct{}{}
		}
	}
	if len(values) != 1 {
		return nil
	}
	for value := range values {
		taxIncluded := value.taxIncluded
		return &Price{Amount: value.amount, Currency: "JPY", TaxIncluded: &taxIncluded, Source: SourceOpenBD}
	}
	return nil
}

// priceTaxIncluded は、対応するONIX PriceTypeの税込情報を返す
func priceTaxIncluded(value string) (bool, bool) {
	switch value {
	case "01":
		return false, true
	case "02":
		return true, true
	default:
		return false, false
	}
}

// isASCIIInteger は、値が空でないASCII数字だけで構成されるか判定する
func isASCIIInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

var explicitVolumePattern = regexp.MustCompile(`(?:第)?[0-9０-９]+巻|#[0-9０-９]+`)

// applyTitleMetadata は、並列タイトルの曖昧な末尾数値を除き安全な付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	metadata := titlemeta.Parse(book.Title)
	if strings.Contains(book.Title, "=") && !explicitVolumePattern.MatchString(book.Title) {
		metadata.Volume = Volume{}
		metadata.IsFinalVolume = false
	}
	if book.Volume.Number == nil && book.Volume.Label == "" {
		book.Volume = metadata.Volume
	}
	if len(book.Editions) == 0 {
		book.Editions = metadata.Editions
	}
	if metadata.IsFinalVolume {
		book.IsFinalVolume = true
	}
}

// collectSourceBook は、優先順位に従ってopenBDの主要値を収集する
func collectSourceBook(response responseBook) sourceBook {
	var result sourceBook
	for _, element := range response.Onix.DescriptiveDetail.TitleDetail.TitleElement {
		if response.Onix.DescriptiveDetail.TitleDetail.TitleType != "01" ||
			element.TitleElementLevel != "01" || element.TitleText.Content == "" {
			continue
		}
		result.Title = element.TitleText.Content
		result.TitleReading = element.TitleText.CollationKey
		result.Subtitle = element.Subtitle.Content
		break
	}
	if result.Title == "" {
		result.Title = response.Summary.Title
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
	for _, collection := range response.Onix.DescriptiveDetail.Collection {
		for _, element := range collection.TitleDetail.TitleElement {
			// Collection内でも商品階層のTitleElementが混在し得るため、Collection階層だけを出版系列として扱う
			if element.TitleElementLevel != "02" {
				continue
			}
			result.PublicationSeries = appendUnique(result.PublicationSeries, element.TitleText.Content)
		}
	}
	result.PublicationSeries = appendUnique(result.PublicationSeries, response.Summary.Series)
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
		}

		roles := mapContributorRoles(value.ContributorRole)
		contributors = append(contributors, Contributor{
			Name:    name,
			Reading: value.PersonName.CollationKey,
			Roles:   roles,
		})
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
func mapContributorRoles(values []string) []string {
	roles := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
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
func mapContributorRole(value string) (string, bool) {
	switch value {
	case "A01":
		return "著者", true
	case "A03", "A14", "A45":
		return "脚本", true
	case "A07", "A12", "A35":
		return "作画", true
	case "B01":
		return "編集", true
	case "B06":
		return "翻訳", true
	default:
		return "", false
	}
}

// containsAuthorRole は、Authorsへ含める主要な創作者役割があるか判定する
func containsAuthorRole(roles []string) bool {
	for _, role := range roles {
		switch role {
		case "著者", "脚本", "作画":
			return true
		}
	}
	return false
}

// selectPublishedDate は、対応可能な出版日を採用順序に従って選ぶ
func selectPublishedDate(response responseBook) string {
	for _, date := range response.Onix.PublishingDetail.PublishingDate {
		if date.PublishingDateRole == "01" && date.Date != "" {
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
