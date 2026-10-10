package platform

// Paging is the page a list asks for; list filters embed it.
type Paging struct {
	Page     int `query:"page" minimum:"1" default:"1"`
	PageSize int `query:"page_size" enum:"20,50,100" default:"50"`
}

// Limit and Offset are the page in SQL terms.
func (p Paging) Limit() int32  { return int32(p.PageSize) }
func (p Paging) Offset() int32 { return int32((p.Page - 1) * p.PageSize) }
