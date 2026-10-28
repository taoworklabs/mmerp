package hrm

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

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

// employeeMaxLen holds the form's maxLength per column, read from the input types' tags.
var employeeMaxLen = func() (out []struct{ col, max int }) {
	for col, f := range map[int]struct {
		t     any
		field string
	}{0: {NewEmployee{}, "Code"}, 1: {NewEmployee{}, "FullName"}, 4: {NewEmployee{}, "Phone"}, 5: {NewEmployee{}, "Email"},
		6: {NewEmployee{}, "Address"}, 11: {SensitiveValues{}, "NationalID"}, 12: {SensitiveValues{}, "SocialInsuranceNo"},
		13: {SensitiveValues{}, "TaxCode"}, 14: {SensitiveValues{}, "BankAccount"}} {
		sf, _ := reflect.TypeOf(f.t).FieldByName(f.field)
		n, err := strconv.Atoi(sf.Tag.Get("maxLength"))
		if err != nil {
			panic("hrm: no maxLength on " + f.field)
		}
		out = append(out, struct{ col, max int }{col, n})
	}
	slices.SortFunc(out, func(a, b struct{ col, max int }) int { return a.col - b.col })
	return out
}()

// Code, phone, manager code and the sensitive columns hold digit strings with leading zeros.
var employeeTextColumns = []int{0, 4, 8, 11, 12, 13, 14}

