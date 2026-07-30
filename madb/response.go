package madb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// sparqlResponse は、SPARQL Results JSONの検索結果を保持する
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
	ID                string
	Titles            []string
	Subtitles         []string
	SeriesNames       []string
	SeriesID          string
	SeriesResourceURI string
	VolumeNumber      string
	Versions          []string
	Creators          []string
	AgentNames        []string
	Publishers        []string
	Brands            []string
	ISBNs             []string
	PublishedDate     string
	ResourceURI       string
}

// sourceBookAccumulator は、複数bindingから1冊分の項目を集約する
type sourceBookAccumulator struct {
	book       sourceBook
	titles     map[string]struct{}
	subtitles  map[string]struct{}
	series     map[string]struct{}
	versions   map[string]struct{}
	creators   map[string]struct{}
	agentNames map[string]struct{}
	publishers map[string]struct{}
	brands     map[string]struct{}
	isbns      map[string]struct{}
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
	title string,
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
		nextCursor, err := encodeCursor(sourceBooks[len(sourceBooks)-1].ResourceURI, title, limit)
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
		book:       sourceBook{ResourceURI: resourceURI},
		titles:     make(map[string]struct{}),
		subtitles:  make(map[string]struct{}),
		series:     make(map[string]struct{}),
		versions:   make(map[string]struct{}),
		creators:   make(map[string]struct{}),
		agentNames: make(map[string]struct{}),
		publishers: make(map[string]struct{}),
		brands:     make(map[string]struct{}),
		isbns:      make(map[string]struct{}),
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
		case "subtitle":
			addValue(accumulator.subtitles, value.Value)
		case "seriesName":
			addValue(accumulator.series, value.Value)
		case "relatedSeriesName":
			addValue(accumulator.series, value.Value)
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
		case "publishedDate":
			if err := setScalar(&accumulator.book.PublishedDate, value.Value, name); err != nil {
				return err
			}
		}
	}
	return nil
}

// finish は、集約した集合を安定した順序のsourceBookへ変換する
func (accumulator *sourceBookAccumulator) finish() sourceBook {
	accumulator.book.Titles = sortedValues(accumulator.titles)
	accumulator.book.Subtitles = sortedValues(accumulator.subtitles)
	accumulator.book.SeriesNames = sortedValues(accumulator.series)
	accumulator.book.Versions = sortedValues(accumulator.versions)
	accumulator.book.Creators = sortedValues(accumulator.creators)
	accumulator.book.AgentNames = sortedValues(accumulator.agentNames)
	accumulator.book.Publishers = sortedValues(accumulator.publishers)
	accumulator.book.Brands = sortedValues(accumulator.brands)
	accumulator.book.ISBNs = sortedValues(accumulator.isbns)
	return accumulator.book
}

// convertBook は、MADB固有のsourceBookを共通のBookへ変換する
func convertBook(source sourceBook) Book {
	authors := source.AgentNames
	if len(authors) == 0 {
		authors = make([]string, 0, len(source.Creators))
		for _, creator := range source.Creators {
			if author := removeCreatorRoles(creator); author != "" {
				authors = append(authors, author)
			}
		}
		authors = uniqueSorted(authors)
	}

	isbn10s := make([]string, 0)
	isbn13s := make([]string, 0)
	for _, value := range source.ISBNs {
		normalized := normalizeISBN(value)
		switch {
		case isValidISBN10(normalized):
			isbn10s = append(isbn10s, normalized)
		case isValidISBN13(normalized):
			isbn13s = append(isbn13s, normalized)
		}
	}

	return Book{
		ID:                source.ID,
		Titles:            nonNilStrings(source.Titles),
		Subtitles:         nonNilStrings(source.Subtitles),
		SeriesNames:       nonNilStrings(source.SeriesNames),
		SeriesID:          source.SeriesID,
		SeriesURL:         source.SeriesResourceURI,
		VolumeNumber:      source.VolumeNumber,
		EditionStatements: nonNilStrings(source.Versions),
		Authors:           nonNilStrings(uniqueSorted(authors)),
		Publishers:        nonNilStrings(source.Publishers),
		Imprints:          nonNilStrings(source.Brands),
		ISBN10s:           nonNilStrings(uniqueSorted(isbn10s)),
		ISBN13s:           nonNilStrings(uniqueSorted(isbn13s)),
		PublishedDate:     source.PublishedDate,
		Source:            SourceMADB,
		SourceURL:         source.ResourceURI,
	}
}

// isKnownVariable は、レスポンスで型を検証する既知変数か判定する
func isKnownVariable(name string) bool {
	switch name {
	case "id", "title", "subtitle", "seriesName", "seriesResource",
		"relatedSeriesName", "seriesID", "volumeNumber", "version",
		"creator", "agentName", "publisher", "brand", "isbn", "publishedDate":
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

// nonNilStrings は、nilスライスを空スライスへ変換する
func nonNilStrings(values []string) []string {
	if values == nil {
		return make([]string, 0)
	}
	return values
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

// removeCreatorRoles は、creator先頭の角括弧で囲まれた役割を取り除く
func removeCreatorRoles(value string) string {
	value = strings.TrimSpace(value)
	for strings.HasPrefix(value, "[") {
		end := strings.Index(value, "]")
		if end < 0 {
			break
		}
		value = strings.TrimSpace(value[end+1:])
	}
	return value
}
