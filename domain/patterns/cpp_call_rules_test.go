package patterns

import (
	"fmt"
	"testing"
)

func TestCPPCallRulesExcludeOpaqueCallSets(t *testing.T) {
	var facts []*MiningFacts
	for i := range 10 {
		facts = append(facts, &MiningFacts{ID: fmt.Sprint(i), Name: "balanced", Pdg: &MiningGraph{Nodes: []PdgNode{{Kind: Call, CalleeID: "lock"}, {Kind: Call, CalleeID: "unlock"}}}})
	}
	facts = append(facts, &MiningFacts{ID: "missing", Name: "missing", Pdg: &MiningGraph{Nodes: []PdgNode{{Kind: Call, CalleeID: "lock"}}}})
	facts = append(facts, &MiningFacts{ID: "opaque", Name: "opaque", Pdg: &MiningGraph{Nodes: []PdgNode{{Kind: Call, CalleeID: "lock"}, {Kind: Call}}}})
	candidates := cppCallRules(facts)
	if len(candidates) != 1 || candidates[0].Kind != MinedRule || candidates[0].Sites[0].Name != "missing" {
		t.Fatalf("wrong closed-callset associations: %+v", candidates)
	}
}
