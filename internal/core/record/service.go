// Package record registers record types, answers what an actor may do with a
// record, and keeps the lifecycle of documents and the period locks.
package record

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/core/audit"
	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/core/record/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// plainNumber is the only number shape accepted from outside: no exponent, bounded size.
var plainNumber = regexp.MustCompile(`^-?[0-9]{1,15}(\.[0-9]{1,6})?$`)

// ParseNumber reads a decimal written plainly ("2.5"). An exponent such as "1e999999999"
// is refused: comparing it would expand into an enormous integer.
func ParseNumber(s string) (decimal.Decimal, bool) {
	if !plainNumber.MatchString(s) {
		return decimal.Decimal{}, false
	}
	d, err := decimal.NewFromString(s)
	return d, err == nil
}

func (a Action) class() platform.Class {
	switch a {
	case View, ViewFiles:
		return platform.ClassRead
	case Export, Print:
		return platform.ClassExport
	}
	return platform.ClassWrite
}

type Service struct {
	d        Deps
	types    map[string]Type
	gate     ApprovalGate
	keepers  []Keeper
	freezers []Freezer
	// history says what another module's audit action needs to show on a timeline.
	history map[string]Action
	// offers holds the print and export actions a type has something registered for.
	offers map[string][]Action
}

func NewService(d Deps) *Service {
	return &Service{d: d, types: map[string]Type{}, history: map[string]Action{}, offers: map[string][]Action{}}
}

// SetApprovalGate wires approval while composing the app. Without a gate nothing needs approval.
func (s *Service) SetApprovalGate(g ApprovalGate) { s.gate = g }

// OnDeleted registers a module that keeps data about records, while wiring modules.
func (s *Service) OnDeleted(k Keeper) { s.keepers = append(s.keepers, k) }

// OnPosted registers a core module told of every document posted, while wiring modules.
func (s *Service) OnPosted(f Freezer) { s.freezers = append(s.freezers, f) }

// RestrictHistory shows audit action on a record's timeline only to whom may perform
// needs on the record; with needs empty it never shows there, only in the audit log.
// Called while wiring modules.
func (s *Service) RestrictHistory(action string, needs Action) { s.history[action] = needs }

// Register adds a record type while wiring modules, before serving.
func (s *Service) Register(t Type) {
	// The product gate goes by Product, so a type without one would never be gated.
	if t.Product == "" || !strings.HasPrefix(t.Code, t.Product+".") {
		panic("record: type " + t.Code + " is not named <product>.<type> after its product " + t.Product)
	}
	if _, dup := s.types[t.Code]; dup {
		panic("record: type " + t.Code + " registered twice")
	}
	if t.Kind == Document && t.NumberPrefix == "" {
		panic("record: document type " + t.Code + " has no number prefix")
	}
	if t.Can == nil {
		s.checkDeclaration(t)
	}
	s.types[t.Code] = t
}

func (s *Service) checkDeclaration(t Type) {
	fail := func(why string) { panic("record: type " + t.Code + " " + why) }
	if t.Perms.View == "" {
		fail("declares no view permission")
	}
	p := t.Perms
	for _, perm := range []string{p.View, p.Edit, p.Post, p.Cancel, p.Export, p.Print, p.ViewFiles, p.Attach, p.Create, p.Import} {
		if perm != "" && !s.d.IAM.Registered(t.Product, perm) {
			fail("declares " + perm + ", which no role of " + t.Product + " grants")
		}
	}
	if t.Kind == Catalog {
		if t.NumberPrefix != "" || t.Fields != nil || t.OnTransition != nil || t.BeforeSubmit != nil || t.Approvers != nil || t.SubjectCreates {
			fail("is a catalog with document-only fields")
		}
		if t.Locate == nil {
			fail("is a catalog without Locate")
		}
	}
}

// Offer says a print template or an export (action Print or Export) exists for a type;
// without one a declared type never answers that action. Called while wiring modules.
func (s *Service) Offer(code string, action Action) {
	s.offers[code] = append(s.offers[code], action)
}

// TypeOf returns a registered type.
func (s *Service) TypeOf(code string) (Type, bool) {
	t, ok := s.types[code]
	return t, ok
}

