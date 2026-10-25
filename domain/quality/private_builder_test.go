package quality

import "testing"

func TestPrivateBuilderEvidence(t *testing.T) {
	for _, callee := range []string{"fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln", "io.WriteString"} {
		f := fact("caller", 1)
		f.Calls = []Call{{Callee: callee, Line: 2, MemoryWriter: true, PrivateBuilder: true, Args: []ArgSource{{Kind: ArgOther}}}}
		report, err := Evaluate([]Function{f}, DefaultConfig())
		if err != nil || len(report.Functions[0].Effects) != 0 {
			t.Fatalf("private builder has effects: %+v, %v", report, err)
		}
		f.Calls[0].PrivateBuilder = false
		report, err = Evaluate([]Function{f}, DefaultConfig())
		if err != nil || len(report.Functions[0].Effects) != 1 || report.Functions[0].Effects[0].Kind != MutationEffect {
			t.Fatalf("legacy facts lose memory write: %+v, %v", report, err)
		}
	}
}

func TestPrivateBuilderRejectsInvalidEvidence(t *testing.T) {
	for _, call := range []Call{
		{Callee: "fmt.Fprint", Line: 2, PrivateBuilder: true},
		{Callee: "fmt.Fprint", Line: 2, PrivateBuilder: true, MemoryWriter: true},
		{Callee: "fmt.Fprint", Line: 2, PrivateBuilder: true, MemoryWriter: true, Dynamic: true, Args: []ArgSource{{Kind: ArgOther}}},
		{Callee: "fmt.Sscan", Line: 2, PrivateBuilder: true, MemoryWriter: true, Args: []ArgSource{{Kind: ArgOther}}},
		{Callee: "encoding/json.Unmarshal", Line: 2, PrivateBuilder: true, MemoryWriter: true, Args: []ArgSource{{Kind: ArgOther}}},
		{Callee: "other.Fprint", Line: 2, PrivateBuilder: true, MemoryWriter: true, Args: []ArgSource{{Kind: ArgOther}}},
	} {
		f := fact("caller", 1)
		f.Calls = []Call{call}
		if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
			t.Fatalf("invalid private builder evidence accepted: %+v", call)
		}
	}
}
