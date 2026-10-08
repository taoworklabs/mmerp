package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// jobsFixture: company C with department P, where E1 (from 05/01) and E2 (from 10/03)
// work and E3 works in department Q. hr has hrm.hr at P; head has it tenant-wide and
// approves timesheets. Jobs are queued, and only worked once work is called.
type jobsFixture struct {
	t                 *testing.T
	pool              *pgxpool.Pool
	env               app.Env
	admin, hr, head   *client
	p, hrID, hrGrant  int64
	e1, e2, timesheet int64
	payID, payGrant   int64
}

func newJobsFixture(t *testing.T) *jobsFixture {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	f := &jobsFixture{t: t, pool: pool, env: env(t, pool, []string{"hrm"})}
	f.admin = f.login("admin")
	c := f.id(f.admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	f.p = f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"P"}`, c))
	q := f.id(f.admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"Q"}`, c))
	f.hrID = f.id(f.admin, "/api/users", `{"login":"hr","name":"Nhân sự","password":"correct horse"}`)
	f.hrGrant = f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", f.hrID), fmt.Sprintf(`{"product":"hrm","role":"hr","org_unit_id":%d}`, f.p))
	head := f.id(f.admin, "/api/users", `{"login":"head","name":"Trưởng phòng","password":"correct horse"}`)
	f.id(f.admin, fmt.Sprintf("/api/users/%d/roles", head), `{"product":"hrm","role":"hr","org_unit_id":null}`)
	f.hr, f.head = f.login("hr"), f.login("head")
	employee := func(code string, unit int64, hire string) int64 {
		return f.id(f.head, "/api/hrm/employees", fmt.Sprintf(`{"code":%q,"full_name":"Nhân viên %s","date_of_birth":null,"gender":null,"phone":null,
			"email":null,"address":null,"org_unit_id":%d,"manager_id":null,"user_login":null,"hire_date":%q,"termination_date":null}`, code, code, unit, hire))
	}
	f.e1, f.e2 = employee("E1", f.p, "2026-01-05"), employee("E2", f.p, "2026-03-10")
	employee("E3", q, "2026-01-05")
	f.ok(f.admin, "PUT", "/api/approval-rules/hrm.timesheet",
		`{"steps":[{"approver":{"kind":"role","product":"hrm","role":"hr"}}],"max_levels":3,"fallback_product":"hrm","fallback_role":"hr"}`, 204)
	f.timesheet = f.id(f.hr, "/api/hrm/timesheets", fmt.Sprintf(`{"org_unit_id":%d,"month":"2026-03"}`, f.p))
	return f
}

func (f *jobsFixture) login(user string) *client {
	c := &client{t: f.t, h: app.New(f.env, app.Modules(), nil)}
	c.cookie = c.do("POST", "/api/auth/login", fmt.Sprintf(`{"login":%q,"password":"correct horse"}`, user)).Result().Cookies()[0]
	return c
}

func (f *jobsFixture) ok(c *client, method, path, body string, status int) []byte {
	f.t.Helper()
	rec := c.do(method, path, body)
	if rec.Code != status {
		f.t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
	}
	return rec.Body.Bytes()
}

func (f *jobsFixture) id(c *client, path, body string) int64 {
	f.t.Helper()
	var out struct{ ID int64 }
	_ = json.Unmarshal(f.ok(c, "POST", path, body, 201), &out)
	return out.ID
}

// upload starts an import of an Excel file made of rows.
func (f *jobsFixture) upload(c *client, kind, params string, rows ...[]any) int64 {
	f.t.Helper()
	x := excelize.NewFile()
	for r, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, r+1)
		if err := x.SetSheetRow("Sheet1", cell, &row); err != nil {
			f.t.Fatal(err)
		}
	}
	file, err := x.WriteToBuffer()
	if err != nil {
		f.t.Fatal(err)
	}
	return f.uploadFile(c, kind, params, file.Bytes())
}

