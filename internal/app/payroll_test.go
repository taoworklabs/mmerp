package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/app"
)

// payrollSetup adds to the jobs fixture a payroll user and posted contracts for E1 and E2.
func (f *jobsFixture) payrollSetup() *client {
	f.t.Helper()
	pay := f.id(f.admin, "/api/users", `{"login":"pay","name":"Kế toán lương","password":"correct horse"}`)
	f.payID = pay
	f.payGrant = f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", pay), `{"product":"hrm","role":"payroll","org_unit_id":null}`)
	f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", pay), `{"product":"hrm","role":"hr","org_unit_id":null}`)
	c := f.login("pay")
	kind := f.id(f.head, "/api/hrm/contract-types", `{"name":"Không xác định thời hạn","fixed_term":false,"active":true}`)
	for _, e := range []int64{f.e1, f.e2} {
		id := f.id(c, "/api/hrm/contracts", fmt.Sprintf(`{"employee_id":%d,"contract_type_id":%d,"start_date":"2026-01-01","end_date":null,
			"terms":{"salary":22000000,"lines":[]}}`, e, kind))
		f.ok(c, "POST", fmt.Sprintf("/api/documents/hrm.contract/%d/transitions", id), `{"to":"posted","version":1}`, 200)
	}
	return c
}

