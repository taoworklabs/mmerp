// Package discussion keeps the comments on records of any type. Whoever may view a record
// reads its comments and, while the record takes changes, adds to them.
package discussion

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/discussion/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/notification"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct{ d Deps }

func NewService(d Deps) *Service {
	s := &Service{d: d}
	d.Record.OnDeleted(s)
	return s
}

// List returns the comments of a record, oldest first.
func (s *Service) List(ctx context.Context, ref record.Ref) (Discussion, error) {
	out := Discussion{AllowedActions: []string{}, Items: []Comment{}, MaxLength: maxBody}
	if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
		return out, err
	}
	open, err := s.d.Record.Open(ctx, ref)
	if err != nil {
		return out, err
	}
	if open {
		out.AllowedActions = append(out.AllowedActions, actionComment)
	}
	rows, err := store.New(platform.DBFrom(ctx)).ListComments(ctx, store.ListCommentsParams{DocType: ref.Type, DocID: ref.ID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, Comment{ID: r.ID, AuthorName: r.AuthorName, Body: r.Body, CreatedAt: r.CreatedAt.Time.Format(time.RFC3339)})
	}
	return out, nil
}

// Add posts a comment on a record. It changes neither the record nor its version, so a
// locked period does not stop it; a cancelled document does.
func (s *Service) Add(ctx context.Context, ref record.Ref, body string) error {
	if err := s.d.Record.WriteGate(ctx, ref); err != nil {
		return err
	}
	if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" || !utf8.ValidString(body) {
		return errBlank
	}
	if utf8.RuneCountInString(body) > maxBody {
		return ErrCommentTooLong
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if err := s.d.Record.LockOpen(ctx, ref); err != nil {
			return err
		}
		actor, _ := platform.ActorFrom(ctx)
		id, err := store.New(platform.DBFrom(ctx)).CreateComment(ctx, store.CreateCommentParams{DocType: ref.Type, DocID: ref.ID, AuthorID: actor, Body: body})
		if err != nil {
			return err
		}
		if err := s.d.Audit.RecordFor(ctx, "discussion.commented", audit.Ref(ref), map[string]any{"id": id}); err != nil {
			return err
		}
		return s.mention(ctx, ref, body)
	})
}

// mentionRe finds @login in a comment; a trailing dot or comma ends the sentence, not the login.
var mentionRe = regexp.MustCompile(`@([\p{L}\p{N}_.-]*[\p{L}\p{N}_-])`)

// mention tells the users a comment mentions, among those who may view the record. Anyone
// else is skipped without an error, so the writer cannot probe who sees the record.
func (s *Service) mention(ctx context.Context, ref record.Ref, body string) error {
	var logins []string
	for _, m := range mentionRe.FindAllStringSubmatch(body, -1) {
		if l := strings.ToLower(m[1]); !slices.Contains(logins, l) {
			logins = append(logins, l)
		}
	}
	if len(logins) == 0 {
		return nil
	}
	ids, err := store.New(platform.DBFrom(ctx)).UsersByLogin(ctx, logins)
	if err != nil {
		return err
	}
	users, err := s.viewers(ctx, ref, ids)
	if err != nil {
		return err
	}
	return s.d.Notification.Send(ctx, notification.Mentioned, ref, users)
}

// viewers keeps the users who may view the record.
// ponytail: one Can per user; filter by role scope first if tenants grow past a few hundred users.
func (s *Service) viewers(ctx context.Context, ref record.Ref, ids []int64) ([]int64, error) {
	var out []int64
	for _, id := range ids {
		ok, err := s.d.Record.Can(s.d.IAM.AsUser(ctx, id), ref.Type, ref.ID, record.View)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// Mentionable lists the users the actor may mention on a record: the others who may view it.
func (s *Service) Mentionable(ctx context.Context, ref record.Ref) ([]Mentionable, error) {
	if err := s.d.Record.Visible(ctx, ref, record.View); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).Users(ctx)
	if err != nil {
		return nil, err
	}
	actor, _ := platform.ActorFrom(ctx)
	out := []Mentionable{}
	for _, r := range rows {
		if r.ID == actor {
			continue
		}
		ok, err := s.d.Record.Can(s.d.IAM.AsUser(ctx, r.ID), ref.Type, ref.ID, record.View)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, Mentionable{Login: r.Login, Name: r.Name})
		}
	}
	return out, nil
}

// Deleted removes the comments of a deleted draft, in its transaction.
func (s *Service) Deleted(ctx context.Context, ref record.Ref) error {
	return store.New(platform.DBFrom(ctx)).DeleteRecordComments(ctx, store.DeleteRecordCommentsParams{DocType: ref.Type, DocID: ref.ID})
}
