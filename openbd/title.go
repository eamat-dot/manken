package openbd

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var titleVolumePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(.*?)[.．]\s*([0-9０-９]{1,3})$`),
	regexp.MustCompile(`^(.*?)\s+([0-9０-９]{1,3})$`),
	regexp.MustCompile(`^(.*?)\s*第\s*([0-9０-９]{1,3})\s*[巻卷]$`),
	regexp.MustCompile(`^(.*?)\s*([0-9０-９]{1,3})\s*[巻卷]$`),
	regexp.MustCompile(`^(.*?)\s*[#＃]\s*([0-9０-９]{1,3})$`),
	regexp.MustCompile(`^(.*?)\s*[\(（]\s*([0-9０-９]{1,3})\s*[\)）]$`),
}

// parseSourceTitle は、明示形式のopenBDタイトルだけを主題、並列題、巻数へ分ける
func parseSourceTitle(value string, hasSeries bool) (string, []string, Volume) {
	if strings.Count(value, " = ") == 1 {
		parts := strings.SplitN(value, " = ", 2)
		if parts[0] != "" && parts[1] != "" && containsLatinLetter(parts[1]) {
			parallelTitle, volume, ok := splitTrailingVolume(parts[1])
			if !ok {
				parallelTitle = parts[1]
			}
			return parts[0], []string{parallelTitle}, volume
		}
	}

	if hasSeries {
		if title, volume, ok := splitTrailingVolume(value); ok {
			return title, nil, volume
		}
	}
	return value, nil, Volume{}
}

// normalizeTitleKana は、タイトルと同じ巻数を安全に分離できる読みだけを返す
func normalizeTitleKana(value, sourceTitle, normalizedTitle string, volume Volume) string {
	if sourceTitle == normalizedTitle {
		return value
	}
	if value == "" || volume.Number == nil {
		return ""
	}
	titleKana, kanaVolume, ok := splitTrailingVolume(value)
	if !ok || kanaVolume.Number == nil || *kanaVolume.Number != *volume.Number {
		return ""
	}
	return titleKana
}

// splitTrailingVolume は、許可した末尾巻表示をタイトルから分ける
func splitTrailingVolume(value string) (string, Volume, bool) {
	for _, pattern := range titleVolumePatterns {
		matches := pattern.FindStringSubmatch(value)
		if matches == nil {
			continue
		}
		title := strings.TrimSpace(matches[1])
		digits, ok := normalizeDigits(matches[2])
		if title == "" || !ok {
			return "", Volume{}, false
		}
		number, err := strconv.Atoi(digits)
		if err != nil {
			return "", Volume{}, false
		}
		return title, Volume{Number: &number, Label: strconv.Itoa(number)}, true
	}
	return "", Volume{}, false
}

// containsLatinLetter は、文字列にASCIIの英字が含まれるか判定する
func containsLatinLetter(value string) bool {
	for _, character := range value {
		if character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' {
			return true
		}
	}
	return false
}

// normalizeDigits は、ASCII数字と全角数字をASCII数字列へ変換する
func normalizeDigits(value string) (string, bool) {
	var builder strings.Builder
	for _, character := range value {
		switch {
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
		case unicode.In(character, unicode.Digit) && character >= '０' && character <= '９':
			builder.WriteRune('0' + character - '０')
		default:
			return "", false
		}
	}
	return builder.String(), builder.Len() > 0
}
