package app_test

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// smtpServer is a mail server on a local port: it keeps every message it receives, and
// refuses every login while refuse is set. Enough SMTP for net/smtp, no more.
type smtpServer struct {
	port   int
	mu     sync.Mutex
	mails  []sentMail
	refuse bool
}

type sentMail struct {
	user, pass string
	to         []string
	data       string
}

func newSMTP(t *testing.T) *smtpServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	s := &smtpServer{port: l.Addr().(*net.TCPAddr).Port}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go s.serve(c)
		}
	}()
	return s
}

func (s *smtpServer) serve(c net.Conn) {
	defer func() { _ = c.Close() }()
	r := bufio.NewReader(c)
	say := func(line string) { _, _ = fmt.Fprintf(c, "%s\r\n", line) }
	var m sentMail
	say("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		cmd := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		switch {
		case cmd == "EHLO" || cmd == "HELO":
			say("250-fake")
			say("250 AUTH PLAIN")
		case strings.HasPrefix(strings.ToUpper(line), "AUTH PLAIN "):
			raw, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(line[len("AUTH PLAIN "):]))
			parts := strings.Split(string(raw), "\x00")
			s.mu.Lock()
			refuse := s.refuse
			s.mu.Unlock()
			if refuse || len(parts) != 3 {
				say("535 5.7.8 authentication failed")
				continue
			}
			m.user, m.pass = parts[1], parts[2]
			say("235 2.7.0 ok")
		case cmd == "MAIL":
			say("250 ok")
		case cmd == "RCPT":
			m.to = append(m.to, strings.Trim(strings.SplitN(line, ":", 2)[1], "<> "))
			say("250 ok")
		case cmd == "DATA":
			say("354 go ahead")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			m.data = b.String()
			s.mu.Lock()
			s.mails = append(s.mails, m)
			s.mu.Unlock()
			m = sentMail{user: m.user, pass: m.pass}
			say("250 ok")
		case cmd == "QUIT":
			say("221 bye")
			return
		default:
			say("250 ok")
		}
	}
}

func (s *smtpServer) received() []sentMail {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sentMail(nil), s.mails...)
}

// mailServer is the configuration pointing at port.
func mailServer(port int, password string) string {
	return fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"security":"none","username":"mmerp","password":%s,
		"from_address":"mmerp@example.com","base_url":"https://erp.example.com"}`, port, password)
}

// systemJob waits until a system job of kind shows in the administrator's list in state.
func (f *jobsFixture) systemJob(kind, state string) (code string) {
	f.t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		var jobs []struct {
			Kind, State string
			Code        *string `json:"error_code"`
		}
		_ = json.Unmarshal(f.ok(f.admin, "GET", "/api/jobs?system=true", "", 200), &jobs)
		for _, j := range jobs {
			if j.Kind == kind && j.State == state {
				if j.Code != nil {
					return *j.Code
				}
				return ""
			}
		}
	}
	f.t.Fatalf("no %s job %s", kind, state)
	return ""
}

// An administrator sets users' addresses and the company's mail server, whose password is
// sealed and never read back, and checks it with a test message to themselves.
func TestMailServer(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	smtp := newSMTP(t)

	f.ok(f.admin, "PUT", "/api/users/1/email", `{"email":"admin@example.com"}`, 204)
	wantCode(t, f.admin.do("PUT", "/api/users/1/email", `{"email":"Admin <admin@example.com>"}`), 422, "invalid_request")
	if body := f.ok(f.admin, "GET", "/api/users/1", "", 200); !strings.Contains(string(body), `"email":"admin@example.com"`) {
		t.Fatalf("user = %s", body)
	}
	wantCode(t, f.hr.do("PUT", "/api/users/1/email", `{"email":"x@example.com"}`), 403, "forbidden")

	wantCode(t, f.admin.do("GET", "/api/mail-server", ""), 404, "not_found")
	wantCode(t, f.admin.do("POST", "/api/mail-server/test", ""), 409, "mail_server_missing")
	wantCode(t, f.hr.do("PUT", "/api/mail-server", mailServer(smtp.port, `"s3cret pass"`)), 403, "forbidden")
	wantCode(t, f.admin.do("PUT", "/api/mail-server", strings.Replace(mailServer(smtp.port, `"s3cret pass"`), "https://erp.example.com", "erp", 1)), 422, "invalid_request")
	f.ok(f.admin, "PUT", "/api/mail-server", mailServer(smtp.port, `"s3cret pass"`), 204)
	// Saved again without a password: the stored one stays.
	f.ok(f.admin, "PUT", "/api/mail-server", mailServer(smtp.port, `null`), 204)
	body := f.ok(f.admin, "GET", "/api/mail-server", "", 200)
	if strings.Contains(string(body), "s3cret") || !strings.Contains(string(body), `"password_set":true`) {
		t.Fatalf("mail server = %s", body)
	}
	wantCode(t, f.hr.do("GET", "/api/mail-server", ""), 403, "forbidden")
	var plain int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM notification.mail_server WHERE position('s3cret'::bytea IN password) > 0`).Scan(&plain); err != nil || plain != 0 {
		t.Fatalf("password in the clear: %d %v", plain, err)
	}
	if a := auditActions(t, f.pool); strings.Count(a, "notification.mail_server_saved") != 2 {
		t.Fatalf("audit = %s", a)
	}
	var details string
	_ = f.pool.QueryRow(t.Context(), `SELECT string_agg(details::text, ' ') FROM audit.log`).Scan(&details)
	if strings.Contains(details, "s3cret") {
		t.Fatalf("password audited: %s", details)
	}

	f.ok(f.admin, "POST", "/api/mail-server/test", "", 202)
	f.systemJob("notification.email", "completed")
	mails := smtp.received()
	if len(mails) != 1 || mails[0].to[0] != "admin@example.com" || mails[0].user != "mmerp" || mails[0].pass != "s3cret pass" ||
		!strings.Contains(mails[0].data, "https://erp.example.com") {
		t.Fatalf("mails = %+v", mails)
	}
	if n := unread(f.admin); n != 0 {
		t.Fatalf("a mail job told someone: %d", n)
	}
	wantCode(t, f.hr.do("POST", "/api/mail-server/test", ""), 403, "forbidden")

	// A refused login cancels at once with its own code.
	smtp.mu.Lock()
	smtp.refuse = true
	smtp.mu.Unlock()
	f.ok(f.admin, "POST", "/api/mail-server/test", "", 202)
	if code := f.systemJob("notification.email", "failed"); code != "mail_auth_failed" {
		t.Fatalf("refused login = %s", code)
	}

	// The stored password never follows the settings to another server.
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	closed := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	wantCode(t, f.admin.do("PUT", "/api/mail-server", mailServer(closed, `null`)), 422, "invalid_request")

	// An unreachable server is retried.
	f.ok(f.admin, "PUT", "/api/mail-server", mailServer(closed, `"s3cret pass"`), 204)
	f.ok(f.admin, "POST", "/api/mail-server/test", "", 202)
	if code := f.systemJob("notification.email", "retrying"); code != "mail_unreachable" {
		t.Fatalf("unreachable = %s", code)
	}
	if n := unread(f.admin); n != 0 {
		t.Fatalf("a failing mail job told someone: %d", n)
	}

	f.ok(f.admin, "DELETE", "/api/mail-server", "", 204)
	wantCode(t, f.admin.do("GET", "/api/mail-server", ""), 404, "not_found")
}

