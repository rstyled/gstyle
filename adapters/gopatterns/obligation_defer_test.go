package gopatterns

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestDeferredReleaseUsesCanonicalCapture(t *testing.T) {
	pkg := loadTestPackage(t, `package p;import "os"
var other *os.File
func caller()error{
 f,err:=os.Open("file");if err!=nil{return err}
 _=other
 defer f.Close()
 return nil
}`)
	fn := findFunc(pkg.Syntax[0], "caller")
	graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	released, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil || len(released) != 1 {
		t.Fatalf("released=%v err=%v", released, err)
	}
	var replacement pdg.Ref
	for i, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category == pdg.Read && len(attrs.Reads) == 1 && graph.Tables.Text(graph.Symbols[attrs.Reads[0].Symbol-1].Name.Value) == "other" {
			replacement = pdg.Ref(i + 1)
		}
	}
	if replacement == 0 {
		t.Fatal("missing replacement input")
	}
	for i, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) == "defer-receiver" {
			graph.Edges[i].Source = replacement
		}
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	released, _, err = fileObligationFacts(pkg, fn, extractCtx{}, &graph)
	if err != nil || len(released) != 0 {
		t.Fatalf("source overrides captured producer: released=%v err=%v", released, err)
	}
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool { return graph.Tables.Text(edge.Subkind) == "defer-registration" })
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fileObligationFacts(pkg, fn, extractCtx{}, &graph); err == nil {
		t.Fatal("source proof replaces missing saved registration")
	}
}
