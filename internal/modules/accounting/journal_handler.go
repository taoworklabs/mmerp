package accounting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type journalEntriesOutput struct{ Body JournalEntryList }

type journalEntryOutput struct{ Body JournalEntry }

type journalActionsOutput struct{ Body JournalEntryActions }

type newJournalEntryInput struct{ Body NewJournalEntry }

type updateJournalEntryInput struct {
	ID   int64 `path:"id"`
	Body JournalEntryUpdate
}

func registerJournalHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-journal-entries", Method: http.MethodGet, Path: "/accounting/journal-entries"},
		func(ctx context.Context, in *JournalEntryFilter) (*journalEntriesOutput, error) {
			l, err := s.JournalEntries(ctx, *in)
			return &journalEntriesOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-journal-entry-actions", Method: http.MethodGet, Path: "/accounting/journal-entries/actions"},
		func(ctx context.Context, _ *struct{}) (*journalActionsOutput, error) {
			a, err := s.JournalEntryActions(ctx)
			return &journalActionsOutput{Body: a}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-journal-entry", Method: http.MethodPost, Path: "/accounting/journal-entries", DefaultStatus: http.StatusCreated,
		Description: "Sending the same request_id again returns the document created the first time."},
		func(ctx context.Context, in *newJournalEntryInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateJournalEntry(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-journal-entry", Method: http.MethodGet, Path: "/accounting/journal-entries/{id}"},
		func(ctx context.Context, in *platform.IDInput) (*journalEntryOutput, error) {
			e, err := s.JournalEntry(ctx, in.ID)
			return &journalEntryOutput{Body: e}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-journal-entry", Method: http.MethodPut, Path: "/accounting/journal-entries/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateJournalEntryInput) (*struct{}, error) {
			return nil, s.UpdateJournalEntry(ctx, in.ID, in.Body)
		})
}
