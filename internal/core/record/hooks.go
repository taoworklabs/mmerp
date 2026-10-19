package record

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// ApprovalGate decides whether an action on a document needs approval; approval implements it.
type ApprovalGate interface {
	// Submit opens an approval instance for d when its rules require one.
	Submit(ctx context.Context, d Doc) (ticket int64, needed bool, err error)
	// Withdrawn closes the open instance of d after its submitter took it back.
	Withdrawn(ctx context.Context, d Doc) error
	// Hold opens a request for action on d when a rule requires one; params travel to the run.
	Hold(ctx context.Context, d Doc, action Action, params json.RawMessage) (needed bool, err error)
	// HeldBy returns the open request holding ref, or nil.
	HeldBy(ctx context.Context, ref Ref) (*Hold, error)
	// HeldInPeriod lists the held documents of a legal entity dated on or before until.
	HeldInPeriod(ctx context.Context, legalEntity int64, until pgtype.Date) ([]HeldDoc, error)
}

// Keeper is a core module that keeps data about records of any type (attachments,
// comments) and registers with OnDeleted.
type Keeper interface {
	// Deleted removes what it keeps about a draft being deleted. It runs in the
	// deletion's transaction, after the document's row is gone; an error keeps the draft.
	Deleted(ctx context.Context, ref Ref) error
}

// Freezer is a core module that keeps what a document showed when it was posted (its
// print) and registers with OnPosted.
type Freezer interface {
	// Posted runs in the transaction moving d to posted, after its type's OnTransition;
	// an error keeps the document from being posted.
	Posted(ctx context.Context, d Doc) error
}
