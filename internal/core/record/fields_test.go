package record_test

import (
	"context"
	"errors"
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

func errCode(err error) string {
	if e, ok := errors.AsType[*platform.Error](err); ok {
		return e.Code
	}
	return ""
}

func noOptions(context.Context, string) ([]record.FieldOption, error) { return nil, nil }

func noLabels(context.Context, []int64) ([]record.FieldOption, error) { return nil, nil }

func TestSetFieldValidated(t *testing.T) {
	f := newDeclared(t)
	f.rec.Register(record.Type{
		Code: "test.set", Product: "test", Kind: record.Document, NumberPrefix: "S",
		Perms: record.Perms{View: "test.view", Edit: "test.edit"}, OnDelete: noRows,
		Fields: []record.Field{{Key: "parts", Kind: record.Set, Search: noOptions, Labels: noLabels}},
	})
	create := func(v string) error {
		_, err := f.rec.Create(f.editor, "test.set", record.Header{Date: "2026-03-10", OrgUnitID: f.a, Fields: map[string]string{"parts": v}})
		return err
	}
	for _, v := range []string{"3,17", ""} {
		if err := create(v); err != nil {
			t.Errorf("%q: %v", v, err)
		}
	}
	for _, v := range []string{"17,3", "3,3", "0", "3,x", " 3", "3,", "03"} {
		if c := errCode(create(v)); c != "invalid_field" {
			t.Errorf("%q: %q", v, c)
		}
	}
	if got := record.SetValue([]int64{17, 3, 3}); got != "3,17" {
		t.Errorf("SetValue %q", got)
	}
	if got := record.SetValue(nil); got != "" {
		t.Errorf("SetValue(nil) %q", got)
	}
}

func TestSetFieldNeedsSearchAndLabels(t *testing.T) {
	f := newDeclared(t)
	for name, field := range map[string]record.Field{
		"set without search": {Key: "parts", Kind: record.Set, Labels: noLabels},
		"set without labels": {Key: "parts", Kind: record.Set, Search: noOptions},
		"number with search": {Key: "days", Kind: record.Number, Search: noOptions, Labels: noLabels},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: registered", name)
				}
			}()
			f.rec.Register(record.Type{Code: "test.bad", Product: "test", Kind: record.Document, NumberPrefix: "B",
				Perms: record.Perms{View: "test.view"}, OnDelete: noRows, Fields: []record.Field{field}})
		}()
	}
}
