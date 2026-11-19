package stock

import "github.com/taoworklabs/mmerp/internal/platform"

// StocktakeLine is what was counted of one item, lot or serial.
type StocktakeLine struct {
	ItemID  int64   `json:"item_id"`
	Lot     *LotRef `json:"lot,omitempty" doc:"Lot-tracked items: a lot found on the shelf is created at posting"`
	Serial  *string `json:"serial,omitempty" minLength:"1" maxLength:"100" doc:"Serial-tracked items"`
	Counted string  `json:"counted" pattern:"^[0-9]{1,12}(\\.[0-9]{1,3})?$" doc:"0 or more, up to 3 decimals; a serial counts 0 or 1"`
}

// StocktakeFields are the editable fields of a stocktake.
type StocktakeFields struct {
	Date        string          `json:"date" format:"date"`
	WarehouseID int64           `json:"warehouse_id"`
	Note        string          `json:"note" maxLength:"500"`
	Lines       []StocktakeLine `json:"lines" minItems:"1" maxItems:"2000" nullable:"false"`
}

type NewStocktake struct {
	RequestID string `json:"request_id" format:"uuid" doc:"Generated once per form: sending it again returns the first document"`
	StocktakeFields
}

type StocktakeUpdate struct {
	Version int32 `json:"version"`
	StocktakeFields
}

type StocktakeLineOut struct {
	StocktakeLine
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name"`
	Unit     string `json:"unit"`
	// Current on hand, until the stocktake posts.
	OnHand *string `json:"on_hand,omitempty"`
	// On hand when it posted, and counted less that.
	Book       *string `json:"book,omitempty"`
	Difference *string `json:"difference,omitempty"`
}

type Stocktake struct {
	ID            int64              `json:"id"`
	Number        string             `json:"number"`
	Status        string             `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Version       int32              `json:"version"`
	Date          string             `json:"date" format:"date"`
	LegalEntityID int64              `json:"legal_entity_id"`
	WarehouseID   int64              `json:"warehouse_id"`
	WarehouseCode string             `json:"warehouse_code"`
	WarehouseName string             `json:"warehouse_name"`
	Note          string             `json:"note"`
	Lines         []StocktakeLineOut `json:"lines" nullable:"false"`
	// edit, delete, submit, withdraw, cancel, print.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

type StocktakeFilter struct {
	Q           string `query:"q" maxLength:"100" doc:"Number or note"`
	Status      string `query:"status" enum:"draft,pending_approval,posted,cancelled"`
	WarehouseID int64  `query:"warehouse_id"`
	Sort        string `query:"sort" enum:"date,-date,number,-number" default:"-date"`
	platform.Paging
}

type StocktakeListItem struct {
	ID            int64  `json:"id"`
	Number        string `json:"number"`
	Status        string `json:"status" enum:"draft,pending_approval,posted,cancelled"`
	Date          string `json:"date" format:"date"`
	WarehouseID   int64  `json:"warehouse_id"`
	WarehouseCode string `json:"warehouse_code"`
	WarehouseName string `json:"warehouse_name"`
	Note          string `json:"note"`
}

type StocktakeList struct {
	Items []StocktakeListItem `json:"items" nullable:"false"`
	Total int64               `json:"total"`
}

// OnHandLine is a balance as a stocktake line counting what is on hand.
type OnHandLine struct {
	StocktakeLine
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name"`
	Unit     string `json:"unit"`
}

// OnHandLines are a warehouse's balances as stocktake lines.
type OnHandLines struct {
	Items []OnHandLine `json:"items" nullable:"false"`
	// The warehouse holds more rows than a stocktake takes; only the first are listed.
	Truncated bool `json:"truncated"`
}

type OnHandLinesFilter struct {
	ID     int64 `path:"id"`
	ItemID int64 `query:"item_id" doc:"Only this item"`
}
