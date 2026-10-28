package hrm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	employeeIO = "hrm.employees"
	dateLayout = "02/01/2006"
)

var employeeHeaderKeys = []string{"code", "name", "date_of_birth", "gender", "phone", "email", "address", "org_unit", "manager",
	"hire_date", "termination_date", "national_id", "social_insurance_no", "tax_code", "bank_account"}

func (s *Service) registerEmployeeIO() {
	s.d.DataIO.RegisterExport(dataio.Export{Kind: employeeIO, Product: "hrm", Run: s.exportEmployees,
		Check: func(ctx context.Context, params json.RawMessage) error {
			_, err := s.exportedEmployees(ctx, params)
			return err
		}})
}

// exportedEmployees lists who the export holds: the filter within the actor's view scope. With
// sensitive values asked for, the actor must also read those of every one of them.
func (s *Service) exportedEmployees(ctx context.Context, params json.RawMessage) ([]store.EmployeeExportBaseRow, error) {
	if err := s.requireEmployeeView(ctx); err != nil {
		return nil, err
	}
	var p EmployeeExportParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, dataio.ErrInvalidParams
		}
	}
	sc, err := s.d.IAM.DataScope(ctx, "hrm", PermView)
	if err != nil {
		return nil, err
	}
	today, err := s.d.Setting.Today(ctx)
	if err != nil {
		return nil, err
	}
	// ponytail: one unpaged read; stream if tenants pass tens of thousands of employees
	page, err := employeeExportList.Run(ctx, employeeQuery(sc, EmployeeFilter{OrgUnitID: p.OrgUnitID, Status: p.Status, Q: p.Q, Paging: platform.Paging{Page: 1, PageSize: 1 << 30}}, today))
	if err != nil {
		return nil, err
	}
	if p.IncludeSensitive {
		sens, err := s.d.IAM.Scope(ctx, "hrm", PermSensitive)
		if err != nil {
			return nil, err
		}
		for _, e := range page.Items {
			if !sens.Has(e.OrgUnitID) {
				return nil, platform.ErrForbidden
			}
		}
	}
	return page.Items, nil
}

// requireEmployeeView refuses whoever may view no employee.
func (s *Service) requireEmployeeView(ctx context.Context) error {
	sc, err := s.d.IAM.Scope(ctx, "hrm", PermView)
	if err == nil && !sc.Any() {
		err = platform.ErrForbidden
	}
	return err
}

// employeeColumns are the sheet's titles in the actor's language.
func (s *Service) employeeColumns(ctx context.Context) ([]string, error) {
	loc, err := s.locale(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(employeeHeaderKeys))
	for i, k := range employeeHeaderKeys {
		out[i] = platform.Translate(loc, "hrm.import.header."+k, nil)
	}
	return out, nil
}

func genderLabel(loc string, g *string) string {
	if g == nil {
		return ""
	}
	return platform.Translate(loc, "hrm.import.gender."+*g, nil)
}

// unitPaths gives every unit its shortest unique path, `name` or `parent / name` up the tree;
// a unit whose full path is still shared gets the full path.
func unitPaths(units []iam.OrgUnit) map[int64]string {
	byID := make(map[int64]iam.OrgUnit, len(units))
	for _, u := range units {
		byID[u.ID] = u
	}
	chain := func(u iam.OrgUnit) []string { // root first
		names := []string{u.Name}
		for p := u.ParentID; p != nil; {
			pu, ok := byID[*p]
			if !ok {
				break
			}
			names = append([]string{pu.Name}, names...)
			p = pu.ParentID
		}
		return names
	}
	out := make(map[int64]string, len(units))
	for _, u := range units {
		names := chain(u)
		out[u.ID] = strings.Join(names, " / ")
		for n := 1; n <= len(names); n++ {
			path := strings.Join(names[len(names)-n:], " / ")
			if _, matches := resolveUnit(units, path); matches == 1 {
				out[u.ID] = path
				break
			}
		}
	}
	return out
}

// resolveUnit finds the units a path names: the last segment is the unit's name, the
// segments before it a suffix of its ancestors' names.
func resolveUnit(units []iam.OrgUnit, path string) (id int64, matches int) {
	var segs []string
	for _, p := range strings.Split(path, "/") {
		segs = append(segs, strings.TrimSpace(p))
	}
	byID := make(map[int64]iam.OrgUnit, len(units))
	for _, u := range units {
		byID[u.ID] = u
	}
	for _, u := range units {
		cur, ok := u, true
		for i := len(segs) - 1; i >= 0 && ok; i-- {
			if cur.Name != segs[i] {
				ok = false
				break
			}
			if i > 0 {
				if cur.ParentID == nil {
					ok = false
					break
				}
				cur, ok = byID[*cur.ParentID]
			}
		}
		if ok {
			id, matches = u.ID, matches+1
		}
	}
	return id, matches
}

// exportEmployees writes the employees the list shows, in the import's layout. Sensitive
// cells are filled only where the actor may read them, and each such read is audited.
func (s *Service) exportEmployees(ctx context.Context, params json.RawMessage) (dataio.Sheet, error) {
	var p EmployeeExportParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return dataio.Sheet{}, dataio.ErrInvalidParams
		}
	}
	items, err := s.exportedEmployees(ctx, params)
	if err != nil {
		return dataio.Sheet{}, err
	}
	header, err := s.employeeColumns(ctx)
	if err != nil {
		return dataio.Sheet{}, err
	}
	loc, err := s.locale(ctx)
	if err != nil {
		return dataio.Sheet{}, err
	}
	units, err := s.d.IAM.OrgUnits(ctx, "", "")
	if err != nil {
		return dataio.Sheet{}, err
	}
	paths := unitPaths(units)
	rows := make([][]any, 0, len(items))
	for _, e := range items {
		row := make([]any, len(header))
		row[0], row[1] = e.Code, e.FullName
		if e.DateOfBirth.Valid {
			row[2] = e.DateOfBirth.Time.Format(dateLayout)
		}
		row[3] = genderLabel(loc, platform.TextPtr(e.Gender))
		row[4], row[5], row[6] = e.Phone.String, e.Email.String, e.Address.String
		row[7], row[8] = paths[e.OrgUnitID], e.ManagerCode.String
		row[9] = e.HireDate.Time.Format(dateLayout)
		if e.TerminationDate.Valid {
			row[10] = e.TerminationDate.Time.Format(dateLayout)
		}
		if p.IncludeSensitive {
			var fields []string
			sealed := sealedOf(store.HrmEmployee{NationalID: e.NationalID, SocialInsuranceNo: e.SocialInsuranceNo, TaxCode: e.TaxCode, BankAccount: e.BankAccount})
			for i, f := range sensitiveFields {
				v, err := open(ctx, f, sealed[f])
				if err != nil {
					return dataio.Sheet{}, err
				}
				if v != nil && *v != "" {
					row[11+i] = *v
					fields = append(fields, f)
				}
			}
			if len(fields) > 0 {
				err := platform.InTx(ctx, func(ctx context.Context) error {
					return s.d.Audit.RecordFor(ctx, "hrm.employee_sensitive_exported", audit.Ref{Type: employeeType, ID: e.ID}, map[string]any{"fields": fields})
				})
				if err != nil {
					return dataio.Sheet{}, err
				}
			}
		}
		rows = append(rows, row)
	}
	return dataio.Sheet{Name: platform.Translate(loc, "hrm.import.sheet.employees", nil), Header: header, Rows: rows}, nil
}