func (f *jobsFixture) uploadFile(c *client, kind, params string, file []byte) int64 {
	f.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", "import.xlsx")
	_, _ = part.Write(file)
	_ = w.WriteField("params", params)
	_ = w.Close()
	req := httptest.NewRequest("POST", "/api/imports/"+kind, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(c.cookie)
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	var out struct {
		JobID int64 `json:"job_id"`
	}
	if rec.Code != 202 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
		f.t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	return out.JobID
}

func (f *jobsFixture) params() string { return fmt.Sprintf(`{"timesheet_id":%d}`, f.timesheet) }

// work starts a job client running with the given products, as a server would.
func (f *jobsFixture) work(products []string) {
	f.t.Helper()
	e := f.env
	e.Products = products
	jobs, err := app.NewJobs(e, app.Modules())
	if err != nil {
		f.t.Fatal(err)
	}
	if err := jobs.Start(f.t.Context()); err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { _ = jobs.Stop(f.t.Context()) })
}

type job struct {
	State     string
	Rows      *int
	FileID    *string `json:"file_id"`
	RowErrors []struct {
		Row     int
		Message string
	} `json:"row_errors"`
	Code *string `json:"error_code"`
}

// wait follows a job as its requester until it ends.
func (f *jobsFixture) wait(c *client, id int64) job {
	f.t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		var j job
		if err := json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/jobs/%d", id), "", 200), &j); err != nil {
			f.t.Fatal(err)
		}
		if j.State == "completed" || j.State == "failed" {
			return j
		}
	}
	f.t.Fatalf("job %d did not end", id)
	return job{}
}

type timesheet struct {
	Status, Number string
	Version        int32
	Lines          []struct {
		EmployeeID int64 `json:"employee_id"`
		Date, Days string
	}
}

func (f *jobsFixture) get(c *client) timesheet {
	f.t.Helper()
	var t timesheet
	_ = json.Unmarshal(f.ok(c, "GET", fmt.Sprintf("/api/hrm/timesheets/%d", f.timesheet), "", 200), &t)
	return t
}

// march fills E1's working days (half a day on Saturdays) and E2's from 10/03.
func march() [][]any {
	head := []any{"Mã NV", "Họ tên"}
	e1, e2 := []any{"E1", "Nhân viên E1"}, []any{"e2", "Nhân viên E2"}
	for d := 1; d <= 31; d++ {
		head = append(head, fmt.Sprintf("%02d/03", d))
		switch time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC).Weekday() {
		case time.Sunday:
			e1, e2 = append(e1, ""), append(e2, "")
		case time.Saturday:
			e1, e2 = append(e1, "0.5"), append(e2, "")
		default:
			e1 = append(e1, 1)
			if d >= 10 {
				e2 = append(e2, "1")
			} else {
				e2 = append(e2, "")
			}
		}
	}
	return [][]any{head, e1, e2}
}

