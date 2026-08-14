package yahooshopping

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/eamat-dot/manken/api"
)

var (
	brPattern                  = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagPattern                 = regexp.MustCompile(`<[^>]*>`)
	volumePattern              = regexp.MustCompile(`^(\d+|#\d+|上|下)巻?$`)
	parenthesizedVolumePattern = regexp.MustCompile(`\s*\((\d+)\)$`)
	trailingNumberPattern      = regexp.MustCompile(`\s+(\d+)\s*$`)
	wholeSetPattern            = regexp.MustCompile(`(?:全\d+巻|\d+\s*[-～]\s*\d+巻)セット`)
	janPattern                 = regexp.MustCompile(`^[0-9]{8}(?:[0-9]{5})?$`)
)

// convertItem は、Tower商品の実測済み項目を共通書籍情報へ変換する
func convertItem(value item, observedAt string) Book {
	description := parseTowerDescription(value.Description)
	labeledTitle := strings.TrimSpace(description.labels["タイトル"])
	title, editions, volume := parseTowerTitle(labeledTitle, value.Name)
	contributors := towerContributors(description.labels["アーティスト"], description.labels["アーティストカナ"])
	book := Book{Normalized: NormalizedBook{Title: title, Authors: contributorNamesOnly(contributors), Contributors: contributors, Medium: PublicationMediumPrint, Images: images(value)}, Sources: []BookSource{{Source: SourceYahooShopping, ID: value.Code, URL: sourceURL(value.URL)}}}
	if labeledTitle != "" && title == labeledTitle {
		book.Normalized.TitleReading = description.labels["タイトルカナ"]
	}
	book.Normalized.EditionStatements = editions
	if publisher := description.labels["レーベル"]; publisher != "" {
		book.Normalized.Publishers = []string{publisher}
	}
	if release := description.labels["発売日"]; release != "" {
		book.Normalized.Dates = []BookDate{{Type: BookDateTypeReleased, Value: release}}
	}
	if volume != nil {
		book.Normalized.Volume = *volume
	}
	if identifier, ok := itemIdentifier(value.JanCode); ok {
		book.Normalized.Identifiers = []Identifier{identifier}
	}
	if value.Price != nil {
		var taxIncluded *bool
		if value.PriceLabel != nil {
			taxIncluded = value.PriceLabel.Taxable
		}
		book.Normalized.Prices = []Price{{Type: PriceTypeCurrent, Amount: *value.Price, Currency: "JPY", TaxIncluded: taxIncluded, Source: SourceYahooShopping, ObservedAt: observedAt}}
	}
	return book
}

type parsedDescription struct {
	labels map[string]string
}

var towerLabelPattern = regexp.MustCompile(`(?:^|\s*/\s*)(発売日|商品ID|ジャンル|フォーマット|構成数|レーベル|アーティスト|アーティストカナ|タイトル|タイトルカナ)\s*[:：]\s*`)

// parseTowerDescription は、Towerの商品説明にある明示ラベルを抽出する
func parseTowerDescription(description string) parsedDescription {
	description = strings.TrimSpace(tagPattern.ReplaceAllString(brPattern.ReplaceAllString(description, " / "), ""))
	result := parsedDescription{labels: make(map[string]string)}
	matches := towerLabelPattern.FindAllStringSubmatchIndex(description, -1)
	for index, match := range matches {
		end := len(description)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		label := description[match[2]:match[3]]
		value := strings.TrimSpace(strings.Trim(description[match[1]:end], " /"))
		if value != "" {
			result.labels[label] = value
		}
	}
	return result
}

// parseTowerTitle は、明示タイトルから安全に巻数と版表示を分離する
func parseTowerTitle(labeledTitle, fallback string) (string, []string, *api.Volume) {
	title := strings.TrimSpace(labeledTitle)
	if title == "" {
		return strings.TrimSpace(strings.TrimPrefix(fallback, "〔予約〕")), nil, nil
	}
	volume, title := towerTrailingVolume(title)
	if volume == nil {
		return title, nil, nil
	}
	for _, edition := range []string{"完全版", "新装版", "特装版", "愛蔵版"} {
		if strings.HasPrefix(title, edition+" ") {
			trimmed := strings.TrimSpace(strings.TrimPrefix(title, edition))
			if trimmed != "" {
				return trimmed, []string{edition}, volume
			}
		}
	}
	return title, nil, volume
}

