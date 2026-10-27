package iam

import (
	"slices"
	"testing"
)

func TestPasswordProblems(t *testing.T) {
	def := LoginPolicy{MinLength: 8}
	all := LoginPolicy{MinLength: 8, Upper: true, Lower: true, Digit: true, Symbol: true}
	for _, c := range []struct {
		name     string
		p        LoginPolicy
		login, n string
		pw       string
		want     []string
	}{
		{"short", def, "", "", "short", []string{"min_length"}},
		{"classes", all, "", "", "abcdefgh", []string{"upper", "digit", "symbol"}},
		{"unicode ok", all, "", "", "Đường-1234", []string{}},
		{"space is not a symbol", all, "", "", "Abc 12345", []string{"symbol"}},
		{"short login skipped", def, "an", "", "xxANxxxx", []string{}},
		{"login", def, "binh", "", "myBINH2026", []string{"contains_login"}},
		{"no diacritics folding", def, "", "Trần Văn Bình", "binhbinh", []string{}},
		{"name part", def, "", "Trần Văn Bình", "bìnhtran!", []string{"contains_name"}},
	} {
		if got := passwordProblems(c.p, c.login, c.n, c.pw); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