// TestTimesheetImportSampleFlow is step 8 of the sample flow: HR imports P's March
// timesheet, the job runs as HR, the timesheet goes draft → approved → posted, and a
// closed payroll period then stops its cancel.
func TestTimesheetImportSampleFlow(t *testing.T) {
	f := newJobsFixture(t)
	rows := march()
	id := f.upload(f.hr, "hrm.timesheet", f.params(), rows...)
	f.work([]string{"hrm"})
	if j := f.wait(f.hr, id); j.State != "completed" || *j.Rows != 2 {
		t.Fatalf("import: %+v", j)
	}
	ts := f.get(f.hr)
	// E1: 22 weekdays and 4 Saturdays; E2: 16 weekdays from 10/03.
	if ts.Status != "draft" || ts.Version != 2 || len(ts.Lines) != 22+4+16 {
		t.Fatalf("timesheet: %+v", ts)
	}
	// The job ran as HR: the save is HR's in the history.
	var history []struct {
		ActorName *string `json:"actor_name"`
		Action    string
	}
	_ = json.Unmarshal(f.ok(f.hr, "GET", fmt.Sprintf("/api/documents/hrm.timesheet/%d/history", f.timesheet), "", 200), &history)
	if !slices.ContainsFunc(history, func(h struct {
		ActorName *string `json:"actor_name"`
		Action    string
	}) bool {
		return h.Action == "hrm.timesheet_saved" && h.ActorName != nil && *h.ActorName == "Nhân sự"
	}) {
		t.Fatalf("history: %+v", history)
	}

	// The export is the form to fill in: imported back, it changes nothing but the version.
	var ex struct {
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(f.hr, "POST", "/api/exports/hrm.timesheet", fmt.Sprintf(`{"params":%s}`, f.params()), 202), &ex)
	exported := f.wait(f.hr, ex.JobID)
	if j := f.wait(f.hr, f.uploadFile(f.hr, "hrm.timesheet", f.params(), f.ok(f.hr, "GET", "/api/files/"+*exported.FileID, "", 200))); j.State != "completed" {
		t.Fatalf("re-import: %+v", j)
	}
	if again := f.get(f.hr); again.Version != 3 || !slices.Equal(again.Lines, ts.Lines) {
		t.Fatalf("re-imported: %+v", again)
	}

	// Draft → pending → approved by head → posted.
	f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"posted","version":3}`, 204)
	var inbox struct {
		Items []struct {
			InstanceID int64 `json:"instance_id"`
		}
	}
	_ = json.Unmarshal(f.ok(f.head, "GET", "/api/approvals/inbox", "", 200), &inbox)
	if len(inbox.Items) != 1 {
		t.Fatalf("inbox: %+v", inbox)
	}
	f.ok(f.head, "POST", fmt.Sprintf("/api/approvals/%d/approve", inbox.Items[0].InstanceID), `{"step":1}`, 204)
	if ts := f.get(f.hr); ts.Status != "posted" {
		t.Fatalf("after approval: %+v", ts)
	}

	// The export is the same layout (01/03 is a Sunday), downloadable by HR only.
	_ = json.Unmarshal(f.ok(f.hr, "POST", "/api/exports/hrm.timesheet", fmt.Sprintf(`{"params":%s}`, f.params()), 202), &ex)
	j := f.wait(f.hr, ex.JobID)
	if j.State != "completed" || j.FileID == nil {
		t.Fatalf("export: %+v", j)
	}
	x, err := excelize.OpenReader(bytes.NewReader(f.ok(f.hr, "GET", "/api/files/"+*j.FileID, "", 200)))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := x.GetRows("Sheet1", excelize.Options{RawCellValue: true})
	if len(got) != 3 || got[1][0] != "E1" || got[1][3] != "1" || got[1][8] != "0.5" || got[2][0] != "E2" || len(got[0]) != 33 {
		t.Fatalf("exported rows: %v", got)
	}
	wantCode(t, f.head.do("GET", "/api/files/"+*j.FileID, ""), 404, "not_found")
	if rec := f.head.do("GET", fmt.Sprintf("/api/jobs/%d", ex.JobID), ""); rec.Code != 404 {
		t.Fatalf("someone else's job: %d", rec.Code)
	}

	// Step 13: March closed by a posted payroll, the timesheet cannot be cancelled.
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO hrm.payroll_periods (legal_entity_id, period_start, period_end, posted_payroll_id)
		SELECT legal_entity_id, '2026-03-01', '2026-03-31', id FROM record.documents WHERE id = $1`, f.timesheet); err != nil {
		t.Fatal(err)
	}
	wantCode(t, f.hr.do("POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"cancelled","version":4}`),
		409, "payroll_period_closed")
}

// A file with bad rows names each row as Excel numbers it, with the reason in the
// requester's language, and writes nothing.
func TestImportRowErrors(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	rows := march()
	rows[1][5] = "2"                      // E1, day 4
	rows[2][3] = 1                        // E2, day 2, before 10/03
	rows = append(rows, []any{"E3", "x"}) // department Q
	rows = append(rows, []any{"", ""}, []any{"NOPE"})
	j := f.wait(f.hr, f.upload(f.hr, "hrm.timesheet", f.params(), rows...))
	if j.State != "failed" || *j.Code != "import_rows_invalid" || len(j.RowErrors) != 4 {
		t.Fatalf("%+v", j)
	}
	want := []struct {
		row  int
		text string
	}{{2, "Ngày 4"}, {3, "Ngày 2 nằm ngoài thời gian làm việc"}, {4, "E3"}, {6, "NOPE"}}
	for i, w := range want {
		if e := j.RowErrors[i]; e.Row != w.row || !strings.Contains(e.Message, w.text) {
			t.Fatalf("row error %d: %+v, want %+v", i, e, w)
		}
	}
	if ts := f.get(f.hr); ts.Version != 1 || len(ts.Lines) != 0 {
		t.Fatalf("written: %+v", ts)
	}

	// A file whose header does not match the period fails as a whole: too few columns,
	// or the days of another month with as many days.
	j = f.wait(f.hr, f.upload(f.hr, "hrm.timesheet", f.params(), []any{"Mã NV", "Họ tên", "1"}, []any{"E1", "", 1}))
	if j.State != "failed" || *j.Code != "import_columns" {
		t.Fatalf("%+v", j)
	}
	may := march()
	for d := range 31 {
		may[0][2+d] = fmt.Sprintf("%02d/05", d+1)
	}
	if j = f.wait(f.hr, f.upload(f.hr, "hrm.timesheet", f.params(), may...)); j.State != "failed" || *j.Code != "import_header" {
		t.Fatalf("%+v", j)
	}
}

