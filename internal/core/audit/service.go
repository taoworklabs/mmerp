// Package audit owns the audit log. Entries are written in the caller's
// transaction, so they commit or roll back with the change they describe.
package audit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{}

func NewService() *Service { return &Service{} }

// Ref names the record an entry is about.
type Ref struct {
	Type string // record type, e.g. hrm.employee
	ID   int64
}

// Change is one field of a record before and after a write.
type Change struct {
	Field    string
	Old, New any
	// Sensitive values are stored encrypted.
	Sensitive bool
}

// Record logs action (prefixed by its module, e.g. "iam.login") by the actor in ctx.
// data must not hold secrets.
func (s *Service) Record(ctx context.Context, action string, data map[string]any) error {
	return s.insert(ctx, action, nil, data)
}

// RecordFor logs action about a record. data must not hold sensitive values.
func (s *Service) RecordFor(ctx context.Context, action string, ref Ref, data map[string]any) error {
	return s.insert(ctx, action, &ref, data)
}

// RecordChanges logs the fields that differ, as {"changes": {field: {"old", "new"}}}.
// A sensitive field's values are JSON-encoded, encrypted and base64-encoded.
// Nothing is logged when no field changed.
func (s *Service) RecordChanges(ctx context.Context, action string, ref Ref, changes []Change) error {
	diff := map[string]any{}
	for _, c := range changes {
		if reflect.DeepEqual(c.Old, c.New) {
			continue
		}
		if !c.Sensitive {
			diff[c.Field] = map[string]any{"old": c.Old, "new": c.New}
			continue
		}
		aad := "audit." + ref.Type + "." + c.Field
		before, err := seal(ctx, aad, c.Old)
		if err != nil {
			return err
		}
		after, err := seal(ctx, aad, c.New)
		if err != nil {
			return err
		}
		diff[c.Field] = map[string]any{"old": before, "new": after, "sensitive": true}
	}
	if len(diff) == 0 {
		return nil
	}
	return s.insert(ctx, action, &ref, map[string]any{"changes": diff})
}

func seal(ctx context.Context, aad string, v any) (*string, error) {
	if v == nil || reflect.ValueOf(v).Kind() == reflect.Pointer && reflect.ValueOf(v).IsNil() {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	enc := base64.StdEncoding.EncodeToString(platform.Encrypt(ctx, aad, raw))
	return &enc, nil
}

func (s *Service) insert(ctx context.Context, action string, ref *Ref, data map[string]any) error {
	p := store.InsertParams{Action: action}
	p.ActorID.Int64, p.ActorID.Valid = platform.ActorFrom(ctx)
	if ref != nil {
		p.DocType = pgtype.Text{String: ref.Type, Valid: true}
		p.DocID = pgtype.Int8{Int64: ref.ID, Valid: true}
	}
	if data != nil {
		var err error
		if p.Data, err = json.Marshal(data); err != nil {
			return err
		}
	}
	return store.New(platform.DBFrom(ctx)).Insert(ctx, p)
}
