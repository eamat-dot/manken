// Package authorrole は、Authorsへ含める共通役割を判定する
package authorrole

var values = map[string]struct{}{
	"著者":         {},
	"原作":         {},
	"脚本":         {},
	"作画":         {},
	"キャラクター原案":   {},
	"キャラクターデザイン": {},
}

// Contains は、役割にAuthorsへ含める共通役割が1つ以上あるか判定する
func Contains(roles []string) bool {
	for _, role := range roles {
		if _, ok := values[role]; ok {
			return true
		}
	}
	return false
}
