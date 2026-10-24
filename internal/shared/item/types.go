package item

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

const (
	// Viewing needs the permission anywhere; managing needs it tenant-wide.
	PermView   = "item.view"
	PermManage = "item.manage"
)

var ErrCodeTaken = &platform.Error{Status: http.StatusConflict, Code: "item_code_taken"}

// ItemFields are what an edit may change; the code is set once, at creation.
type ItemFields struct {
	Name    string `json:"name" minLength:"1" maxLength:"300"`
	Unit    string `json:"unit" minLength:"1" maxLength:"30"`
	Price   int64  `json:"price" minimum:"0" doc:"Default unit price in đồng"`
	VatRate string `json:"vat_rate" enum:"none,0,5,8,10"`
	Active  bool   `json:"active"`
}

type NewItem struct {
	Code string `json:"code,omitempty" maxLength:"50" doc:"Empty or absent: issued from the item.item code series"`
	ItemFields
}

type Item struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	ItemFields
}

type ItemFilter struct {
	Q      string `query:"q" maxLength:"100"`
	Active string `query:"active" enum:"true,false"`
	Sort   string `query:"sort" enum:"code,-code,name,-name" default:"code"`
	platform.Paging
}

type ItemList struct {
	Items []Item `json:"items" nullable:"false"`
	Total int64  `json:"total"`
}
