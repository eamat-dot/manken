package authorrole

import "testing"

// TestContains は、Authorsへ含める共通役割だけを判定することを検証する
func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{name: "author", roles: []string{"著者"}, want: true},
		{name: "original work", roles: []string{"原作"}, want: true},
		{name: "screenplay", roles: []string{"脚本"}, want: true},
		{name: "art", roles: []string{"作画"}, want: true},
		{name: "character concept", roles: []string{"キャラクター原案"}, want: true},
		{name: "character design", roles: []string{"キャラクターデザイン"}, want: true},
		{name: "editor", roles: []string{"編集"}},
		{name: "translator", roles: []string{"翻訳"}},
		{name: "supervisor", roles: []string{"監修"}},
		{name: "empty", roles: []string{""}},
		{name: "no roles"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := Contains(test.roles); got != test.want {
				t.Errorf("Contains(%q) = %t, want %t", test.roles, got, test.want)
			}
		})
	}
}
