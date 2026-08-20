package isbn

import "testing"

// TestNormalize は、ISBN表記の整形、検証、対応形式の生成を検証する
func TestNormalize(t *testing.T) {
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
			got, err := Normalize(test.value)
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("Normalize() = %#v, want %#v", got, test.want)
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Fatalf("Normalize() = %#v, want %#v", got, test.want)
				}
			}
		})
	}
}

// TestNormalizeRejectsInvalidValues は、不正なISBNを入力エラーにする
func TestNormalizeRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{
		"", "4088466361", "4088466362", "9784088466362",
		"4901234567894", "9784778031404 (set)", "123456789",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := Normalize(value); err == nil {
				t.Fatal("Normalize() error = nil")
			}
		})
	}
}

// TestCanonical13 は、ISBN入力を問い合わせ用のISBN-13へ正準化することを検証する
func TestCanonical13(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "ISBN-10", value: "4088466365", want: "9784088466361"},
		{name: "ISBN-13", value: "9784088466361", want: "9784088466361"},
		{name: "979 ISBN-13", value: "9791234567896", want: "9791234567896"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Canonical13(test.value)
			if err != nil {
				t.Fatalf("Canonical13() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Canonical13() = %q, want %q", got, test.want)
			}
		})
	}
	if _, err := Canonical13("9784088466362"); err == nil {
		t.Fatal("Canonical13() error = nil")
	}
}

// TestValidation は、ISBN-10とISBN-13のチェックディジットを検証する
func TestValidation(t *testing.T) {
	tests := []struct {
		value  string
		isbn10 bool
		isbn13 bool
	}{
		{value: "080442957X", isbn10: true},
		{value: "080442957x"},
		{value: "0804429570"},
		{value: "9780306406157", isbn13: true},
		{value: "9780306406150"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if got := IsValidISBN10(test.value); got != test.isbn10 {
				t.Fatalf("IsValidISBN10() = %t, want %t", got, test.isbn10)
			}
			if got := IsValidISBN13(test.value); got != test.isbn13 {
				t.Fatalf("IsValidISBN13() = %t, want %t", got, test.isbn13)
			}
		})
	}
}
