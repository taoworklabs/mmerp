package notification

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/notification/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/setting"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// passwordAAD names the sealed column, so its value cannot be moved to another one.
const passwordAAD = "notification.mail_server.password"

var (
	errMailServerMissing = &platform.Error{Status: http.StatusConflict, Code: "mail_server_missing"}
	errNoAddress         = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "user_email_missing"}
	// Retried: the server or the network may come back.
	errMailUnreachable = &platform.Error{Status: http.StatusServiceUnavailable, Code: "mail_unreachable"}
	// Not retried: the configuration or the address has to change first.
	errMailAuth     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "mail_auth_failed"}
	errMailRejected = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "mail_rejected"}
)

func invalid(field string) error {
	return &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{field}}}
}

// MailServer returns the tenant's mail server; not found when there is none.
func (s *Service) MailServer(ctx context.Context) (MailServer, error) {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManageMail); err != nil {
		return MailServer{}, err
	}
	m, err := store.New(platform.DBFrom(ctx)).GetMailServer(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return MailServer{}, platform.ErrNotFound
	}
	return MailServer{Host: m.Host, Port: m.Port, Security: m.Security, Username: m.Username, PasswordSet: m.Password != nil,
		FromAddress: m.FromAddress, BaseURL: m.BaseUrl}, err
}

func (s *Service) SaveMailServer(ctx context.Context, in MailServerInput) error {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManageMail); err != nil {
		return err
	}
	if a, err := mail.ParseAddress(in.FromAddress); err != nil || a.Address != in.FromAddress {
		return invalid("from_address")
	}
	if u, err := url.Parse(in.BaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return invalid("base_url")
	}
	var password []byte
	if in.Password != nil {
		password = platform.Encrypt(ctx, passwordAAD, []byte(*in.Password))
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		q := store.New(platform.DBFrom(ctx))
		// The stored password only goes on to the server it was given for: pointing it
		// anywhere else would hand it to whoever runs that host.
		if cur, err := q.GetMailServer(ctx); err == nil && in.Password == nil && cur.Password != nil &&
			(cur.Host != in.Host || cur.Port != in.Port || cur.Security != in.Security || cur.Username != in.Username) {
			return invalid("password")
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := q.SaveMailServer(ctx, store.SaveMailServerParams{
			Host: in.Host, Port: in.Port, Security: in.Security, Username: in.Username, Password: password,
			FromAddress: in.FromAddress, BaseUrl: strings.TrimRight(in.BaseURL, "/"),
		}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "notification.mail_server_saved", map[string]any{
			"host": in.Host, "port": in.Port, "security": in.Security, "password_changed": in.Password != nil})
	})
}

// DeleteMailServer stops email; notifications in the app go on.
func (s *Service) DeleteMailServer(ctx context.Context) error {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManageMail); err != nil {
		return err
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		n, err := store.New(platform.DBFrom(ctx)).DeleteMailServer(ctx)
		if err != nil || n == 0 {
			return platform.OrErr(err, platform.ErrNotFound)
		}
		return s.d.Audit.Record(ctx, "notification.mail_server_deleted", nil)
	})
}

// SendTestMail queues a test message to the actor; its outcome shows among the system jobs.
func (s *Service) SendTestMail(ctx context.Context) error {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManageMail); err != nil {
		return err
	}
	q := store.New(platform.DBFrom(ctx))
	if _, err := q.GetMailServer(ctx); errors.Is(err, pgx.ErrNoRows) {
		return errMailServerMissing
	} else if err != nil {
		return err
	}
	actor, _ := platform.ActorFrom(ctx)
	r, err := q.Recipient(ctx, actor)
	if err != nil {
		return err
	}
	if !r.Email.Valid {
		return errNoAddress
	}
	_, err = platform.Enqueue(ctx, emailArgs{TestTo: actor})
	return err
}

// message is one email: a subject and a plain-text body.
type message struct{ subject, body string }

