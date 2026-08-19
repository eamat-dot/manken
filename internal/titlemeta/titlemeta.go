// Package titlemeta は、取得元タイトルを変更せずに安全な付加情報候補を抽出する
package titlemeta

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/eamat-dot/manken/model"
)

var (
	suppressedVolumePattern    = regexp.MustCompile(`(?:分冊版|単話|話売り|連載版|合本版|期間限定無料版|試読版|無料お試し|全\s*\d+\s*巻|\d+\s*[-~～〜]\s*\d+\s*巻|\d+\s*巻\s*[-~～〜]\s*\d+\s*巻|(?i:vol\.\d+))`)
	parenthesizedVolumePattern = regexp.MustCompile(`[（(](\d+)[）)]`)
	explicitVolumePattern      = regexp.MustCompile(`(?:第)?(\d+)巻`)
	hashVolumePattern          = regexp.MustCompile(`#(\d+)(?:\s|$|完|[（(<＜])`)
	trailingVolumePattern      = regexp.MustCompile(`(?:\s+|[.:：]\s*)(\d+)\s*$`)
	upperLowerVolumePattern    = regexp.MustCompile(`[（(]?(上|下)巻[）)]?\s*$`)
	finalVolumePattern         = regexp.MustCompile(`^\s*(?:完|[（(]\s*完\s*[）)]|＜\s*完\s*＞|<\s*完\s*>)\s*$`)
)

var editionWords = []string{"完全版", "新装版", "特装版", "愛蔵版", "文庫版", "通常版"}

// Metadata は、タイトルから安全に抽出できた付加情報候補を表す
type Metadata struct {
	Volume        model.Volume
	Editions      []string
	IsFinalVolume bool
}

// Parse は、タイトルを変更せずに安全な巻数、版表示、完結巻候補を抽出する
func Parse(title string) Metadata {
	result := Metadata{Editions: editions(title)}
	normalized := normalizeDigits(title)
	if suppressedVolumePattern.MatchString(normalized) {
		return result
	}

	volume, end, ok := volume(normalized)
	if !ok {
		return result
	}
	result.Volume = volume
	result.IsFinalVolume = finalVolumePattern.MatchString(normalized[end:])
	return result
}

// Apply は、タイトルから抽出した付加情報で未設定のBook項目だけを補う
func Apply(book *model.Book) {
	metadata := Parse(book.Title)
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

// editions は、タイトルに明確に含まれる既知の版表示を出現順で重複なく返す
func editions(title string) []string {
	type editionMatch struct {
		word  string
		index int
	}
	matches := make([]editionMatch, 0, len(editionWords))
	for _, edition := range editionWords {
		if index := strings.Index(title, edition); index >= 0 {
			matches = append(matches, editionMatch{word: edition, index: index})
		}
	}
	sort.Slice(matches, func(left, right int) bool {
		return matches[left].index < matches[right].index
	})
	result := make([]string, len(matches))
	for index, match := range matches {
		result[index] = match.word
	}
	return result
}

// volume は、許可した巻表示を優先順に解析し、巻表示の末尾位置も返す
func volume(title string) (model.Volume, int, bool) {
	if match := explicitVolumePattern.FindStringSubmatchIndex(title); match != nil {
		return parsedNumericVolume(title[match[2]:match[3]], match[1])
	}
	if match := hashVolumePattern.FindStringSubmatchIndex(title); match != nil {
		return parsedNumericVolume(title[match[2]:match[3]], match[3])
	}
	if volume, end, ok := parenthesizedVolume(title); ok {
		return volume, end, true
	}
	if match := upperLowerVolumePattern.FindStringSubmatchIndex(title); match != nil {
		return model.Volume{Label: title[match[2]:match[3]]}, match[1], true
	}
	if match := trailingVolumePattern.FindStringSubmatchIndex(title); match != nil {
		return parsedNumericVolume(title[match[2]:match[3]], match[1])
	}
	return model.Volume{}, 0, false
}

// parenthesizedVolume は、後続が安全な末尾表示だけの括弧数値を巻表示として返す
func parenthesizedVolume(title string) (model.Volume, int, bool) {
	for _, match := range parenthesizedVolumePattern.FindAllStringSubmatchIndex(title, -1) {
		if suffix := strings.TrimSpace(title[match[1]:]); suffix == "" || finalVolumePattern.MatchString(suffix) || isEditionSuffix(suffix) {
			return parsedNumericVolume(title[match[2]:match[3]], match[1])
		}
	}
	return model.Volume{}, 0, false
}

// isEditionSuffix は、既知の版表示だけで構成される末尾かを返す
func isEditionSuffix(value string) bool {
	for _, edition := range editionWords {
		value = strings.ReplaceAll(value, edition, "")
	}
	return strings.TrimSpace(value) == ""
}

// parsedNumericVolume は、ASCII数字の巻表示を変換し、成功時だけ末尾位置とともに返す
func parsedNumericVolume(value string, end int) (model.Volume, int, bool) {
	volume, ok := numericVolume(value)
	return volume, end, ok
}

// numericVolume は、ASCII数字の巻表示を共通の数値巻へ変換する
func numericVolume(value string) (model.Volume, bool) {
	number, err := strconv.Atoi(value)
	if err != nil {
		return model.Volume{}, false
	}
	return model.Volume{Number: &number, Label: strconv.Itoa(number)}, true
}

// normalizeDigits は、巻表示の候補を判定するため全角数字をASCII数字へ変換する
func normalizeDigits(value string) string {
	return strings.Map(func(character rune) rune {
		if character >= '０' && character <= '９' {
			return character - '０' + '0'
		}
		return character
	}, value)
}