func (f *jobsFixture) legalEntity() int64 {
	f.t.Helper()
	var c int64
	if err := f.system.QueryRow(f.t.Context(), `SELECT id FROM iam.org_units WHERE kind = 'company'`).Scan(&c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

// approveAll approves, as head, every step waiting in head's inbox; status is what each answer must be.
func (f *jobsFixture) approveAll(status int) []byte {
	f.t.Helper()
	var inbox struct {
		Items []struct {
			InstanceID int64 `json:"instance_id"`
		}
	}
	_ = json.Unmarshal(f.ok(f.head, "GET", "/api/approvals/inbox", "", 200), &inbox)
	var last []byte
	for _, i := range inbox.Items {
		last = f.ok(f.head, "POST", fmt.Sprintf("/api/approvals/%d/approve", i.InstanceID), `{"step":1}`, status)
	}
	return last
}

type payrollView struct {
	Status         string
	Version        int32
	SourcesChanged bool `json:"sources_changed"`
	Lines          []struct {
		EmployeeID int64 `json:"employee_id"`
		Gross      int64
	}
}

func (f *jobsFixture) payroll(c *client, id int64) payrollView {
	f.t.Helper()
	var p payrollView
	_ = json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/hrm/payrolls/%d", id), "", 200), &p)
	return p
}

func (f *jobsFixture) createPayroll(c *client) int64 {
	f.t.Helper()
	var out struct {
		ID    int64 `json:"id"`
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(c, "POST", "/api/hrm/payrolls", fmt.Sprintf(`{"legal_entity_id":%d,"month":"2026-03"}`, f.legalEntity()), 201), &out)
	if j := f.wait(c, out.JobID); j.State != "completed" {
		f.t.Fatalf("compute: %+v", j)
	}
	return out.ID
}

// Steps 9 to 13 of the sample flow, HRM only: no accounting, so posting a payroll leaves
// posting lines and queues no posting job.
func TestPayrollSampleFlow(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	f.wait(f.hr, f.upload(f.hr, "hrm.timesheet", f.params(), march()...))
	f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"posted","version":2}`, 200)
	f.approveAll(204)
	pay := f.payrollSetup()
	f.ok(f.admin, "PUT", "/api/approval-rules/hrm.overtime_request",
		`{"flow":[{"step":{"approver":{"kind":"role","product":"hrm","role":"hr"},"approvals":1,"rejections":1}}],"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`, 204)
	overtime := func(date string) int64 {
		id := f.id(f.hr, "/api/hrm/overtimes", fmt.Sprintf(`{"employee_id":%d,"date":%q,"day_kind":"weekday","day_hours":"2","night_hours":"0","reason":null}`, f.e1, date))
		f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.overtime_request/%d/transitions", id), `{"to":"posted","version":1}`, 200)
		return id
	}

	// 9: the payroll of March, computed by a job.
	id := f.createPayroll(pay)
	if p := f.payroll(pay, id); p.Status != "draft" || len(p.Lines) != 2 || p.SourcesChanged {
		t.Fatalf("payroll: %+v", p)
	}
	// 10: an overtime request approved since: sending is refused until computed again.
	overtime("2026-03-16")
	f.approveAll(204)
	p := f.payroll(pay, id)
	if !p.SourcesChanged {
		t.Fatal("sources unchanged")
	}
	wantCode(t, pay.do("POST", fmt.Sprintf("/api/documents/hrm.payroll/%d/transitions", id), fmt.Sprintf(`{"to":"posted","version":%d}`, p.Version)),
		409, "payroll_sources_changed")
	// 11: compute, send, posted.
	var queued struct {
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(pay, "POST", fmt.Sprintf("/api/hrm/payrolls/%d/compute", id), fmt.Sprintf(`{"version":%d}`, p.Version), 202), &queued)
	f.wait(pay, queued.JobID)
	if told := notifications(pay).Items; len(told) == 0 || told[0].Kind != "job_completed" || told[0].JobID == nil || *told[0].JobID != queued.JobID {
		t.Fatalf("pay told %+v", told)
	}
	f.ok(pay, "POST", fmt.Sprintf("/api/documents/hrm.payroll/%d/transitions", id), fmt.Sprintf(`{"to":"posted","version":%d}`, f.payroll(pay, id).Version), 200)
	var lines, others int
	if err := f.system.QueryRow(t.Context(), `SELECT count(*) FROM posting.lines WHERE doc_id = $1 AND voided_at IS NULL`, id).Scan(&lines); err != nil {
		t.Fatal(err)
	}
	if err := f.system.QueryRow(t.Context(), `SELECT count(*) FROM river_job WHERE kind NOT IN ('dataio.import', 'dataio.export', 'dataio.cleanup', 'attachment.cleanup', 'approval.overdue', 'hrm.leave_accrual', 'hrm.payroll_compute')`).Scan(&others); err != nil {
		t.Fatal(err)
	}
	if lines != 5 || others != 0 {
		t.Fatalf("%d posting lines, %d other jobs", lines, others)
	}
	// 11': another March overtime cannot be approved into the closed period; it stays pending.
	late := overtime("2026-03-17")
	if body := f.approveAll(409); !bytes.Contains(body, []byte("payroll_period_closed")) {
		t.Fatalf("approve: %s", body)
	}
	// 12': nor can March be locked while it waits.
	wantCode(t, f.admin.do("PUT", fmt.Sprintf("/api/period-locks/%d", f.legalEntity()), `{"locked_until":"2026-03-31"}`), 409, "period_has_pending_documents")
	f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.overtime_request/%d/transitions", late), `{"to":"draft","version":1}`, 200)
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/period-locks/%d", f.legalEntity()), `{"locked_until":"2026-03-31"}`, 204)
	// 13: the timesheet cannot be cancelled.
	wantCode(t, f.hr.do("POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"cancelled","version":3}`), 409, "period_locked")

	// The export: a row per employee, the department's total, the grand total; HR without
	// the payroll role may not run it.
	export := func(c *client) job {
		var ex struct {
			JobID int64 `json:"job_id"`
		}
		_ = json.Unmarshal(f.ok(c, "POST", "/api/exports/hrm.payroll", fmt.Sprintf(`{"params":{"payroll_id":%d}}`, id), 202), &ex)
		return f.wait(c, ex.JobID)
	}
	j := export(pay)
	x, err := excelize.OpenReader(bytes.NewReader(f.ok(pay, "GET", "/api/files/"+*j.FileID, "", 200)))
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := x.GetRows("Sheet1", excelize.Options{RawCellValue: true})
	if len(rows) != 5 || rows[1][0] != "E1" || rows[3][1] != "Tổng P" || rows[4][1] != "Tổng cộng" {
		t.Fatalf("exported rows: %v", rows)
	}
	if j := export(f.head); j.Code == nil || *j.Code != "forbidden" {
		t.Fatalf("head's export: %+v", j)
	}
	// Once the payroll role is revoked, the file already exported is gone too.
	f.ok(f.admin, "DELETE", fmt.Sprintf("/api/users/%d/roles/%d", f.payID, f.payGrant), "", 204)
	f.ok(pay, "GET", "/api/files/"+*j.FileID, "", 404)
}

// With HRM turned off while jobs wait, a payroll export still runs and a computation fails.
func TestPayrollJobsWithProductOff(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	// No timesheet is needed when nobody is paid: drop the contracts.
	if _, err := f.system.Exec(t.Context(), `UPDATE record.documents SET status = 'cancelled' WHERE doc_type = 'hrm.contract'`); err != nil {
		t.Fatal(err)
	}
	jobs, err := app.NewJobs(f.env, app.Modules())
	if err != nil {
		t.Fatal(err)
	}
	if err := jobs.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	id := f.createPayroll(pay)
	if err := jobs.Stop(t.Context()); err != nil {
		t.Fatal(err)
	}

	var ex, comp struct {
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(pay, "POST", "/api/exports/hrm.payroll", fmt.Sprintf(`{"params":{"payroll_id":%d}}`, id), 202), &ex)
	_ = json.Unmarshal(f.ok(pay, "POST", fmt.Sprintf("/api/hrm/payrolls/%d/compute", id), `{"version":2}`, 202), &comp)
	f.work(nil)
	if j := f.wait(pay, ex.JobID); j.State != "completed" {
		t.Fatalf("export: %+v", j)
	}
	if j := f.wait(pay, comp.JobID); j.State != "failed" || *j.Code != "product_not_enabled" {
		t.Fatalf("compute: %+v", j)
	}
	told := notifications(pay).Items
	if !slices.ContainsFunc(told, func(n notification) bool { return n.Kind == "job_failed" && n.JobID != nil && *n.JobID == comp.JobID }) {
		t.Fatalf("pay told %+v", told)
	}
}

// A posted payroll reaches the ledger through the reconcile job, and its cancellation
// reverses it; with accounting off, payrolls still post and cancel and queue nothing.
func TestPayrollReachesLedger(t *testing.T) {
	f := newJobsFixture(t)
	f.env.Products = []string{"hrm", "accounting"}
	f.admin, f.hr, f.head = f.login("admin"), f.login("hr"), f.login("head")
	f.work(f.env.Products)
	f.wait(f.hr, f.upload(f.hr, "hrm.timesheet", f.params(), march()...))
	f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"posted","version":2}`, 200)
	f.approveAll(204)
	pay := f.payrollSetup()

	accounts := map[string]int64{}
	for code, typ := range map[string]string{"642": "expense", "334": "liability", "338": "liability", "3335": "liability"} {
		accounts[code] = f.id(f.admin, "/api/accounting/accounts", fmt.Sprintf(`{"code":%q,"name":"Tài khoản %s","type":%q,"parent_id":null,"active":true}`, code, code, typ))
	}
	for kind, rule := range map[string][2]string{"salary_expense": {"642", "debit"}, "employer_insurance_expense": {"642", "debit"},
		"salary_payable": {"334", "credit"}, "insurance_payable": {"338", "credit"}, "pit_payable": {"3335", "credit"}} {
		f.ok(f.admin, "PUT", "/api/accounting/rules/"+kind, fmt.Sprintf(`{"account_id":%d,"side":%q}`, accounts[rule[0]], rule[1]), 204)
	}
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/accounting/starts/%d", f.legalEntity()), `{"start_date":"2026-01-01"}`, 204)

	// entries waits for the payroll to have n entries, then returns each entry's debit and credit totals.
	type entry struct{ debit, credit int64 }
	entries := func(id int64, n int) []entry {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			rows, err := f.system.Query(t.Context(), `SELECT sum(l.debit), sum(l.credit) FROM accounting.entries e JOIN accounting.entry_lines l ON l.entry_id = e.id
				WHERE e.doc_type = 'hrm.payroll' AND e.doc_id = $1 GROUP BY e.id ORDER BY e.id`, id)
			if err != nil {
				t.Fatal(err)
			}
			var out []entry
			for rows.Next() {
				var e entry
				if err := rows.Scan(&e.debit, &e.credit); err != nil {
					t.Fatal(err)
				}
				out = append(out, e)
			}
			rows.Close()
			if len(out) >= n {
				return out
			}
		}
		t.Fatalf("payroll %d did not reach %d entries", id, n)
		return nil
	}
	post := func(c *client) (int64, int64) {
		t.Helper()
		id := f.createPayroll(c)
		var p struct {
			Version int32
			Lines   []struct{ Cost int64 }
		}
		_ = json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/hrm/payrolls/%d", id), "", 200), &p)
		f.ok(c, "POST", fmt.Sprintf("/api/documents/hrm.payroll/%d/transitions", id), fmt.Sprintf(`{"to":"posted","version":%d}`, p.Version), 200)
		var cost int64
		for _, l := range p.Lines {
			cost += l.Cost
		}
		return id, cost
	}
	cancel := func(c *client, id int64) {
		t.Helper()
		f.ok(c, "POST", fmt.Sprintf("/api/documents/hrm.payroll/%d/transitions", id), fmt.Sprintf(`{"to":"cancelled","version":%d}`, f.payroll(c, id).Version), 200)
	}

	id, cost := post(pay)
	if es := entries(id, 1); len(es) != 1 || es[0].debit != cost || es[0].credit != cost || cost == 0 {
		t.Fatalf("entries %+v, cost %d", es, cost)
	}
	var led struct {
		Closing int64
		Lines   []struct {
			DocType string `json:"doc_type"`
		}
	}
	q := fmt.Sprintf("legal_entity_id=%d&from=2026-01-01&to=2026-12-31", f.legalEntity())
	_ = json.Unmarshal(f.ok(f.admin, "GET", fmt.Sprintf("/api/accounting/ledger?%s&account_id=%d", q, accounts["642"]), "", 200), &led)
	if led.Closing != cost || len(led.Lines) == 0 || led.Lines[0].DocType != "hrm.payroll" {
		t.Fatalf("ledger %+v, cost %d", led, cost)
	}
	var tb struct {
		TotalDebit  int64 `json:"total_debit"`
		TotalCredit int64 `json:"total_credit"`
	}
	_ = json.Unmarshal(f.ok(f.admin, "GET", "/api/accounting/trial-balance?"+q, "", 200), &tb)
	if tb.TotalDebit != cost || tb.TotalCredit != cost {
		t.Fatalf("trial balance %+v, cost %d", tb, cost)
	}
	cancel(pay, id)
	if es := entries(id, 2); len(es) != 2 || es[1].debit != cost || es[1].credit != cost {
		t.Fatalf("after cancel %+v", es)
	}
	var reverses int64
	if err := f.system.QueryRow(t.Context(), `SELECT count(*) FROM accounting.entries WHERE doc_id = $1 AND reverses IS NOT NULL`, id).Scan(&reverses); err != nil || reverses != 1 {
		t.Fatalf("reversals %d %v", reverses, err)
	}

	// Accounting off: HR posts and cancels as before, and nothing is queued for the ledger.
	f.env.Products = []string{"hrm"}
	pay = f.login("pay")
	var before, after int
	if err := f.system.QueryRow(t.Context(), `SELECT count(*) FROM river_job WHERE kind = 'accounting.reconcile'`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	id, _ = post(pay)
	cancel(pay, id)
	if err := f.system.QueryRow(t.Context(), `SELECT count(*) FROM river_job WHERE kind = 'accounting.reconcile'`).Scan(&after); err != nil || after != before {
		t.Fatalf("reconcile jobs %d then %d %v", before, after, err)
	}
}
