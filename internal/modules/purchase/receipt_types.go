package purchase

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/shared/stock"
)

// ReceiptLineInput brings in a quantity of one order line, into a lot or as a serial when the item has them.
type ReceiptLineInput struct {
	OrderPosition int           `json:"order_position" minimum:"1"`
	Quantity      string        `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"More than 0, up to 3 decimals; 1 for a serial"`
	Lot           *stock.LotRef `json:"lot,omitempty" doc:"Lot-tracked items: created at posting when missing"`
	Serial        *string       `json:"serial,omitempty" maxLength:"100" doc:"Serial-tracked items: one per line"`
}

type ReceiptFields struct {
	Date        string             `json:"date" format:"date"`
	WarehouseID int64              `json:"warehouse_id"`
	Note        *string            `json:"note" maxLength:"2000"`
	Lines       []ReceiptLineInput `json:"lines" minItems:"1" maxItems:"1000" nullable:"false"`
}

type NewGoodsReceipt struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first goods receipt"`
	OrderID   int64  `json:"order_id"`
	ReceiptFields
}

type GoodsReceiptUpdate struct {
	Version int32 `json:"version"`
	ReceiptFields
}

// ReceiptLine is a line as saved; item and texts are the order line's.
type ReceiptLine struct {
	ReceiptLineInput
	ItemID      int64  `json:"item_id"`
	ItemCode    string `json:"item_code"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

type GoodsReceipt struct {
	ID            int64         `json:"id"`
	OrderID       int64         `json:"order_id"`
	OrderNumber   string        `json:"order_number"`
	Number        string        `json:"number"`
	Status        string        `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32         `json:"version"`
	Date          string        `json:"date" format:"date"`
	OrgUnitID     int64         `json:"org_unit_id"`
	WarehouseID   int64         `json:"warehouse_id"`
	WarehouseCode string        `json:"warehouse_code"`
	WarehouseName string        `json:"warehouse_name"`
	Note          *string       `json:"note"`
	Lines         []ReceiptLine `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type GoodsReceiptFilter struct {
	Q           string `query:"q" maxLength:"100" doc:"Number, order number or supplier name"`
	Status      string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrderID     int64  `query:"order_id"`
	WarehouseID int64  `query:"warehouse_id"`
	Sort        string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type GoodsReceiptListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date          string `json:"date" format:"date"`
	OrderID       int64  `json:"order_id"`
	OrderNumber   string `json:"order_number"`
	SupplierID    int64  `json:"supplier_id"`
	SupplierName  string `json:"supplier_name"`
	WarehouseID   int64  `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
	OrgUnitName   string `json:"org_unit_name"`
}

type GoodsReceiptList struct {
	Items []GoodsReceiptListItem `json:"items" nullable:"false"`
	Total int64                  `json:"total"`
}

type WarehouseOption struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// ReceiptOptionLine is a stock line of the order with what is left to receive.
type ReceiptOptionLine struct {
	OrderPosition int    `json:"order_position"`
	ItemID        int64  `json:"item_id"`
	ItemCode      string `json:"item_code"`
	Description   string `json:"description"`
	Unit          string `json:"unit"`
	Tracking      string `json:"tracking" enum:"none,quantity,lot,serial"`
	Ordered       string `json:"ordered"`
	Remaining     string `json:"remaining"`
}

type ReceiptOptions struct {
	Warehouses []WarehouseOption   `json:"warehouses" nullable:"false"`
	Lines      []ReceiptOptionLine `json:"lines" nullable:"false"`
}

var (
	ErrReceiptOverOrder    = &platform.Error{Status: http.StatusConflict, Code: "purchase_receipt_over_order"}
	ErrReceiptLineNotStock = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_receipt_line_not_stock"}
	// The order has no line at the given order_position.
	ErrReceiptPosition = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_receipt_position"}
	// The warehouse is inactive or of another legal entity than the order.
	ErrReceiptWarehouse   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_receipt_warehouse"}
	ErrReceiptBeforeOrder = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "purchase_receipt_before_order"}
)