// DocumentTypes lists the registered document types by code.
func (s *Service) DocumentTypes() []Type {
	var out []Type
	for _, c := range slices.Sorted(maps.Keys(s.types)) {
		if s.types[c].Kind == Document {
			out = append(out, s.types[c])
		}
	}
	return out
}

// ProductsWithData lists, sorted, the products that have a document or master data.
func (s *Service) ProductsWithData(ctx context.Context) ([]string, error) {
	var codes []string
	for _, t := range s.DocumentTypes() {
		codes = append(codes, t.Code)
	}
	used, err := store.New(platform.DBFrom(ctx)).DocTypesInUse(ctx, codes)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, c := range used {
		if p := s.types[c].Product; !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	for _, t := range s.types {
		if t.HasData == nil || slices.Contains(out, t.Product) {
			continue
		}
		if ok, err := t.HasData(ctx); err != nil {
			return nil, err
		} else if ok {
			out = append(out, t.Product)
		}
	}
	slices.Sort(out)
	return out, nil
}

// Can reports whether the actor may perform action on the record.
func (s *Service) Can(ctx context.Context, code string, id int64, action Action) (bool, error) {
	t, ok := s.types[code]
	if !ok {
		return false, fmt.Errorf("record: unknown type %q", code)
	}
	if t.Can != nil {
		ok, err := t.Can(ctx, id, action)
		if err != nil || !ok {
			return false, err
		}
		return platform.ProductGate(ctx, t.Product, action.class()) == nil, nil
	}
	var unit int64
	var err error
	if t.Kind == Document {
		var d Doc
		d, err = s.Get(ctx, Ref{code, id})
		unit = d.OrgUnitID
	} else {
		unit, err = t.Locate(ctx, id)
	}
	if errors.Is(err, platform.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s.answer(ctx, t, id, unit, action)
}

// answer evaluates a declared type's action on record id at org unit unit.
func (s *Service) answer(ctx context.Context, t Type, id, unit int64, action Action) (bool, error) {
	perm := t.Perms.of(action)
	if perm == "" || (action == Print || action == Export) && !slices.Contains(s.offers[t.Code], action) {
		return false, nil
	}
	ok, err := s.d.IAM.Allowed(ctx, t.Product, perm, unit)
	if err == nil && !ok && t.Grant != nil {
		ok, err = t.Grant(ctx, id, action)
	}
	if err == nil && ok && t.Refine != nil {
		ok, err = t.Refine(ctx, id, action)
	}
	if err != nil || !ok {
		return false, err
	}
	return platform.ProductGate(ctx, t.Product, action.class()) == nil, nil
}

// Read returns a document the actor may view; one they may not does not exist.
func (s *Service) Read(ctx context.Context, ref Ref) (Doc, error) {
	if err := s.Visible(ctx, ref, View); err != nil {
		return Doc{}, err
	}
	return s.Get(ctx, ref)
}

// Actions is the whole allowed_actions of d: its lifecycle actions, print and export
// when available, then extra, the module's own actions it has already decided on.
func (s *Service) Actions(ctx context.Context, d Doc, extra ...string) ([]string, error) {
	out, err := s.DocumentActions(ctx, d)
	if err != nil {
		return nil, err
	}
	for _, a := range []Action{Print, Export} {
		ok, err := s.Can(ctx, d.Type, d.ID, a)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, string(a))
		}
	}
	return append(out, extra...), nil
}

// TypeActions lists what the actor may do with a type before picking a record: create
// and import, where its declared permission holds anywhere in their scope.
func (s *Service) TypeActions(ctx context.Context, code string) ([]string, error) {
	t, ok := s.types[code]
	if !ok {
		return nil, fmt.Errorf("record: unknown type %q", code)
	}
	out := []string{}
	if platform.ProductGate(ctx, t.Product, platform.ClassWrite) != nil {
		return out, nil
	}
	for _, a := range []Action{Create, Import} {
		perm := t.Perms.of(a)
		if perm == "" {
			continue
		}
		sc, err := s.d.IAM.Scope(ctx, t.Product, perm)
		if err != nil {
			return nil, err
		}
		if sc.Any() {
			out = append(out, string(a))
		}
	}
	return out, nil
}

