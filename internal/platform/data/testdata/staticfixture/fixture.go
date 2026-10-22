//go:build staticcheck_fixture

package staticfixture

import "github.com/taoworklabs/mmerp/internal/platform/data"

func Fragment() data.Frag {
	s := "x"
	return data.SQL(s)
}
