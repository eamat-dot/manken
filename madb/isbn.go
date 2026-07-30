package madb

import "strings"

// normalizeISBN は、ISBNからASCIIのハイフンと空白を取り除く
func normalizeISBN(value string) string {
	value = strings.ReplaceAll(value, "-", "")
	return strings.ReplaceAll(value, " ", "")
}

// isValidISBN10 は、ISBN-10の文字種とチェックディジットを検証する
func isValidISBN10(value string) bool {
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

// isValidISBN13 は、ISBN-13の文字種とチェックディジットを検証する
func isValidISBN13(value string) bool {
	if len(value) != 13 {
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
	checkDigit := (10 - sum%10) % 10
	return checkDigit == int(value[12]-'0')
}
