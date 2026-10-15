package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestNilGuardReceiverContract(t *testing.T) {
	for _, tc := range []struct {
		name, source          string
		wantFinding, receiver bool
	}{
		{"unrelated argument guard", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; return b.N }`, true, true},
		{"receiver set to nil", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; b = nil; return b.N }`, true, false},
		{"receiver guarded", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; if b == nil { return 0 }; return b.N }`, false, false},
		{"nil safe receiver method", `type Builder struct{}; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; return 1 }`, false, false},
		{"same name is a parameter", `type Builder struct { N int }; func Visit(b *Builder, candidate *int) int { if candidate == nil { return 0 }; return b.N }`, true, false},
		{"receiver guard leaves argument unchecked", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if b == nil { return 0 }; return *candidate }`, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidates := nilGuardTestCandidates(t, "package example\n"+tc.source+"\n")
			if (len(candidates) != 0) != tc.wantFinding {
				t.Fatalf("got %d findings, want finding=%v", len(candidates), tc.wantFinding)
			}
			if !tc.wantFinding {
				return
			}
			if len(candidates) != 1 {
				t.Fatalf("got %d findings, want one", len(candidates))
			}
			candidate := candidates[0]
			if tc.receiver {
				if !strings.Contains(candidate.Observation, "receiver b") || !strings.Contains(candidate.Inference, "receiver contract") {
					t.Fatalf("missing receiver context: %+v", candidate)
				}
				if strings.Contains(candidate.Inference, "wrong one") {
					t.Fatalf("unrelated guard implies wrong variable: %+v", candidate)
				}
				if !strings.Contains(strings.Join(candidate.CounterEvidence, " "), "caller") {
					t.Fatalf("missing caller contract limit: %+v", candidate)
				}
			} else if !strings.Contains(candidate.Inference, "wrong one") {
				t.Fatalf("parameter inference changes: %+v", candidate)
			}
			if candidate.FixSpec != nil {
				t.Fatal("receiver evidence must not suggest an automatic fix")
			}
		})
	}
}

func TestReceiverSymbolUsesFunctionInterface(t *testing.T) {
	graph := pdg.Graph{
		Function: pdg.Function{Inputs: []pdg.Parameter{{Role: pdg.ReceiverRole, Symbol: 1}, {Role: pdg.ParameterRole, Symbol: 2}}},
		Symbols:  []pdg.Symbol{{Role: pdg.ParameterRole}, {Role: pdg.ParameterRole}},
	}
	for _, tc := range []struct {
		symbol pdg.Ref
		want   bool
	}{{0, false}, {1, true}, {2, false}, {3, false}} {
		if got := receiverSymbol(&graph, tc.symbol); got != tc.want {
			t.Fatalf("symbol %d: got %v, want %v", tc.symbol, got, tc.want)
		}
	}
}