// towerTrailingVolume は、タイトル末尾の明確な巻表示を分離する
func towerTrailingVolume(title string) (*api.Volume, string) {
	if matches := parenthesizedVolumePattern.FindStringSubmatchIndex(title); len(matches) != 0 {
		withoutParentheses := strings.TrimSpace(title[:matches[0]])
		if volume, stripped := towerTrailingVolume(withoutParentheses); volume != nil {
			return volume, stripped
		}
		number, _ := strconv.Atoi(title[matches[2]:matches[3]])
		return &api.Volume{Number: &number, Label: strconv.Itoa(number)}, withoutParentheses
	}
	if fields := strings.Fields(title); len(fields) >= 2 {
		if volume, ok := parseVolume(fields[len(fields)-1]); ok {
			return &volume, strings.TrimSpace(strings.TrimSuffix(title, fields[len(fields)-1]))
		}
	}
	if matches := trailingNumberPattern.FindStringSubmatchIndex(title); len(matches) != 0 {
		number, _ := strconv.Atoi(title[matches[2]:matches[3]])
		return &api.Volume{Number: &number, Label: strconv.Itoa(number)}, strings.TrimSpace(title[:matches[0]])
	}
	return nil, title
}

// towerContributors は、Towerの寄与者表示を役割を推測せずに変換する
func towerContributors(names, readings string) []Contributor {
	nameParts := splitTowerPeople(names)
	if len(nameParts) == 0 {
		return nil
	}
	readingParts := splitTowerPeople(readings)
	if len(readingParts) != len(nameParts) {
		readingParts = nil
	}
	contributors := make([]Contributor, 0, len(nameParts))
	for index, name := range nameParts {
		contributor := Contributor{Name: name}
		if readingParts != nil {
			contributor.Reading = readingParts[index]
		}
		contributors = append(contributors, contributor)
	}
	return contributors
}

// splitTowerPeople は、Towerの読点区切り表示から安全に人物名を分離する
func splitTowerPeople(value string) []string {
	parts := strings.Split(value, "、")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" && name != "他" {
			result = append(result, name)
		}
	}
	return result
}

// contributorNamesOnly は、寄与者一覧から著者名表示を作成する
func contributorNamesOnly(contributors []Contributor) []string {
	if len(contributors) == 0 {
		return nil
	}
	authors := make([]string, 0, len(contributors))
	for _, contributor := range contributors {
		authors = append(authors, contributor.Name)
	}
	return authors
}

// parseVolume は、実測済みの巻表示を共通巻情報へ変換する
func parseVolume(value string) (api.Volume, bool) {
	value = strings.TrimSpace(value)
	if !volumePattern.MatchString(value) {
		return api.Volume{}, false
	}
	numberText := strings.TrimSuffix(strings.TrimPrefix(value, "#"), "巻")
	if number, err := strconv.Atoi(numberText); err == nil {
		return api.Volume{Number: &number, Label: strconv.Itoa(number)}, true
	}
	return api.Volume{Label: strings.TrimSuffix(value, "巻")}, true
}

// itemIdentifier は、JANまたはISBN-13を種類付き識別子へ変換する
func itemIdentifier(value string) (Identifier, bool) {
	value = strings.TrimSpace(value)
	if canonicalItemISBN(value) != "" {
		return Identifier{Type: IdentifierTypeISBN13, Value: value}, true
	}
	if janPattern.MatchString(value) {
		return Identifier{Type: IdentifierTypeJAN, Value: value}, true
	}
	return Identifier{}, false
}

// images は、small、medium、取得できたexImageを共通画像情報へ変換する
func images(value item) []Image {
	values := []struct {
		purpose, url  string
		width, height *int
	}{{"small", value.Image.Small, nil, nil}, {"medium", value.Image.Medium, nil, nil}, {"exImage", value.ExImage.URL, value.ExImage.Width, value.ExImage.Height}}
	result := make([]Image, 0, len(values))
	for _, image := range values {
		if sourceURL(image.url) != "" {
			result = append(result, Image{URL: image.url, Purpose: image.purpose, Width: image.width, Height: image.height})
		}
	}
	return result
}

// sourceURL は、HTTPまたはHTTPSの取得元URLだけを返す
func sourceURL(value string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return ""
}

// UnmarshalJSON は、Yahoo!ショッピングの商品JSONをitemへ復号する
func (value *item) UnmarshalJSON(data []byte) error {
	type rawItem item
	var decoded rawItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*value = item(decoded)
	return nil
}
