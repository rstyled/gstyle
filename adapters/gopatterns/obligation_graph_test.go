package gopatterns

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

func resourceGraphFixture(t *testing.T) (*releaseGraph, resourceflow.Policy) {
	t.Helper()
	graph := parseIR(t, `package p;func f(resource,other *T,err error){if err!=nil{return};_=other;resource.Close()};type T struct{};func(*T)Close(){}`, pdg.NewBuilder())
	facts, err := newReleaseGraph(&graph, nil)
	if err != nil {
		t.Fatal(err)
	}
	return facts, resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, AcquisitionError: graph.Function.Inputs[2].Symbol}
}
func TestResourceReleaseFollowsCanonicalGraph(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pdg.Graph)
	}{
		{"receiver producer", func(g *pdg.Graph) {
			var other pdg.Ref
			for i, node := range g.Nodes {
				attrs := g.NodeAttributes(node)
				if node.Category == pdg.Read && len(attrs.Reads) == 1 && g.Tables.Text(g.Symbols[attrs.Reads[0].Symbol-1].Name.Value) == "other" {
					other = pdg.Ref(i + 1)
				}
			}
			if other == 0 {
				t.Fatal("missing other read")
			}
			for i, edge := range g.Edges {
				if g.Nodes[edge.Target-1].Category == pdg.Call && edge.Position == 1 && canonicalResourceValueEdge(g, edge) {
					g.Edges[i].Source = other
				}
			}
		}},
		{"normal successor", func(g *pdg.Graph) {
			var returnBlock pdg.Ref
			for i, block := range g.ValueFlow.Blocks {
				for _, ref := range block.Operations {
					if g.Nodes[ref-1].Category == pdg.Return {
						returnBlock = pdg.Ref(i + 1)
					}
				}
			}
			if returnBlock == 0 {
				t.Fatal("missing return block")
			}
			g.ValueFlow.Blocks[g.ValueFlow.Entry-1].Successors = []pdg.Ref{returnBlock}
			g.ValueFlow.RefreshReachability()
		}},
		{"missing guard producer", func(g *pdg.Graph) {
			g.Edges = slices.DeleteFunc(g.Edges, func(edge pdg.Edge) bool {
				attrs := g.NodeAttributes(g.Nodes[edge.Target-1])
				return attrs.Operator.Status == pdg.Known && g.Tables.Text(attrs.Operator.Value) == "!=" && edge.Position == 1
			})
		}},
		{"missing method contract", func(g *pdg.Graph) {
			for i, extensions := range g.AttributeTables.Extensions {
				g.AttributeTables.Extensions[i] = slices.DeleteFunc(extensions, func(extension pdg.Extension) bool {
					return g.Tables.Text(extension.Name) == "gstyle.go:method-declaration"
				})
			}
		}},
		{"missing body admission", func(g *pdg.Graph) {
			g.Extensions = slices.DeleteFunc(g.Extensions, func(extension pdg.Extension) bool {
				return g.Tables.Text(extension.Name) == "gstyle.pdg:operation-flow"
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facts, policy := resourceGraphFixture(t)
			if !facts.query.FromEntry(policy) {
				t.Fatal("baseline does not release")
			}
			tc.change(facts.graph)
			if err := pdg.Validate(facts.graph); err != nil {
				t.Fatal(err)
			}
			query, err := resourceflow.New(facts.graph)
			if err != nil {
				t.Fatal(err)
			}
			if query.FromEntry(policy) {
				t.Fatal("source syntax overrides changed canonical graph")
			}
		})
	}
}
func TestResourceQueryDoesNotModifyInputs(t *testing.T) {
	facts, policy := resourceGraphFixture(t)
	before, _ := json.Marshal(facts.graph)
	policy.Helpers = map[string]bool{"p.cleanup": true}
	helpers, _ := json.Marshal(policy.Helpers)
	_ = facts.query.FromEntry(policy)
	policy.TransferCalls = true
	_ = facts.query.FromEntry(policy)
	after, _ := json.Marshal(facts.graph)
	afterHelpers, _ := json.Marshal(policy.Helpers)
	if !bytes.Equal(before, after) || !bytes.Equal(helpers, afterHelpers) {
		t.Fatal("resource query changes borrowed inputs")
	}
}

func TestFileReturnRequiresCanonicalOperand(t *testing.T) {
	for _, body := range []string{`return os.Open("file")`, `f,err:=os.Open("file");if err!=nil{return nil,err};return f,nil`} {
		t.Run(body, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import "os";func caller()(*os.File,error){`+body+`}`)
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			released, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
			if err != nil || len(released) != 1 {
				t.Fatalf("released=%v err=%v", released, err)
			}
			graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
				return graph.Nodes[edge.Target-1].Category == pdg.Return && edge.Position == 1 && canonicalResourceValueEdge(&graph, edge)
			})
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			released, _, err = fileObligationFacts(pkg, fn, extractCtx{}, &graph)
			if err != nil || len(released) != 0 {
				t.Fatalf("source replaces missing transfer: released=%v err=%v", released, err)
			}
		})
	}
}

func canonicalResourceValueEdge(graph *pdg.Graph, edge pdg.Edge) bool {
	if edge.Kind != pdg.Data {
		return false
	}
	switch graph.Tables.Text(edge.Subkind) {
	case "source-value", "binding-value", "scalar-value":
		return true
	default:
		return false
	}
}

func TestFileAcquisitionRequiresCanonicalProducer(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "os";func caller()error{f,err:=os.Open("file");if err!=nil{return err};defer f.Close();return nil}`)
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	released, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil || len(released) != 1 {
		t.Fatalf("released=%v err=%v", released, err)
	}
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
		return canonicalResourceValueEdge(&graph, edge) && edge.Position == 1 && graph.Nodes[edge.Target-1].Category == pdg.Declaration
	})
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	released, _, err = fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil || len(released) != 0 {
		t.Fatalf("source replaces missing acquisition input: released=%v err=%v", released, err)
	}
}

func TestFileTransferWaitsForDeferredExit(t *testing.T) {
	for _, tc := range []struct {
		deferred string
		released int
	}{{`defer println("exit")`, 1}, {`defer os.Exit(0)`, 0}} {
		t.Run(tc.deferred, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import "os";func caller()(*os.File,error){f,err:=os.Open("file");if err!=nil{return nil,err};`+tc.deferred+`;return f,nil}`)
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			released, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
			if err != nil || len(released) != tc.released {
				t.Fatalf("released=%v want=%d err=%v", released, tc.released, err)
			}
		})
	}
}
