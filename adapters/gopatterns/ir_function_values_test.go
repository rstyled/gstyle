package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestFunctionValuesInsideEvaluatedMethodReceivers(t *testing.T) {
	source := `package example
type box struct{}
func(box) run(){}
func makeBox(func()int)box{return box{}}
func plain()int{return 1}
func First()int{return left()}
func Second()int{return right()}
func left()int{return 1}
func right()int{return 2}
func receiver(){makeBox(plain).run()}
`
	extraction, _ := proofExtraction(t, source)
	graph := namedFrameGraph(t, extraction, "receiver")
	var declaration, constructor, method pdg.Ref
	for i, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category == pdg.Call && graph.Tables.Text(attrs.Callee.Value) == "example.makeBox" {
			constructor = pdg.Ref(i + 1)
		}
		if node.Category == pdg.Call && graph.Tables.Text(attrs.Callee.Value) == "example.box.run" {
			method = pdg.Ref(i + 1)
		}
		for _, extension := range attrs.Extensions {
			if graph.Tables.Text(extension.Name) == "gstyle.go:function-value" && graph.Tables.Text(extension.Value) == `"example.plain"` {
				declaration = pdg.Ref(i + 1)
			}
		}
	}
	if declaration == 0 || constructor == 0 {
		t.Fatal("evaluated receiver loses its stored function operand")
	}
	query, err := bindingflow.New(graph)
	if err != nil {
		t.Fatal(err)
	}
	if query.Operand(method, 1) != constructor {
		t.Fatal("method name conflicts with its evaluated receiver operand")
	}
	for _, edge := range graph.Edges {
		if canonicalResourceValueEdge(graph, edge) && edge.Source == declaration && edge.Target == constructor && edge.Position == 1 {
			return
		}
	}
	t.Fatal("function declaration is not the canonical constructor argument")
}