// sendEmail sends the email of a job, if there is still a server and an address.
func (s *Service) sendEmail(ctx context.Context, a emailArgs) error {
	q := store.New(platform.DBFrom(ctx))
	srv, err := q.GetMailServer(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	// A test message, or a notification: only a sentence for its kind and a link, so
	// nothing of the record leaves through the mail server.
	user, key, link := a.TestTo, "notification.email.test", srv.BaseUrl
	if a.Notification != 0 {
		n, err := q.GetNotification(ctx, a.Notification)
		if err != nil {
			return err
		}
		user, key, link = n.UserID, "notification.email."+n.Kind, srv.BaseUrl+"/notifications"
	}
	r, err := q.Recipient(ctx, user)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !r.Email.Valid) {
		return nil
	}
	if err != nil {
		return err
	}
	// A user without a language of their own reads the tenant's.
	locale := r.Locale.String
	if !r.Locale.Valid {
		if locale, err = s.d.Setting.Get(ctx, setting.Locale); err != nil {
			return err
		}
	}
	m := message{
		subject: platform.Translate(locale, key+".subject", nil),
		body: platform.Translate(locale, key+".body", nil) + "\n\n" +
			platform.Translate(locale, "notification.email.footer", map[string]any{"url": link}),
	}
	return s.deliver(ctx, srv, r.Email.String, m)
}

// deliver hands one message to the server. Errors carry only a code: the server's own
// words may echo the address, and they end up in the job's errors and the logs.
func (s *Service) deliver(ctx context.Context, srv store.GetMailServerRow, to string, m message) error {
	password := ""
	if srv.Password != nil {
		pw, err := platform.Decrypt(ctx, passwordAAD, srv.Password)
		if err != nil {
			return err
		}
		password = string(pw)
	}
	addr := net.JoinHostPort(srv.Host, strconv.Itoa(int(srv.Port)))
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	var conn net.Conn
	var err error
	if srv.Security == "tls" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: srv.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return errMailUnreachable
	}
	_ = conn.SetDeadline(time.Now().Add(2 * time.Minute))
	c, err := smtp.NewClient(conn, srv.Host)
	if err != nil {
		_ = conn.Close()
		return classify(err)
	}
	defer func() { _ = c.Close() }()
	if srv.Security == "starttls" {
		if err := c.StartTLS(&tls.Config{ServerName: srv.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return classify(err)
		}
	}
	if srv.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", srv.Username, password, srv.Host)); err != nil {
			// net/smtp refuses a password over plain text before asking the server.
			var tp *textproto.Error
			if !errors.As(err, &tp) || tp.Code >= 500 {
				return errMailAuth
			}
			return classify(err)
		}
	}
	if err := c.Mail(srv.FromAddress); err != nil {
		return classify(err)
	}
	if err := c.Rcpt(to); err != nil {
		return classify(err)
	}
	w, err := c.Data()
	if err != nil {
		return classify(err)
	}
	if _, err := w.Write(compose(srv.FromAddress, to, m)); err != nil {
		return classify(err)
	}
	if err := w.Close(); err != nil {
		return classify(err)
	}
	return classify(c.Quit())
}

// classify turns an SMTP exchange's error into a code: a permanent refusal is not retried.
func classify(err error) error {
	if err == nil {
		return nil
	}
	var tp *textproto.Error
	if errors.As(err, &tp) && tp.Code >= 500 {
		if tp.Code == 530 || tp.Code == 534 || tp.Code == 535 {
			return errMailAuth
		}
		return errMailRejected
	}
	return errMailUnreachable
}

// compose writes a plain-text UTF-8 message, its subject encoded for any server.
func compose(from, to string, m message) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\n",
		from, to, mime.BEncoding.Encode("UTF-8", m.subject), time.Now().Format(time.RFC1123Z))
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n")
	w := quotedprintable.NewWriter(&b)
	_, _ = w.Write([]byte(strings.ReplaceAll(m.body, "\n", "\r\n")))
	_ = w.Close()
	return []byte(b.String())
}
