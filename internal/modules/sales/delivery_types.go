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
	ErrOrderHasDelivery     = &platform.Error{Status: http.StatusConflict, Code: "sales_order_has_delivery"}
	ErrOrderNotPosted       = &platform.Error{Status: http.StatusConflict, Code: "sales_order_not_posted"}
	ErrDeliveryOverOrdered  = &platform.Error{Status: http.StatusConflict, Code: "sales_delivery_over_ordered"}
	ErrDeliveryLineNotStock = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_delivery_line_not_stock"}
	// The warehouse is inactive or of another legal entity than the order.
	ErrDeliveryWarehouse = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "sales_delivery_warehouse"}
)

type DeliveryFilter struct {
	Q           string `query:"q" maxLength:"100" doc:"Number, order number or customer name"`
	Status      string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	OrderID     int64  `query:"order_id"`
	WarehouseID int64  `query:"warehouse_id"`
	Sort        string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type DeliveryListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date          string `json:"date" format:"date"`
	OrderID       int64  `json:"order_id"`
	OrderNumber   string `json:"order_number"`
	CustomerID    int64  `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	WarehouseID   int64  `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
	OrgUnitName   string `json:"org_unit_name"`
}

type DeliveryList struct {
	Items []DeliveryListItem `json:"items" nullable:"false"`
	Total int64              `json:"total"`
}

// ItemAvailability: available is on hand less committed, and may be negative.
type ItemAvailability struct {
	ItemID    int64  `json:"item_id"`
	OnHand    string `json:"on_hand"`
	Committed string `json:"committed" doc:"Still to deliver on posted orders"`
	Available string `json:"available"`
}

type WarehouseOption struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// DeliveryOptionLine is a stock line of the order with what is left to deliver and where it
// could come from: suggested lots (first expiring first) or serials on hand.
type DeliveryOptionLine struct {
	OrderPosition int           `json:"order_position"`
	ItemID        int64         `json:"item_id"`
	ItemCode      string        `json:"item_code"`
	Description   string        `json:"description"`
	Unit          string        `json:"unit"`
	Tracking      string        `json:"tracking" enum:"quantity,lot,serial"`
	Ordered       string        `json:"ordered"`
	Remaining     string        `json:"remaining"`
	Lots          []DeliveryLot `json:"lots" nullable:"false"`
	Serials       []string      `json:"serials" nullable:"false"`
}

type DeliveryLot struct {
	Code     string  `json:"code"`
	Expiry   *string `json:"expiry" format:"date"`
	Quantity string  `json:"quantity"`
}

type DeliveryOptions struct {
	Warehouses  []WarehouseOption    `json:"warehouses" nullable:"false"`
	WarehouseID int64                `json:"warehouse_id" doc:"The warehouse the suggestions are for; 0 when the legal entity has none"`
	Lines       []DeliveryOptionLine `json:"lines" nullable:"false"`
}
