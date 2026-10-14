package patterns

import (
	"fmt"
	"testing"
)

func TestAuditValueObjectCommonSubset(t *testing.T) {
	common := []ParamInfo{{Name: "amount", Type: "int"}, {Name: "currency", Type: "string"}}
	makeFact := func(id string, extras ...ParamInfo) *MiningFacts {
		return &MiningFacts{ID: id, Name: id, Path: "p.go", Line: 1, Params: append(append([]ParamInfo{}, common...), extras...)}
	}
	facts := []*MiningFacts{
		makeFact("f", ParamInfo{Name: "region", Type: "string"}),
		makeFact("g", ParamInfo{Name: "account", Type: "string"}),
		makeFact("h", ParamInfo{Name: "region", Type: "string"}, ParamInfo{Name: "account", Type: "string"}),
	}
	if got := valueObjectCandidates(facts); len(got) != 1 {
		t.Fatalf("three functions share amount/currency, got %d candidates", len(got))
	}
}
func TestAuditTaintSanitizerContext(t *testing.T) {
	g := &MiningGraph{Nodes: []PdgNode{{Kind: Call, CalleeID: "os.Getenv", Line: 1}, {Kind: Call, CalleeID: "html.EscapeString", Line: 2}, {Kind: Call, CalleeID: "os.Open", Line: 3}}, Edges: []PdgEdge{{From: 0, To: 1, Kind: Data, ArgPos: 0}, {From: 1, To: 2, Kind: Data, ArgPos: 0}}}
	if got := FindTaintFlows([]*MiningFacts{{ID: "path", Name: "path", Pdg: g}}); len(got) != 1 {
		t.Fatalf("HTML escaping does not stop path traversal: %d flows", len(got))
	}
}
func TestAuditObligationLongAlias(t *testing.T) {
	g := &MiningGraph{Nodes: []PdgNode{{Kind: Call, CalleeID: "os.Open", Line: 1}, {Kind: Let}, {Kind: Let}, {Kind: Let}, {Kind: Let}, {Kind: Call, CalleeID: "os.File.Close"}, {Kind: Defer}}}
	for i := 0; i < 6; i++ {
		g.Edges = append(g.Edges, PdgEdge{From: i, To: i + 1, Kind: Data, ArgPos: 0})
	}
	if got := FindObligationViolations([]*MiningFacts{{ID: "closed", Name: "closed", Pdg: g}}); len(got) != 0 {
		t.Fatalf("linked deferred release rejected after three hops: %+v", got)
	}
}
func TestAuditMinedRuleStableTop(t *testing.T) {
	sets := []CallSet{}
	for i := 0; i < 9; i++ {
		sets = append(sets, CallSet{FuncID: fmt.Sprint(i), Calls: map[string]bool{"A": true, "B": true, "C": true, "D": true}})
	}
	sets = append(sets, CallSet{FuncID: "missing", Calls: map[string]bool{"A": true}})
	facts := []*MiningFacts{{ID: "missing", Name: "missing", Path: "p.go", Line: 1}}
	first := ""
	for range 100 {
		m := Mined{Candidates: MinedRuleCandidates(FindViolations(sets, MineRules(sets)), facts)}
		sortMined(&m)
		if len(m.Candidates) == 0 {
			t.Fatal("no mined violations")
		}
		now := m.Candidates[0].Observation
		if first == "" {
			first = now
			continue
		}
		if first != now {
			t.Fatalf("unchanged input changes top result: %q vs %q", first, now)
		}
	}
}

func TestAuditAlattinStableTop(t *testing.T) {
	var sets []CallSet
	for i := 0; i < 9; i++ {
		calls := map[string]bool{"A": true, "F": true}
		if i < 5 {
			calls["B"] = true
		} else {
			calls["C"] = true
		}
		sets = append(sets, CallSet{FuncID: fmt.Sprint(i), Calls: calls})
	}
	sets = append(sets, CallSet{FuncID: "missing", Calls: map[string]bool{"A": true, "F": true}})
	facts := []*MiningFacts{{ID: "missing", Name: "missing", Path: "p.go", Line: 1}}
	first := ""
	for range 100 {
		m := Mined{Candidates: AlattinRuleCandidates(FindAlternativeViolations(sets, MineDisjunctiveRules(sets)), facts)}
		sortMined(&m)
		if len(m.Candidates) == 0 {
			t.Fatal("no alternative violations")
		}
		now := m.Candidates[0].Observation
		if first == "" {
			first = now
			continue
		}
		if first != now {
			t.Fatalf("unchanged input changes top result: %q vs %q", first, now)
		}
	}
}
