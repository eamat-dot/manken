package madb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	internalisbn "github.com/eamat-dot/manken/internal/isbn"
)

// sparqlResponse は、SPARQL Results JSONの結果を保持する
type sparqlResponse struct {
	Results *sparqlResults `json:"results"`
}

// sparqlResults は、SPARQL Results JSONのbinding配列を保持する
type sparqlResults struct {
	Bindings []map[string]sparqlValue `json:"bindings"`
}

// sparqlValue は、SPARQL Results JSONの1つのbinding値を保持する
type sparqlValue struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	Language string `json:"xml:lang,omitempty"`
	Datatype string `json:"datatype,omitempty"`
}

// sourceBook は、MADB固有の項目を共通モデルへ変換する前に保持する
type sourceBook struct {
	ID                 string
	Titles             []string
	TitleKana          []string
	Subtitles          []string
	SeriesNames        []string
	RelatedSeriesNames []string
	SeriesID           string
	SeriesResourceURI  string
	VolumeNumber       string
	Versions           []string
	Creators           []string
	AgentNames         []string
	Publishers         []string
	Brands             []string
	ISBNs              []string
	MatchedISBNs       []string
	PublishedDate      string
	PageCount          string
	Size               string
	ResourceURI        string
}

// sourceBookAccumulator は、複数bindingから1冊分の項目を集約する
type sourceBookAccumulator struct {
	book               sourceBook
	titles             map[string]struct{}
	titleKana          map[string]struct{}
	subtitles          map[string]struct{}
	seriesNames        map[string]struct{}
	relatedSeriesNames map[string]struct{}
	versions           map[string]struct{}
	creators           map[string]struct{}
	agentNames         map[string]struct{}
	publishers         map[string]struct{}
	brands             map[string]struct{}
	isbns              map[string]struct{}
	matchedISBNs       map[string]struct{}
}

// buildISBNLookupResult は、SPARQLレスポンスを入力ISBNごとの参照結果へ変換する
func buildISBNLookupResult(
	response sparqlResponse,
	inputs []isbnLookupInput,
) (ISBNLookupResult, error) {
	if response.Results == nil {
		return ISBNLookupResult{}, newError(
			operationISBNLookup,
			ErrorKindInvalidResponse,
			errors.New("SPARQL response does not contain results"),
		)
	}

	sourceBooks, err := aggregateBindings(response.Results.Bindings)
	if err != nil {
		return ISBNLookupResult{}, newError(operationISBNLookup, ErrorKindInvalidResponse, err)
	}

	items := make([]ISBNLookupItem, len(inputs))
	for index, input := range inputs {
		items[index] = ISBNLookupItem{
			RequestedISBN: input.Requested,
			Books:         make([]Book, 0),
		}
	}
	for _, source := range sourceBooks {
		if len(source.MatchedISBNs) == 0 {
			return ISBNLookupResult{}, newError(
				operationISBNLookup,
				ErrorKindInvalidResponse,
				errors.New("SPARQL binding does not contain matchedISBN"),
			)
		}
		book := convertBook(source)
		for index, input := range inputs {
			if stringsIntersect(input.Candidates, source.MatchedISBNs) {
				items[index].Books = append(items[index].Books, book)
			}
		}
	}
	return ISBNLookupResult{Items: items}, nil
}

// stringsIntersect は、2つの文字列集合に共通する値があるか判定する
func stringsIntersect(left []string, right []string) bool {
	for _, leftValue := range left {
		for _, rightValue := range right {
			if leftValue == rightValue {
				return true
			}
		}
	}
	return false
}

// decodeSPARQLResponse は、SPARQL Results JSONを非公開レスポンス型へ変換する
func decodeSPARQLResponse(body []byte) (sparqlResponse, error) {
	var response sparqlResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&response); err != nil {
		return sparqlResponse{}, fmt.Errorf("decode SPARQL response: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return sparqlResponse{}, errors.New("SPARQL response contains trailing data")
	}
	if response.Results == nil || response.Results.Bindings == nil {
		return sparqlResponse{}, errors.New("SPARQL response does not contain results.bindings")
	}
	return response, nil
}

