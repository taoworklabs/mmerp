package hrm

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/dataio"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

const monthLayout = "01/2006"

func (s *Service) registerReports() {
	for kind, run := range map[string]func(context.Context, json.RawMessage) (dataio.Sheet, error){
		"hrm.report.d02lt":      s.exportD02LT,
		"hrm.report.pit_05kk":   s.exportPIT05KK,
		"hrm.report.pit_05qtt":  s.exportPIT05QTT,
		"hrm.report.pit_05_1bk": s.exportPIT05_1BK,
		"hrm.report.pit_05_3bk": s.exportPIT05_3BK,
	} {
		s.d.DataIO.RegisterExport(dataio.Export{Kind: kind, Product: "hrm", Run: run, Check: s.checkReportFile})
	}
}

// reportMonth is one posted payroll of the legal entity: each employee's decrypted line.
type reportMonth struct {
	Start time.Time
	Lines map[int64]payrollLine
}

type reportBasis struct {
	Entity iam.OrgUnit
	Months []reportMonth
}

// reportBasis reads the posted payrolls of months months from first. Every one must exist:
// a report built on part of the data would look right and be wrong.
func (s *Service) reportBasis(ctx context.Context, entityID int64, first time.Time, months int) (reportBasis, error) {
	var b reportBasis
	units, err := s.d.IAM.OrgUnits(ctx, "", "")
	if err != nil {
		return b, err
	}
	i := slices.IndexFunc(units, func(u iam.OrgUnit) bool { return u.ID == entityID && u.Kind == "company" })
	if i < 0 {
		return b, dataio.ErrInvalidParams
	}
	b.Entity = units[i]
	if err := s.require(ctx, PermSalaryView, entityID); err != nil {
		return b, err
	}
	var missing []string
	for m := range months {
		start := first.AddDate(0, m, 0)
		id, err := store.PostedPayroll.Run(ctx, store.PostedPayrollParams{LegalEntityID: entityID, PeriodStart: pgtype.Date{Time: start, Valid: true}})
		if errors.Is(err, pgx.ErrNoRows) || err == nil && !id.Valid {
			missing = append(missing, start.Format(monthLayout))
			continue
		}
		if err != nil {
			return b, err
		}
		rows, err := store.PayrollLines.Run(ctx, id.Int64)
		if err != nil {
			return b, err
		}
		rm := reportMonth{Start: start, Lines: make(map[int64]payrollLine, len(rows))}
		for _, r := range rows {
			if rm.Lines[r.EmployeeID], err = openLine(ctx, r.Data); err != nil {
				return b, err
			}
		}
		b.Months = append(b.Months, rm)
	}
	if missing != nil {
		return b, &platform.Error{Status: ErrReportPayrollMissing.Status, Code: ErrReportPayrollMissing.Code,
			Params: map[string]any{"months": strings.Join(missing, ", ")}}
	}
	return b, nil
}

