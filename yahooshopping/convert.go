package yahooshopping

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/eamat-dot/manken/internal/titlemeta"
)

var (
	brPattern       = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagPattern      = regexp.MustCompile(`<[^>]*>`)
	wholeSetPattern = regexp.MustCompile(`(?:全\d+巻|\d+\s*[-～]\s*\d+巻)セット`)
	janPattern      = regexp.MustCompile(`^[0-9]{8}(?:[0-9]{5})?$`)
)

// convertItem は、Tower商品の実測済み項目を共通書籍情報へ変換する
func convertItem(value item, observedAt string) Book {
	description := parseTowerDescription(value.Description)
	labeledTitle := strings.TrimSpace(description.labels["タイトル"])
	title := sourceTitle(labeledTitle, value.Name)
	contributors := towerContributors(description.labels["アーティスト"], description.labels["アーティストカナ"])
	book := Book{Title: title, Authors: contributorNamesOnly(contributors), Contributors: contributors, Medium: PublicationMediumPrint, CoverURL: coverURL(value), Sources: []BookSource{{Source: SourceYahooShopping, ID: value.Code, URL: sourceURL(value.URL)}}}
	if labeledTitle != "" {
		book.TitleReading = description.labels["タイトルカナ"]
	}
	if publisher := description.labels["レーベル"]; publisher != "" {
		book.Publishers = []string{publisher}
	}
	if release := description.labels["発売日"]; release != "" {
		book.ReleaseDate = release
	}
	applyTitleMetadata(&book)
	if isbn := canonicalItemISBN(strings.TrimSpace(value.JanCode)); isbn != "" {
		book.ISBN13 = []string{isbn}
	} else if jan := canonicalJAN(value.JanCode); jan != "" {
		book.JAN = []string{jan}
	}
	if value.Price != nil {
		var taxIncluded *bool
		if value.PriceLabel != nil {
			taxIncluded = value.PriceLabel.Taxable
		}
		book.CurrentPrice = &Price{Amount: *value.Price, Currency: "JPY", TaxIncluded: taxIncluded, Source: SourceYahooShopping, ObservedAt: observedAt}
	}
	return book
}

// applyTitleMetadata は、Tower固有のタイトル選択後に安全な付加情報だけを未設定のBook項目へ補う
func applyTitleMetadata(book *Book) {
	titlemeta.Apply(book)
}

// sourceTitle は、Towerの明示タイトルを優先して取得元の商品タイトルをそのまま返す
func sourceTitle(labeledTitle, fallback string) string {
	if labeledTitle != "" {
		return labeledTitle
	}
	return strings.TrimSpace(fallback)
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

// canonicalJAN は、有効なJANコードを空白を除いて返す
func canonicalJAN(value string) string {
	value = strings.TrimSpace(value)
	if !janPattern.MatchString(value) || !isValidJAN(value) {
		return ""
	}
	return value
}

// isValidJAN は、8桁または13桁のJANチェックディジットを検証する
func isValidJAN(value string) bool {
	sum := 0
	for index := len(value) - 2; index >= 0; index-- {
		digit := int(value[index] - '0')
		if (len(value)-2-index)%2 == 0 {
			sum += digit * 3
		} else {
			sum += digit
		}
	}
	return (10-sum%10)%10 == int(value[len(value)-1]-'0')
}

// coverURL は、600pxで要求するexImageを優先し、欠落時はmedium、smallの順に返す
func coverURL(value item) string {
	for _, candidate := range []string{value.ExImage.URL, value.Image.Medium, value.Image.Small} {
		if url := sourceURL(candidate); url != "" {
			return url
		}
	}
	return ""
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
