package hrm

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/taoworklabs/mmerp/internal/modules/hrm/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// payrollList shows payrolls of the legal entities in the actor's scope, latest
// period first, then legal entity name.
var payrollList = data.Register(data.List[store.PayrollListBaseRow]{
	Name:     "hrm.PayrollList",
	Base:     store.PayrollListBase,
	Scope:    data.ByColumn("legal_entity_id"),
	Default:  []data.Order{data.Desc("period_start"), data.Asc("legal_entity_name")},
	Tiebreak: data.Desc("id"),
})

// payrollQuery turns the API filter into a query over payrollList.
func payrollQuery(sc data.Scope, f PayrollFilter, start pgtype.Date) data.Query {
	return data.Query{
		Scope: sc, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.LegalEntityID != 0, data.Eq("legal_entity_id", f.LegalEntityID)),
			data.When(start.Valid, data.Eq("period_start", start)),
		),
	}
}