// The test message goes to the administrator who asks; without an address there is nowhere to send it.
func TestTestMailNeedsAnAddress(t *testing.T) {
	f := newJobsFixture(t)
	f.ok(f.admin, "PUT", "/api/mail-server", mailServer(25, `"x"`), 204)
	wantCode(t, f.admin.do("POST", "/api/mail-server/test", ""), 422, "user_email_missing")
}

// Approval and mention notifications are emailed too, in the recipient's language and with
// only a link; a recipient without an address gets none, and a dead server stops nothing.
func TestNotificationEmails(t *testing.T) {
	f := newJobsFixture(t)
	f.work([]string{"hrm"})
	smtp := newSMTP(t)
	f.ok(f.admin, "PUT", "/api/mail-server", mailServer(smtp.port, `"pw"`), 204)
	var headID int64
	_ = f.pool.QueryRow(t.Context(), `SELECT id FROM iam.users WHERE login = 'head'`).Scan(&headID)
	f.ok(f.admin, "PUT", fmt.Sprintf("/api/users/%d/email", headID), `{"email":"head@example.com"}`, 204)
	// head has no language of their own: the tenant's applies.
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO setting.values (key, value) VALUES ('setting.locale', 'en')`); err != nil {
		t.Fatal(err)
	}
	number := f.get(f.hr).Number

	f.ok(f.hr, "POST", fmt.Sprintf("/api/documents/hrm.timesheet/%d/transitions", f.timesheet), `{"to":"posted","version":1}`, 204)
	f.systemJob("notification.email", "completed")
	mails := smtp.received()
	if len(mails) != 1 || mails[0].to[0] != "head@example.com" || !strings.Contains(mails[0].data, "waiting for your approval") ||
		!strings.Contains(mails[0].data, "https://erp.example.com/notifications") || strings.Contains(mails[0].data, number) ||
		strings.Contains(mails[0].data, "Nhân sự") {
		t.Fatalf("mails = %+v", mails)
	}

	// hr, told of the approval, has no address: no email for them.
	f.approveAll(204)
	if n := unread(f.hr); n != 1 {
		t.Fatalf("hr unread = %d", n)
	}
	var queued int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM river_job WHERE kind = 'notification.email'`).Scan(&queued); err != nil || queued != 1 {
		t.Fatalf("%d email jobs %v", queued, err)
	}

	// The server goes away: the document still moves, the notification is still there,
	// and the email waits to be retried where the administrator sees it.
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	closed := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	f.ok(f.admin, "PUT", "/api/mail-server", mailServer(closed, `"pw"`), 204)
	comments := fmt.Sprintf("/api/records/hrm.timesheet/%d/comments", f.timesheet)
	f.ok(f.hr, "POST", comments, `{"body":"@head xem lại giúp"}`, 204)
	if n := unread(f.head); n != 2 {
		t.Fatalf("head unread = %d", n)
	}
	if code := f.systemJob("notification.email", "retrying"); code != "mail_unreachable" {
		t.Fatalf("code = %s", code)
	}
	if n := unread(f.hr) + unread(f.admin); n != 1 {
		t.Fatalf("a mail job told someone: %d", n)
	}
}
