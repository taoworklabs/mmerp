package stock

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	// Viewing needs the permission anywhere; managing needs it tenant-wide.
	PermView   = "stock.view"
	PermManage = "stock.manage"
)

var ErrWarehouseCodeTaken = &platform.Error{Status: http.StatusConflict, Code: "warehouse_code_taken"}

// WarehouseFields are what an edit may change; code and legal entity are set at creation.
type WarehouseFields struct {
	Name          string `json:"name" minLength:"1" maxLength:"300"`
	LegalEntityID int64  `json:"legal_entity_id" doc:"A company org unit; ignored on update"`
	Active        bool   `json:"active"`
}

type NewWarehouse struct {
	Code string `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the stock.warehouse code series"`
	WarehouseFields
}

type Warehouse struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	IsDefault bool   `json:"is_default"`
	WarehouseFields
}

type WarehouseFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	Sort   string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	platform.Paging
}

type WarehouseList struct {
	Items []Warehouse `json:"items" nullable:"false"`
	Total int64       `json:"total"`
}
