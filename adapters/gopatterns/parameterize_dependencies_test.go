package gopatterns

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/rewriteflow"
)

func TestParameterizationPreservesCanonicalMemoryBehavior(t *testing.T) {
	source := strings.Replace(dependencyProofFixture, "func consume(*int){}", "func consume(p *int){*p+=10}", 1)
	extraction, report := proofExtraction(t, source)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	spec := report.Candidates[0].FixSpec
	if spec == nil {
		t.Fatalf("memory fixture lacks proof: %+v", report.Candidates[0])
	}
	test := `package example
import "testing"
func TestMemory(t *testing.T) {
 for _,c:=range []bool{false,true} {
  extra:=0;if c{extra=10}
  if First(3,c)!=4+extra||Second(3,c)!=5+extra {t.Fatal("memory or branch behavior changes")}
 }
}
`
	if err := os.WriteFile("sample_test.go", []byte(test), 0600); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
	out, err := applyProjectedFix(t, spec, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	spec.Params["substitution_policy"] = "gstyle.parameterization/0.3.0"
	if _, err := applyProjectedFix(t, spec, []byte(source)); err == nil {
		t.Fatal("old dependency policy accepted")
	}
	if err := os.WriteFile("sample.go", out, 0600); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)

}

const dependencyProofFixture = `package example
func First(x int,c bool)int{value:=left(x);p:=&value;if c{consume(p)};return value}
func Second(x int,c bool)int{value:=right(x);p:=&value;if c{consume(p)};return value}
func left(x int)int{return x+1}
func right(x int)int{return x+2}
func consume(*int){}
`

