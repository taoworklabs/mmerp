// Package printing prints records to PDF from the templates modules register. A print is
// a dataio export, so it runs as a job of the requester and its file is theirs alone.
package printing

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"slices"
	"sync"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{ d Deps }

//go:embed i18n/*.json
var i18nFiles embed.FS

// loadTranslations adds the watermarks to the catalog, once.
var loadTranslations = sync.OnceValue(func() error {
	sub, err := fs.Sub(i18nFiles, "i18n")
	if err != nil {
		return err
	}
	return platform.LoadTranslations(sub)
})

func NewService(d Deps) *Service {
	if err := loadTranslations(); err != nil {
		// The files are embedded: failing here is a build defect.
		panic(err)
	}
	// Who printed what tells nothing to someone who could not print it.
	d.Record.RestrictHistory("printing.printed", record.Print)
	return &Service{d: d}
}

// Register adds a print template while wiring modules, before serving.
func (s *Service) Register(t Template) {
	if len(t.Layouts) == 0 || t.Data == nil {
		panic("printing: template " + t.Code + " needs Data and a layout")
	}
	s.d.DataIO.RegisterExport(dataio.Export{
		Kind: "printing." + t.Code, Product: t.Product,
		Render: func(ctx context.Context, raw json.RawMessage) (dataio.File, error) { return s.render(ctx, t, raw) },
		Check: func(ctx context.Context, raw json.RawMessage) error {
			_, err := s.check(ctx, t, raw)
			return err
		},
	})
}

// check reads a print's params if the actor may print the record now; one they cannot
// see does not exist.
func (s *Service) check(ctx context.Context, t Template, raw json.RawMessage) (params, error) {
	var in params
	if err := json.Unmarshal(raw, &in); err != nil || in.ID == 0 {
		return in, dataio.ErrInvalidParams
	}
	ref := record.Ref{Type: t.DocType, ID: in.ID}
	if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
		return in, err
	}
	if ok, err := s.d.Record.Can(ctx, ref.Type, ref.ID, record.Print); err != nil || !ok {
		return in, platform.OrErr(err, platform.ErrForbidden)
	}
	return in, nil
}

// render draws the asked parts of a record in the actor's locale, a page or more each, and
// audits the print on the record.
func (s *Service) render(ctx context.Context, t Template, raw json.RawMessage) (dataio.File, error) {
	in, err := s.check(ctx, t, raw)
	if err != nil {
		return dataio.File{}, err
	}
	ref := record.Ref{Type: t.DocType, ID: in.ID}
	d, err := s.d.Record.Get(ctx, ref)
	if err != nil {
		return dataio.File{}, err
	}
	parts, err := t.Data(ctx, in.ID)
	if err != nil {
		return dataio.File{}, err
	}
	if parts, err = pick(parts, in.Parts); err != nil {
		return dataio.File{}, err
	}
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return dataio.File{}, err
	}
	layout := len(t.Layouts)
	p, err := newPage(me.Locale, watermark(me.Locale, d.Status))
	if err != nil {
		return dataio.File{}, err
	}
	for _, part := range parts {
		p.next()
		if err := t.Layouts[layout-1](p, part.Data); err != nil {
			return dataio.File{}, err
		}
	}
	b, err := p.bytes()
	if err != nil {
		return dataio.File{}, err
	}
	err = platform.InTx(ctx, func(ctx context.Context) error {
		return s.d.Audit.RecordFor(ctx, "printing.printed", audit.Ref(ref), printedData(t, layout, parts))
	})
	return dataio.File{Name: d.Number + ".pdf", Body: b}, err
}

// pick keeps the asked parts, in print order; none asked means all.
func pick(parts []Part, keys []int64) ([]Part, error) {
	if len(keys) == 0 {
		return parts, nil
	}
	out := make([]Part, 0, len(keys))
	for _, p := range parts {
		if slices.Contains(keys, p.Key) {
			out = append(out, p)
		}
	}
	if len(out) != len(slices.Compact(slices.Sorted(slices.Values(keys)))) {
		return nil, dataio.ErrInvalidParams
	}
	return out, nil
}

// printedData is what the audit keeps of a print: never amounts, only which parts.
func printedData(t Template, layout int, parts []Part) map[string]any {
	data := map[string]any{"template": t.Code, "layout": layout}
	var keys []int64
	for _, p := range parts {
		if p.Key != 0 {
			keys = append(keys, p.Key)
		}
	}
	if keys != nil {
		data["parts"] = keys
	}
	return data
}

// watermark marks a print of a document not in effect.
func watermark(locale string, s record.Status) string {
	switch s {
	case record.Draft, record.PendingApproval:
		return platform.Translate(locale, "printing.watermark.draft", nil)
	case record.Cancelled:
		return platform.Translate(locale, "printing.watermark.cancelled", nil)
	}
	return ""
}
