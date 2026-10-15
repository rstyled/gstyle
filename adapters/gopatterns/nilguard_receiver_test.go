package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func TestNilGuardIgnoresUnrelatedReceiverContract(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		wantFinding  bool
	}{
		{"unrelated argument guard", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; return b.N }`, false},
		{"receiver set to nil", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; b = nil; return b.N }`, true},
		{"receiver known nil in guard", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; if b == nil { return b.N }; return 0 }`, true},
		{"receiver replaced by argument", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int, other *Builder) int { if candidate == nil { return 0 }; b = other; return b.N }`, true},
		{"receiver may become nil at join", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; if *candidate > 0 { b = nil }; return b.N }`, true},
		{"receiver replaced by call", `type Builder struct { N int }; func unknown() *Builder { return nil }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; b = unknown(); return b.N }`, true},
		{"receiver address escapes", `type Builder struct { N int }; func overwrite(p **Builder) { *p = nil }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; overwrite(&b); return b.N }`, true},
		{"closure changes receiver", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; change := func() { b = nil }; change(); return b.N }`, true},
		{"receiver replaced by allocation", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; b = &Builder{}; return b.N }`, false},
		{"entry receiver restored", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; saved := b; b = &Builder{}; b = saved; return b.N }`, false},
		{"receiver guarded", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; if b == nil { return 0 }; return b.N }`, false},
		{"nil safe receiver method", `type Builder struct{}; func (b *Builder) Visit(candidate *int) int { if candidate == nil { return 0 }; return 1 }`, false},
		{"same name is a parameter", `type Builder struct { N int }; func Visit(b *Builder, candidate *int) int { if candidate == nil { return 0 }; return b.N }`, true},
		{"receiver guard leaves argument unchecked", `type Builder struct { N int }; func (b *Builder) Visit(candidate *int) int { if b == nil { return 0 }; return *candidate }`, true},
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
			if !strings.Contains(candidate.Inference, "wrong one") {
				t.Fatalf("existing inference changes: %+v", candidate)
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

// Bind the diagnostic policy to the actual flow facts, not receiver spelling.
func TestNilGuardReceiverValueEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body              string
		entryIdentity, knownNil bool
	}{
		{"entry", `return b.N`, true, false},
		{"replacement", `b = other; return b.N`, false, false},
		{"join", `if *candidate > 0 { b = nil }; return b.N`, false, false},
		{"call", `b = unknown(); return b.N`, false, false},
		{"escape", `overwrite(&b); return b.N`, false, false},
		{"closure", `change := func() { b = nil }; change(); return b.N`, false, false},
		{"nil", `b = nil; return b.N`, false, true},
		{"nil branch", `if b == nil { return b.N }; return 0`, true, true},
		{"restore", `saved := b; b = &Builder{}; b = saved; return b.N`, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package example
 type Builder struct { N int }
 func unknown() *Builder { return nil }
 func overwrite(p **Builder) { *p = nil }
 func (b *Builder) Visit(candidate *int, other *Builder) int {
  if candidate == nil { return 0 }
  ` + tc.body + `
 }
`
			checked := 0
			for _, function := range runtimeTestGraphs(t, source) {
				graph := &function.Pdg
				if len(graph.Function.Inputs) == 0 || graph.Function.Inputs[0].Role != pdg.ReceiverRole {
					continue
				}
				if err := pdg.Validate(graph); err != nil {
					t.Fatal(err)
				}
				if graph.ValueFlow == nil {
					t.Fatal("missing canonical value flow")
				}
				receiver := graph.Function.Inputs[0].Symbol
				report, err := runtimeguard.Analyze(graph)
				if err != nil {
					t.Fatal(err)
				}
				for _, site := range report.Accesses {
					if site.Symbol != receiver {
						continue
					}
					checked++
					t.Logf("receiver=%d storage-role=%d node=%d value=%+v guards=%v blocks=%d", receiver, graph.Symbols[receiver-1].Role, site.Node, site.Value, site.Guards, len(graph.ValueFlow.Blocks))
					if (site.Value.Identity == uint64(receiver)*2) != tc.entryIdentity || site.Value.Nil != tc.knownNil {
						t.Fatalf("unexpected receiver state: %+v", site.Value)
					}
					if site.Value.NonNil || len(site.Guards) == 0 {
						t.Fatalf("fixture loses unchecked access or unrelated guard: %+v", site)
					}
				}
			}
			if checked != 1 {
				t.Fatalf("checked %d receiver accesses, want one", checked)
			}
		})
	}
}
