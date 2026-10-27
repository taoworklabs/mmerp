package iam

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/taoworklabs/mmerp/internal/core/iam/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// LoginPolicy holds the password rules and the sign-in lockout of the tenant.
type LoginPolicy struct {
	MinLength   int  `json:"min_length" minimum:"8" maximum:"64"`
	Upper       bool `json:"upper"`
	Lower       bool `json:"lower"`
	Digit       bool `json:"digit"`
	Symbol      bool `json:"symbol"`
	MaxFailures int  `json:"max_failures" minimum:"3" maximum:"20"`
	LockMinutes int  `json:"lock_minutes" minimum:"1" maximum:"1440"`
}

// LoginPolicy returns the current policy.
func (s *Service) LoginPolicy(ctx context.Context) (LoginPolicy, error) {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return LoginPolicy{}, err
	}
	return s.policy(ctx)
}

// policy reads the policy without a permission check, for sign-in and password changes.
func (s *Service) policy(ctx context.Context) (LoginPolicy, error) {
	r, err := store.GetLoginPolicy.Run(ctx)
	if err != nil {
		return LoginPolicy{}, err
	}
	return LoginPolicy{
		MinLength: int(r.MinLength), Upper: r.Upper, Lower: r.Lower, Digit: r.Digit, Symbol: r.Symbol,
		MaxFailures: int(r.MaxFailures), LockMinutes: int(r.LockMinutes),
	}, nil
}

// SetLoginPolicy replaces the policy.
func (s *Service) SetLoginPolicy(ctx context.Context, p LoginPolicy) error {
	if err := s.require(ctx, PermManageUsers); err != nil {
		return err
	}
	var bad []string
	for _, c := range []struct {
		field     string
		v, lo, hi int
	}{{"min_length", p.MinLength, 8, 64}, {"max_failures", p.MaxFailures, 3, 20}, {"lock_minutes", p.LockMinutes, 1, 1440}} {
		if c.v < c.lo || c.v > c.hi {
			bad = append(bad, c.field)
		}
	}
	if bad != nil {
		return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": bad}}
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := store.SetLoginPolicy.Run(ctx, store.SetLoginPolicyParams{
			MinLength: int32(p.MinLength), Upper: p.Upper, Lower: p.Lower, Digit: p.Digit, Symbol: p.Symbol,
			MaxFailures: int32(p.MaxFailures), LockMinutes: int32(p.LockMinutes),
		}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "iam.login_policy_set", map[string]any{
			"min_length": p.MinLength, "upper": p.Upper, "lower": p.Lower, "digit": p.Digit, "symbol": p.Symbol,
			"max_failures": p.MaxFailures, "lock_minutes": p.LockMinutes,
		})
	})
}

// checkPassword fails with password_weak when password breaks the policy.
func (s *Service) checkPassword(ctx context.Context, login, name, password string) error {
	p, err := s.policy(ctx)
	if err != nil {
		return err
	}
	if problems := passwordProblems(p, login, name, password); len(problems) > 0 {
		return errPasswordWeak(p, problems)
	}
	return nil
}

func errPasswordWeak(p LoginPolicy, problems []string) *platform.Error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "password_weak", Params: map[string]any{"rules": problems, "min": p.MinLength}}
}

// minPart is the shortest login or name part checked against the password.
const minPart = 3

// passwordProblems lists the rules password breaks, in a fixed order.
func passwordProblems(p LoginPolicy, login, name, password string) []string {
	problems := []string{}
	if utf8.RuneCountInString(password) < p.MinLength {
		problems = append(problems, "min_length")
	}
	has := func(f func(rune) bool) bool { return strings.IndexFunc(password, f) >= 0 }
	if p.Upper && !has(unicode.IsUpper) {
		problems = append(problems, "upper")
	}
	if p.Lower && !has(unicode.IsLower) {
		problems = append(problems, "lower")
	}
	if p.Digit && !has(unicode.IsDigit) {
		problems = append(problems, "digit")
	}
	if p.Symbol && !has(func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) }) {
		problems = append(problems, "symbol")
	}
	lower := strings.ToLower(password)
	long := func(s string) bool { return utf8.RuneCountInString(s) >= minPart }
	if l := strings.ToLower(login); long(l) && strings.Contains(lower, l) {
		problems = append(problems, "contains_login")
	}
	if slices.ContainsFunc(strings.Fields(strings.ToLower(name)), func(part string) bool { return long(part) && strings.Contains(lower, part) }) {
		problems = append(problems, "contains_name")
	}
	return problems
}