// employeeIDs lists everyone with a line in any of the months.
func (b reportBasis) employeeIDs() []int64 {
	var ids []int64
	for _, m := range b.Months {
		for id := range m.Lines {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

// reportEmployees reads the employees' plain fields, ordered by code.
func reportEmployees(ctx context.Context, ids []int64) ([]store.ReportEmployeesRow, error) {
	rows, err := store.ReportEmployees.Run(ctx, ids)
	slices.SortFunc(rows, func(a, b store.ReportEmployeesRow) int {
		return cmp.Compare(strings.ToLower(a.Code), strings.ToLower(b.Code))
	})
	return rows, err
}

// reportPerson is a listed employee with the identifiers the forms need, decrypted.
type reportPerson struct {
	store.ReportEmployeesRow
	SocialInsuranceNo, TaxCode, NationalID string
}

// reportPeople decrypts the identifiers of the employees a report lists. The actor must read
// sensitive values at each one's current unit; every read is audited, never with the values.
func (s *Service) reportPeople(ctx context.Context, rows []store.ReportEmployeesRow) ([]reportPerson, error) {
	if err := s.requireReportSensitive(ctx, rows); err != nil {
		return nil, err
	}
	out := make([]reportPerson, len(rows))
	for i, r := range rows {
		p := reportPerson{ReportEmployeesRow: r}
		var fields []string
		for f, dst := range map[string]*string{"social_insurance_no": &p.SocialInsuranceNo, "tax_code": &p.TaxCode, "national_id": &p.NationalID} {
			v, err := open(ctx, f, sealedOf(store.HrmEmployee{NationalID: r.NationalID, SocialInsuranceNo: r.SocialInsuranceNo, TaxCode: r.TaxCode})[f])
			if err != nil {
				return nil, err
			}
			if v != nil && *v != "" {
				*dst = *v
				fields = append(fields, f)
			}
		}
		if len(fields) > 0 {
			slices.Sort(fields)
			err := platform.InTx(ctx, func(ctx context.Context) error {
				return s.d.Audit.RecordFor(ctx, "hrm.employee_sensitive_exported", audit.Ref{Type: employeeType, ID: r.ID}, map[string]any{"fields": fields})
			})
			if err != nil {
				return nil, err
			}
		}
		out[i] = p
	}
	return out, nil
}

// requireReportSensitive refuses unless the actor reads sensitive values at every employee's
// current unit: even a report of totals is only for those who may see what it adds up.
func (s *Service) requireReportSensitive(ctx context.Context, rows []store.ReportEmployeesRow) error {
	sens, err := s.d.IAM.Scope(ctx, "hrm", PermSensitive)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if !sens.Has(r.OrgUnitID) {
			return platform.ErrForbidden
		}
	}
	return nil
}

// reportFile is what a report file holds, pinned when it is made.
type reportFile struct {
	LegalEntityID int64   `json:"legal_entity_id"`
	EmployeeIDs   []int64 `json:"employee_ids"`
}

// checkReportFile lets a download through only while the actor still sees the legal entity's
// pay and the sensitive values of every employee in the file.
func (s *Service) checkReportFile(ctx context.Context, pin json.RawMessage) error {
	var p reportFile
	if err := json.Unmarshal(pin, &p); err != nil || p.EmployeeIDs == nil || p.LegalEntityID == 0 {
		return platform.ErrForbidden
	}
	if err := s.require(ctx, PermSalaryView, p.LegalEntityID); err != nil {
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
	if len(units) != len(p.EmployeeIDs) {
		return platform.ErrForbidden
	}
	for _, u := range units {
		if !sens.Has(u.OrgUnitID) {
			return platform.ErrForbidden
		}
	}
	return nil
}

// reportSheet pins the legal entity and the employees a sheet is built from.
func reportSheet(name string, header []string, rows [][]any, text []int, entity int64, ids []int64) (dataio.Sheet, error) {
	ids = slices.Compact(slices.Sorted(slices.Values(ids)))
	if ids == nil {
		ids = []int64{}
	}
	pin, err := json.Marshal(reportFile{LegalEntityID: entity, EmployeeIDs: ids})
	return dataio.Sheet{Name: name, Header: header, Rows: rows, TextColumns: text, Pin: pin}, err
}

// entityHeader is the name, tax code and address the forms' headers show.
func entityHeader(u iam.OrgUnit) (name, taxCode, address string) {
	name = u.Name
	if u.LegalName != nil && *u.LegalName != "" {
		name = *u.LegalName
	}
	if u.TaxCode != nil {
		taxCode = *u.TaxCode
	}
	if u.Address != nil {
		address = *u.Address
	}
	return name, taxCode, address
}

func parseReportParams(params json.RawMessage) (ReportParams, error) {
	var p ReportParams
	if err := json.Unmarshal(params, &p); err != nil || p.LegalEntityID == 0 {
		return p, dataio.ErrInvalidParams
	}
	return p, nil
}

// reportMonthOf parses a YYYY-MM month to its first day.
func reportMonthOf(s string) (time.Time, error) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return t, dataio.ErrInvalidParams
	}
	return t, nil
}

// dottedThousands writes n the Vietnamese way: 22.000.000.
func dottedThousands(n int64) string {
	s := strings.TrimPrefix(strconv.FormatInt(n, 10), "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	if n < 0 {
		return "-" + s
	}
	return s
}

func personIDs(people []reportPerson) []int64 {
	ids := make([]int64, len(people))
	for i, p := range people {
		ids[i] = p.ID
	}
	return ids
}
