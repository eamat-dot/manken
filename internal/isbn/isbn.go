// Package isbn は、取得元に依存しないISBN入力の検証と対応形式の生成を提供する
package isbn

import (
	"errors"
	"sort"
	"strings"
	"unicode"
)

// Normalize は、入力ISBNを検証し同じ書籍を表すISBN-10とISBN-13の候補を返す
func Normalize(value string) ([]string, error) {
	normalized := normalizeInput(value)
	var candidates []string
	switch {
	case IsValidISBN10(normalized):
		candidates = []string{normalized, convertISBN10To13(normalized)}
	case IsValidISBN13(normalized):
		candidates = []string{normalized}
		if converted, ok := convertISBN13To10(normalized); ok {
			candidates = append(candidates, converted)
		}
	default:
		return nil, errors.New("isbn must be a valid ISBN-10 or ISBN-13")
	}

	sort.Strings(candidates)
	return candidates, nil
}

// Canonical13 は、ISBN-10またはISBN-13を問い合わせ用のISBN-13へ正準化する
func Canonical13(value string) (string, error) {
	candidates, err := Normalize(value)
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if IsValidISBN13(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("isbn cannot be represented as ISBN-13")
}

// normalizeInput は、ISBNからASCIIハイフンとUnicode空白を取り除き末尾のxを大文字にする
func normalizeInput(value string) string {
	value = strings.Map(func(character rune) rune {
		if character == '-' || unicode.IsSpace(character) {
			return -1
		}
		return character
	}, value)
	if strings.HasSuffix(value, "x") {
		value = value[:len(value)-1] + "X"
	}
	return value
}

// convertISBN10To13 は、妥当なISBN-10を978で始まるISBN-13へ変換する
func convertISBN10To13(value string) string {
	prefix := "978" + value[:9]
	return prefix + string(rune('0'+isbn13CheckDigit(prefix)))
}

// convertISBN13To10 は、978で始まる妥当なISBN-13をISBN-10へ変換する
func convertISBN13To10(value string) (string, bool) {
	// ISBN-10へ対応付けられるISBN-13は978 Booklandだけで、979にはISBN-10表現がない
	if !strings.HasPrefix(value, "978") {
		return "", false
	}
	body := value[3:12]
	sum := 0
	for index := 0; index < len(body); index++ {
		sum += (10 - index) * int(body[index]-'0')
	}
	checkDigit := (11 - sum%11) % 11
	if checkDigit == 10 {
		return body + "X", true
	}
	return body + string(rune('0'+checkDigit)), true
}

// isbn13CheckDigit は、12桁のISBN-13本体からチェックディジットを算出する
func isbn13CheckDigit(value string) int {
	sum := 0
	for index := 0; index < len(value); index++ {
		digit := int(value[index] - '0')
		if index%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}
	return (10 - sum%10) % 10
}

// IsValidISBN10 は、ISBN-10の文字種とチェックディジットを検証する
func IsValidISBN10(value string) bool {
	if len(value) != 10 {
		return false
	}

	sum := 0
	for index := 0; index < 10; index++ {
		digit := 0
		if index == 9 && value[index] == 'X' {
			digit = 10
		} else {
			if value[index] < '0' || value[index] > '9' {
				return false
			}
			digit = int(value[index] - '0')
		}
		sum += (10 - index) * digit
	}
	return sum%11 == 0
}

// IsValidISBN13 は、ISBN-13の文字種とチェックディジットを検証する
func IsValidISBN13(value string) bool {
	if len(value) != 13 || (!strings.HasPrefix(value, "978") && !strings.HasPrefix(value, "979")) {
		return false
	}

	sum := 0
	for index := 0; index < 12; index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
		digit := int(value[index] - '0')
		if index%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}
	if value[12] < '0' || value[12] > '9' {
		return false
	}
	return (10-sum%10)%10 == int(value[12]-'0')
}
