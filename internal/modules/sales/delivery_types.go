package sales

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// DeliveryLineInput takes a quantity of one order line, from one lot or serial when the item has them.
type DeliveryLineInput struct {
	OrderPosition int     `json:"order_position" minimum:"1"`
	LotCode       *string `json:"lot_code,omitempty" maxLength:"100"`
	SerialCode    *string `json:"serial_code,omitempty" maxLength:"100"`
	Quantity      string  `json:"quantity" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"More than 0, up to 3 decimals"`
}

type DeliveryFields struct {
	Date        string              `json:"date" format:"date"`
	WarehouseID int64               `json:"warehouse_id"`
	Note        *string             `json:"note" maxLength:"2000"`
	Lines       []DeliveryLineInput `json:"lines" minItems:"1" maxItems:"500" nullable:"false"`
}

type NewDelivery struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first delivery"`
	OrderID   int64  `json:"order_id"`
	DeliveryFields
}

type DeliveryUpdate struct {
	Version int32 `json:"version"`
	DeliveryFields
}

// DeliveryLine is a line as saved; item and texts are the order line's.
type DeliveryLine struct {
	DeliveryLineInput
	ItemID      int64  `json:"item_id"`
	ItemCode    string `json:"item_code"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

type Delivery struct {
	ID            int64          `json:"id"`
	OrderID       int64          `json:"order_id"`
	OrderNumber   string         `json:"order_number"`
	Number        string         `json:"number"`
	Status        string         `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32          `json:"version"`
	Date          string         `json:"date" format:"date"`
	OrgUnitID     int64          `json:"org_unit_id"`
	WarehouseID   int64          `json:"warehouse_id"`
	WarehouseCode string         `json:"warehouse_code"`
	WarehouseName string         `json:"warehouse_name"`
	Note          *string        `json:"note"`
	Lines         []DeliveryLine `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

var (
	ErrOrderNotPosted       = &platform.Error{Status: http.StatusConflict, Code: "sales_order_not_posted"}
	ErrDeliveryOverOrdered  = &platform.Error{Status: http.StatusConflict, Code: "sales_delivery_over_ordered"}
	ErrDeliveryLineNotStock = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_delivery_line_not_stock"}
	// The warehouse is inactive or of another legal entity than the order.
	ErrDeliveryWarehouse = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_delivery_warehouse"}
)
