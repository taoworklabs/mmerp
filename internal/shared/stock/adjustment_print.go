package stock

import (
	"context"
	"encoding/json"

	"github.com/taoworklabs/mmerp/internal/core/printing"
)

func (s *Service) registerPrints() {
	s.d.Printing.Register(printing.Template{Code: adjustmentType, Name: "stock.print.adjustment.name", DocType: adjustmentType,
		Blocks: []printing.Block{{Key: "footer", Label: "stock.print.block.footer", Default: "stock.print.block.footer.default"}},
		Data:   s.adjustmentPrintData, Layouts: []printing.Layout{adjustmentLayout1}})
}

// adjustmentPrint is what an adjustment prints.
type adjustmentPrint struct {
	Adjustment
	Company printing.LegalEntity `json:"company"`
}

// adjustmentPrintData reads an adjustment as printed; printing checked the actor may print it.
func (s *Service) adjustmentPrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	a, err := readAdjustment(ctx, id)
	if err != nil {
		return nil, err
	}
	le, err := s.d.Printing.LegalEntity(ctx, a.LegalEntityID)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(adjustmentPrint{Adjustment: a, Company: le})
	return []printing.Part{{Data: b}}, err
}

// adjustmentLayout1 prints an adjustment: company, warehouse, reason, lines, footer.
func adjustmentLayout1(p *printing.Page, raw json.RawMessage) error {
	var d adjustmentPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t := func(k string) string { return p.T("stock.print."+k, nil) }
	p.Title(t("adjustment.title"))
	p.Center(p.T("stock.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.LegalEntity(t("company"), t("name"), d.Company)
	p.Field(t("warehouse"), d.WarehouseCode+" · "+d.WarehouseName)
	p.Field(t("reason"), d.Reason)
	cols := []printing.Column{{Title: t("item_code"), Share: 0.14}, {Title: t("item_name"), Share: 0.32}, {Title: t("unit"), Share: 0.1},
		{Title: t("lot"), Share: 0.14}, {Title: t("expiry"), Share: 0.12}, {Title: t("serial"), Share: 0.08}, {Title: t("quantity"), Share: 0.1, Right: true}}
	rows := make([][]string, len(d.Lines))
	for i, l := range d.Lines {
		var lot, expiry, serial string
		if l.Lot != nil {
			lot = l.Lot.Code
			if l.Lot.Expiry != nil {
				expiry = p.Date(*l.Lot.Expiry)
			}
		}
		if l.Serial != nil {
			serial = *l.Serial
		}
		rows[i] = []string{l.ItemCode, l.ItemName, l.Unit, lot, expiry, serial, p.Decimal(l.Quantity)}
	}
	p.Heading(t("lines"))
	p.Table(cols, rows, false)
	p.Gap()
	p.Block("footer", nil)
	return nil
}
