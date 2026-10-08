package app_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/taoworklabs/mmerp/internal/app"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

type attachmentList struct {
	Hidden         bool
	AllowedActions []string `json:"allowed_actions"`
	Items          []struct {
		ID             int64
		Name           string
		Size           int64
		ContentType    string   `json:"content_type"`
		UploadedByName string   `json:"uploaded_by_name"`
		AllowedActions []string `json:"allowed_actions"`
	}
}

func upload(c *client, path, name string, content []byte) *httptest.ResponseRecorder {
	c.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", name)
	_, _ = part.Write(content)
	_ = w.Close()
	req := httptest.NewRequest("POST", path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	// As a server receives it; httptest only sets the field.
	req.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))
	req.AddCookie(c.cookie)
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec
}

func listAttachments(c *client, path string) attachmentList {
	c.t.Helper()
	var l attachmentList
	rec := c.do("GET", path, "")
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &l) != nil {
		c.t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body)
	}
	return l
}

func zipWith(t *testing.T, name string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	if _, err := z.Create(name); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

var pdf = []byte("%PDF-1.4\n% giấy khám bệnh\n")

// Attachments of a leave request: seen by whoever sees the request, added by whoever may
// change it, read-only once cancelled or when the product is off, invisible to others.
func TestLeaveAttachments(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	on := env(t, pool, []string{"hrm"})
	c := &client{t: t, h: app.New(on, app.Modules(), nil)}
	c.cookie = c.do("POST", "/api/auth/login", `{"login":"admin","password":"correct horse"}`).Result().Cookies()[0]
	ok := func(c *client, method, path, body string, status int) []byte {
		t.Helper()
		rec := c.do(method, path, body)
		if rec.Code != status {
			t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
		return rec.Body.Bytes()
	}
	id := func(path, body string) int64 {
		t.Helper()
		var out struct{ ID int64 }
		_ = json.Unmarshal(ok(c, "POST", path, body, 201), &out)
		return out.ID
	}
	company := id("/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	dept := id("/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	id("/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	id("/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	emp := id("/api/hrm/employees", fmt.Sprintf(`{"code":"E1","full_name":"An","date_of_birth":null,"gender":null,"phone":null,
		"email":null,"address":null,"org_unit_id":%d,"manager_id":null,"user_login":"admin","hire_date":"2026-01-05","termination_date":null}`, dept))
	annual := id("/api/hrm/leave-types", `{"name":"Phép năm","deducts_balance":true,"paid":true,"active":true}`)
	ok(c, "POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", emp), `{"delta":"12","reason":"đầu năm"}`, 204)
	lv := id("/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":"2026-03-10","end_date":"2026-03-11","days":"1.5","reason":null}`, annual))
	files := fmt.Sprintf("/api/records/hrm.leave_request/%d/attachments", lv)
	docPath := fmt.Sprintf("/api/documents/hrm.leave_request/%d", lv)

	if rec := c.do("GET", files, ""); !strings.Contains(rec.Body.String(), `"max_mb":20`) || !strings.Contains(rec.Body.String(), `".pdf"`) {
		t.Fatalf("limits = %s", rec.Body)
	}
	if l := listAttachments(c, files); l.Hidden || len(l.Items) != 0 || !slices.Equal(l.AllowedActions, []string{"attach"}) {
		t.Fatalf("empty = %+v", l)
	}
	if rec := upload(c, files, "giay-kham.pdf", pdf); rec.Code != 204 {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	// The type comes from the content: a renamed text file is refused, a Word file is not.
	wantCode(t, upload(c, files, "fake.pdf", []byte("just text")), 422, "file_type_not_allowed")
	wantCode(t, upload(c, files, "fake.docx", zipWith(t, "other.xml")), 422, "file_type_not_allowed")
	if rec := upload(c, files, "don.docx", zipWith(t, "word/document.xml")); rec.Code != 204 {
		t.Fatalf("docx = %d %s", rec.Code, rec.Body)
	}
	big := append(bytes.Clone(pdf), make([]byte, 20<<20)...)
	wantCode(t, upload(c, files, "big.pdf", big), 413, "file_too_large")
	wantCode(t, upload(c, files, "huge.pdf", append(bytes.Clone(pdf), make([]byte, 30<<20)...)), 413, "file_too_large")
	// The refused body is read to the end, so the browser that sent it gets the answer
	// rather than a reset connection.
	big25 := append(bytes.Clone(pdf), make([]byte, 25<<20)...)
	counted := &countingReader{r: bytes.NewReader(big25)}
	req25 := httptest.NewRequest("POST", files, counted)
	req25.ContentLength = int64(len(big25))
	req25.Header.Set("Content-Length", strconv.Itoa(len(big25)))
	req25.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	req25.AddCookie(c.cookie)
	rec25 := httptest.NewRecorder()
	c.h.ServeHTTP(rec25, req25)
	wantCode(t, rec25, 413, "file_too_large")
	if counted.n != len(big25) {
		t.Fatalf("read %d of %d bytes before answering", counted.n, len(big25))
	}
	// An import has its own limit, said the same way.
	wantCode(t, upload(c, "/api/imports/hrm.timesheet", "big.xlsx", make([]byte, 6<<20)), 413, "file_too_large")
	// A body that hides its length is cut by the app's cap before the handler sees a file.
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "hidden.pdf")
	_, _ = part.Write(append(bytes.Clone(pdf), make([]byte, 40<<20)...))
	_ = mw.Close()
	req := httptest.NewRequest("POST", files, io.MultiReader(&body))
	req.ContentLength = -1
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(c.cookie)
	hidden := httptest.NewRecorder()
	c.h.ServeHTTP(hidden, req)
	if hidden.Code < 400 || hidden.Code >= 500 || strings.Contains(hidden.Body.String(), "file_too_large") {
		t.Fatalf("unbounded body = %d %s", hidden.Code, hidden.Body)
	}

	l := listAttachments(c, files)
	if len(l.Items) != 2 || l.Items[0].Name != "giay-kham.pdf" || l.Items[0].Size != int64(len(pdf)) || l.Items[0].ContentType != "application/pdf" ||
		l.Items[0].UploadedByName != "Admin" || l.Items[1].ContentType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Fatalf("list = %+v", l)
	}
	download := fmt.Sprintf("/api/attachments/%d", l.Items[0].ID)
	if rec := c.do("GET", download, ""); rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), pdf) ||
		rec.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(rec.Header().Get("Content-Disposition"), "giay-kham.pdf") {
		t.Fatalf("download = %d %v %q", rec.Code, rec.Header(), rec.Body)
	}
	if a := auditActions(t, pool); !strings.Contains(a, "attachment.added:1") || !strings.Contains(a, "attachment.downloaded:1") {
		t.Fatalf("audit = %s", a)
	}
	if h := string(ok(c, "GET", docPath+"/history", "", 200)); !strings.Contains(h, "attachment.added") || strings.Contains(h, "attachment.downloaded") {
		t.Fatalf("history = %s", h)
	}

	// Someone who cannot see the request learns nothing, even with the right ids.
	id("/api/users", `{"login":"stranger","name":"Stranger","password":"correct horse"}`)
	stranger := &client{t: t, h: c.h}
	stranger.cookie = stranger.do("POST", "/api/auth/login", `{"login":"stranger","password":"correct horse"}`).Result().Cookies()[0]
	wantCode(t, stranger.do("GET", files, ""), 404, "not_found")
	wantCode(t, stranger.do("GET", download, ""), 404, "not_found")
	wantCode(t, upload(stranger, files, "x.pdf", pdf), 404, "not_found")
	wantCode(t, c.do("GET", "/api/records/hrm.nothing/1/attachments", ""), 404, "not_found")
	wantCode(t, c.do("GET", "/api/attachments/999999", ""), 404, "not_found")

	// Posted: the lock date does not hold attachments back.
	ok(c, "POST", docPath+"/transitions", `{"to":"posted","version":1}`, 204)
	ok(c, "PUT", fmt.Sprintf("/api/period-locks/%d", company), `{"locked_until":"2026-03-31"}`, 204)
	if rec := upload(c, files, "sau-khoa.pdf", pdf); rec.Code != 204 {
		t.Fatalf("upload in a locked period = %d %s", rec.Code, rec.Body)
	}
	ok(c, "PUT", fmt.Sprintf("/api/period-locks/%d", company), `{"locked_until":null}`, 204)

	// A disabled product still reads, never adds.
	offEnv := env(t, pool, nil)
	offEnv.Files = on.Files
	off := &client{t: t, h: app.New(offEnv, app.Modules(), nil), cookie: c.cookie}
	if l := listAttachments(off, files); len(l.Items) != 3 || len(l.AllowedActions) != 0 {
		t.Fatalf("disabled product = %+v", l)
	}
	if rec := off.do("GET", download, ""); rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), pdf) {
		t.Fatalf("download with the product off = %d", rec.Code)
	}
	wantCode(t, upload(off, files, "x.pdf", pdf), 403, "product_not_enabled")

	// Cancelled: still read, no more added.
	ok(c, "POST", docPath+"/transitions", `{"to":"cancelled","version":2}`, 204)
	if l := listAttachments(c, files); len(l.Items) != 3 || len(l.AllowedActions) != 0 {
		t.Fatalf("cancelled = %+v", l)
	}
	if rec := c.do("GET", download, ""); rec.Code != 200 {
		t.Fatalf("download cancelled = %d", rec.Code)
	}
	wantCode(t, upload(c, files, "x.pdf", pdf), 409, "document_not_editable")
}

// Removing attachments: the uploader removes their own, attach removes anyone's, and
// nothing is removed once the request is cancelled or the product is off.
func TestRemoveLeaveAttachments(t *testing.T) {
	pool := pgtest.New(t)
	if err := app.CreateAdmin(t.Context(), pool, "admin", "Admin", "correct horse"); err != nil {
		t.Fatal(err)
	}
	on := env(t, pool, []string{"hrm"})
	h := app.New(on, app.Modules(), nil)
	login := func(name string) *client {
		c := &client{t: t, h: h}
		c.cookie = c.do("POST", "/api/auth/login", fmt.Sprintf(`{"login":%q,"password":"correct horse"}`, name)).Result().Cookies()[0]
		return c
	}
	admin := login("admin")
	ok := func(c *client, method, path, body string, status int) []byte {
		t.Helper()
		rec := c.do(method, path, body)
		if rec.Code != status {
			t.Fatalf("%s %s = %d %s", method, path, rec.Code, rec.Body)
		}
		return rec.Body.Bytes()
	}
	id := func(c *client, path, body string) int64 {
		t.Helper()
		var out struct{ ID int64 }
		_ = json.Unmarshal(ok(c, "POST", path, body, 201), &out)
		return out.ID
	}
	added := func(c *client, path, name string) {
		t.Helper()
		if rec := upload(c, path, name, pdf); rec.Code != 204 {
			t.Fatalf("upload %s = %d %s", name, rec.Code, rec.Body)
		}
	}

	company := id(admin, "/api/org-units", `{"parent_id":null,"kind":"company","name":"C"}`)
	dept := id(admin, "/api/org-units", fmt.Sprintf(`{"parent_id":%d,"kind":"department","name":"A"}`, company))
	id(admin, "/api/users/1/roles", `{"product":"hrm","role":"hr","org_unit_id":null}`)
	id(admin, "/api/users/1/roles", `{"product":"hrm","role":"leave_admin","org_unit_id":null}`)
	for _, u := range []string{"boss", "staff", "stranger"} {
		id(admin, "/api/users", fmt.Sprintf(`{"login":%q,"name":%q,"password":"correct horse"}`, u, u))
	}
	employee := func(code, login string, manager string) int64 {
		return id(admin, "/api/hrm/employees", fmt.Sprintf(`{"code":%q,"full_name":%q,"date_of_birth":null,"gender":null,"phone":null,
			"email":null,"address":null,"org_unit_id":%d,"manager_id":%s,"user_login":%q,"hire_date":"2026-01-05","termination_date":null}`, code, login, dept, manager, login))
	}
	boss := employee("E1", "boss", "null")
	staff := employee("E2", "staff", fmt.Sprint(boss))
	annual := id(admin, "/api/hrm/leave-types", `{"name":"Phép năm","deducts_balance":true,"paid":true,"active":true}`)
	ok(admin, "POST", fmt.Sprintf("/api/hrm/employees/%d/leave-balances/2026/adjustments", staff), `{"delta":"12","reason":"đầu năm"}`, 204)

	staffC, bossC, stranger := login("staff"), login("boss"), login("stranger")
	lv := id(staffC, "/api/hrm/leaves", fmt.Sprintf(`{"leave_type_id":%d,"start_date":"2026-03-10","end_date":"2026-03-11","days":"1.5","reason":null}`, annual))
	files := fmt.Sprintf("/api/records/hrm.leave_request/%d/attachments", lv)
	docPath := fmt.Sprintf("/api/documents/hrm.leave_request/%d", lv)

	// boss (user 2) uploads while HR, then keeps only what a manager has: seeing the request.
	grant := id(admin, "/api/users/2/roles", fmt.Sprintf(`{"product":"hrm","role":"hr","org_unit_id":%d}`, dept))
	added(bossC, files, "boss.pdf")
	added(admin, files, "admin.pdf")
	added(staffC, files, "staff.pdf")
	ok(admin, "DELETE", fmt.Sprintf("/api/users/2/roles/%d", grant), "", 204)

	l := listAttachments(bossC, files)
	if len(l.AllowedActions) != 0 || len(l.Items) != 3 || !slices.Equal(l.Items[0].AllowedActions, []string{"delete"}) || len(l.Items[1].AllowedActions) != 0 {
		t.Fatalf("boss sees %+v", l)
	}
	bossFile, adminFile, staffFile := l.Items[0].ID, l.Items[1].ID, l.Items[2].ID
	wantCode(t, bossC.do("DELETE", fmt.Sprintf("/api/attachments/%d", adminFile), ""), 403, "forbidden")
	wantCode(t, stranger.do("DELETE", fmt.Sprintf("/api/attachments/%d", adminFile), ""), 404, "not_found")
	ok(bossC, "DELETE", fmt.Sprintf("/api/attachments/%d", bossFile), "", 204)
	// The submitter may change the request, so removes anyone's.
	if l := listAttachments(staffC, files); len(l.Items) != 2 || !slices.Equal(l.Items[0].AllowedActions, []string{"delete"}) {
		t.Fatalf("staff sees %+v", l)
	}
	ok(staffC, "DELETE", fmt.Sprintf("/api/attachments/%d", adminFile), "", 204)
	wantCode(t, staffC.do("DELETE", fmt.Sprintf("/api/attachments/%d", adminFile), ""), 404, "not_found")
	if h := string(ok(admin, "GET", docPath+"/history", "", 200)); strings.Count(h, "attachment.removed") != 2 {
		t.Fatalf("history = %s", h)
	}

	offEnv := env(t, pool, nil)
	offEnv.Files = on.Files
	off := &client{t: t, h: app.New(offEnv, app.Modules(), nil), cookie: admin.cookie}
	if l := listAttachments(off, files); len(l.Items[0].AllowedActions) != 0 {
		t.Fatalf("disabled product = %+v", l)
	}
	wantCode(t, off.do("DELETE", fmt.Sprintf("/api/attachments/%d", staffFile), ""), 403, "product_not_enabled")

	ok(admin, "POST", docPath+"/transitions", `{"to":"posted","version":1}`, 204)
	ok(admin, "POST", docPath+"/transitions", `{"to":"cancelled","version":2}`, 204)
	if l := listAttachments(staffC, files); len(l.Items) != 1 || len(l.Items[0].AllowedActions) != 0 {
		t.Fatalf("cancelled = %+v", l)
	}
	wantCode(t, staffC.do("DELETE", fmt.Sprintf("/api/attachments/%d", staffFile), ""), 409, "document_not_editable")
}

// A contract's scan may show the salary: its attachments need hrm.salary.view on top of
// seeing the contract, asked again at every download, and are added after posting.
func TestContractAttachmentsNeedSalaryView(t *testing.T) {
	f := newJobsFixture(t)
	pay := f.payrollSetup()
	var contract int64
	if err := f.pool.QueryRow(t.Context(), `SELECT min(id) FROM hrm.contracts`).Scan(&contract); err != nil {
		t.Fatal(err)
	}
	files := fmt.Sprintf("/api/records/hrm.contract/%d/attachments", contract)

	if rec := upload(pay, files, "hop-dong-da-ky.pdf", pdf); rec.Code != 204 {
		t.Fatalf("upload to a posted contract = %d %s", rec.Code, rec.Body)
	}
	l := listAttachments(pay, files)
	if len(l.Items) != 1 || !slices.Equal(l.AllowedActions, []string{"attach"}) {
		t.Fatalf("pay sees %+v", l)
	}
	download := fmt.Sprintf("/api/attachments/%d", l.Items[0].ID)
	if rec := pay.do("GET", download, ""); rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), pdf) {
		t.Fatalf("download = %d", rec.Code)
	}

	// hr sees the contract, not its files: told they are hidden, never what they are.
	f.ok(f.hr, "GET", fmt.Sprintf("/api/hrm/contracts/%d", contract), "", 200)
	if l := listAttachments(f.hr, files); !l.Hidden || len(l.Items) != 0 || len(l.AllowedActions) != 0 {
		t.Fatalf("hr sees %+v", l)
	}
	history := fmt.Sprintf("/api/documents/hrm.contract/%d/history", contract)
	if h := string(f.ok(f.hr, "GET", history, "", 200)); strings.Contains(h, "attachment.") || strings.Contains(h, "hop-dong-da-ky") {
		t.Fatalf("hr's history shows the file: %s", h)
	}
	if h := string(f.ok(pay, "GET", history, "", 200)); !strings.Contains(h, "attachment.added") {
		t.Fatalf("pay's history = %s", h)
	}
	wantCode(t, f.hr.do("GET", download, ""), 404, "not_found")
	wantCode(t, upload(f.hr, files, "x.pdf", pdf), 404, "not_found")

	// Revoked after the upload: the next download is refused.
	f.ok(f.admin, "DELETE", fmt.Sprintf("/api/users/%d/roles/%d", f.payID, f.payGrant), "", 204)
	wantCode(t, pay.do("GET", download, ""), 404, "not_found")
	if l := listAttachments(pay, files); !l.Hidden || len(l.Items) != 0 {
		t.Fatalf("revoked pay sees %+v", l)
	}
}

type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}
