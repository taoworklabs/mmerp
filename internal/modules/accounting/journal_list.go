package accounting

import (
	"github.com/taoworklabs/mmerp/internal/modules/accounting/internal/store"
	"github.com/taoworklabs/mmerp/internal/platform"
	"github.com/taoworklabs/mmerp/internal/platform/data"
)

// journalEntryList shows the entries of legal entities within the actor's scope.
var journalEntryList = data.Register(data.List[store.JournalEntryListBaseRow]{
	Name:   "accounting.JournalEntryList",
	Base:   store.JournalEntryListBase,
	Scope:  data.ByColumn("org_unit_id"),
	Search: []string{"number", "description"},
	Sorts: map[string][]data.Order{
		"date":    {data.Asc("date")},
		"-date":   {data.Desc("date")},
		"number":  {data.Asc("number")},
		"-number": {data.Desc("number")},
	},
	Default:  []data.Order{data.Desc("date")},
	Tiebreak: data.Desc("id"),
})

func journalEntryQuery(sc data.Scope, f JournalEntryFilter) data.Query {
	from, to := platform.NullDate(platform.Optional(f.From)), platform.NullDate(platform.Optional(f.To))
	return data.Query{
		Scope: sc, Q: f.Q, Sort: f.Sort, Paging: f.Paging,
		Where: data.And(
			data.When(f.Status != "", data.Eq("status", f.Status)),
			data.When(f.LegalEntityID != 0, data.Eq("legal_entity_id", f.LegalEntityID)),
			data.When(from.Valid, data.Gte("date", from)),
			data.When(to.Valid, data.Lte("date", to)),
		),
	}
}