// mayCreate checks a declared type's create permission at the header's org unit; the
// subject the header names may create without it if the type says so. Refused for a
// named subject, the subject stays hidden as if it did not exist.
func (s *Service) mayCreate(ctx context.Context, t Type, h Header) error {
	actor, _ := platform.ActorFrom(ctx)
	if t.SubjectCreates && h.Subject != 0 && h.Subject == actor {
		return nil
	}
	ok, err := s.d.IAM.Allowed(ctx, t.Product, t.Perms.of(Create), h.OrgUnitID)
	switch {
	case err != nil:
		return err
	case ok:
		return nil
	case h.Subject != 0:
		return platform.ErrNotFound
	}
	return platform.ErrForbidden
}

// require fails with the product gate's own error first, so a disabled product says so.
func (s *Service) require(ctx context.Context, ref Ref, action Action) error {
	if err := platform.ProductGate(ctx, s.types[ref.Type].Product, action.class()); err != nil {
		return err
	}
	ok, err := s.Can(ctx, ref.Type, ref.ID, action)
	if err == nil && !ok {
		return platform.ErrForbidden
	}
	return err
}

// AllowedActions filters actions down to those the actor may perform, for allowed_actions.
func (s *Service) AllowedActions(ctx context.Context, code string, id int64, actions ...Action) ([]string, error) {
	out := []string{}
	for _, a := range actions {
		ok, err := s.Can(ctx, code, id, a)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, string(a))
		}
	}
	return out, nil
}

// documentActions maps each status to the API actions and the permission each needs.
var documentActions = map[Status][]struct {
	name   string
	action Action
}{
	Draft:           {{"edit", Edit}, {"delete", Edit}, {"submit", Post}},
	PendingApproval: {{"withdraw", Post}},
	Posted:          {{"cancel", Cancel}},
}

// DocumentActions lists the write actions on d for allowed_actions; none once its date is locked.
func (s *Service) DocumentActions(ctx context.Context, d Doc) ([]string, error) {
	out := []string{}
	rows, err := store.New(platform.DBFrom(ctx)).LockedUntil(ctx, d.LegalEntityID)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 && checkPeriod(d.Date, rows[0]) != nil {
		return out, nil
	}
	for _, a := range documentActions[d.Status] {
		if a.name == "withdraw" && !submittedBy(ctx, d) {
			continue
		}
		ok, err := s.Can(ctx, d.Type, d.ID, a.action)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, a.name)
		}
	}
	return out, nil
}

// Get reads a document without locking it.
func (s *Service) Get(ctx context.Context, ref Ref) (Doc, error) {
	row, err := store.New(platform.DBFrom(ctx)).GetDocument(ctx, store.GetDocumentParams{ID: ref.ID, DocType: ref.Type})
	if errors.Is(err, pgx.ErrNoRows) {
		return Doc{}, platform.ErrNotFound
	}
	if err != nil {
		return Doc{}, err
	}
	return doc(row)
}

