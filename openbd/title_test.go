package openbd

import "testing"

// TestParseSourceTitle は、許可したタイトル形式だけを巻数と並列タイトルへ分ける
func TestParseSourceTitle(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		hasSeries    bool
		title        string
		parallel     []string
		volumeNumber int
	}{
		{
			name:         "parallel title and volume",
			value:        "とんがり帽子のアトリエ = ATELIER OF WITCH HAT. 16",
			title:        "とんがり帽子のアトリエ",
			parallel:     []string{"ATELIER OF WITCH HAT"},
			volumeNumber: 16,
		},
		{name: "series title", value: "好きって言わせる方法 4", hasSeries: true, title: "好きって言わせる方法", volumeNumber: 4},
		{name: "full width brackets", value: "寄生獣（１０）", hasSeries: true, title: "寄生獣", volumeNumber: 10},
		{name: "no series", value: "1984 2", title: "1984 2"},
		{name: "ambiguous equals", value: "A = B = C 2", hasSeries: true, title: "A = B = C", volumeNumber: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			title, parallel, volume := parseSourceTitle(test.value, test.hasSeries)
			if title != test.title {
				t.Fatalf("title = %q, want %q", title, test.title)
			}
			assertStrings(t, parallel, test.parallel)
			if test.volumeNumber == 0 {
				if volume.Number != nil || volume.Label != "" {
					t.Fatalf("volume = %#v, want empty", volume)
				}
				return
			}
			if volume.Number == nil || *volume.Number != test.volumeNumber {
				t.Fatalf("volume = %#v, want %d", volume, test.volumeNumber)
			}
		})
	}
}

// TestNormalizeTitleKana は、タイトルと同じ巻数を持つ読みだけを分離する
func TestNormalizeTitleKana(t *testing.T) {
	number := 4
	volume := Volume{Number: &number, Label: "4"}
	if got := normalizeTitleKana("スキッテイワセルホウホウ 4", "好きって言わせる方法 4", "好きって言わせる方法", volume); got != "スキッテイワセルホウホウ" {
		t.Fatalf("normalizeTitleKana() = %q", got)
	}
	if got := normalizeTitleKana("スキッテイワセルホウホウ 5", "好きって言わせる方法 4", "好きって言わせる方法", volume); got != "" {
		t.Fatalf("normalizeTitleKana() = %q, want empty", got)
	}
}