// buildSearchResult は、SPARQLレスポンスを公開検索結果へ変換する
func buildSearchResult(
	response sparqlResponse,
	conditions searchConditions,
	limit int,
) (SearchBooksResult, error) {
	if response.Results == nil {
		return SearchBooksResult{}, newError(
			operationSearchBooks,
			ErrorKindInvalidResponse,
			errors.New("SPARQL response does not contain results"),
		)
	}

	sourceBooks, err := aggregateBindings(response.Results.Bindings)
	if err != nil {
		return SearchBooksResult{}, newError(
			operationSearchBooks,
			ErrorKindInvalidResponse,
			err,
		)
	}

	hasNext := len(sourceBooks) > limit
	if hasNext {
		sourceBooks = sourceBooks[:limit]
	}

	books := make([]Book, 0, len(sourceBooks))
	for _, source := range sourceBooks {
		books = append(books, convertBook(source))
	}

	result := SearchBooksResult{Books: books}
	if hasNext {
		nextCursor, err := encodeCursor(sourceBooks[len(sourceBooks)-1].ResourceURI, conditions, limit)
		if err != nil {
			return SearchBooksResult{}, newError(
				operationSearchBooks,
				ErrorKindInvalidResponse,
				err,
			)
		}
		result.NextCursor = nextCursor
	}
	return result, nil
}

// aggregateBindings は、resourceごとのbindingをsourceBookへ集約する
func aggregateBindings(bindings []map[string]sparqlValue) ([]sourceBook, error) {
	accumulators := make(map[string]*sourceBookAccumulator)
	for _, binding := range bindings {
		resource, ok := binding["resource"]
		if !ok || resource.Type != "uri" || !isValidResourceURI(resource.Value) {
			return nil, errors.New("binding contains an invalid or missing resource")
		}

		accumulator, ok := accumulators[resource.Value]
		if !ok {
			accumulator = newSourceBookAccumulator(resource.Value)
			accumulators[resource.Value] = accumulator
		}
		if err := accumulator.addBinding(binding); err != nil {
			return nil, err
		}
	}

	resourceURIs := make([]string, 0, len(accumulators))
	for resourceURI := range accumulators {
		resourceURIs = append(resourceURIs, resourceURI)
	}
	sort.Strings(resourceURIs)

	books := make([]sourceBook, 0, len(resourceURIs))
	for _, resourceURI := range resourceURIs {
		books = append(books, accumulators[resourceURI].finish())
	}
	return books, nil
}

// newSourceBookAccumulator は、指定resourceのbindingを集約する準備をする
func newSourceBookAccumulator(resourceURI string) *sourceBookAccumulator {
	return &sourceBookAccumulator{
		book:               sourceBook{ResourceURI: resourceURI},
		titles:             make(map[string]struct{}),
		titleKana:          make(map[string]struct{}),
		subtitles:          make(map[string]struct{}),
		seriesNames:        make(map[string]struct{}),
		relatedSeriesNames: make(map[string]struct{}),
		versions:           make(map[string]struct{}),
		creators:           make(map[string]struct{}),
		agentNames:         make(map[string]struct{}),
		publishers:         make(map[string]struct{}),
		brands:             make(map[string]struct{}),
		isbns:              make(map[string]struct{}),
		matchedISBNs:       make(map[string]struct{}),
	}
}