// Create adds a draft and gives it a number; the caller then inserts its own row with
// the returned id. A declared type's create permission is checked here; for a type with
// a hand-written Can, the caller checks it.
func (s *Service) Create(ctx context.Context, code string, h Header) (Doc, error) {
	t, ok := s.types[code]
	if !ok || t.Kind != Document {
		return Doc{}, fmt.Errorf("record: %q is not a document type", code)
	}
	if err := platform.ProductGate(ctx, t.Product, platform.ClassWrite); err != nil {
		return Doc{}, err
	}
	if t.Can == nil {
		if err := s.mayCreate(ctx, t, h); err != nil {
			return Doc{}, err
		}
	}
	date, fields, err := s.check(ctx, t, h)
	if err != nil {
		return Doc{}, err
	}
	var out Doc
	err = platform.InTx(ctx, func(ctx context.Context) error {
		if h.RequestID != "" {
			found, ok, err := byRequest(ctx, code, h.RequestID)
			if ok || err != nil {
				out = found
				return err
			}
		}
		// Before the legal entity is read, so a tree move cannot place the document wrongly.
		if err := s.d.IAM.ShareTree(ctx); err != nil {
			return err
		}
		le, err := s.d.IAM.LegalEntityOf(ctx, h.OrgUnitID)
		if err != nil {
			return err
		}
		locked, err := s.sharePeriod(ctx, le)
		if err != nil {
			return err
		}
		if err := checkPeriod(h.Date, locked); err != nil {
			return err
		}
		year := int32(date.Time.Year())
		n, err := s.d.Numbering.Next(ctx, code, le, year)
		if err != nil {
			return err
		}
		number := fmt.Sprintf("%s-%d-%05d", t.NumberPrefix, year, n)
		id, err := store.New(platform.DBFrom(ctx)).InsertDocument(ctx, store.InsertDocumentParams{
			DocType: code, Number: number, Date: date, LegalEntityID: le, OrgUnitID: h.OrgUnitID,
			Amount: platform.NullInt8(h.Amount), Fields: fields,
			RequestID: pgtype.Text{String: h.RequestID, Valid: h.RequestID != ""}, RequestedBy: requester(ctx, h.RequestID),
		})
		if err != nil {
			return err
		}
		out = Doc{Ref: Ref{code, id}, Number: number, Status: Draft, Version: 1, Date: h.Date,
			LegalEntityID: le, OrgUnitID: h.OrgUnitID, Amount: h.Amount, Fields: h.Fields}
		return s.d.Audit.RecordFor(ctx, "record.created", audit.Ref(out.Ref), map[string]any{"number": number})
	})
	return out, err
}

// byRequest finds the document the actor created earlier with requestID. Its lock makes
// a racing twin wait for the first to commit, then find it.
func byRequest(ctx context.Context, code, requestID string) (Doc, bool, error) {
	q := store.New(platform.DBFrom(ctx))
	if err := q.LockRequest(ctx, requestID); err != nil {
		return Doc{}, false, err
	}
	row, err := q.DocumentByRequest(ctx, store.DocumentByRequestParams{RequestID: requestID, RequestedBy: requester(ctx, requestID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return Doc{}, false, nil
	}
	if err != nil {
		return Doc{}, false, err
	}
	if row.DocType != code {
		return Doc{}, false, ErrRequestReused
	}
	d, err := doc(row)
	d.Reused = true
	return d, true, err
}

// requester is who a create-once key belongs to; none without a key.
func requester(ctx context.Context, requestID string) pgtype.Int8 {
	actor, ok := platform.ActorFrom(ctx)
	return pgtype.Int8{Int64: actor, Valid: ok && requestID != ""}
}

// Edit replaces the header of a draft the actor may edit, at the version they saw.
func (s *Service) Edit(ctx context.Context, ref Ref, version int32, h Header) (Doc, error) {
	var out Doc
	err := platform.InTx(ctx, func(ctx context.Context) error {
		d, locked, err := s.lockDraft(ctx, ref, version)
		if err != nil {
			return err
		}
		date, fields, err := s.check(ctx, s.types[ref.Type], h)
		if err != nil {
			return err
		}
		if err := checkPeriod(h.Date, locked); err != nil {
			return err
		}
		if err := s.d.IAM.ShareTree(ctx); err != nil {
			return err
		}
		if le, err := s.d.IAM.LegalEntityOf(ctx, h.OrgUnitID); err != nil {
			return err
		} else if le != d.LegalEntityID {
			return ErrLegalEntityChanged
		}
		if err := store.New(platform.DBFrom(ctx)).UpdateHeader(ctx, store.UpdateHeaderParams{
			ID: ref.ID, Date: date, OrgUnitID: h.OrgUnitID, Amount: platform.NullInt8(h.Amount), Fields: fields,
		}); err != nil {
			return err
		}
		d.Date, d.OrgUnitID, d.Amount, d.Fields, d.Version = h.Date, h.OrgUnitID, h.Amount, h.Fields, d.Version+1
		out = d
		return nil
	})
	return out, err
}

// Touch locks a draft like Edit and bumps its version without changing the header,
// for module writes that change only child rows.
func (s *Service) Touch(ctx context.Context, ref Ref, version int32) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		if _, _, err := s.lockDraft(ctx, ref, version); err != nil {
			return err
		}
		return store.New(platform.DBFrom(ctx)).BumpVersion(ctx, ref.ID)
	})
}

