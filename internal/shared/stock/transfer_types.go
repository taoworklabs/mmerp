package stock

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// ErrTransferSameWarehouse: a transfer's source and destination are one warehouse.
var ErrTransferSameWarehouse = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_transfer_same_warehouse"}

// TransferLine is one line of a transfer: a quantity out of the source and into the destination.
type TransferLine struct {
	ItemID   int64   `json:"item_id"`
	Lot      *string `json:"lot,omitempty" minLength:"1" maxLength:"100" doc:"Lot code, for lot-tracked items"`
	Serial   *string `json:"serial,omitempty" minLength:"1" maxLength:"100" doc:"Serial-tracked items"`
	Quantity string  `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"Above 0, up to 3 decimals"`
}

// TransferFields are the editable fields of a transfer.
type TransferFields struct {
	Date            string         `json:"date" format:"date"`
	FromWarehouseID int64          `json:"from_warehouse_id"`
	ToWarehouseID   int64          `json:"to_warehouse_id"`
	Note            string         `json:"note" maxLength:"500"`
	Lines           []TransferLine `json:"lines" minItems:"1" maxItems:"500" nullable:"false"`
}

type NewTransfer struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	TransferFields
}

type TransferUpdate struct {
	Version int32 `json:"version"`
	TransferFields
}

type TransferLineOut struct {
	TransferLine
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name"`
	Unit     string `json:"unit"`
}

type Transfer struct {
	ID                int64             `json:"id"`
	Number            string            `json:"number"`
	Status            string            `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version           int32             `json:"version"`
	Date              string            `json:"date" format:"date"`
	LegalEntityID     int64             `json:"legal_entity_id"`
	FromWarehouseID   int64             `json:"from_warehouse_id"`
	FromWarehouseCode string            `json:"from_warehouse_code"`
	FromWarehouseName string            `json:"from_warehouse_name"`
	ToWarehouseID     int64             `json:"to_warehouse_id"`
	ToWarehouseCode   string            `json:"to_warehouse_code"`
	ToWarehouseName   string            `json:"to_warehouse_name"`
	Note              string            `json:"note"`
	Lines             []TransferLineOut `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type TransferFilter struct {
	Q           string `query:"q" maxLength:"100" doc:"Number or note"`
	Status      string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	WarehouseID int64  `query:"warehouse_id" doc:"Source or destination"`
	Sort        string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type TransferListItem struct {
	ID                int64  `json:"id"`
	Number            string `json:"number"`
	Status            string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date              string `json:"date" format:"date"`
	FromWarehouseID   int64  `json:"from_warehouse_id"`
	FromWarehouseCode string `json:"from_warehouse_code"`
	FromWarehouseName string `json:"from_warehouse_name"`
	ToWarehouseID     int64  `json:"to_warehouse_id"`
	ToWarehouseCode   string `json:"to_warehouse_code"`
	ToWarehouseName   string `json:"to_warehouse_name"`
	Note              string `json:"note"`
}

type TransferList struct {
	Items []TransferListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}
