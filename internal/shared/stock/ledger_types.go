package stock

import (
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/taoworklabs/mmerp/internal/platform"
)

var (
	// ErrInsufficient: a balance would go below zero; params name the key and what is missing.
	ErrInsufficient      = &platform.Error{Status: http.StatusConflict, Code: "stock_insufficient"}
	ErrAlreadyRecorded   = &platform.Error{Status: http.StatusConflict, Code: "stock_already_recorded"}
	ErrWarehouseEntity   = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_warehouse_entity"}
	ErrWarehouseInactive = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_warehouse_inactive"}
	ErrUntracked         = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_untracked"}
	ErrLotNotTracked     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_lot_not_tracked"}
	ErrLotRequired       = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_lot_required"}
	ErrLotExpiryMismatch = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_lot_expiry_mismatch"}
	ErrSerialRequired    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_serial_required"}
	// ErrSerialQuantity: a serial moves by one, and is on hand at most once.
	ErrSerialQuantity = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_serial_quantity"}
	// ErrLotNotFound, ErrSerialNotFound: an outgoing move names a lot or serial the item does not have.
	ErrLotNotFound    = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_lot_not_found"}
	ErrSerialNotFound = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_serial_not_found"}
	// ErrQuantityInvalid: zero, more than three decimals, or too large for a balance.
	ErrQuantityInvalid = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_quantity_invalid"}
	// ErrCountDuplicate: two counts of one item, lot or serial; params name the second line.
	ErrCountDuplicate = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "stock_count_duplicate"}
)

// Move is one document line's effect on stock: positive in, negative out.
type Move struct {
	LineNo      int
	WarehouseID int64
	ItemID      int64
	Lot         *LotRef
	Serial      *string
	Quantity    decimal.Decimal
}

// Count is what a stocktake found for one key of one warehouse; Quantity ≥ 0.
type Count struct {
	LineNo   int
	ItemID   int64
	Lot      *LotRef
	Serial   *string
	Quantity decimal.Decimal
}

// CountResult is a count's on hand when it posted.
type CountResult struct {
	LineNo int
	Book   decimal.Decimal
}

// LotRef names a lot by code; a move creates it when missing, with Expiry (YYYY-MM-DD).
type LotRef struct {
	Code   string  `json:"code" minLength:"1" maxLength:"100"`
	Expiry *string `json:"expiry,omitempty" format:"date"`
}

// LotPick is a lot suggested for an outgoing quantity.
type LotPick struct {
	LotID    int64
	Code     string
	Expiry   *string
	Quantity decimal.Decimal
}
