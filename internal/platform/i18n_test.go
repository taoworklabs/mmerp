package platform_test

import (
	"testing"
	"testing/fstest"

	"github.com/taoworklabs/mmerp/internal/platform"
)

func TestTranslate(t *testing.T) {
	err := platform.LoadTranslations(fstest.MapFS{
		"vi.json": {Data: []byte(`{"demo.greet": "Chào {name}", "demo.vi_only": "chỉ tiếng Việt"}`)},
		"en.json": {Data: []byte(`{"demo.greet": "Hello {name}"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ locale, key, want string }{
		{"en", "demo.greet", "Hello An"},
		{"vi", "demo.greet", "Chào An"},
		{"en", "demo.vi_only", "chỉ tiếng Việt"},
		{"en", "demo.missing", "demo.missing"},
	} {
		if got := platform.Translate(c.locale, c.key, map[string]any{"name": "An"}); got != c.want {
			t.Errorf("Translate(%s, %s) = %q, want %q", c.locale, c.key, got, c.want)
		}
	}
}
