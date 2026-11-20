package accounting

import (
	"cmp"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/posting"
)

var ErrChartNotEmpty = &platform.Error{Status: http.StatusConflict, Code: "chart_not_empty"}

// TemplateInfo names a chart template.
type TemplateInfo struct {
	Code string `json:"code"`
	// Name by language: vi, en.
	Name map[string]string `json:"name"`
}

// chartTemplate is a template file: a chart, a rule per business line kind, the closing
// accounts and the statement forms of one Vietnamese accounting regime.
type chartTemplate struct {
	TemplateInfo
	Accounts []struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Parent    string `json:"parent"`
		ByPartner bool   `json:"by_partner"`
	} `json:"accounts"`
	Rules []struct {
		Kind    string `json:"kind"`
		Account string `json:"account"`
		Side    string `json:"side"`
	} `json:"rules"`
	Closing struct {
		Result   string `json:"result"`
		Retained string `json:"retained"`
	} `json:"closing"`
	Forms map[string][]StatementRow `json:"forms"`
}

//go:embed templates/*.json
var templateFiles embed.FS

// templates are loaded once; a broken file fails at start, not at apply.
var templates = func() []chartTemplate {
	entries, err := templateFiles.ReadDir("templates")
	if err != nil {
		panic(err)
	}
	var out []chartTemplate
	for _, e := range entries {
		b, err := templateFiles.ReadFile("templates/" + e.Name())
		if err != nil {
			panic(err)
		}
		var t chartTemplate
		if err := json.Unmarshal(b, &t); err != nil {
			panic(fmt.Sprintf("accounting: template %s: %v", e.Name(), err))
		}
		for form, rows := range t.Forms {
			if _, err := evaluateRows(rows, statementInput{}); !knownForm(form) || err != nil {
				panic(fmt.Sprintf("accounting: template %s, form %s: %v", e.Name(), form, err))
			}
		}
		out = append(out, t)
	}
	// Circular 133 before 99: the longer number first, then by code.
	slices.SortFunc(out, func(a, b chartTemplate) int {
		return cmp.Or(cmp.Compare(len(b.Code), len(a.Code)), strings.Compare(a.Code, b.Code))
	})
	return out
}()

// Templates lists the chart templates shipped with the product.
func Templates() []TemplateInfo {
	out := make([]TemplateInfo, len(templates))
	for i, t := range templates {
		out[i] = t.TemplateInfo
	}
	return out
}

func templateOf(code string) (chartTemplate, bool) {
	i := slices.IndexFunc(templates, func(t chartTemplate) bool { return t.Code == code })
	if i < 0 {
		return chartTemplate{}, false
	}
	return templates[i], true
}

// ApplyTemplate writes a template's accounts, its rules for the kinds declared now, the
// closing accounts and the statement forms, once, on a tenant without accounts.
func (s *Service) ApplyTemplate(ctx context.Context, code string) error {
	t, ok := templateOf(code)
	if !ok {
		return platform.ErrNotFound
	}
	return s.writeAccount(ctx, func(ctx context.Context) error {
		if any, err := store.AnyAccount.Run(ctx); err != nil {
			return err
		} else if any {
			return ErrChartNotEmpty
		}
		ids := map[string]int64{}
		for _, a := range t.Accounts {
			var parent *int64
			if a.Parent != "" {
				p := ids[a.Parent]
				parent = &p
			}
			id, err := store.CreateAccount.Run(ctx, store.CreateAccountParams{Code: a.Code, Name: a.Name, Type: a.Type,
				ParentID: platform.NullInt8(parent), Active: true, ByPartner: a.ByPartner})
			if err != nil {
				return err
			}
			ids[a.Code] = id
		}
		tracked := map[string]bool{}
		for _, a := range t.Accounts {
			tracked[a.Code] = a.ByPartner
		}
		for _, r := range t.Rules {
			d, declared := posting.Declared(posting.Kind(r.Kind))
			if !declared {
				continue
			}
			if d.ByPartner != tracked[r.Account] {
				return fmt.Errorf("accounting: template %s maps %s to %s", code, r.Kind, r.Account)
			}
			if err := store.SaveRule.Run(ctx, store.SaveRuleParams{Kind: r.Kind, AccountID: ids[r.Account], Side: r.Side}); err != nil {
				return err
			}
		}
		if err := store.SaveClosingAccounts.Run(ctx, store.SaveClosingAccountsParams{ResultAccountID: ids[t.Closing.Result], RetainedAccountID: ids[t.Closing.Retained]}); err != nil {
			return err
		}
		for form, rows := range t.Forms {
			if err := saveStatementRows(ctx, form, rows); err != nil {
				return err
			}
		}
		if err := store.InsertChartTemplate.Run(ctx, code); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "accounting.template_applied", map[string]any{"code": code, "accounts": len(t.Accounts)})
	})
}
