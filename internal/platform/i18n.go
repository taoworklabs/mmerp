package platform

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"strings"
)

// Locales the product ships; every key must exist in each.
var Locales = []string{"vi", "en"}

const defaultLocale = "vi"

// Written once at startup, read-only afterwards.
var catalog = map[string]map[string]string{}

// LoadTranslations merges a module's <locale>.json files (flat key → text) into the catalog.
func LoadTranslations(fsys fs.FS) error {
	for _, loc := range Locales {
		b, err := fs.ReadFile(fsys, loc+".json")
		if err != nil {
			return fmt.Errorf("i18n %s: %w", loc, err)
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("i18n %s: %w", loc, err)
		}
		if catalog[loc] == nil {
			catalog[loc] = map[string]string{}
		}
		maps.Copy(catalog[loc], m)
	}
	return nil
}

// Translate looks up key in locale (falling back to the default locale) and
// replaces {name} placeholders. A missing key returns the key itself.
func Translate(locale, key string, params map[string]any) string {
	s, ok := catalog[locale][key]
	if !ok {
		if s, ok = catalog[defaultLocale][key]; !ok {
			return key
		}
	}
	for k, v := range params {
		s = strings.ReplaceAll(s, "{"+k+"}", fmt.Sprint(v))
	}
	return s
}