// addBinding は、1つのbindingに含まれる既知項目を集約する
func (accumulator *sourceBookAccumulator) addBinding(binding map[string]sparqlValue) error {
	for name, value := range binding {
		if name == "resource" {
			continue
		}
		if !isKnownVariable(name) {
			continue
		}
		if name == "seriesResource" {
			if value.Type != "uri" || !isValidSeriesResourceURI(value.Value) {
				return fmt.Errorf("binding %q contains an invalid resource URI", name)
			}
			if err := setScalar(&accumulator.book.SeriesResourceURI, value.Value, name); err != nil {
				return err
			}
			continue
		}
		if value.Type != "literal" {
			return fmt.Errorf("binding %q has unexpected type %q", name, value.Type)
		}

		switch name {
		case "id":
			if err := setScalar(&accumulator.book.ID, value.Value, name); err != nil {
				return err
			}
		case "title":
			addValue(accumulator.titles, value.Value)
		case "titleKana":
			addValue(accumulator.titleKana, value.Value)
		case "subtitle":
			addValue(accumulator.subtitles, value.Value)
		case "seriesName":
			addValue(accumulator.seriesNames, value.Value)
		case "relatedSeriesName":
			addValue(accumulator.relatedSeriesNames, value.Value)
		case "seriesID":
			if err := setScalar(&accumulator.book.SeriesID, value.Value, name); err != nil {
				return err
			}
		case "volumeNumber":
			if err := setScalar(&accumulator.book.VolumeNumber, value.Value, name); err != nil {
				return err
			}
		case "version":
			addValue(accumulator.versions, value.Value)
		case "creator":
			addValue(accumulator.creators, value.Value)
		case "agentName":
			addValue(accumulator.agentNames, value.Value)
		case "publisher":
			addValue(accumulator.publishers, value.Value)
		case "brand":
			addValue(accumulator.brands, value.Value)
		case "isbn":
			addValue(accumulator.isbns, value.Value)
		case "matchedISBN":
			addValue(accumulator.matchedISBNs, value.Value)
		case "publishedDate":
			if err := setScalar(&accumulator.book.PublishedDate, value.Value, name); err != nil {
				return err
			}
		case "pageCount":
			if err := setScalar(&accumulator.book.PageCount, value.Value, name); err != nil {
				return err
			}
		case "size":
			if err := setScalar(&accumulator.book.Size, value.Value, name); err != nil {
				return err
			}
		}
	}
	return nil
}

// finish は、集約した集合を安定した順序のsourceBookへ変換する
func (accumulator *sourceBookAccumulator) finish() sourceBook {
	accumulator.book.Titles = sortedValues(accumulator.titles)
	accumulator.book.TitleKana = sortedValues(accumulator.titleKana)
	accumulator.book.Subtitles = sortedValues(accumulator.subtitles)
	accumulator.book.SeriesNames = sortedValues(accumulator.seriesNames)
	accumulator.book.RelatedSeriesNames = sortedValues(accumulator.relatedSeriesNames)
	accumulator.book.Versions = sortedValues(accumulator.versions)
	accumulator.book.Creators = sortedValues(accumulator.creators)
	accumulator.book.AgentNames = sortedValues(accumulator.agentNames)
	accumulator.book.Publishers = sortedValues(accumulator.publishers)
	accumulator.book.Brands = sortedValues(accumulator.brands)
	accumulator.book.ISBNs = sortedValues(accumulator.isbns)
	accumulator.book.MatchedISBNs = sortedValues(accumulator.matchedISBNs)
	return accumulator.book
}

// convertBook は、MADB固有のsourceBookを共通のBookへ変換する
func convertBook(source sourceBook) Book {
	authors, contributors := convertCreators(source)

	identifiers := make([]Identifier, 0, len(source.ISBNs))
	for _, value := range source.ISBNs {
		normalized := normalizeISBN(value)
		switch {
		case internalisbn.IsValidISBN10(normalized):
			identifiers = append(identifiers, Identifier{
				Type:  IdentifierTypeISBN10,
				Value: normalized,
			})
		case internalisbn.IsValidISBN13(normalized):
			identifiers = append(identifiers, Identifier{
				Type:  IdentifierTypeISBN13,
				Value: normalized,
			})
		}
	}

	dates := make([]BookDate, 0, 1)
	if source.PublishedDate != "" {
		dates = append(dates, BookDate{
			Type:  BookDateTypePublished,
			Value: source.PublishedDate,
		})
	}

	pageCount := normalizePageCount(source.PageCount)
	physicalSize := normalizePhysicalSize(source.Size)
	medium := PublicationMediumUnknown
	if physicalSize != nil {
		medium = PublicationMediumPrint
	}

	return Book{
		Normalized: NormalizedBook{
			Title:             firstValue(source.Titles),
			TitleReading:      normalizeTitleKana(source.TitleKana),
			Subtitle:          firstValue(source.Subtitles),
			Series:            convertSeries(source),
			Volume:            normalizeVolume(source.VolumeNumber),
			EditionStatements: source.Versions,
			Authors:           authors,
			Contributors:      contributors,
			Publishers:        normalizePublishers(source.Publishers),
			Imprints:          source.Brands,
			Identifiers:       identifiers,
			Dates:             dates,
			PageCount:         pageCount,
			Medium:            medium,
			PhysicalSize:      physicalSize,
		},
		Sources: []BookSource{{
			Source: SourceMADB,
			ID:     source.ID,
			URL:    source.ResourceURI,
		}},
	}
}