// Rights are checked when the job runs: a role revoked while the job waited fails it.
func TestImportRecheckedWhenRun(t *testing.T) {
	f := newJobsFixture(t)
	id := f.upload(f.hr, "hrm.timesheet", f.params(), march()...)
	f.ok(f.admin, "DELETE", fmt.Sprintf("/api/users/%d/roles/%d", f.hrID, f.hrGrant), "", 204)
	f.work([]string{"hrm"})
	if j := f.wait(f.hr, id); j.State != "failed" || *j.Code != "forbidden" {
		t.Fatalf("%+v", j)
	}
	if ts := f.get(f.head); ts.Version != 1 {
		t.Fatalf("written: %+v", ts)
	}
}

// With HRM turned off while jobs wait, the import fails and the export still runs.
func TestJobsWithProductOff(t *testing.T) {
	f := newJobsFixture(t)
	imp := f.upload(f.hr, "hrm.timesheet", f.params(), march()...)
	var ex struct {
		JobID int64 `json:"job_id"`
	}
	_ = json.Unmarshal(f.ok(f.hr, "POST", "/api/exports/hrm.timesheet", fmt.Sprintf(`{"params":%s}`, f.params()), 202), &ex)
	f.work(nil)
	if j := f.wait(f.hr, imp); j.State != "failed" || *j.Code != "product_not_enabled" {
		t.Fatalf("import: %+v", j)
	}
	if j := f.wait(f.hr, ex.JobID); j.State != "completed" || j.FileID == nil {
		t.Fatalf("export: %+v", j)
	}
	var list []struct{ ID int64 }
	_ = json.Unmarshal(f.ok(f.hr, "GET", "/api/jobs", "", 200), &list)
	if len(list) != 2 || list[0].ID != ex.JobID {
		t.Fatalf("my jobs: %+v", list)
	}
	if _ = json.Unmarshal(f.ok(f.head, "GET", "/api/jobs", "", 200), &list); len(list) != 0 {
		t.Fatalf("head's jobs: %+v", list)
	}
}

// Opening leave balances import all or nothing, and never leave a balance below zero.
func TestLeaveBalanceImport(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	head := []any{"Mã NV", "Năm", "Số ngày", "Lý do"}
	// hr is no leave_admin.
	if j := f.wait(f.hr, f.upload(f.hr, "hrm.leave_balance", "", head, []any{"E1", 2026, 12, "Đầu năm"})); j.Code == nil || *j.Code != "forbidden" {
		t.Fatalf("%+v", j)
	}
	f.id(f.admin, "/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	balance := func(emp int64) string {
		var bs []struct {
			Year int32
			Days string
		}
		_ = json.Unmarshal(f.ok(f.admin, "GET", fmt.Sprintf("/api/hrm/employees/%d/leave-balances", emp), "", 200), &bs)
		if len(bs) == 0 {
			return "0"
		}
		return bs[0].Days
	}
	j := f.wait(f.admin, f.upload(f.admin, "hrm.leave_balance", "", head,
		[]any{"E1", 2026, 12, "Đầu năm"}, []any{"E2", 2026, "1,5", "Đầu năm"}, []any{"E1", 2026, -13, "Trừ"}, []any{"E2", "năm", 1, ""}))
	// E1 would end at -1 (row 4); row 5 has a bad year.
	if j.State != "failed" || len(j.RowErrors) != 2 || j.RowErrors[0].Row != 4 || !strings.Contains(j.RowErrors[0].Message, "âm") ||
		j.RowErrors[1].Row != 5 || !strings.Contains(j.RowErrors[1].Message, "Năm") {
		t.Fatalf("%+v", j)
	}
	if balance(f.e1) != "0" || balance(f.e2) != "0" {
		t.Fatalf("written: %s %s", balance(f.e1), balance(f.e2))
	}
	j = f.wait(f.admin, f.upload(f.admin, "hrm.leave_balance", "", head,
		// A deduction before the grant it relies on still applies: the file's end result counts.
		[]any{"E1", 2026, -2, "Trừ"}, []any{"e2", 2026, "1,5", "Đầu năm"}, []any{"E1", 2026, 12, "Đầu năm"}))
	if j.State != "completed" || balance(f.e1) != "10" || balance(f.e2) != "1.5" {
		t.Fatalf("%+v %s %s", j, balance(f.e1), balance(f.e2))
	}
}

