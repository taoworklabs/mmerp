package stock

import (
	"context"
	"encoding/json"

	"github.com/taoworklabs/mmerp/internal/core/printing"
)

func (s *Service) registerStocktakePrint() {
	s.d.Printing.Register(printing.Template{Code: stocktakeType, Name: "stock.print.stocktake.name", DocType: stocktakeType,
		Blocks: []printing.Block{{Key: "footer", Label: "stock.print.block.footer", Default: "stock.print.stocktake.footer.default"}},
		Data:   s.stocktakePrintData, Layouts: []printing.Layout{stocktakeLayout1}})
}

// stocktakePrint is what a stocktake prints.
type stocktakePrint struct {
	Stocktake
	Company printing.LegalEntity `json:"company"`
}

// stocktakePrintData reads a stocktake as printed; printing checked the actor may print it.
func (s *Service) stocktakePrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	t, err := readStocktake(ctx, id)
	if err != nil {
		return nil, err
	}
	le, err := s.d.Printing.LegalEntity(ctx, t.LegalEntityID)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(stocktakePrint{Stocktake: t, Company: le})
	return []printing.Part{{Data: b}}, err
}

// stocktakeLayout1 prints a count sheet: company, warehouse, note, lines with book, counted and
// difference once posted, footer.
func stocktakeLayout1(p *printing.Page, raw json.RawMessage) error {
	var d stocktakePrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t := func(k string) string { return p.T("stock.print."+k, nil) }
	p.Title(t("stocktake.title"))
	p.Center(p.T("stock.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.LegalEntity(t("company"), t("name"), d.Company)
	p.Field(t("warehouse"), d.WarehouseCode+" · "+d.WarehouseName)
	if d.Note != "" {
		p.Field(t("note"), d.Note)
	}
	cols := []printing.Column{{Title: t("item_code"), Share: 0.12}, {Title: t("item_name"), Share: 0.28}, {Title: t("unit"), Share: 0.08},
		{Title: t("lot"), Share: 0.12}, {Title: t("serial"), Share: 0.1}, {Title: t("book"), Share: 0.1, Right: true},
		{Title: t("counted"), Share: 0.1, Right: true}, {Title: t("difference"), Share: 0.1, Right: true}}
	rows := make([][]string, len(d.Lines))
	dec := func(v *string) string {
		if v == nil {
			return ""
		}
		return p.Decimal(*v)
	}
	for i, l := range d.Lines {
		var lot string
		if l.Lot != nil {
			lot = l.Lot.Code
		}
		rows[i] = []string{l.ItemCode, l.ItemName, l.Unit, lot, deref(l.Serial), dec(l.Book), p.Decimal(l.Counted), dec(l.Difference)}
	}
	p.Heading(t("lines"))
	p.Table(cols, rows, false)
	p.Gap()
	p.Block("footer", nil)
	return nil
}