// convertCreators は、MADBのcreator文字列とAgent名を共通の著者と寄与者へ変換する
func convertCreators(source sourceBook) ([]string, []Contributor) {
	if len(source.Creators) == 0 {
		contributors := make([]Contributor, 0, len(source.AgentNames))
		for _, name := range source.AgentNames {
			contributors = append(contributors, Contributor{Name: name})
		}
		return source.AgentNames, contributors
	}

	authors := make([]string, 0, len(source.Creators))
	authorNames := make(map[string]struct{}, len(source.Creators))
	contributors := make([]Contributor, 0, len(source.Creators))
	contributorIndexes := make(map[string]int, len(source.Creators))
	for _, creator := range source.Creators {
		name, roles, hasRole, usable := parseCreator(creator)
		if !usable {
			continue
		}
		if !hasRole || containsAuthorRole(roles) {
			authors = appendUniqueString(authors, authorNames, name)
		}
		contributors = mergeContributor(contributors, contributorIndexes, name, roles)
	}
	return authors, contributors
}

// parseCreator は、creator文字列から人物名、確定済みの共通役割、役割表記の有無を取り出す
func parseCreator(value string) (string, []ContributorRole, bool, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil, false, false
	}
	if !strings.HasPrefix(value, "[") {
		return value, nil, false, true
	}
	if strings.HasPrefix(value, "[[") {
		return "", nil, false, false
	}

	roleEnd := strings.Index(value, "]")
	if roleEnd < 0 {
		return "", nil, false, false
	}
	roles, rolesOK := mapCreatorRoles(value[1:roleEnd])
	name, ok := parseCreatorName(value[roleEnd+1:])
	if !ok {
		return "", nil, false, false
	}
	if !rolesOK {
		return name, nil, true, true
	}
	return name, roles, true, true
}

