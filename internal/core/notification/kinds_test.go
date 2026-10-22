package notification

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"mime"
	"net"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/numbering"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/pgtest"
)

// A product's own kind, with its email texts in the product's translations, reaches the
// inbox and the mail like a core kind; a kind nobody registered is refused.
func TestProductKinds(t *testing.T) {
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	check(platform.LoadTranslations(fstest.MapFS{
		"en.json": {Data: []byte(`{"test.pinged.email.subject":"You were pinged","test.pinged.email.body":"Someone pinged you."}`)},
		"vi.json": {Data: []byte(`{"test.pinged.email.subject":"Bạn được gọi","test.pinged.email.body":"Có người gọi bạn."}`)},
	}))
	pool := pgtest.New(t)
	jobs, err := river.NewClient(riverpgxv5.New(pgtest.System(t, pool)), &river.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	check(err)
	ctx := platform.WithJobs(platform.WithKeyring(platform.WithProducts(platform.WithDB(t.Context(), pool), []string{"test"}), pgtest.Keyring()), jobs)
	set := setting.NewService()
	ids := iam.NewService(iam.Deps{Setting: set, Audit: audit.NewService()})
	rec := record.NewService(record.Deps{IAM: ids, Numbering: numbering.NewService(numbering.Deps{}), Audit: audit.NewService()})
	ids.RegisterRoles("test", map[string][]string{"member": {"test.all"}})
	rec.Register(record.Type{Code: "test.doc", Product: "test", Kind: record.Document, NumberPrefix: "T",
		Perms:    record.Perms{View: "test.all", Edit: "test.all", ViewFiles: "test.all", Attach: "test.all"},
		Grant:    func(context.Context, int64, record.Action) (bool, error) { return true, nil },
		OnDelete: func(context.Context, int64) error { return nil }})
	s := NewService(Deps{Record: rec, IAM: ids, Audit: audit.NewService(), Setting: set})
	s.RegisterKind("test.pinged", true)

	adminID, err := ids.CreateAdmin(ctx, "admin", "Admin", "long enough")
	check(err)
	admin := platform.WithActor(ctx, adminID)
	unit, err := ids.CreateOrgUnit(admin, iam.OrgUnitInput{Kind: "company", Name: "C"})
	check(err)
	user, err := ids.CreateUser(admin, "user", "User", "long enough")
	check(err)
	email := "user@example.com"
	check(ids.SetEmail(admin, user, &email))
	smtp := fakeSMTP(t)
	pw := "pw"
	check(s.SaveMailServer(admin, MailServerInput{Host: "127.0.0.1", Port: int32(smtp.port), Security: "none", Password: &pw,
		FromAddress: "mmerp@example.com", BaseURL: "https://erp.example.com"}))
	d, err := rec.Create(admin, "test.doc", record.Header{Date: "2026-03-10", OrgUnitID: unit})
	check(err)

	check(platform.InTx(admin, func(ctx context.Context) error { return s.Send(ctx, "test.pinged", d.Ref, []int64{user}) }))
	as := platform.WithActor(ctx, user)
	page, err := s.List(as, nil)
	check(err)
	if len(page.Items) != 1 || page.Items[0].Kind != "test.pinged" || page.Items[0].RecordID == nil {
		t.Fatalf("inbox = %+v", page.Items)
	}
	check(s.sendEmail(ctx, emailArgs{Notification: page.Items[0].ID}))
	got := <-smtp.mails
	_, subject, _ := strings.Cut(got, "Subject: ")
	subject, _, _ = strings.Cut(subject, "\r\n")
	if subject, _ = new(mime.WordDecoder).DecodeHeader(subject); !strings.Contains(got, "RCPT TO:<user@example.com>") || subject != "Bạn được gọi" {
		t.Fatalf("mail = %s", got)
	}

	if err := s.Send(admin, "test.unknown", d.Ref, []int64{user}); err == nil {
		t.Fatal("an unregistered kind was sent")
	}
	if n, err := s.Unread(as); err != nil || n != 1 {
		t.Fatalf("unread = %d %v", n, err)
	}
	var queued int
	check(pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'notification.email'`).Scan(&queued))
	if queued != 1 {
		t.Fatalf("%d email jobs", queued)
	}
}

type smtpFake struct {
	port  int
	mails chan string
}

// fakeSMTP accepts every command and passes on each message's envelope and data.
func fakeSMTP(t *testing.T) *smtpFake {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	f := &smtpFake{port: l.Addr().(*net.TCPAddr).Port, mails: make(chan string, 4)}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				r := bufio.NewReader(c)
				say := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
				say("220 fake")
				var b strings.Builder
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(line))
					switch cmd {
					case "DATA":
						say("354 go")
						for {
							l, err := r.ReadString('\n')
							if err != nil || l == ".\r\n" {
								break
							}
							b.WriteString(l)
						}
						f.mails <- b.String()
						b.Reset()
						say("250 ok")
					case "QUIT":
						say("221 bye")
						return
					default:
						b.WriteString(line)
						say("250 ok")
					}
				}
			}()
		}
	}()
	return f
}
