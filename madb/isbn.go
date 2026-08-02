package madb

import (
	"strings"
)

// normalizeISBN は、取得元ISBNからASCIIハイフンとASCII空白を取り除く
func normalizeISBN(value string) string {
	value = strings.ReplaceAll(value, "-", "")
	return strings.ReplaceAll(value, " ", "")
}