// mapCreatorRoles は、MADBの単一または複合役割を共通役割へ変換する
func mapCreatorRoles(value string) ([]ContributorRole, bool) {
	parts := strings.Split(value, "・")
	roles := make([]ContributorRole, 0, len(parts))
	seen := make(map[ContributorRole]struct{}, len(parts))
	for _, part := range parts {
		role, ok := mapCreatorRole(part)
		if !ok {
			return nil, false
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles, true
}

// mapCreatorRole は、MADBの単一役割を共通役割へ変換する
func mapCreatorRole(value string) (ContributorRole, bool) {
	switch value {
	case "著", "著者", "作", "共著", "ほか著", "他著":
		return ContributorRoleAuthor, true
	case "原作", "原案", "共原作":
		return ContributorRoleOriginalCreator, true
	case "脚本", "シナリオ", "構成", "脚色", "文", "ストーリー", "ライター":
		return ContributorRoleWriter, true
	case "漫画", "作画", "画", "劇画", "まんが", "絵",
		"comic", "Comic", "COMIC", "comics", "コミック", "マンガ", "アーティスト":
		return ContributorRoleArtist, true
	case "キャラクター原案":
		return ContributorRoleCharacterCreator, true
	case "キャラクターデザイン":
		return ContributorRoleCharacterDesigner, true
	case "編", "編集":
		return ContributorRoleEditor, true
	case "訳":
		return ContributorRoleTranslator, true
	case "監修":
		return ContributorRoleSupervisor, true
	case "解説":
		return ContributorRoleCommentator, true
	case "カバーデザイン", "装丁", "装幀", "デザイン":
		return ContributorRoleDesigner, true
	default:
		return "", false
	}
}

// parseCreatorName は、既知役割より後ろの人物名候補を安全に整形する
func parseCreatorName(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") {
		nameEnd := strings.Index(value, "]")
		if nameEnd < 0 {
			return "", false
		}
		value = value[1:nameEnd] + value[nameEnd+1:]
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

// containsAuthorRole は、役割にAuthorsへ含める主要な創作者があるか判定する
func containsAuthorRole(roles []ContributorRole) bool {
	for _, role := range roles {
		switch role {
		case ContributorRoleAuthor,
			ContributorRoleOriginalCreator,
			ContributorRoleWriter,
			ContributorRoleArtist,
			ContributorRoleCharacterCreator,
			ContributorRoleCharacterDesigner:
			return true
		}
	}
	return false
}

// appendUniqueString は、未追加の文字列だけを順序を変えずに追加する
func appendUniqueString(values []string, seen map[string]struct{}, value string) []string {
	if _, exists := seen[value]; exists {
		return values
	}
	seen[value] = struct{}{}
	return append(values, value)
}

// mergeContributor は、同名の寄与者をまとめて未追加の役割を加える
func mergeContributor(
	contributors []Contributor,
	indexes map[string]int,
	name string,
	roles []ContributorRole,
) []Contributor {
	index, exists := indexes[name]
	if !exists {
		indexes[name] = len(contributors)
		return append(contributors, Contributor{Name: name, Roles: roles})
	}

	seen := make(map[ContributorRole]struct{}, len(contributors[index].Roles))
	for _, role := range contributors[index].Roles {
		seen[role] = struct{}{}
	}
	for _, role := range roles {
		if _, duplicated := seen[role]; duplicated {
			continue
		}
		seen[role] = struct{}{}
		contributors[index].Roles = append(contributors[index].Roles, role)
	}
	return contributors
}

// normalizeTitleKana は、空白を除いた最多のタイトル読みが一意な場合に返す
func normalizeTitleKana(values []string) string {
	counts := make(map[string]int, len(values))
	for _, value := range values {
		normalized := strings.Map(func(character rune) rune {
			if unicode.IsSpace(character) {
				return -1
			}
			return character
		}, value)
		if normalized != "" {
			counts[normalized]++
		}
	}

	var selected string
	maxCount := 0
	tied := false
	for value, count := range counts {
		switch {
		case count > maxCount:
			selected = value
			maxCount = count
			tied = false
		case count == maxCount:
			tied = true
		}
	}
	if tied {
		return ""
	}
	return selected
}

// normalizePublishers は、出版社名に付記されたカナ読みを除き重複を取り除く
func normalizePublishers(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if separator := strings.IndexRune(name, '∥'); separator >= 0 {
			reading := strings.TrimSpace(name[separator+len("∥"):])
			if isKatakanaReading(reading) {
				name = strings.TrimSpace(name[:separator])
			}
		}
		if name != "" {
			normalized = append(normalized, name)
		}
	}
	return uniqueSorted(normalized)
}

// isKatakanaReading は、文字列が出版社名のカナ読みに見えるか判定する
func isKatakanaReading(value string) bool {
	hasKatakana := false
	for _, character := range value {
		switch {
		case unicode.In(character, unicode.Katakana):
			hasKatakana = true
		case unicode.IsSpace(character):
		case character == 'ー', character == '・', character == '･', character == 'ヽ', character == 'ヾ':
		default:
			return false
		}
	}
	return hasKatakana
}

var pageCountPattern = regexp.MustCompile(`^([0-9０-９]+)[pPｐＰ]?$`)

// normalizePageCount は、MADBのページ数表記を整数へ変換する
func normalizePageCount(value string) *int {
	value = removeUnicodeSpaces(value)
	matches := pageCountPattern.FindStringSubmatch(value)
	if matches == nil {
		return nil
	}
	digits, ok := normalizeDigits(matches[1])
	if !ok {
		return nil
	}
	pageCount, err := strconv.Atoi(digits)
	if err != nil {
		return nil
	}
	return &pageCount
}

var physicalSizePattern = regexp.MustCompile(
	`(?i)^([0-9０-９]+(?:[.．][0-9０-９])?)cm(?:[×xX＊*]([0-9０-９]+(?:[.．][0-9０-９])?)cm)?$`,
)

// normalizePhysicalSize は、MADBのセンチメートル表記をミリメートル単位へ変換する
func normalizePhysicalSize(value string) *PhysicalSize {
	value = removeUnicodeSpaces(value)
	matches := physicalSizePattern.FindStringSubmatch(value)
	if matches == nil {
		return nil
	}
	height, ok := centimetersToMillimeters(matches[1])
	if !ok {
		return nil
	}
	result := &PhysicalSize{HeightMM: &height}
	if matches[2] == "" {
		return result
	}
	width, ok := centimetersToMillimeters(matches[2])
	if !ok {
		return nil
	}
	result.WidthMM = &width
	return result
}

// removeUnicodeSpaces は、文字列からUnicode空白文字を取り除く
func removeUnicodeSpaces(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return -1
		}
		return character
	}, value)
}