func secondProofGraph(t *testing.T, extraction *Extraction) *pdg.Graph {
	t.Helper()
	for i := range extraction.Funcs {
		if extraction.Funcs[i].Name == "example.Second" {
			return &extraction.Funcs[i].Pdg
		}
	}
	t.Fatal("Second graph missing")
	return nil
}
func removeDependency(t *testing.T, graph *pdg.Graph, match func(pdg.Edge) bool) {
	t.Helper()
	for i, edge := range graph.Edges {
		if match(edge) {
			graph.Edges = append(graph.Edges[:i], graph.Edges[i+1:]...)
			return
		}
	}
	t.Fatal("dependency fixture is missing selected edge")
}
func TestParameterizationRejectsChangedCanonicalDependencies(t *testing.T) {
	for _, change := range []string{"value", "control", "execution", "memory", "operand slot", "loop state", "successors", "input inventory", "storage abstraction", "access slot", "definition location", "unknown extension", "fixed call target", "admission"} {
		t.Run(change, func(t *testing.T) {
			extraction, report := proofExtraction(t, dependencyProofFixture)
			baseline := report
			baseline.Candidates = slices.Clone(report.Candidates)
			if err := VerifyParameterizations(&baseline, extraction); err != nil {
				t.Fatal(err)
			}
			if baseline.Candidates[0].FixSpec == nil {
				t.Fatalf("fixture has no baseline proof: %+v", baseline.Candidates[0])
			}
			graph := secondProofGraph(t, extraction)
			switch change {
			case "fixed call target":
				changed := false
				for _, node := range graph.Nodes {
					if node.Category != pdg.Call {
						continue
					}
					attrs := &graph.Attributes[node.Attributes-1]
					if graph.Tables.Text(attrs.Callee.Value) != "example.consume" {
						continue
					}
					graph.Tables.Strings = append(graph.Tables.Strings, "example.unknown")
					attrs.Callee.Value = pdg.Text(len(graph.Tables.Strings))
					changed = true
					break
				}
				if !changed {
					t.Fatal("fixed call target missing")
				}
			case "admission":
				graph.Extensions = slices.DeleteFunc(graph.Extensions, func(ext pdg.Extension) bool { return graph.Tables.Text(ext.Name) == "gstyle.pdg:operation-flow" })

			case "unknown extension":
				graph.Tables.Strings = append(graph.Tables.Strings, "review:unknown-dependency", "true")
				graph.Extensions = append(graph.Extensions, pdg.Extension{Name: pdg.Text(len(graph.Tables.Strings) - 1), Value: pdg.Text(len(graph.Tables.Strings))})

			case "value":
				removeDependency(t, graph, func(edge pdg.Edge) bool { return canonicalResourceValueEdge(graph, edge) })
			case "control":
				removeDependency(t, graph, func(edge pdg.Edge) bool { return edge.Kind == pdg.ControlEdge })
			case "execution":
				removeDependency(t, graph, func(edge pdg.Edge) bool { return edge.Kind == pdg.Execution })
			case "memory":
				removeDependency(t, graph, func(edge pdg.Edge) bool { return edge.Location != 0 })
			case "operand slot", "loop state":
				changed := false
				for i, edge := range graph.Edges {
					if !canonicalResourceValueEdge(graph, edge) {
						continue
					}
					if change == "operand slot" {
						graph.Edges[i].Position++
					} else {
						graph.Edges[i].LoopCarried = 2
					}
					changed = true
					break
				}
				if !changed {
					t.Fatal("value edge missing")
				}
			case "successors":
				changed := false
				for i, block := range graph.ValueFlow.Blocks {
					if len(block.Successors) != 2 {
						continue
					}
					graph.ValueFlow.Blocks[i].Successors = []pdg.Ref{block.Successors[1], block.Successors[0]}
					changed = true
					break
				}
				if !changed {
					t.Fatal("branch missing")
				}
			case "input inventory":
				changed := false
				for i, extensions := range graph.AttributeTables.Extensions {
					kept := slices.DeleteFunc(extensions, func(ext pdg.Extension) bool { return graph.Tables.Text(ext.Name) == "gstyle.go:value-operands" })
					if len(kept) == len(extensions) {
						continue
					}
					graph.AttributeTables.Extensions[i] = kept
					changed = true
					break
				}
				if !changed {
					t.Fatal("inventory missing")
				}
			case "storage abstraction":
				if len(graph.Locations) == 0 {
					t.Fatal("storage missing")
				}
				graph.Tables.Strings = append(graph.Tables.Strings, "unknown-storage-model")
				graph.Locations[0].Abstraction = pdg.Text(len(graph.Tables.Strings))
			case "access slot":
				changed := false
				for i, accesses := range graph.AttributeTables.Accesses {
					if len(accesses) == 0 {
						continue
					}
					graph.AttributeTables.Accesses[i] = slices.Clone(accesses)
					graph.AttributeTables.Accesses[i][0].Position++
					changed = true
					break
				}
				if !changed {
					t.Fatal("access missing")
				}
			case "definition location":
				if len(graph.Locations) < 2 {
					t.Fatal("two locations required")
				}
				changed := false
				for i, definition := range graph.Definitions {
					if definition.Location == 0 {
						continue
					}
					graph.Definitions[i].Location = definition.Location%pdg.Ref(len(graph.Locations)) + 1
					changed = true
					break
				}
				if !changed {
					t.Fatal("storage definition missing")
				}
			}
			if err := pdg.Validate(graph); err != nil {
				t.Fatalf("control must remain structurally valid: %v", err)
			}
			if err := VerifyParameterizations(&report, extraction); err != nil {
				t.Fatal(err)
			}
			candidate := report.Candidates[0]
			if candidate.FixSpec != nil || !strings.Contains(strings.Join(candidate.CounterEvidence, " "), "dependencies") {
				t.Fatalf("changed graph accepted: %+v", candidate)
			}
		})
	}
}
func TestParameterizationDependenciesIgnoreEdgePresentation(t *testing.T) {
	extraction, report := proofExtraction(t, dependencyProofFixture)
	graph := secondProofGraph(t, extraction)
	slices.Reverse(graph.Edges)
	duplicate := graph.Edges[0]
	graph.Tables.Strings = append(graph.Tables.Strings, "duplicate-semantic-edge")
	duplicate.ID = pdg.Text(len(graph.Tables.Strings))
	graph.Edges = append(graph.Edges, duplicate)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	if report.Candidates[0].FixSpec == nil {
		t.Fatalf("presentation order changes proof: %+v", report.Candidates[0])
	}
}
func TestRewriteDependenciesNeedCurrentFlowAndValidSubstitutions(t *testing.T) {
	graph := parseIR(t, `package p;func f(x int)int{return x+1}`, pdg.NewBuilder())
	if same, err := rewriteflow.SameDependencies(&graph, &graph, nil); err != nil || !same {
		t.Fatalf("identity=%v,%v", same, err)
	}
	for _, sub := range []rewriteflow.CallSubstitution{{Left: 0, Right: 0}, {Left: 1, Right: 2}, {Left: 1, Right: 1}} {
		if same, err := rewriteflow.SameDependencies(&graph, &graph, []rewriteflow.CallSubstitution{sub}); err != nil || same {
			t.Fatalf("invalid substitution=%v,%v", same, err)
		}
	}
	withoutFlow := graph
	withoutFlow.NormalFlow = nil
	withoutFlow.ValueFlow = nil
	if same, err := rewriteflow.SameDependencies(&withoutFlow, &withoutFlow, nil); err != nil || same {
		t.Fatalf("missing flow=%v,%v", same, err)
	}
	invalid := graph
	invalid.Nodes = slices.Clone(graph.Nodes)
	invalid.Nodes[0].Span = pdg.Ref(len(graph.Spans) + 1)
	if _, err := rewriteflow.SameDependencies(&graph, &invalid, nil); err == nil {
		t.Fatal("invalid graph accepted")
	}
	if same, err := rewriteflow.SameDependencies(nil, &graph, nil); err != nil || same {
		t.Fatalf("nil graph=%v,%v", same, err)
	}
}
