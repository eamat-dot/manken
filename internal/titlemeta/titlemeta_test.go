package titlemeta

import "testing"

// TestParse は、許可した巻表示、版表示、完結表示と誤抽出抑止を検証する
func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		title    string
		number   *int
		label    string
		editions []string
		isFinal  bool
	}{
		{name: "full width parentheses", title: "作品（１）", number: integer(1), label: "1"},
		{name: "ascii parentheses", title: "作品(12)", number: integer(12), label: "12"},
		{name: "volume suffix", title: "作品 3巻", number: integer(3), label: "3"},
		{name: "numbered volume", title: "作品 第4巻", number: integer(4), label: "4"},
		{name: "trailing number", title: "作品 5", number: integer(5), label: "5"},
		{name: "period trailing number", title: "作品. 6", number: integer(6), label: "6"},
		{name: "edition before volume", title: "作品 完全版 7", number: integer(7), label: "7", editions: []string{"完全版"}},
		{name: "edition prefix", title: "新装版 作品（8）", number: integer(8), label: "8", editions: []string{"新装版"}},
		{name: "edition after volume", title: "作品 9巻 特装版", number: integer(9), label: "9", editions: []string{"特装版"}},
		{name: "parenthesized volume before edition", title: "作品（1） 特装版", number: integer(1), label: "1", editions: []string{"特装版"}},
		{name: "final volume", title: "作品（10）完", number: integer(10), label: "10", isFinal: true},
		{name: "parenthesized final volume", title: "作品（10）（完）", number: integer(10), label: "10", isFinal: true},
		{name: "angle bracket final volume", title: "新装版 寄生獣(10)＜完＞", number: integer(10), label: "10", editions: []string{"新装版"}, isFinal: true},
		{name: "ascii angle bracket final volume", title: "寄生獣（完全版）（8）<完>", number: integer(8), label: "8", editions: []string{"完全版"}, isFinal: true},
		{name: "hash angle bracket final volume", title: "作品 #10＜完＞", number: integer(10), label: "10", isFinal: true},
		{name: "mixed angle brackets are not final", title: "作品(10)<完＞"},
		{name: "standalone final marker", title: "作品＜完＞"},
		{name: "explicit volume before edition details", title: "作品 6巻特装版 小冊子付き", number: integer(6), label: "6", editions: []string{"特装版"}},
		{name: "upper volume", title: "作品 上巻", label: "上"},
		{name: "lower volume", title: "作品 下巻", label: "下"},
		{name: "magazine volume", title: "月刊ビッグガンガン 2017 Vol.07"},
		{name: "number in title", title: "犬が幸せになれる10のお話"},
		{name: "large number in title", title: "レベル1000超えの転生者"},
		{name: "out of range volume", title: "作品 999999999999999999999999999999999999999999999999999999"},
		{name: "serial edition", title: "作品【分冊版】 71"},
		{name: "single chapter", title: "作品【単話】（単話）"},
		{name: "whole set", title: "作品 全24巻セット"},
		{name: "range set", title: "作品 1-24巻セット"},
		{name: "whole volumes", title: "作品 全24巻"},
		{name: "range volumes", title: "作品 1-24巻"},
		{name: "range volumes full width tilde", title: "作品 1～24巻"},
		{name: "range with volume suffix", title: "作品 1巻～24巻"},
		{name: "range with wave dash", title: "作品 1巻〜24巻"},
		{name: "omnibus", title: "作品 合本版 1巻"},
		{name: "free edition", title: "作品【期間限定無料版】 1"},
		{name: "deduplicates editions", title: "完全版 完全版 作品 1", number: integer(1), label: "1", editions: []string{"完全版"}},
		{name: "editions preserve title order", title: "特装版 新装版 作品 1", number: integer(1), label: "1", editions: []string{"特装版", "新装版"}},
		{name: "unsafe final word after parenthesized number", title: "作品（10）完結"},
		{name: "parenthesized number in title", title: "作品（1）周年記念"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(test.title)
			if !sameInteger(got.Volume.Number, test.number) || got.Volume.Label != test.label || !sameStrings(got.Editions, test.editions) || got.IsFinalVolume != test.isFinal {
				t.Fatalf("Parse(%q) = %#v", test.title, got)
			}
		})
	}
}

// integer は、テスト用の整数ポインターを返す
func integer(value int) *int {
	return &value
}

// sameInteger は、整数ポインターが同じ値またはともにnilかを返す
func sameInteger(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

// sameStrings は、文字列スライスが同じ順序と要素かを返す
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