// centimetersToMillimeters は、整数または小数第1位までのセンチメートル値をミリメートルへ変換する
func centimetersToMillimeters(value string) (int, bool) {
	value = strings.ReplaceAll(value, "．", ".")
	parts := strings.Split(value, ".")
	integerDigits, ok := normalizeDigits(parts[0])
	if !ok {
		return 0, false
	}
	integerPart, err := strconv.Atoi(integerDigits)
	if err != nil {
		return 0, false
	}
	millimeters := integerPart * 10
	if len(parts) == 1 {
		return millimeters, true
	}
	decimalDigits, ok := normalizeDigits(parts[1])
	if !ok || len(decimalDigits) != 1 {
		return 0, false
	}
	decimalPart, err := strconv.Atoi(decimalDigits)
	if err != nil {
		return 0, false
	}
	return millimeters + decimalPart, true
}

// firstValue は、文字列スライスの先頭または空文字列を返す
func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// convertSeries は、MADBの直接指定と参照先のシリーズを共通モデルへ変換する
func convertSeries(source sourceBook) []Series {
	series := make([]Series, 0, len(source.SeriesNames)+len(source.RelatedSeriesNames))
	indexes := make(map[string]int)

	for _, name := range source.SeriesNames {
		indexes[name] = len(series)
		series = append(series, Series{Name: name})
	}
	for _, name := range source.RelatedSeriesNames {
		value := Series{
			Name:   name,
			ID:     source.SeriesID,
			URL:    source.SeriesResourceURI,
			Source: SourceMADB,
		}
		if index, ok := indexes[name]; ok {
			series[index] = value
			continue
		}
		indexes[name] = len(series)
		series = append(series, value)
	}

	return series
}

var volumeNumberPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^([0-9０-９]{1,3})$`),
	regexp.MustCompile(`^第\s*([0-9０-９]{1,3})\s*[巻卷]$`),
	regexp.MustCompile(`^([0-9０-９]{1,3})\s*[巻卷]$`),
	regexp.MustCompile(`(?i)^v\.?\s*([0-9０-９]{1,3})$`),
	regexp.MustCompile(`(?i)^vol(?:ume)?\.?\s*([0-9０-９]{1,3})$`),
	regexp.MustCompile(`(?i)^no\.?\s*([0-9０-９]{1,3})$`),
	regexp.MustCompile(`^[#＃]\s*([0-9０-９]{1,3})$`),
	regexp.MustCompile(`^\(([0-9０-９]{1,3})\)$`),
	regexp.MustCompile(`^（([0-9０-９]{1,3})）$`),
	regexp.MustCompile(`^\[([0-9０-９]{1,3})\]$`),
	regexp.MustCompile(`^［([0-9０-９]{1,3})］$`),
}

var decimalVolumePattern = regexp.MustCompile(
	`^([0-9０-９]+)[.．]([0-9０-９]+)(?:[巻卷])?$`,
)

// normalizeVolume は、許可したMADB巻数表記を共通の巻数へ変換する
func normalizeVolume(value string) Volume {
	value = strings.TrimSpace(value)
	if value == "" {
		return Volume{}
	}

	separated := strings.ReplaceAll(value, "／", "/")
	if strings.Contains(separated, "/") {
		parts := strings.Split(separated, "/")
		for _, part := range parts {
			if strings.TrimSpace(part) == "" {
				return Volume{}
			}
		}
		return normalizeRepeatedVolume(parts)
	}

	return normalizeAtomicVolume(value)
}

// normalizeRepeatedVolume は、同じ巻を併記した表記だけを共通の巻数へ変換する
func normalizeRepeatedVolume(parts []string) Volume {
	var normalized Volume
	for index, part := range parts {
		current := normalizeAtomicVolume(strings.TrimSpace(part))
		if current.Number == nil && current.Label == "" {
			return Volume{}
		}
		if index == 0 {
			normalized = current
			continue
		}
		if !equalVolumes(normalized, current) {
			return Volume{}
		}
	}
	return normalized
}