// An administrator follows system jobs, failing ones on their own, each with the code of its
// last error and never its arguments; everyone else still sees only their own jobs.
func TestSystemJobsForAdmin(t *testing.T) {
	f := newJobsFixture(t)
	errs := func(msg string) string {
		return fmt.Sprintf(`ARRAY['{"at":"2026-10-07T00:00:00Z","attempt":1,"error":%q,"trace":""}'::jsonb]`, msg)
	}
	for _, q := range []string{
		`INSERT INTO river_job (kind, args, state, attempt, max_attempts, errors, metadata, finalized_at, queue, priority)
		 VALUES ('attachment.cleanup', '{"secret":"x@example.com"}', 'discarded', 3, 3, ` + errs("open /data/files/x: permission denied") + `, '{}', now(), 'default', 1)`,
		`INSERT INTO river_job (kind, args, state, attempt, max_attempts, errors, metadata, scheduled_at, queue, priority)
		 VALUES ('dataio.cleanup', '{}', 'retryable', 1, 25, ` + errs("mail_unreachable") + `, '{}', now() + interval '1 hour', 'default', 1)`,
		`INSERT INTO river_job (kind, args, state, attempt, max_attempts, metadata, finalized_at, queue, priority)
		 VALUES ('dataio.cleanup', '{}', 'completed', 1, 25, '{}', now(), 'default', 1)`,
	} {
		if _, err := f.pool.Exec(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	f.upload(f.hr, "hrm.timesheet", f.params(), march()...)

	type sysJob struct {
		Kind     string
		State    string
		Attempts int
		Code     *string `json:"error_code"`
	}
	list := func(c *client, query string) []sysJob {
		t.Helper()
		var out []sysJob
		body := f.ok(c, "GET", "/api/jobs"+query, "", 200)
		if bytes.Contains(body, []byte("x@example.com")) || bytes.Contains(body, []byte("permission denied")) {
			t.Fatalf("leaks: %s", body)
		}
		_ = json.Unmarshal(body, &out)
		return out
	}
	if all := list(f.admin, "?system=true"); len(all) != 3 {
		t.Fatalf("system jobs: %+v", all)
	}
	failed := list(f.admin, "?system=true&failed=true")
	if len(failed) != 2 || failed[0].Kind != "dataio.cleanup" || failed[0].State != "retrying" || *failed[0].Code != "mail_unreachable" ||
		failed[1].State != "failed" || *failed[1].Code != "internal_error" || failed[1].Attempts != 3 {
		t.Fatalf("failed system jobs: %+v", failed)
	}
	if own := list(f.admin, ""); len(own) != 0 {
		t.Fatalf("admin's own jobs: %+v", own)
	}
	wantCode(t, f.hr.do("GET", "/api/jobs?system=true", ""), 403, "forbidden")
	if own := list(f.hr, ""); len(own) != 1 || own[0].Kind != "dataio.import" {
		t.Fatalf("hr's own jobs: %+v", own)
	}
}

// The requester hears when a job of a declaring kind ends, done or failed; system jobs never tell anyone.
func TestJobNotifications(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	bad := march()
	bad[1][5] = "2"
	failed := f.upload(f.hr, "hrm.timesheet", f.params(), bad...)
	f.wait(f.hr, failed)
	done := f.upload(f.hr, "hrm.timesheet", f.params(), march()...)
	f.wait(f.hr, done)

	// The failure is told after River writes the job's end; wait for it.
	var p notificationPage
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if p = notifications(f.hr); len(p.Items) == 2 {
			break
		}
	}
	if len(p.Items) != 2 || p.Items[0].Kind != "job_completed" || *p.Items[0].JobID != done ||
		p.Items[1].Kind != "job_failed" || *p.Items[1].JobID != failed || p.Items[0].RecordType != nil || p.Items[0].ActorName != nil {
		t.Fatalf("hr: %+v", p.Items)
	}
	var n int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM notification.notifications WHERE user_id <> $1`, f.hrID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("%d others told %v", n, err)
	}
}
