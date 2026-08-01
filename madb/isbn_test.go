package madb

import "testing"

// TestNormalizeSearchISBN は、ISBN表記の整形、検証、対応形式の生成を検証する
func TestNormalizeSearchISBN(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "ISBN-10", value: "4088466365", want: []string{"4088466365", "9784088466361"}},
		{name: "ISBN-13", value: "9784088466361", want: []string{"4088466365", "9784088466361"}},
		{name: "hyphens and Unicode spaces", value: " 978-4-08\u3000846636-1\n", want: []string{"4088466365", "9784088466361"}},
		{name: "lowercase x", value: "0-8044-2957-x", want: []string{"080442957X", "9780804429573"}},
		{name: "979 ISBN-13", value: "9791234567896", want: []string{"9791234567896"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeSearchISBN(test.value)
			if err != nil {
				t.Fatalf("normalizeSearchISBN() error = %v", err)
			}
			assertStrings(t, got, test.want)
		})
	}
}

// TestNormalizeSearchISBN_RejectsInvalidValues は、不正なISBNを入力エラーにする
func TestNormalizeSearchISBN_RejectsInvalidValues(t *testing.T) {
	for _, value := range []string{
		"",
		"4088466361",
		"4088466362",
		"9784088466362",
		"4901234567894",
		"9784778031404 (set)",
		"123456789",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := normalizeSearchISBN(value); err == nil {
				t.Fatal("normalizeSearchISBN() error = nil")
			}
		})
	}
}

// TestISBNConversions は、ISBN-10と978 ISBN-13を相互変換する
func TestISBNConversions(t *testing.T) {
	if got := convertISBN10To13("080442957X"); got != "9780804429573" {
		t.Fatalf("convertISBN10To13() = %q", got)
	}
	if got, ok := convertISBN13To10("9780804429573"); !ok || got != "080442957X" {
		t.Fatalf("convertISBN13To10() = %q, %t", got, ok)
	}
	if got, ok := convertISBN13To10("9791234567896"); ok || got != "" {
		t.Fatalf("convertISBN13To10(979...) = %q, %t", got, ok)
	}
}