// equalVolumes は、2つの正規化済み巻数が同じ値か判定する
func equalVolumes(left, right Volume) bool {
	if left.Label != right.Label {
		return false
	}
	if left.Number == nil || right.Number == nil {
		return left.Number == nil && right.Number == nil
	}
	return *left.Number == *right.Number
}

// normalizeAtomicVolume は、単一の巻数表記を共通の巻数へ変換する
func normalizeAtomicVolume(value string) Volume {
	trimmed := strings.TrimSpace(value)
	if label, ok := normalizeVolumeLabel(trimmed); ok {
		return Volume{Label: label}
	}
	if matches := decimalVolumePattern.FindStringSubmatch(trimmed); matches != nil {
		integerPart, integerOK := normalizeDigits(matches[1])
		decimalPart, decimalOK := normalizeDigits(matches[2])
		if integerOK && decimalOK {
			return Volume{Label: integerPart + "." + decimalPart}
		}
	}

	for _, pattern := range volumeNumberPatterns {
		matches := pattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}
		digits, ok := normalizeDigits(matches[1])
		if !ok {
			return Volume{}
		}
		number, err := strconv.Atoi(digits)
		if err != nil {
			return Volume{}
		}
		return Volume{Number: &number, Label: strconv.Itoa(number)}
	}

	return Volume{}
}

// normalizeVolumeLabel は、認識する非整数巻の表記を共通ラベルへ変換する
func normalizeVolumeLabel(value string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, brackets := range [][2]string{{"(", ")"}, {"（", "）"}, {"[", "]"}, {"［", "］"}} {
		if strings.HasPrefix(value, brackets[0]) && strings.HasSuffix(value, brackets[1]) {
			value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, brackets[0]), brackets[1]))
			break
		}
	}

	switch value {
	case "上", "上巻":
		return "上", true
	case "中", "中巻":
		return "中", true
	case "下", "下巻":
		return "下", true
	case "前編":
		return "前編", true
	case "後編":
		return "後編", true
	case "別巻":
		return "別巻", true
	case "外伝":
		return "外伝", true
	case "番外編":
		return "番外編", true
	default:
		return "", false
	}
}

// normalizeDigits は、ASCII数字と全角数字をASCII数字列へ変換する
func normalizeDigits(value string) (string, bool) {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, character := range value {
		switch {
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
		case character >= '０' && character <= '９':
			builder.WriteRune('0' + character - '０')
		default:
			return "", false
		}
	}
	if builder.Len() == 0 {
		return "", false
	}
	return builder.String(), true
}

// isKnownVariable は、レスポンスで型を検証する既知変数か判定する
func isKnownVariable(name string) bool {
	switch name {
	case "id", "title", "subtitle", "seriesName", "seriesResource",
		"relatedSeriesName", "seriesID", "volumeNumber", "version",
		"creator", "agentName", "publisher", "brand", "isbn", "matchedISBN", "publishedDate",
		"titleKana", "pageCount", "size":
		return true
	default:
		return false
	}
}

// setScalar は、単一値の重複を許可しつつ競合する値を拒否する
func setScalar(target *string, value, name string) error {
	if value == "" {
		return nil
	}
	if *target == "" || *target == value {
		*target = value
		return nil
	}
	return fmt.Errorf("binding %q contains conflicting values", name)
}

// addValue は、空でない値を集合へ追加する
func addValue(values map[string]struct{}, value string) {
	if value != "" {
		values[value] = struct{}{}
	}
}

// sortedValues は、集合を文字列昇順のスライスへ変換する
func sortedValues(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// uniqueSorted は、空値と重複を除いた文字列を昇順で返す
func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		addValue(set, value)
	}
	return sortedValues(set)
}

// isValidSeriesResourceURI は、MADBのマンガ単行本シリーズリソースURIか検証する
func isValidSeriesResourceURI(resourceURI string) bool {
	const prefix = "https://mediaarts-db.artmuseums.go.jp/id/C"
	if !strings.HasPrefix(resourceURI, prefix) {
		return false
	}
	id := strings.TrimPrefix(resourceURI, prefix)
	if id == "" {
		return false
	}
	for _, character := range id {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
