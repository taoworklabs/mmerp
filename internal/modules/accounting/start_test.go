package accounting_test

import (
	"testing"

	"github.com/taoworklabs/mmerp/internal/core/iam"
	"github.com/taoworklabs/mmerp/internal/modules/accounting"
)

func TestSetStartPerLegalEntity(t *testing.T) {
	core, s := newAccounting(t)
	a, err := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	d := "2026-01-01"
	if err := s.SetStart(core.Admin, a, &d); err != nil {
		t.Fatal(err)
	}
	st, err := s.Starts(core.Admin)
	if err != nil || len(st) != 2 {
		t.Fatalf("starts %+v %v", st, err)
	}
	for _, x := range st {
		switch x.LegalEntityID {
		case a:
			if x.StartDate == nil || *x.StartDate != d {
				t.Errorf("a %+v", x)
			}
		case b:
			if x.StartDate != nil {
				t.Errorf("b %+v", x)
			}
		}
	}
	if err := s.SetStart(core.Admin, a, nil); err != nil {
		t.Fatal(err)
	}
	if st, _ = s.Starts(core.Admin); st[0].StartDate != nil || st[1].StartDate != nil {
		t.Fatalf("cleared %+v", st)
	}
}

func TestSetStartRefusesNonLegalEntity(t *testing.T) {
	core, s := newAccounting(t)
	c, _ := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "A"})
	dept, _ := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{ParentID: &c, Kind: "department", Name: "D"})
	d := "2026-01-01"
	if err := s.SetStart(core.Admin, dept, &d); err == nil {
		t.Fatal("department accepted")
	}
}

func TestStartOf(t *testing.T) {
	core, s := newAccounting(t)
	c, _ := core.IAM.CreateOrgUnit(core.Admin, iam.OrgUnitInput{Kind: "company", Name: "A"})
	if d, err := accounting.StartOfForTest(core.Admin, c); err != nil || d.Valid {
		t.Fatalf("unset %v %v", d, err)
	}
	day := "2026-03-01"
	if err := s.SetStart(core.Admin, c, &day); err != nil {
		t.Fatal(err)
	}
	if d, err := accounting.StartOfForTest(core.Admin, c); err != nil || !d.Valid || d.Time.Format("2006-01-02") != day {
		t.Fatalf("set %v %v", d, err)
	}
}
