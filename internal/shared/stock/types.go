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

type OnHandFilter struct {
	WarehouseID    int64  `query:"warehouse_id"`
	ItemID         int64  `query:"item_id"`
	Q              string `query:"q" maxLength:"100" doc:"Item code or name, lot or serial code"`
	ExpiringBefore string `query:"expiring_before" format:"date" doc:"Only lots expiring before this date"`
	platform.Paging
}

type OnHandRow struct {
	WarehouseID   int64   `json:"warehouse_id"`
	WarehouseCode string  `json:"warehouse_code"`
	ItemID        int64   `json:"item_id"`
	ItemCode      string  `json:"item_code"`
	ItemName      string  `json:"item_name"`
	Unit          string  `json:"unit"`
	LotCode       *string `json:"lot_code,omitempty"`
	Expiry        *string `json:"expiry,omitempty" format:"date"`
	SerialCode    *string `json:"serial_code,omitempty"`
	Quantity      string  `json:"quantity" doc:"Decimal"`
}

type OnHandList struct {
	Items []OnHandRow `json:"items" nullable:"false"`
	Total int64       `json:"total"`
}

var (
	ErrStockCardRange    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_card_range_invalid"}
	ErrStockCardRangeLen = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_card_range_too_long"}
)

// maxStockCardDays bounds a stock card, counting both ends.
const maxStockCardDays = 366

type StockCardFilter struct {
	WarehouseID int64  `query:"warehouse_id" required:"true"`
	ItemID      int64  `query:"item_id" required:"true"`
	From        string `query:"from" format:"date" required:"true"`
	To          string `query:"to" format:"date" required:"true"`
}

// StockCardRow is one move; Quantity is signed and Balance the running total after it.
type StockCardRow struct {
	Date       string  `json:"date" format:"date"`
	DocType    string  `json:"doc_type"`
	DocID      int64   `json:"doc_id"`
	Number     string  `json:"number"`
	LotCode    *string `json:"lot_code,omitempty"`
	SerialCode *string `json:"serial_code,omitempty"`
	Quantity   string  `json:"quantity" doc:"Decimal, signed"`
	Balance    string  `json:"balance" doc:"Decimal"`
}

type StockCard struct {
	Opening string         `json:"opening" doc:"Decimal: balance before the first day"`
	Rows    []StockCardRow `json:"rows" nullable:"false"`
	Closing string         `json:"closing" doc:"Decimal"`
}
