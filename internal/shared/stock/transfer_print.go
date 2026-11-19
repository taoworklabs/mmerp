package stock

import (
	"context"
	"encoding/json"

	"github.com/taoworklabs/mmerp/internal/core/printing"
)

func (s *Service) registerTransferPrint() {
	s.d.Printing.Register(printing.Template{Code: transferType, Name: "stock.print.transfer.name", DocType: transferType,
		Blocks: []printing.Block{{Key: "footer", Label: "stock.print.block.footer", Default: "stock.print.transfer.footer.default"}},
		Data:   s.transferPrintData, Layouts: []printing.Layout{transferLayout1}})
}

// transferPrint is what a transfer prints.
type transferPrint struct {
	Transfer
	Company printing.LegalEntity `json:"company"`
}

// transferPrintData reads a transfer as printed; printing checked the actor may print it.
func (s *Service) transferPrintData(ctx context.Context, id int64) ([]printing.Part, error) {
	t, err := readTransfer(ctx, id)
	if err != nil {
		return nil, err
	}
	le, err := s.d.Printing.LegalEntity(ctx, t.LegalEntityID)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(transferPrint{Transfer: t, Company: le})
	return []printing.Part{{Data: b}}, err
}

// transferLayout1 prints a transfer: company, both warehouses, note, lines, footer.
func transferLayout1(p *printing.Page, raw json.RawMessage) error {
	var d transferPrint
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	t := func(k string) string { return p.T("stock.print."+k, nil) }
	p.Title(t("transfer.title"))
	p.Center(p.T("stock.print.number", map[string]any{"number": d.Number, "date": p.Date(d.Date)}))
	p.LegalEntity(t("company"), t("name"), d.Company)
	p.Field(t("from_warehouse"), d.FromWarehouseCode+" · "+d.FromWarehouseName)
	p.Field(t("to_warehouse"), d.ToWarehouseCode+" · "+d.ToWarehouseName)
	if d.Note != "" {
		p.Field(t("note"), d.Note)
	}
	cols := []printing.Column{{Title: t("item_code"), Share: 0.14}, {Title: t("item_name"), Share: 0.36}, {Title: t("unit"), Share: 0.1},
		{Title: t("lot"), Share: 0.15}, {Title: t("serial"), Share: 0.13}, {Title: t("quantity"), Share: 0.12, Right: true}}
	rows := make([][]string, len(d.Lines))
	for i, l := range d.Lines {
		rows[i] = []string{l.ItemCode, l.ItemName, l.Unit, deref(l.Lot), deref(l.Serial), p.Decimal(l.Quantity)}
	}
	p.Heading(t("lines"))
	p.Table(cols, rows, false)
	p.Gap()
	p.Block("footer", nil)
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
