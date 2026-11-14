package accounting

import (
	"context"
	"strings"

	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
)

// Partners serves the line editor of manual entries: whoever may write entries somewhere picks
// partners without a partner role, as Sales serves items to its document writers.
func (s *Service) Partners(ctx context.Context, q string) (AccountingPartnerList, error) {
	sc, err := s.d.IAM.Scope(ctx, "accounting", PermEntryEdit)
	if err != nil {
		return AccountingPartnerList{}, err
	}
	if !sc.Any() {
		return AccountingPartnerList{}, platform.ErrForbidden
	}
	rows, err := store.FindPartners.Run(ctx, strings.TrimSpace(q))
	if err != nil {
		return AccountingPartnerList{}, err
	}
	out := AccountingPartnerList{Items: make([]AccountingPartner, len(rows))}
	for i, r := range rows {
		out.Items[i] = AccountingPartner{ID: r.ID, Code: r.Code, Name: r.Name}
	}
	return out, nil
}
