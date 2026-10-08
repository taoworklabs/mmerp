package record

import "context"

// ApprovalGate decides whether sending a document needs approval; approval implements it.
type ApprovalGate interface {
	// Submit opens an approval instance for d when its rules require one.
	Submit(ctx context.Context, d Doc) (ticket int64, needed bool, err error)
	// Withdrawn closes the open instance of d after its submitter took it back.
	Withdrawn(ctx context.Context, d Doc) error
}

// Keeper is a core module that keeps data about records of any type (attachments,
// comments) and registers with OnDeleted.
type Keeper interface {
	// Deleted removes what it keeps about a draft being deleted. It runs in the
	// deletion's transaction, after the document's row is gone; an error keeps the draft.
	Deleted(ctx context.Context, ref Ref) error
}
