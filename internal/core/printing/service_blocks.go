package printing

import (
	"context"
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/printing/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// texts are a template's saved blocks: block → locale → text.
type texts map[string]map[string]string

// placeholder finds the {name} placeholders of a text.
var placeholder = regexp.MustCompile(`\{([^{}]*)\}`)

// latest reads the last save of every template's blocks, by template.
func latest(ctx context.Context) (map[string]store.LatestBlocksRow, error) {
	rows, err := store.New(platform.DBFrom(ctx)).LatestBlocks(ctx)
	out := map[string]store.LatestBlocksRow{}
	for _, r := range rows {
		out[r.Template] = r
	}
	return out, err
}

// resolve gives each block of t in locale: its last saved text, else its default.
func resolve(t Template, saved store.LatestBlocksRow, locale string) (map[string]string, error) {
	var tx texts
	if saved.Texts != nil {
		if err := json.Unmarshal(saved.Texts, &tx); err != nil {
			return nil, err
		}
	}
	out := map[string]string{}
	for _, b := range t.Blocks {
		if v, ok := tx[b.Key][locale]; ok {
			out[b.Key] = v
		} else {
			out[b.Key] = platform.Translate(locale, b.Default, nil)
		}
	}
	return out, nil
}

// current gives the blocks of t in locale as a first print draws them now.
func (s *Service) current(ctx context.Context, t Template, locale string) (map[string]string, error) {
	if len(t.Blocks) == 0 {
		return map[string]string{}, nil
	}
	saved, err := latest(ctx)
	if err != nil {
		return nil, err
	}
	return resolve(t, saved[t.Code], locale)
}

func listed(t Template, saved store.LatestBlocksRow, locale string) PrintTemplate {
	out := PrintTemplate{Code: t.Code, Name: platform.Translate(locale, t.Name, nil), Product: t.product, Version: int(saved.Version)}
	if saved.Version > 0 {
		at := saved.SavedAt.Time.UTC().Format(time.RFC3339)
		out.SavedAt, out.SavedByName = &at, &saved.SavedByName
	}
	return out
}

// Templates lists the print templates, by code, for administrators.
func (s *Service) Templates(ctx context.Context) ([]PrintTemplate, error) {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManagePrint); err != nil {
		return nil, err
	}
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := latest(ctx)
	if err != nil {
		return nil, err
	}
	out := []PrintTemplate{}
	for _, code := range slices.Sorted(maps.Keys(s.byCode)) {
		out = append(out, listed(s.byCode[code], saved[code], me.Locale))
	}
	return out, nil
}

// TemplateBlocks returns a template with the current text of its blocks in both languages.
func (s *Service) TemplateBlocks(ctx context.Context, code string) (PrintTemplateBlocks, error) {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManagePrint); err != nil {
		return PrintTemplateBlocks{}, err
	}
	t, ok := s.byCode[code]
	if !ok {
		return PrintTemplateBlocks{}, platform.ErrNotFound
	}
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return PrintTemplateBlocks{}, err
	}
	saved, err := latest(ctx)
	if err != nil {
		return PrintTemplateBlocks{}, err
	}
	vi, err := resolve(t, saved[code], "vi")
	if err != nil {
		return PrintTemplateBlocks{}, err
	}
	en, err := resolve(t, saved[code], "en")
	out := PrintTemplateBlocks{PrintTemplate: listed(t, saved[code], me.Locale), Blocks: []PrintBlock{}}
	for _, b := range t.Blocks {
		out.Blocks = append(out.Blocks, PrintBlock{Key: b.Key, Label: platform.Translate(me.Locale, b.Label, nil),
			Placeholders: append([]string{}, b.Placeholders...), Vi: vi[b.Key], En: en[b.Key]})
	}
	return out, err
}

// SaveBlocks saves every block of a template as its next version. Documents already
// printed keep the texts they were first printed with.
func (s *Service) SaveBlocks(ctx context.Context, code string, in []BlockText) error {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManagePrint); err != nil {
		return err
	}
	t, ok := s.byCode[code]
	if !ok {
		return platform.ErrNotFound
	}
	if err := platform.ProductGate(ctx, t.product, platform.ClassWrite); err != nil {
		return err
	}
	if len(in) != len(t.Blocks) {
		return errBlocks
	}
	tx := texts{}
	for _, b := range in {
		i := slices.IndexFunc(t.Blocks, func(d Block) bool { return d.Key == b.Key })
		if i < 0 || tx[b.Key] != nil {
			return errBlocks
		}
		for _, text := range []string{b.Vi, b.En} {
			if utf8.RuneCountInString(text) > maxBlockText {
				return errBlocks
			}
			for _, m := range placeholder.FindAllStringSubmatch(text, -1) {
				if !slices.Contains(t.Blocks[i].Placeholders, m[1]) {
					return errPlaceholder(b.Key, m[1])
				}
			}
		}
		tx[b.Key] = map[string]string{"vi": b.Vi, "en": b.En}
	}
	raw, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	actor, _ := platform.ActorFrom(ctx)
	return platform.InTx(ctx, func(ctx context.Context) error {
		v, err := store.New(platform.DBFrom(ctx)).SaveBlocks(ctx, store.SaveBlocksParams{Template: code, Texts: raw, SavedBy: actor})
		if err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "printing.blocks_saved", map[string]any{"template": code, "version": v})
	})
}
