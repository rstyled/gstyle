package gopatterns

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestExtractionCertifiesVerifiedScalarFunctions(t *testing.T) {
	pkg := loadTestPackage(t, `package p
func Identity(a int32) int32 { return a }
func Chain(a int32) int32 { x := -a; y := -x; return y }
func Overwrite(a int32) int32 { x := a; x = 1; return x }
func Branch(a int32, b bool) int32 { x := a; if b { x = 1 } else { x = 2 }; return x }
func Early(a int32, b bool) int32 { if b { return a }; return 2 }
func ShortCircuit(a, b bool) bool { return a && b }
func Renamed(input_value int32, decision bool) int32 {
	temporary := input_value
	if decision { temporary = 1 } else { temporary = 2 }
	return temporary
}
func Arithmetic(a int32) int32 { return a + 1 }
func Loop(a int32) int32 { for a > 0 { a-- }; return a }
func Call(a int32) int32 { return Identity(a) }
func Memory(a *int32) int32 { return *a }
func Closure(a int32) func() int32 { return func() int32 { return a } }
`)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Identity": true, "Chain": true, "Overwrite": true, "Branch": true, "Early": true, "ShortCircuit": true, "Renamed": true}
	seen := map[string]bool{}
	for _, graph := range extraction.Callables {
		name := graph.Tables.Text(graph.Function.Name.Value)
		seen[name] = true
		if got := graph.Conformance.Status == pdg.Conformant; got != want[name] {
			t.Fatalf("%s conformance=%v reasons=%v", name, got, graph.Conformance.Reasons)
		}
		if !want[name] {
			if len(graph.Conformance.Reasons) == 0 || len(graph.Diagnostics) == 0 {
				t.Fatalf("%s loses incomplete extraction evidence", name)
			}
			continue
		}
		policy := graph.Tables.Policies[graph.Profile-1]
		if graph.Tables.Text(policy.ID) != ScalarNormalProfileID || graph.Tables.Text(policy.Version) != NativeExtractionVersion {
			t.Fatalf("%s claims the wrong extraction profile", name)
		}
		if len(graph.Diagnostics) != 0 {
			t.Fatalf("%s retains incomplete dependency diagnostic", name)
		}
		for _, capability := range graph.Capabilities {
			if graph.Tables.Text(capability.Name) == "node_extraction" && graph.Tables.Evidence[capability.Evidence-1].Status != pdg.Supported {
				t.Fatalf("%s lacks verified node extraction", name)
			}
		}
	}
	for name := range want {
		if !seen[name] {
			t.Fatalf("missing graph %s", name)
		}
	}
}

func TestScalarAssessmentRejectsMissingAndCorruptEvidence(t *testing.T) {
	pkg := loadTestPackage(t, "package p; func f(a int32) int32 { return a }")
	fn := findFunc(pkg.Syntax[0], "f")
	for _, fault := range []string{"source", "target", "inventory", "dependencies"} {
		t.Run(fault, func(t *testing.T) {
			pool := pdg.NewBuilder()
			graph, err := buildGraph(pkg, fn, "p.go", pool)
			if err != nil {
				t.Fatal(err)
			}
			graph.Sources[0].ContentIdentity = pool.Known("source-snapshot")
			loaded := *pkg
			switch fault {
			case "source":
				graph.Sources[0].ContentIdentity = pool.Missing(pdg.Unknown, "missing snapshot")
			case "target":
				loaded.TypesSizes = nil
			case "inventory":
				graph.Nodes[0].Category = pdg.Computation
			case "dependencies":
				graph.Edges = nil
			}
			analysis := packageAnalysis{pkg: &loaded, pool: pool}
			if err := analysis.assessScalarExtraction(functionGraph{declaration: fn, graph: &graph}); err != nil {
				t.Fatal(err)
			}
			if graph.Conformance.Status != pdg.NonConformant || len(graph.Conformance.Reasons) == 0 {
				t.Fatal("invalid extraction is certified")
			}
		})
	}
}
