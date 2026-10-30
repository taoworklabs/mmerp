package stock

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var (
	ErrReasonRequired = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_reason_required"}
	// ErrCodeBlank: a lot code or serial of only spaces; params name the line.
	ErrCodeBlank = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_code_blank"}
)

// AdjustmentLine is one line of a stock adjustment: positive brings stock in, negative takes it out.
type AdjustmentLine struct {
	ItemID   int64   `json:"item_id"`
	Lot      *LotRef `json:"lot,omitempty" doc:"Lot-tracked items: created at posting when missing"`
	Serial   *string `json:"serial,omitempty" minLength:"1" maxLength:"100" doc:"Serial-tracked items"`
	Quantity string  `json:"quantity" pattern:"^-?[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"Signed, not 0, up to 3 decimals"`
}

// AdjustmentFields are the editable fields of a stock adjustment.
type AdjustmentFields struct {
	Date        string           `json:"date" format:"date"`
	WarehouseID int64            `json:"warehouse_id"`
	Reason      string           `json:"reason" minLength:"1" maxLength:"500"`
	Lines       []AdjustmentLine `json:"lines" minItems:"1" maxItems:"500" nullable:"false"`
}

type NewAdjustment struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	AdjustmentFields
}

type AdjustmentUpdate struct {
	Version int32 `json:"version"`
	AdjustmentFields
}

type AdjustmentLineOut struct {
	AdjustmentLine
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name"`
	Unit     string `json:"unit"`
}

type Adjustment struct {
	ID            int64               `json:"id"`
	Number        string              `json:"number"`
	Status        string              `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32               `json:"version"`
	Date          string              `json:"date" format:"date"`
	LegalEntityID int64               `json:"legal_entity_id"`
	WarehouseID   int64               `json:"warehouse_id"`
	WarehouseCode string              `json:"warehouse_code"`
	WarehouseName string              `json:"warehouse_name"`
	Reason        string              `json:"reason"`
	Lines         []AdjustmentLineOut `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type AdjustmentFilter struct {
	Q           string `query:"q" maxLength:"100" doc:"Number or reason"`
	Status      string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	WarehouseID int64  `query:"warehouse_id"`
	Sort        string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type AdjustmentListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date          string `json:"date" format:"date"`
	WarehouseID   int64  `json:"warehouse_id"`
	WarehouseCode string `json:"warehouse_code"`
	WarehouseName string `json:"warehouse_name"`
	Reason        string `json:"reason"`
}

type AdjustmentList struct {
	Items []AdjustmentListItem `json:"items" nullable:"false"`
	Total int64                `json:"total"`
}

// AdjustmentActions are what the actor may do before picking an adjustment.
type AdjustmentActions struct {
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}
