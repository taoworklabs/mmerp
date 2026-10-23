package hrm

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/taoworklabs/mmerp/internal/platform"
)

type contractTypesOutput struct{ Body []ContractType }

type contractTypeInput struct{ Body ContractTypeInput }

type updateContractTypeInput struct {
	ID   int64 `path:"id"`
	Body ContractTypeInput
}

type contractsOutput struct{ Body []ContractListItem }

type contractOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         Contract
}

type newContractInput struct{ Body NewContract }

type updateContractInput struct {
	ID   int64 `path:"id"`
	Body ContractUpdate
}

type contractListOutput struct{ Body ContractList }

func registerContractHandlers(api huma.API, s *Service) {
	huma.Register(api, huma.Operation{OperationID: "list-contract-types", Method: http.MethodGet, Path: "/hrm/contract-types"},
		func(ctx context.Context, _ *struct{}) (*contractTypesOutput, error) {
			ts, err := s.ContractTypes(ctx)
			return &contractTypesOutput{Body: ts}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-contract-type", Method: http.MethodPost, Path: "/hrm/contract-types", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *contractTypeInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.SaveContractType(ctx, 0, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "update-contract-type", Method: http.MethodPut, Path: "/hrm/contract-types/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateContractTypeInput) (*struct{}, error) {
			_, err := s.SaveContractType(ctx, in.ID, in.Body)
			return nil, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-contracts", Method: http.MethodGet, Path: "/hrm/contracts",
		Description: "Every contract and appendix in scope; without terms."},
		func(ctx context.Context, in *ContractFilter) (*contractListOutput, error) {
			l, err := s.Contracts(ctx, *in)
			return &contractListOutput{Body: l}, err
		})
	huma.Register(api, huma.Operation{OperationID: "list-employee-contracts", Method: http.MethodGet, Path: "/hrm/employees/{id}/contracts",
		Description: "Originals, each followed by its appendices; without terms."},
		func(ctx context.Context, in *platform.IDInput) (*contractsOutput, error) {
			cs, err := s.EmployeeContracts(ctx, in.ID)
			return &contractsOutput{Body: cs}, err
		})
	huma.Register(api, huma.Operation{OperationID: "create-contract", Method: http.MethodPost, Path: "/hrm/contracts", DefaultStatus: http.StatusCreated},
		func(ctx context.Context, in *newContractInput) (*platform.CreatedOutput, error) {
			return platform.Created(s.CreateContract(ctx, in.Body))
		})
	huma.Register(api, huma.Operation{OperationID: "get-contract", Method: http.MethodGet, Path: "/hrm/contracts/{id}",
		Description: "Terms come with hrm.salary.view; each such read is audited."},
		func(ctx context.Context, in *platform.IDInput) (*contractOutput, error) {
			c, err := s.Contract(ctx, in.ID)
			return &contractOutput{CacheControl: "no-store", Body: c}, err
		})
	huma.Register(api, huma.Operation{OperationID: "update-contract", Method: http.MethodPut, Path: "/hrm/contracts/{id}", DefaultStatus: http.StatusNoContent},
		func(ctx context.Context, in *updateContractInput) (*struct{}, error) {
			return nil, s.UpdateContract(ctx, in.ID, in.Body)
		})
}
