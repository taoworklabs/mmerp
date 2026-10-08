// Package printing prints records to PDF from the templates modules register. A print is
// a dataio export, so it runs as a job of the requester and its file is theirs alone.
package printing

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"sync"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/printing/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct {
	d Deps
	// templates by document type: one each.
	templates map[string]Template
	byCode    map[string]Template
}

// snapshotAAD binds the encrypted print data to its column.
const snapshotAAD = "printing.snapshots.data"

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
	s := &Service{d: d, templates: map[string]Template{}, byCode: map[string]Template{}}
	d.Record.OnPosted(s)
	// Who printed what tells nothing to someone who could not print it.
	d.Record.RestrictHistory("printing.printed", record.Print)
	return s
}

// Register adds a print template while wiring modules, before serving.
func (s *Service) Register(t Template) {
	if len(t.Layouts) == 0 || t.Data == nil {
		panic("printing: template " + t.Code + " needs Data and a layout")
	}
	if _, dup := s.templates[t.DocType]; dup {
		panic("printing: " + t.DocType + " has two templates")
	}
	s.templates[t.DocType], s.byCode[t.Code] = t, t
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

// render draws the asked parts of a record, a page or more each, and audits the print on
// the record. A posted document prints its snapshot as its first print pinned it; any
// other prints its current data in the actor's locale.
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
	me, err := s.d.IAM.Me(ctx)
	if err != nil {
		return dataio.File{}, err
	}
	parts, err := s.frozen(ctx, t, d)
	if err != nil {
		return dataio.File{}, err
	}
	layout, locale := len(t.Layouts), me.Locale
	blocks, err := s.current(ctx, t, locale)
	if err != nil {
		return dataio.File{}, err
	}
	if parts != nil {
		if layout, locale, blocks, err = s.pin(ctx, ref, layout, locale, blocks); err != nil {
			return dataio.File{}, err
		}
		if layout > len(t.Layouts) {
			return dataio.File{}, fmt.Errorf("printing: %s pinned to layout %d, which is gone", t.Code, layout)
		}
	} else if parts, err = t.Data(ctx, in.ID); err != nil {
		return dataio.File{}, err
	}
	if parts, err = pick(parts, in.Parts); err != nil {
		return dataio.File{}, err
	}
	p, err := newPage(locale, watermark(locale, d.Status), blocks)
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

// Posted freezes the print data of a document with a template, in its posting.
func (s *Service) Posted(ctx context.Context, d record.Doc) error {
	t, ok := s.templates[d.Type]
	if !ok {
		return nil
	}
	_, err := s.freeze(ctx, t, d.Ref)
	return err
}

// frozen returns the snapshot of a posted or cancelled document, nil for one without. A
// posted one without (posted before printing existed) is frozen now, under its row lock.
func (s *Service) frozen(ctx context.Context, t Template, d record.Doc) ([]Part, error) {
	if d.Status != record.Posted && d.Status != record.Cancelled {
		return nil, nil
	}
	parts, err := s.snapshots(ctx, d.Ref)
	if err != nil || parts != nil || d.Status != record.Posted {
		return parts, err
	}
	err = platform.InTx(ctx, func(ctx context.Context) error {
		if _, err := s.d.Record.Lock(ctx, d.Ref); err != nil {
			return err
		}
		// Another print may have frozen it while this one waited for the lock.
		if parts, err = s.snapshots(ctx, d.Ref); err != nil || parts != nil {
			return err
		}
		parts, err = s.freeze(ctx, t, d.Ref)
		return err
	})
	return parts, err
}

// freeze writes the current print data of every part of a record as its snapshot.
func (s *Service) freeze(ctx context.Context, t Template, ref record.Ref) ([]Part, error) {
	parts, err := t.Data(ctx, ref.ID)
	if err != nil {
		return nil, err
	}
	in := store.CreateSnapshotsParams{DocType: ref.Type, DocID: ref.ID}
	for i, p := range parts {
		in.Parts = append(in.Parts, p.Key)
		in.Positions = append(in.Positions, int32(i))
		in.Data = append(in.Data, platform.Encrypt(ctx, snapshotAAD, p.Data))
	}
	return parts, store.New(platform.DBFrom(ctx)).CreateSnapshots(ctx, in)
}

// snapshots decrypts the snapshot of a record, in print order; nil when it has none.
func (s *Service) snapshots(ctx context.Context, ref record.Ref) ([]Part, error) {
	rows, err := store.New(platform.DBFrom(ctx)).Snapshots(ctx, store.SnapshotsParams{DocType: ref.Type, DocID: ref.ID})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	parts := make([]Part, len(rows))
	for i, r := range rows {
		data, err := platform.Decrypt(ctx, snapshotAAD, r.Data)
		if err != nil {
			return nil, err
		}
		parts[i] = Part{Key: r.Part, Data: data}
	}
	return parts, nil
}

// pin returns the layout, locale and text blocks a posted document's first print fixed,
// fixing them to these at the first print.
func (s *Service) pin(ctx context.Context, ref record.Ref, layout int, locale string, blocks map[string]string) (int, string, map[string]string, error) {
	raw, err := json.Marshal(blocks)
	if err != nil {
		return 0, "", nil, err
	}
	var pin store.GetPinRow
	err = platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		err := q.CreatePin(ctx, store.CreatePinParams{DocType: ref.Type, DocID: ref.ID, Layout: int32(layout), Locale: locale, Blocks: raw})
		if err != nil {
			return err
		}
		pin, err = q.GetPin(ctx, store.GetPinParams{DocType: ref.Type, DocID: ref.ID})
		return err
	})
	if err != nil {
		return 0, "", nil, err
	}
	blocks = map[string]string{}
	return int(pin.Layout), pin.Locale, blocks, json.Unmarshal(pin.Blocks, &blocks)
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
