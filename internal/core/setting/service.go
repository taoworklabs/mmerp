// Package setting owns tenant and legal-entity settings: keys prefixed by their module,
// string values, and a default in code for every key without a stored value.
package setting

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/setting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	Locale   = "setting.locale"
	Timezone = "setting.timezone"
	// Rounding is a legal entity's platform.Rounding.
	Rounding = "setting.rounding"
	// PermManage reads and writes legal-entity settings; a core admin permission.
	PermManage = "core.setting.manage"
)

var defaults = map[string]string{
	Locale:   "vi",
	Timezone: "Asia/Ho_Chi_Minh",
}

// key is a legal-entity setting: its product (empty for core), default and the values it may take.
type key struct {
	product string
	def     string
	values  []string
}

type Service struct {
	keys  map[string]key
	authz Authz
	audit *audit.Service
}

func NewService() *Service {
	s := &Service{keys: map[string]key{}, audit: audit.NewService()}
	s.RegisterLegalEntityKey("", Rounding, string(platform.RoundLine), []string{string(platform.RoundLine), string(platform.RoundTotal)})
	return s
}

// SetAuthz wires the permission check while composing the app.
func (s *Service) SetAuthz(a Authz) { s.authz = a }

// Get returns the stored value of key, or its default.
func (s *Service) Get(ctx context.Context, key string) (string, error) {
	def, ok := defaults[key]
	if !ok {
		// Unknown keys are wiring bugs; failing loudly beats returning "".
		return "", fmt.Errorf("setting: unknown key %q", key)
	}
	v, err := store.New(platform.DBFrom(ctx)).GetValue(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return def, nil
	}
	return v, err
}

// RegisterLegalEntityKey adds a legal-entity setting of product (empty for core) while
// wiring modules, before serving; it is written only while its product is enabled.
func (s *Service) RegisterLegalEntityKey(product, name, def string, values []string) {
	if _, dup := s.keys[name]; dup {
		panic("setting: key " + name + " registered twice")
	}
	s.keys[name] = key{product: product, def: def, values: values}
}

// GetFor returns a legal entity's value of key, or its default.
func (s *Service) GetFor(ctx context.Context, legalEntity int64, name string) (string, error) {
	k, ok := s.keys[name]
	if !ok {
		return "", fmt.Errorf("setting: unknown key %q", name)
	}
	v, err := store.New(platform.DBFrom(ctx)).GetLegalEntityValue(ctx, store.GetLegalEntityValueParams{LegalEntityID: legalEntity, Key: name})
	if errors.Is(err, pgx.ErrNoRows) {
		return k.def, nil
	}
	return v, err
}

// LegalEntitySettings lists every legal-entity setting of one legal entity, by key.
func (s *Service) LegalEntitySettings(ctx context.Context, legalEntity int64) ([]LegalEntitySetting, error) {
	if err := s.checkEntity(ctx, legalEntity); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).LegalEntityValues(ctx, legalEntity)
	if err != nil {
		return nil, err
	}
	stored := map[string]string{}
	for _, r := range rows {
		stored[r.Key] = r.Value
	}
	out := []LegalEntitySetting{}
	for _, name := range slices.Sorted(maps.Keys(s.keys)) {
		k := s.keys[name]
		v, ok := stored[name]
		if !ok {
			v = k.def
		}
		out = append(out, LegalEntitySetting{Key: name, Value: v, Values: k.values})
	}
	return out, nil
}

// SetFor stores a legal entity's value of key, audited.
func (s *Service) SetFor(ctx context.Context, legalEntity int64, name, value string) error {
	k, ok := s.keys[name]
	if !ok {
		return platform.ErrNotFound
	}
	if !slices.Contains(k.values, value) {
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_setting_value", Params: map[string]any{"key": name}}
	}
	if err := platform.ProductGate(ctx, k.product, platform.ClassWrite); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.checkEntity(ctx, legalEntity); err != nil {
			return err
		}
		old, err := s.GetFor(ctx, legalEntity, name)
		if err != nil {
			return err
		}
		if err := store.New(platform.DBFrom(ctx)).SetLegalEntityValue(ctx, store.SetLegalEntityValueParams{LegalEntityID: legalEntity, Key: name, Value: value}); err != nil {
			return err
		}
		return s.audit.Record(ctx, "setting.changed", map[string]any{"legal_entity_id": legalEntity, "key": name, "old": old, "new": value})
	})
}

func (s *Service) checkEntity(ctx context.Context, legalEntity int64) error {
	if err := s.authz.RequireCore(ctx, PermManage); err != nil {
		return err
	}
	ok, err := store.New(platform.DBFrom(ctx)).IsLegalEntity(ctx, legalEntity)
	if err == nil && !ok {
		return platform.ErrNotFound
	}
	return err
}
