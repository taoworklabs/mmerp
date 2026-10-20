// Package notification tells users of events that concern them: a document waiting for
// their approval, the outcome of one they sent, a mention, one of their jobs finishing.
package notification

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/core/notification/internal/store"
	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type Service struct {
	d Deps
	// kinds says which kinds may be sent, and whether each is emailed too.
	kinds map[Kind]bool
}

//go:embed i18n/*.json
var i18nFiles embed.FS

// loadTranslations adds the email texts to the catalog, once.
var loadTranslations = sync.OnceValue(func() error {
	sub, err := fs.Sub(i18nFiles, "i18n")
	if err != nil {
		return err
	}
	return platform.LoadTranslations(sub)
})

func NewService(d Deps) *Service {
	if err := loadTranslations(); err != nil {
		// The files are embedded: failing here is a build defect.
		panic(err)
	}
	// A job's requester is waiting in the app: job kinds are not emailed.
	return &Service{d: d, kinds: map[Kind]bool{ApprovalRequested: true, ApprovalApproved: true, ApprovalRejected: true, ApprovalOverdue: true,
		Mentioned: true, JobCompleted: false, JobFailed: false}}
}

// RegisterKind lets a product send kind, named <product>.<name>, while wiring modules.
// An emailed kind needs <kind>.email.subject and <kind>.email.body in the product's translations.
func (s *Service) RegisterKind(kind Kind, emailed bool) {
	if product, name, _ := strings.Cut(string(kind), "."); product == "" || name == "" {
		panic("notification: kind " + string(kind) + " is not named <product>.<name>")
	}
	if _, dup := s.kinds[kind]; dup {
		panic("notification: kind " + string(kind) + " registered twice")
	}
	s.kinds[kind] = emailed
}

// Send tells users of an event on a record, in the event's transaction. The actor is
// never told of their own action.
func (s *Service) Send(ctx context.Context, kind Kind, ref record.Ref, users []int64) error {
	emailed, ok := s.kinds[kind]
	if !ok {
		return fmt.Errorf("notification: kind %q is not registered", kind)
	}
	actor, _ := platform.ActorFrom(ctx)
	users = slices.DeleteFunc(slices.Clone(users), func(u int64) bool { return u == actor })
	if len(users) == 0 {
		return nil
	}
	q := store.New(platform.DBFrom(ctx))
	ids, err := q.Insert(ctx, store.InsertParams{
		Users: users, Kind: string(kind),
		DocType: pgtype.Text{String: ref.Type, Valid: true}, DocID: pgtype.Int8{Int64: ref.ID, Valid: true},
		ActorID: pgtype.Int8{Int64: actor, Valid: actor != 0},
	})
	if err != nil || !emailed {
		return err
	}
	// Queued in the event's transaction; sending happens later, so a dead server stops nothing.
	mail, err := q.Mailable(ctx, ids)
	if err != nil {
		return err
	}
	for _, id := range mail {
		if _, err := platform.Enqueue(ctx, emailArgs{Notification: id}); err != nil {
			return err
		}
	}
	return nil
}

// List returns a page of the actor's notifications, newest first, older than before
// when given. Whatever names a record is left out once the actor may no longer view it.
func (s *Service) List(ctx context.Context, before *int64) (Page, error) {
	actor, _ := platform.ActorFrom(ctx)
	rows, err := store.New(platform.DBFrom(ctx)).List(ctx, store.ListParams{UserID: actor, Before: platform.NullInt8(before), Lim: pageSize + 1})
	if err != nil {
		return Page{}, err
	}
	out := Page{Items: []Notification{}}
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		out.NextBefore = &rows[pageSize-1].ID
	}
	visible := map[record.Ref]bool{}
	for _, r := range rows {
		n := Notification{ID: r.ID, Kind: r.Kind, RecordType: platform.TextPtr(r.DocType), JobID: platform.Int8Ptr(r.JobID),
			CreatedAt: r.CreatedAt.Time.Format(time.RFC3339), Read: r.ReadAt.Valid}
		if r.DocType.Valid {
			ref := record.Ref{Type: r.DocType.String, ID: r.DocID.Int64}
			ok, seen := visible[ref]
			if !seen {
				if ok, err = s.canView(ctx, ref); err != nil {
					return Page{}, err
				}
				visible[ref] = ok
			}
			if ok {
				n.RecordID, n.Label, n.ActorName = &ref.ID, platform.TextPtr(r.Number), platform.TextPtr(r.ActorName)
			}
		}
		out.Items = append(out.Items, n)
	}
	return out, nil
}

// canView asks the record's type; a type no longer registered is never viewable.
func (s *Service) canView(ctx context.Context, ref record.Ref) (bool, error) {
	if _, ok := s.d.Record.TypeOf(ref.Type); !ok {
		return false, nil
	}
	return s.d.Record.Can(ctx, ref.Type, ref.ID, record.View)
}

// Unread counts the actor's unread notifications.
func (s *Service) Unread(ctx context.Context) (int64, error) {
	actor, _ := platform.ActorFrom(ctx)
	return store.New(platform.DBFrom(ctx)).Unread(ctx, actor)
}

// MarkRead marks one of the actor's notifications read; anyone else's does not exist.
func (s *Service) MarkRead(ctx context.Context, id int64) error {
	actor, _ := platform.ActorFrom(ctx)
	n, err := store.New(platform.DBFrom(ctx)).MarkRead(ctx, store.MarkReadParams{ID: id, UserID: actor})
	if err == nil && n == 0 {
		return platform.ErrNotFound
	}
	return err
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	actor, _ := platform.ActorFrom(ctx)
	return store.New(platform.DBFrom(ctx)).MarkAllRead(ctx, actor)
}

// JobEnded tells a requester that their job ended; it is the platform's job notifier. Unlike
// Send it keeps the actor, who is the requester the job runs as.
func JobEnded(ctx context.Context, jobID, requester int64, failed bool) error {
	kind := JobCompleted
	if failed {
		kind = JobFailed
	}
	_, err := store.New(platform.DBFrom(ctx)).Insert(ctx, store.InsertParams{
		Users: []int64{requester}, Kind: string(kind), JobID: pgtype.Int8{Int64: jobID, Valid: true},
	})
	return err
}
