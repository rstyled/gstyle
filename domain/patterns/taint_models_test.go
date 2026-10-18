package patterns

import "testing"

func TestSecurityModelCatalog(t *testing.T) {
	seen := map[string]bool{}
	for _, sink := range taintSinks {
		if sink.callee == "" || seen[sink.callee] {
			t.Fatalf("empty or repeated sink: %q", sink.callee)
		}
		seen[sink.callee] = true
		rule := securityForRisk(sink.desc)
		if rule == nil || rule.RuleID == "" || rule.CWE == "" {
			t.Fatalf("missing security metadata: %+v", sink)
		}
		for _, position := range sink.dangerousArgs {
			if position < 0 {
				t.Fatalf("negative argument position: %+v", sink)
			}
		}
	}
}

func TestSecurityBoundariesRequireFullPackageIdentity(t *testing.T) {
	for _, pair := range [][2]string{
		{"example.com/net/http.Request.FormValue", "database/sql.DB.Exec"},
		{"net/http.Request.FormValue", "example.com/database/sql.DB.Exec"},
	} {
		graph := testTaintPdg([]PdgNode{{Kind: Call, CalleeID: pair[0]}, {Kind: Call, CalleeID: pair[1]}},
			[]PdgEdge{{From: 0, To: 1, Kind: Data, ArgPos: 1}})
		if flows := FindTaintFlows(testTaintFacts(graph)); len(flows) != 0 {
			t.Fatalf("application lookalike is not a stdlib security boundary: %+v", flows)
		}
	}
}