// Delete removes a draft, its type's own rows (OnDelete) and what keepers hold about it.
func (s *Service) Delete(ctx context.Context, ref Ref, version int32) error {
	t, ok := s.types[ref.Type]
	if !ok || t.Kind != Document {
		return platform.ErrNotFound
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, _, err := s.lockDraft(ctx, ref, version)
		if err != nil {
			return err
		}
		if err := store.New(platform.DBFrom(ctx)).DeleteDocument(ctx, ref.ID); err != nil {
			return err
		}
		if t.OnDelete != nil {
			if err := t.OnDelete(ctx, ref.ID); err != nil {
				return err
			}
		}
		for _, k := range s.keepers {
			if err := k.Deleted(ctx, ref); err != nil {
				return err
			}
		}
		return s.d.Audit.RecordFor(ctx, "record.deleted", audit.Ref(ref), map[string]any{"number": d.Number})
	})
}

func (s *Service) lockDraft(ctx context.Context, ref Ref, version int32) (Doc, pgtype.Date, error) {
	d, locked, err := s.lockVisible(ctx, ref)
	if err != nil {
		return d, locked, err
	}
	if err := s.require(ctx, ref, Edit); err != nil {
		return d, locked, err
	}
	if d.Status != Draft {
		return d, locked, ErrNotEditable
	}
	if d.Version != version {
		return d, locked, ErrVersionConflict
	}
	return d, locked, checkPeriod(d.Date, locked)
}

// Transition moves a document the actor may act on, at the version they saw:
// to posted (send; may stop at pending_approval), back to draft (withdraw), or to cancelled.
func (s *Service) Transition(ctx context.Context, ref Ref, version int32, to Status) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, locked, err := s.lockVisible(ctx, ref)
		if err != nil {
			return err
		}
		var action Action
		switch {
		case d.Status == Draft && to == Posted:
			action = Post
		case d.Status == PendingApproval && to == Draft:
			// Taking back a submission needs the right that sent it, not the right to edit.
			action = Post
		case d.Status == Posted && to == Cancelled:
			action = Cancel
		default:
			return ErrInvalidTransition
		}
		if err := s.require(ctx, ref, action); err != nil {
			return err
		}
		if d.Status == PendingApproval && !submittedBy(ctx, d) {
			return platform.ErrForbidden
		}
		if d.Version != version {
			return ErrVersionConflict
		}
		if err := checkPeriod(d.Date, locked); err != nil {
			return err
		}
		if t := s.types[ref.Type]; to == Posted && t.BeforeSubmit != nil {
			if err := t.BeforeSubmit(ctx, d); err != nil {
				return err
			}
		}
		switch {
		case to == Posted && s.gate != nil:
			ticket, needed, err := s.gate.Submit(ctx, d)
			if err != nil {
				return err
			}
			if !needed {
				break
			}
			actor, _ := platform.ActorFrom(ctx)
			if err := store.New(platform.DBFrom(ctx)).SetStatus(ctx, store.SetStatusParams{
				ID: ref.ID, Status: string(PendingApproval), ApprovalTicket: pgtype.Int8{Int64: ticket, Valid: true},
				SubmittedBy: pgtype.Int8{Int64: actor, Valid: true},
			}); err != nil {
				return err
			}
			return s.d.Audit.RecordFor(ctx, "record.transitioned", audit.Ref(ref), map[string]any{"from": d.Status, "to": PendingApproval})
		case d.Status == PendingApproval:
			if err := s.gate.Withdrawn(ctx, d); err != nil {
				return err
			}
		}
		return s.apply(ctx, d, to)
	})
}

// CompleteApproval posts a document whose last approval step was just approved.
// approval calls it in the transaction recording the decision; permission comes
// from the approval rule, not from Can. ErrApprovalStale means the instance no
// longer matches the document; any other error leaves the document pending.
func (s *Service) CompleteApproval(ctx context.Context, ref Ref, ticket int64, version int32) error {
	return s.decide(ctx, ref, ticket, version, Posted)
}

