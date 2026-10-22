package iam

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/iam/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// ListSettings returns the signed-in user's settings for a list; none saved reads as empty.
func (s *Service) ListSettings(ctx context.Context, key string) (ListSettings, error) {
	id, _ := platform.ActorFrom(ctx)
	raw, err := store.New(platform.DBFrom(ctx)).ListSettings(ctx, store.ListSettingsParams{UserID: id, ListKey: key})
	out := ListSettings{Views: []ListView{}, Columns: []ListColumn{}}
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(raw, &out)
}

// SetListSettings replaces the signed-in user's settings for a list.
func (s *Service) SetListSettings(ctx context.Context, key string, in ListSettings) error {
	seen := map[string]bool{}
	for i, v := range in.Views {
		name := strings.TrimSpace(v.Name)
		if seen[strings.ToLower(name)] {
			return ErrListViewNameTaken
		}
		seen[strings.ToLower(name)] = true
		in.Views[i].Name = name
	}
	if in.Views == nil {
		in.Views = []ListView{}
	}
	if in.Columns == nil {
		in.Columns = []ListColumn{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	id, _ := platform.ActorFrom(ctx)
	return platform.InTx(ctx, func(ctx context.Context) error {
		return store.New(platform.DBFrom(ctx)).SetListSettings(ctx, store.SetListSettingsParams{UserID: id, ListKey: key, Settings: raw})
	})
}
