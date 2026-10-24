package gopatterns

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestFactsKeepReportAndFixEvidenceBoundaries(t *testing.T) {
	extraction := &Extraction{Funcs: []FuncPdg{{
		Name: "first", Path: "first.go", Line: 2, EndLine: 8,
		AstTypes:          map[string]int{"call": 1},
		ErrorCheckSites:   []patterns.ErrorCheckSite{{}},
		ReleasedFileLines: []int{3}, FileObligationSites: []patterns.ResourceAcquisition{{}},
		DeferredReleaseLines: []int{4}, SuccessReturnOutputSites: []int{0},
		GuardClauses:      []GuardClauseHit{{Line: 5}},
		EntityIdentities:  []patterns.EntityIdentityHit{{Line: 6, TypeName: "Order", StructPath: "order.go", StructLine: 1}},
		MutableIdentities: []patterns.MutableIdentityHit{{Line: 7}},
		Params:            []patterns.ParamInfo{{Name: "id", Type: "string"}},
	}, {Name: "second", Path: "second.go"}}}
	report, fixes := extraction.PatternFacts(), extraction.FixFacts()
	if len(report) != 2 || len(fixes) != 2 || report[0].Pdg == report[1].Pdg || fixes[0].Pdg == fixes[1].Pdg {
		t.Fatal("facts lose distinct function graphs")
	}
	if report[0].ID != "first" || report[1].ID != "second" || fixes[0].Line != 2 || fixes[0].EndLine != 8 {
		t.Fatal("facts lose function identity or source range")
	}
	if report[0].AstTypes == nil || len(report[0].ErrorCheckSites) != 1 || len(report[0].FileObligationSites) != 1 || len(report[0].ReleasedFileLines) != 1 || len(report[0].DeferredReleaseLines) != 1 || len(report[0].SuccessReturnOutputSites) != 1 {
		t.Fatal("report loses full analysis evidence")
	}
	if fixes[0].AstTypes != nil || fixes[0].ErrorCheckSites != nil || fixes[0].FileObligationSites != nil || fixes[0].ReleasedFileLines != nil || fixes[0].DeferredReleaseLines != nil || fixes[0].SuccessReturnOutputSites != nil {
		t.Fatal("fix projection widens its established evidence boundary")
	}
	if report[0].EntityIdentities[0].StructPath != "" || fixes[0].EntityIdentities[0].StructPath != "order.go" {
		t.Fatal("report and fix entity locations change")
	}
	report[0].GuardClauses[0].Line = 50
	report[0].MutableIdentities[0].Line = 70
	fixes[0].Params[0].Name = "changed"
	if extraction.Funcs[0].GuardClauses[0].Line != 5 || extraction.Funcs[0].MutableIdentities[0].Line != 7 || extraction.Funcs[0].Params[0].Name != "id" {
		t.Fatal("projection exposes previously copied slice storage")
	}
	if !reflect.DeepEqual(report[0].Pdg, fixes[0].Pdg) {
		t.Fatal("projections disagree about the canonical graph")
	}
}

func TestEmptyFactsKeepTheirEstablishedShapes(t *testing.T) {
	extraction := &Extraction{}
	if extraction.PatternFacts() == nil || extraction.FixFacts() != nil {
		t.Fatal("empty projection array shapes change")
	}
	extraction.Funcs = []FuncPdg{{GuardClauses: []GuardClauseHit{}, Params: []patterns.ParamInfo{}}}
	report, fixes := extraction.PatternFacts()[0], extraction.FixFacts()[0]
	if report.GuardClauses != nil || fixes.GuardClauses != nil || report.Params == nil || fixes.Params != nil {
		t.Fatal("empty hit and parameter array shapes change")
	}
}

func TestAdapterHitTypesKeepTheirIdentity(t *testing.T) {
	for _, value := range []any{GuardClauseHit{}, EnumDispatchHit{}, TypeSwitchHit{}} {
		if reflect.TypeOf(value).PkgPath() != "github.com/rstyled/gstyle/adapters/gopatterns" {
			t.Fatalf("adapter hit becomes a different public type: %T", value)
		}
	}
}