// Reject returns a pending document to draft; see CompleteApproval.
func (s *Service) Reject(ctx context.Context, ref Ref, ticket int64, version int32) error {
	return s.decide(ctx, ref, ticket, version, Draft)
}

func (s *Service) decide(ctx context.Context, ref Ref, ticket int64, version int32, to Status) error {
	return platform.InTx(ctx, func(ctx context.Context) error {
		d, locked, err := s.lock(ctx, ref)
		if err != nil {
			return err
		}
		if d.Status != PendingApproval || d.ApprovalTicket == nil || *d.ApprovalTicket != ticket || d.Version != version {
			return ErrApprovalStale
		}
		if err := checkPeriod(d.Date, locked); err != nil {
			return err
		}
		return s.apply(ctx, d, to)
	})
}

func (s *Service) apply(ctx context.Context, d Doc, to Status) error {
	if err := store.New(platform.DBFrom(ctx)).SetStatus(ctx, store.SetStatusParams{ID: d.ID, Status: string(to), Bump: 1}); err != nil {
		return err
	}
	if err := s.d.Audit.RecordFor(ctx, "record.transitioned", audit.Ref(d.Ref), map[string]any{"from": d.Status, "to": to}); err != nil {
		return err
	}
	from := d.Status
	d.Status, d.Version, d.ApprovalTicket, d.SubmittedBy = to, d.Version+1, nil, nil
	if t := s.types[d.Type]; t.OnTransition != nil {
		if err := t.OnTransition(ctx, d, from); err != nil {
			return err
		}
	}
	if to != Posted {
		return nil
	}
	for _, f := range s.freezers {
		if err := f.Posted(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// Lock takes the document's locks in the shared order (period lock, then the
// document) for a core module that locks its own rows after them.
func (s *Service) Lock(ctx context.Context, ref Ref) (Doc, error) {
	d, _, err := s.lock(ctx, ref)
	return d, err
}

// Visible fails with not found unless ref's type exists and the actor may perform
// action (a read) on it, so a core module attached to records lets none be probed.
func (s *Service) Visible(ctx context.Context, ref Ref, action Action) error {
	if _, ok := s.types[ref.Type]; !ok {
		return platform.ErrNotFound
	}
	ok, err := s.Can(ctx, ref.Type, ref.ID, action)
	if err != nil || !ok {
		return platform.OrErr(err, platform.ErrNotFound)
	}
	return nil
}

// WriteGate fails with the product gate's own error when ref's product takes no
// writes, and with not found for an unknown type.
func (s *Service) WriteGate(ctx context.Context, ref Ref) error {
	t, ok := s.types[ref.Type]
	if !ok {
		return platform.ErrNotFound
	}
	return platform.ProductGate(ctx, t.Product, platform.ClassWrite)
}

// Open reports whether ref still takes additions (attachments, comments): its product
// takes writes and, for a document, it is not cancelled.
func (s *Service) Open(ctx context.Context, ref Ref) (bool, error) {
	if s.WriteGate(ctx, ref) != nil {
		return false, nil
	}
	if s.types[ref.Type].Kind != Document {
		return true, nil
	}
	d, err := s.Get(ctx, ref)
	return err == nil && d.Status != Cancelled, err
}

// LockOpen takes a document's locks before a core module adds to it what changes
// neither its data nor its version (attachments, comments), so the period lock does not
// apply; a cancelled one is refused. A catalog record takes no lock.
func (s *Service) LockOpen(ctx context.Context, ref Ref) error {
	if s.types[ref.Type].Kind != Document {
		return nil
	}
	d, _, err := s.lock(ctx, ref)
	if err == nil && d.Status == Cancelled {
		return ErrNotEditable
	}
	return err
}

// lockVisible locks a document the actor may view; one they may not does not exist,
// so its status cannot be probed through write errors.
func (s *Service) lockVisible(ctx context.Context, ref Ref) (Doc, pgtype.Date, error) {
	d, locked, err := s.lock(ctx, ref)
	if err != nil {
		return d, locked, err
	}
	if ok, err := s.Can(ctx, ref.Type, ref.ID, View); err != nil || !ok {
		return d, locked, platform.OrErr(err, platform.ErrNotFound)
	}
	return d, locked, nil
}

func (s *Service) lock(ctx context.Context, ref Ref) (Doc, pgtype.Date, error) {
	// The legal entity never changes, so reading it before the locks is safe.
	d, err := s.Get(ctx, ref)
	if err != nil {
		return d, pgtype.Date{}, err
	}
	locked, err := s.sharePeriod(ctx, d.LegalEntityID)
	if err != nil {
		return d, locked, err
	}
	row, err := store.New(platform.DBFrom(ctx)).LockDocument(ctx, ref.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, locked, platform.ErrNotFound
	}
	if err != nil {
		return d, locked, err
	}
	d, err = doc(row)
	return d, locked, err
}

func (s *Service) sharePeriod(ctx context.Context, legalEntity int64) (pgtype.Date, error) {
	q := store.New(platform.DBFrom(ctx))
	if err := q.EnsurePeriodLock(ctx, legalEntity); err != nil {
		return pgtype.Date{}, err
	}
	return q.SharePeriodLock(ctx, legalEntity)
}

func checkPeriod(date string, locked pgtype.Date) error {
	if until := locked.Time.Format(time.DateOnly); locked.Valid && date <= until {
		return errPeriodLocked(until)
	}
	return nil
}

// check validates a header against the type's approval fields.
func (s *Service) check(ctx context.Context, t Type, h Header) (pgtype.Date, []byte, error) {
	date, err := time.Parse(time.DateOnly, h.Date)
	if err != nil {
		return pgtype.Date{}, nil, errInvalidField("date")
	}
	for k, v := range h.Fields {
		i := slices.IndexFunc(t.Fields, func(f Field) bool { return f.Key == k })
		if i < 0 {
			return pgtype.Date{}, nil, errInvalidField(k)
		}
		ok := false
		switch f := t.Fields[i]; f.Kind {
		case Number:
			_, ok = ParseNumber(v)
		case OrgUnit:
			_, err := strconv.ParseInt(v, 10, 64)
			ok = err == nil
		case Choice:
			opts, err := f.Options(ctx)
			if err != nil {
				return pgtype.Date{}, nil, err
			}
			ok = slices.ContainsFunc(opts, func(o Option) bool { return o.Value == v })
		}
		if !ok {
			return pgtype.Date{}, nil, errInvalidField(k)
		}
	}
	fields, err := json.Marshal(cmpMap(h.Fields))
	return pgtype.Date{Time: date, Valid: true}, fields, err
}

func cmpMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// PeriodLocks lists every legal entity with its lock date.
func (s *Service) PeriodLocks(ctx context.Context) ([]PeriodLock, error) {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManagePeriods); err != nil {
		return nil, err
	}
	rows, err := store.New(platform.DBFrom(ctx)).PeriodLocks(ctx)
	out := make([]PeriodLock, len(rows))
	for i, r := range rows {
		out[i] = PeriodLock{LegalEntityID: r.ID, LegalEntityName: r.Name, LockedUntil: platform.DatePtr(r.LockedUntil)}
	}
	return out, err
}

// SetPeriodLock moves a legal entity's lock date; nil unlocks everything. Moving it
// later is refused while documents in the newly locked days await approval, since
// they could then neither be approved nor withdrawn.
func (s *Service) SetPeriodLock(ctx context.Context, legalEntity int64, until *string) error {
	if err := s.d.IAM.RequireCore(ctx, iam.PermManagePeriods); err != nil {
		return err
	}
	var date pgtype.Date
	if until != nil {
		t, err := time.Parse(time.DateOnly, *until)
		if err != nil {
			return errInvalidField("locked_until")
		}
		date = pgtype.Date{Time: t, Valid: true}
	}
	return platform.InTx(ctx, func(ctx context.Context) error {
		if le, err := s.d.IAM.LegalEntityOf(ctx, legalEntity); err != nil || le != legalEntity {
			return platform.OrErr(err, platform.ErrNotFound)
		}
		q := store.New(platform.DBFrom(ctx))
		if err := q.EnsurePeriodLock(ctx, legalEntity); err != nil {
			return err
		}
		old, err := q.LockPeriodLock(ctx, legalEntity)
		if err != nil {
			return err
		}
		if date.Valid && (!old.Valid || date.Time.After(old.Time)) {
			pending, err := q.PendingInPeriod(ctx, store.PendingInPeriodParams{LegalEntityID: legalEntity, Date: date})
			if err != nil {
				return err
			}
			if len(pending) > 0 {
				docs := make([]map[string]any, len(pending))
				for i, p := range pending {
					docs[i] = map[string]any{"doc_type": p.DocType, "id": p.ID, "number": p.Number, "date": platform.DatePtr(p.Date)}
				}
				return &platform.Error{Status: http.StatusConflict, Code: "period_has_pending_documents", Params: map[string]any{"documents": docs}}
			}
		}
		if err := q.SetLockedUntil(ctx, store.SetLockedUntilParams{LegalEntityID: legalEntity, LockedUntil: date}); err != nil {
			return err
		}
		return s.d.Audit.Record(ctx, "record.period_locked", map[string]any{
			"legal_entity_id": legalEntity, "old": platform.DatePtr(old), "new": until,
		})
	})
}

// TreeChanged refuses an org-unit tree write that moves documents to another legal entity.
func (s *Service) TreeChanged(ctx context.Context) error {
	n, err := store.New(platform.DBFrom(ctx)).MisplacedDocuments(ctx)
	if err == nil && n > 0 {
		return ErrOrgUnitHasDocuments
	}
	return err
}

// History lists the audit entries of a document the actor may view. Sensitive
// values are always masked.
func (s *Service) History(ctx context.Context, ref Ref) ([]HistoryEntry, error) {
	if ok, err := s.Can(ctx, ref.Type, ref.ID, View); err != nil || !ok {
		return nil, platform.OrErr(err, platform.ErrNotFound)
	}
	rows, err := store.New(platform.DBFrom(ctx)).History(ctx, store.HistoryParams{
		DocType: pgtype.Text{String: ref.Type, Valid: true}, DocID: pgtype.Int8{Int64: ref.ID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	allowed := map[Action]bool{}
	out := make([]HistoryEntry, 0, len(rows))
	for _, r := range rows {
		if needs, ok := s.history[r.Action]; ok {
			if _, asked := allowed[needs]; !asked && needs != "" {
				if allowed[needs], err = s.Can(ctx, ref.Type, ref.ID, needs); err != nil {
					return nil, err
				}
			}
			if !allowed[needs] {
				continue
			}
		}
		e := HistoryEntry{ID: r.ID, At: r.At.Time.Format(time.RFC3339), ActorName: platform.TextPtr(r.ActorName), Action: r.Action}
		if r.Data != nil {
			if err := json.Unmarshal(r.Data, &e.Data); err != nil {
				return nil, err
			}
		}
		if changes, ok := e.Data["changes"].(map[string]any); ok {
			for _, c := range changes {
				if c, ok := c.(map[string]any); ok && c["sensitive"] == true {
					c["old"], c["new"] = nil, nil
				}
			}
		}
		out = append(out, e)
	}
	return out, nil
}

func doc(r store.RecordDocument) (Doc, error) {
	d := Doc{
		Ref: Ref{r.DocType, r.ID}, Number: r.Number, Status: Status(r.Status), Version: r.Version,
		Date: *platform.DatePtr(r.Date), LegalEntityID: r.LegalEntityID, OrgUnitID: r.OrgUnitID,
		Amount: platform.Int8Ptr(r.Amount), ApprovalTicket: platform.Int8Ptr(r.ApprovalTicket), SubmittedBy: platform.Int8Ptr(r.SubmittedBy),
	}
	return d, json.Unmarshal(r.Fields, &d.Fields)
}

// submittedBy reports whether the actor sent d for approval.
func submittedBy(ctx context.Context, d Doc) bool {
	actor, ok := platform.ActorFrom(ctx)
	return ok && d.SubmittedBy != nil && *d.SubmittedBy == actor
}
