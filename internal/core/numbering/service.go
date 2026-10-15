// Package numbering hands out document numbers and catalogue codes.
package numbering

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// series is a registered code series: its product and default prefix.
type series struct {
	product string
	prefix  string
}

type Service struct {
	d      Deps
	series map[string]series
}

func NewService(d Deps) *Service { return &Service{d: d, series: map[string]series{}} }

// Next returns the next number of a document type for a legal entity and year.
// Call it inside the transaction that creates the document.
func (s *Service) Next(ctx context.Context, docType string, legalEntity int64, year int32) (int64, error) {
	return store.New(platform.DBFrom(ctx)).Next(ctx, store.NextParams{DocType: docType, LegalEntityID: legalEntity, Year: year})
}

// RegisterSeries adds a code series of product while wiring modules, before serving.
func (s *Service) RegisterSeries(code, product, prefix string) {
	if _, dup := s.series[code]; dup {
		panic("numbering: series " + code + " registered twice")
	}
	s.series[code] = series{product: product, prefix: prefix}
}

// Code returns typed, trimmed, or when it is empty the next code of a series, skipping codes
// taken reports as used. Call it inside the transaction that creates the entry.
func (s *Service) Code(ctx context.Context, code, typed string, taken func(context.Context, string) (bool, error)) (string, error) {
	if typed = strings.TrimSpace(typed); typed != "" {
		return typed, nil
	}
	sr, ok := s.series[code]
	if !ok {
		return "", fmt.Errorf("numbering: unknown series %q", code)
	}
	for {
		r, err := store.New(platform.DBFrom(ctx)).NextInSeries(ctx, store.NextInSeriesParams{Code: code, Prefix: sr.prefix})
		if err != nil {
			return "", err
		}
		c := format(r.Prefix, r.Last)
		used, err := taken(ctx, c)
		if err != nil || !used {
			return c, err
		}
	}
}

// Series lists every code series with its prefix and next code.
func (s *Service) Series(ctx context.Context) ([]CodeSeries, error) {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManageCodeSeries); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListSeries(ctx)
	if err != nil {
		return nil, err
	}
	stored := map[string]store.NumberingSeries{}
	for _, r := range rows {
		stored[r.Code] = r
	}
	out := []CodeSeries{}
	for _, code := range slices.Sorted(maps.Keys(s.series)) {
		sr := s.series[code]
		r, ok := stored[code]
		if !ok {
			r = store.NumberingSeries{Prefix: sr.prefix}
		}
		actions := []string{}
		if platform.ProductGate(ctx, sr.product, platform.ClassWrite) == nil {
			actions = append(actions, "edit")
		}
		out = append(out, CodeSeries{Code: code, Product: sr.product, Prefix: r.Prefix, Next: format(r.Prefix, r.Last+1), AllowedActions: actions})
	}
	return out, nil
}

// SetPrefix changes a series' prefix, audited; the count goes on.
func (s *Service) SetPrefix(ctx context.Context, code, prefix string) error {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManageCodeSeries); err != nil {
		return err
	}
	sr, ok := s.series[code]
	if !ok {
		return platform.ErrNotFound
	}
	if utf8.RuneCountInString(prefix) > 10 || strings.IndexFunc(prefix, unicode.IsSpace) >= 0 {
		return ErrInvalidPrefix
	}
	if err := platform.ProductGate(ctx, sr.product, platform.ClassWrite); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		old := sr.prefix
		r, err := q.LockSeries(ctx, code)
		if err == nil {
			old = r.Prefix
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := q.SetPrefix(ctx, store.SetPrefixParams{Code: code, Prefix: prefix}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "numbering.prefix_changed", map[string]any{"series": code, "old": old, "new": prefix})
	})
}

func format(prefix string, n int64) string { return fmt.Sprintf("%s%05d", prefix, n) }
