package approval

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/core/record"
	"github.com/taoworklabs/mmerp/internal/platform"
)

type inboxOutput struct {
	Body struct {
		Items []InboxItem `json:"items" nullable:"false"`
	}
}

type documentInput struct {
	Type string `path:"type"`
	ID   int64  `path:"id"`
}

type instanceOutput struct {
	Body struct {
		Instance *Instance `json:"instance,omitempty" doc:"The latest submission; absent when never sent for approval"`
	}
}

// Step in each body is the step the actor saw; another step fails with approval_closed.
type approveInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Step int32 `json:"step"`
	}
}

type rejectInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Step   int32  `json:"step"`
		Reason string `json:"reason" minLength:"1" maxLength:"1000"`
	}
}

type reassignInput struct {
	ID   int64 `path:"id"`
	Body struct {
		Step  int32  `json:"step"`
		Login string `json:"login" minLength:"1" maxLength:"200"`
	}
}

type rulesOutput struct{ Body []TypeRule }

type rulesInput struct {
	Product string `query:"product" doc:"Only the document types of this product"`
}

type usersOutput struct{ Body []ApproverUser }

type docTypeInput struct {
	DocType string `path:"doc_type"`
}

type saveRuleInput struct {
	DocType string `path:"doc_type"`
	Body    RuleInput
}

func registerHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "get-inbox", Method: http.MethodGet, Path: "/approvals/inbox"},
		func(ctx context.Context, _ *struct{}) (*inboxOutput, error) {
			items, err := s.Inbox(ctx)
			out := &inboxOutput{}
			out.Body.Items = items
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "get-document-approval", Method: http.MethodGet, Path: "/documents/{type}/{id}/approval"},
		func(ctx context.Context, in *documentInput) (*instanceOutput, error) {
			if _, ok := s.d.Record.TypeOf(in.Type); !ok {
				return nil, platform.ErrNotFound
			}
			inst, err := s.DocumentInstance(ctx, record.Ref{Type: in.Type, ID: in.ID})
			out := &instanceOutput{}
			out.Body.Instance = inst
			return out, err
		})
	huma.Register(api, huma.Operation{OperationID: "approve", Method: http.MethodPost, Path: "/approvals/{id}/approve", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *approveInput) (*struct{}, error) {
			return nil, s.Approve(ctx, in.ID, in.Body.Step)
		})
	huma.Register(api, huma.Operation{OperationID: "reject", Method: http.MethodPost, Path: "/approvals/{id}/reject", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *rejectInput) (*struct{}, error) {
			return nil, s.Reject(ctx, in.ID, in.Body.Step, in.Body.Reason)
		})
	huma.Register(api, huma.Operation{OperationID: "reassign", Method: http.MethodPost, Path: "/approvals/{id}/reassign", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *reassignInput) (*struct{}, error) {
			return nil, s.Reassign(ctx, in.ID, in.Body.Step, in.Body.Login)
		})
	huma.Register(api, huma.Operation{OperationID: "list-approval-rules", Method: http.MethodGet, Path: "/approval-rules"},
		func(ctx context.Context, in *rulesInput) (*rulesOutput, error) {
			r, err := s.Rules(ctx, in.Product)
			return &rulesOutput{Body: r}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-approval-rule-users", Method: http.MethodGet, Path: "/approval-rules/users"},
		func(ctx context.Context, _ *struct{}) (*usersOutput, error) {
			u, err := s.Users(ctx)
			return &usersOutput{Body: u}, err
		})
	huma.Register(api, huma.Operation{OperationID: "save-approval-rule", Method: http.MethodPut, Path: "/approval-rules/{doc_type}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *saveRuleInput) (*struct{}, error) {
			return nil, s.SaveRule(ctx, in.DocType, in.Body)
		})
	huma.Register(api, huma.Operation{OperationID: "delete-approval-rule", Method: http.MethodDelete, Path: "/approval-rules/{doc_type}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *docTypeInput) (*struct{}, error) {
			return nil, s.DeleteRule(ctx, in.DocType)
		})
}