func (s *Service) registerEmployeeIO() {
	s.d.DataIO.RegisterExport(dataio.Export{Kind: employeeIO, Product: "hrm", Run: s.exportEmployees,
		Check: s.checkEmployeeFile})
	s.d.DataIO.RegisterImport(dataio.Import{Kind: employeeIO, Product: "hrm", Run: s.importEmployees, TextColumns: employeeTextColumns,
		Header: func(ctx context.Context, _ json.RawMessage) ([]string, error) {
			sc, err := s.d.IAM.Scope(ctx, "hrm", PermEdit)
			if err == nil && !sc.Any() {
				err = platform.ErrForbidden
			}
			if err != nil {
				return nil, err
			}
			return s.employeeColumns(ctx)
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

// employeeFile is what an employee export file holds, pinned when it is made.
type employeeFile struct {
	EmployeeIDs      []int64 `json:"employee_ids"`
	IncludeSensitive bool    `json:"include_sensitive"`
}

// checkEmployeeFile lets a download through only while the actor may still read every
// employee in the file, and their sensitive values when the file has them.
func (s *Service) checkEmployeeFile(ctx context.Context, pin json.RawMessage) error {
	var p employeeFile
	// A file without a pin predates it, or is not one; refuse rather than guess what it holds.
	if err := json.Unmarshal(pin, &p); err != nil || p.EmployeeIDs == nil {
		return platform.ErrForbidden
	}
	view, err := s.d.IAM.Scope(ctx, "hrm", PermView)
	if err != nil {
		return err
	}
	sens, err := s.d.IAM.Scope(ctx, "hrm", PermSensitive)
	if err != nil {
		return err
	}
	units, err := store.EmployeeUnits.Run(ctx, p.EmployeeIDs)
	if err != nil {
		return err
	}
	if len(units) != len(slices.Compact(slices.Sorted(slices.Values(p.EmployeeIDs)))) {
		return platform.ErrForbidden
	}
	for _, u := range units {
		if !view.Has(u.OrgUnitID) || (p.IncludeSensitive && !sens.Has(u.OrgUnitID)) {
			return platform.ErrForbidden
		}
	}
	return nil
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

// unitSuffixes gives each unit the paths that name it, shortest first: its trimmed name, then
// preceded by each trimmed ancestor's name in turn, joined by " / ".
func unitSuffixes(units []iam.OrgUnit) map[int64][]string {
	byID := make(map[int64]iam.OrgUnit, len(units))
	for _, u := range units {
		byID[u.ID] = u
	}
	out := make(map[int64][]string, len(units))
	for _, u := range units {
		paths := []string{strings.TrimSpace(u.Name)}
		for p := u.ParentID; p != nil; {
			pu, ok := byID[*p]
			if !ok {
				break
			}
			paths = append(paths, strings.TrimSpace(pu.Name)+" / "+paths[len(paths)-1])
			p = pu.ParentID
		}
		out[u.ID] = paths
	}
	return out
}

// unitIndex maps every path to the units it names.
func unitIndex(units []iam.OrgUnit) map[string][]int64 {
	out := map[string][]int64{}
	for id, paths := range unitSuffixes(units) {
		for _, p := range paths {
			out[p] = append(out[p], id)
		}
	}
	return out
}

// unitPaths gives every unit its shortest unique path, `name` or `parent / name` up the tree;
// a unit whose full path is still shared gets the full path.
func unitPaths(units []iam.OrgUnit) map[int64]string {
	idx := unitIndex(units)
	out := make(map[int64]string, len(units))
	for id, paths := range unitSuffixes(units) {
		out[id] = paths[len(paths)-1]
		for _, p := range paths {
			if len(idx[p]) == 1 {
				out[id] = p
				break
			}
		}
	}
	return out
}

// resolveUnit finds the units a cell names, as unitIndex spells their paths.
func resolveUnit(idx map[string][]int64, cell string) (id int64, matches int) {
	ids := idx[strings.TrimSpace(cell)]
	if len(ids) == 0 {
		return 0, 0
	}
	return ids[0], len(ids)
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
	ids := make([]int64, len(items))
	for i, e := range items {
		ids[i] = e.ID
	}
	pin, err := json.Marshal(employeeFile{EmployeeIDs: ids, IncludeSensitive: p.IncludeSensitive})
	if err != nil {
		return dataio.Sheet{}, err
	}
	return dataio.Sheet{Name: platform.Translate(loc, "hrm.import.sheet.employees", nil), Header: header, Rows: rows, TextColumns: employeeTextColumns, Pin: pin}, nil
}

// importEmployees creates the rows whose code is new or empty and edits the others. An empty
// cell keeps the current value, so a re-imported export loses nothing, sensitive cells included.
func (s *Service) importEmployees(ctx context.Context, _ json.RawMessage, rows [][]string) ([]dataio.RowError, error) {
	header, err := s.employeeColumns(ctx)
	if err != nil {
		return nil, err
	}
	units, err := s.d.IAM.OrgUnits(ctx, "", "")
	if err != nil {
		return nil, err
	}
	unitIdx := unitIndex(units)
	view, err := s.d.IAM.Scope(ctx, "hrm", PermView)
	if err != nil {
		return nil, err
	}
	var codes []string
	for _, r := range rows {
		if r != nil {
			codes = append(codes, strings.ToLower(strings.TrimSpace(r[0])), strings.ToLower(strings.TrimSpace(r[8])))
		}
	}
	found, err := store.EmployeesByCodes.Run(ctx, codes)
	if err != nil {
		return nil, err
	}
	known := make(map[string]store.EmployeesByCodesRow, len(found))
	for _, e := range found {
		known[strings.ToLower(e.Code)] = e
	}
	// Codes this file creates, so a row may name one as manager.
	inFile := map[string]bool{}
	for _, r := range rows {
		if r == nil {
			continue
		}
		if key := strings.ToLower(strings.TrimSpace(r[0])); key != "" {
			if _, ok := known[key]; !ok {
				inFile[key] = true
			}
		}
	}
	var errs []dataio.RowError
	var creates, updates []parsedRow
	seen := map[string]bool{}
	for i, r := range rows {
		if r == nil {
			continue
		}
		code := strings.TrimSpace(r[0])
		n := len(errs)
		fail := func(c string, params map[string]any) {
			errs = append(errs, dataio.RowError{Row: i, Code: c, Params: params})
		}
		key := strings.ToLower(code)
		if code != "" && seen[key] {
			fail("duplicate_employee", map[string]any{"code": code})
			continue
		}
		seen[key] = true
		cur, exists := known[key]
		exists = exists && code != ""
		var in EmployeeInput
		var old store.GetEmployeeRow
		if exists {
			// Elsewhere such an employee does not exist, but here that would mean "create".
			if !view.Has(cur.OrgUnitID) {
				fail("employee_code_taken", map[string]any{"code": code})
				continue
			}
			// Locked before reading, so a form edit cannot land between this read and the write.
			if _, err := store.LockEmployee.Run(ctx, cur.ID); err != nil {
				return nil, err
			}
			if old, err = store.GetEmployee.Run(ctx, cur.ID); err != nil {
				return nil, err
			}
			in.EmployeeFields = EmployeeFields{
				FullName: old.FullName, DateOfBirth: platform.DatePtr(old.DateOfBirth), Gender: platform.TextPtr(old.Gender),
				Phone: platform.TextPtr(old.Phone), Email: platform.TextPtr(old.Email), Address: platform.TextPtr(old.Address), OrgUnitID: old.OrgUnitID,
				ManagerID: platform.Int8Ptr(old.ManagerID), UserLogin: platform.TextPtr(old.UserLogin), HireDate: *platform.DatePtr(old.HireDate),
				TerminationDate: platform.DatePtr(old.TerminationDate),
			}
		}
		changed := false
		text := func(dst **string, v string) {
			if v != "" && (*dst == nil || **dst != v) {
				*dst, changed = &v, true
			}
		}
		date := func(col int, dst **string) {
			v, ok := cellDate(r[col])
			if !ok {
				fail("invalid_date", map[string]any{"column": header[col], "value": r[col]})
			}
			text(dst, v)
		}
		if v := strings.TrimSpace(r[1]); v != "" && v != in.FullName {
			in.FullName, changed = v, true
		}
		date(2, &in.DateOfBirth)
		if g, ok := cellGender(r[3]); !ok {
			fail("invalid_gender", map[string]any{"value": r[3]})
		} else if g != nil {
			text(&in.Gender, *g)
		}
		text(&in.Phone, strings.TrimSpace(r[4]))
		text(&in.Email, strings.TrimSpace(r[5]))
		text(&in.Address, strings.TrimSpace(r[6]))
		if v := strings.TrimSpace(r[7]); v != "" {
			switch id, matches := resolveUnit(unitIdx, v); {
			case matches == 0:
				fail("unknown_org_unit", map[string]any{"value": v})
			case matches > 1:
				fail("ambiguous_org_unit", map[string]any{"value": v})
			case id != in.OrgUnitID:
				in.OrgUnitID, changed = id, true
			}
		}
		manager := strings.TrimSpace(r[8])
		mkey, newManager := strings.ToLower(manager), ""
		switch m, ok := known[mkey]; {
		case manager == "":
		case code != "" && mkey == key:
			fail("manager_self", map[string]any{"code": code})
		case inFile[mkey]:
			newManager, changed = mkey, true
		case !ok:
			fail("unknown_manager", map[string]any{"code": manager})
		case in.ManagerID == nil || *in.ManagerID != m.ID:
			in.ManagerID, changed = &m.ID, true
		}
		var hire *string
		if in.HireDate != "" {
			hire = &in.HireDate
		}
		date(9, &hire)
		if hire != nil {
			in.HireDate = *hire
		}
		date(10, &in.TerminationDate)
		for _, l := range employeeMaxLen {
			if utf8.RuneCountInString(strings.TrimSpace(r[l.col])) > l.max {
				fail("too_long", map[string]any{"column": header[l.col], "max": l.max})
			}
		}
		if !exists {
			for _, req := range []struct {
				col  int
				code string
			}{{1, "name_required"}, {7, "org_unit_required"}, {9, "hire_date_required"}} {
				if strings.TrimSpace(r[req.col]) == "" {
					fail(req.code, nil)
				}
			}
		}
		if len(errs) > n {
			continue
		}
		var sens SensitiveValues
		cells := []**string{&sens.NationalID, &sens.SocialInsuranceNo, &sens.TaxCode, &sens.BankAccount}
		filled := false
		for j, dst := range cells {
			if v := strings.TrimSpace(r[11+j]); v != "" {
				*dst, filled = &v, true
			}
		}
		if filled {
			// Checked before comparing, so a guess at a hidden value learns nothing.
			var was []int64
			if exists && old.OrgUnitID != in.OrgUnitID {
				was = append(was, old.OrgUnitID)
			}
			if ok, err := s.allowed(ctx, PermSensitive, in.OrgUnitID, was...); err != nil {
				return nil, err
			} else if !ok {
				fail("sensitive_forbidden", map[string]any{"code": code})
				continue
			}
			sealed := sealedOf(store.HrmEmployee{NationalID: old.NationalID, SocialInsuranceNo: old.SocialInsuranceNo, TaxCode: old.TaxCode, BankAccount: old.BankAccount})
			for j, f := range sensitiveFields {
				if *cells[j] == nil {
					continue
				}
				// An equal value stays as stored, so the audit shows no change.
				v, err := open(ctx, f, sealed[f])
				if err != nil {
					return nil, err
				}
				if v != nil && *v == **cells[j] {
					*cells[j] = nil
				} else {
					in.Sensitive, changed = &sens, true
				}
			}
		}
		w := parsedRow{row: i, code: code, manager: manager, newManager: newManager, in: in}
		switch {
		case !exists:
			creates = append(creates, w)
		case changed:
			w.id, w.code = old.ID, old.Code
			updates = append(updates, w)
		}
	}
	newCodes := map[string]int{}
	for i, w := range creates {
		if w.code != "" {
			newCodes[strings.ToLower(w.code)] = i
		}
	}
	order, err := createOrder(creates, newCodes)
	if err != nil {
		w := creates[order[0]]
		m := creates[newCodes[w.newManager]]
		errs = append(errs, dataio.RowError{Row: w.row, Code: ErrManagerCycle.Code, Params: map[string]any{"code": w.code, "name": m.code + " · " + m.in.FullName}})
	}
	if len(errs) > 0 {
		return errs, nil
	}
	todo := make([]parsedRow, 0, len(creates)+len(updates))
	for _, i := range order {
		todo = append(todo, creates[i])
	}
	ids := map[string]int64{}
	var managed []parsedRow
	for _, w := range append(todo, updates...) {
		if w.newManager != "" {
			id, ok := ids[w.newManager]
			if !ok {
				continue // the manager's row failed and reports it
			}
			w.in.ManagerID = &id
		}
		if w.id == 0 {
			var id int64
			if id, err = s.CreateEmployee(ctx, NewEmployee{Code: w.code, EmployeeInput: w.in}); err == nil {
				w.id, ids[strings.ToLower(w.code)] = id, id
			}
		} else {
			err = s.updateEmployee(ctx, w.id, w.in, false)
		}
		if err == nil {
			if w.manager != "" {
				managed = append(managed, w)
			}
			continue
		}
		e, ok := writeRowError(w.row, w.code, w.manager, err)
		if !ok {
			return nil, err
		}
		errs = append(errs, e)
		if _, ok := errors.AsType[*pgconn.PgError](err); ok {
			return errs, nil // the transaction is aborted
		}
	}
	if len(errs) > 0 || len(managed) == 0 {
		return errs, nil
	}
	// The end state is checked, so a file may swap managers; the lock comes after the
	// employee rows, in the form's order.
	if err := store.LockManagerTree.Run(ctx); err != nil {
		return nil, err
	}
	for _, w := range managed {
		chain, err := store.ManagerChain.Run(ctx, *w.in.ManagerID)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(chain, w.id) {
			continue
		}
		m, err := store.GetEmployee.Run(ctx, *w.in.ManagerID)
		if err != nil {
			return nil, err
		}
		errs = append(errs, dataio.RowError{Row: w.row, Code: ErrManagerCycle.Code, Params: map[string]any{"code": w.code, "name": m.Code + " · " + m.FullName}})
	}
	return errs, nil
}

// parsedRow is a checked import row waiting to be written.
type parsedRow struct {
	row           int
	id            int64 // 0 creates
	code, manager string
	newManager    string // lower-cased code of a manager another row creates
	in            EmployeeInput
}

// createOrder orders the new rows so a manager created by the file comes before its reports, and
// typed codes before issued ones, so the series never issues a code a later row types. Only typed
// rows can be named as managers. On a cycle it returns the index of one row in it.
func createOrder(rows []parsedRow, newCodes map[string]int) ([]int, error) {
	const visiting, done = 1, 2
	state := make([]int, len(rows))
	var order []int
	var visit func(i int) int // -1, or a row in a cycle
	visit = func(i int) int {
		switch state[i] {
		case done:
			return -1
		case visiting:
			return i
		}
		state[i] = visiting
		if m, ok := newCodes[rows[i].newManager]; ok {
			if c := visit(m); c >= 0 {
				return c
			}
		}
		state[i] = done
		order = append(order, i)
		return -1
	}
	for _, typed := range []bool{true, false} {
		for i, r := range rows {
			if (r.code != "") == typed {
				if c := visit(i); c >= 0 {
					return []int{c}, ErrManagerCycle
				}
			}
		}
	}
	return order, nil
}

// writeRowError turns a service's refusal of a row into the row's error; other errors are not the row's.
func writeRowError(row int, code, manager string, err error) (dataio.RowError, bool) {
	switch pe, ok := errors.AsType[*platform.Error](err); {
	case errors.Is(err, platform.ErrForbidden):
		return dataio.RowError{Row: row, Code: "forbidden_row", Params: map[string]any{"code": code}}, true
	case errors.Is(err, ErrManagerNotFound):
		return dataio.RowError{Row: row, Code: "unknown_manager", Params: map[string]any{"code": manager}}, true
	case ok:
		params := map[string]any{"code": code}
		maps.Copy(params, pe.Params)
		return dataio.RowError{Row: row, Code: pe.Code, Params: params}, true
	}
	return dataio.RowError{}, false
}

// cellDate reads dd/mm/yyyy, yyyy-mm-dd or an Excel date; blank is no date.
func cellDate(cell string) (string, bool) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return "", true
	}
	for _, layout := range []string{"2/1/2006", time.DateOnly} {
		if t, err := time.Parse(layout, cell); err == nil {
			return t.Format(time.DateOnly), true
		}
	}
	// Excel counts days from 30/12/1899.
	// Below 1910 it is more likely a year typed alone.
	if n, err := strconv.Atoi(cell); err == nil && n >= 3653 && n < 2958466 {
		return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n).Format(time.DateOnly), true
	}
	return "", false
}

// cellGender reads a gender label in either language, or its code.
func cellGender(cell string) (*string, bool) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return nil, true
	}
	for _, g := range []string{"male", "female", "other"} {
		key := "hrm.import.gender." + g
		if strings.EqualFold(cell, g) || strings.EqualFold(cell, platform.Translate("vi", key, nil)) || strings.EqualFold(cell, platform.Translate("en", key, nil)) {
			return &g, true
		}
	}
	return nil, false
}
