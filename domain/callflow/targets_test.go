package callflow

import (
	"testing"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestFunctionDeclarationFactRequiresOneNonemptyString(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	name := pool.Text("gstyle.go:function-value")
	for _, data := range []string{"null", `""`, "true", "{}", `["p.f"]`, "{", "7"} {
		if value, err := testDeclarationIdentity(graph, []pdg.Extension{{Name: name, Value: pool.Text(data)}}); err == nil {
			t.Fatalf("invalid identity=%q from %s", value, data)
		}
	}
	for _, first := range []string{`""`, `"p.f"`} {
		extensions := []pdg.Extension{{Name: name, Value: pool.Text(first)}, {Name: name, Value: pool.Text(`"p.f"`)}}
		if _, err := testDeclarationIdentity(graph, extensions); err == nil {
			t.Fatal("duplicate inventory accepted")
		}
	}
	if value, err := testDeclarationIdentity(graph, []pdg.Extension{{Name: name, Value: pool.Text(`"p.f"`)}}); err != nil || value != "p.f" {
		t.Fatalf("known declaration=%q,%v", value, err)
	}
}

func TestClosureTargetRequiresUniqueBodyIdentity(t *testing.T) {
	pool := pdg.NewBuilder()
	attrs := pdg.Attributes{Construct: pool.Text("closure"), Extensions: pool.SingletonExtension("gstyle.closure:body-id", `"p.f.closure@12"`)}
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Attributes: 1}}}
	set := func(value pdg.Attributes) {
		graph = &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Attributes: 1}}}
		if err := pdg.PackAttributes(graph, []pdg.Attributes{value}); err != nil {
			t.Fatal(err)
		}
	}
	set(attrs)
	source := bindingflow.ValueSource{Node: 1}
	if got, err := functionValue(graph, source); err != nil || got != "p.f.closure@12" {
		t.Fatalf("closure identity=%q error=%v", got, err)
	}
	duplicate := attrs
	duplicate.Extensions = append(append([]pdg.Extension{}, attrs.Extensions...), attrs.Extensions...)
	set(duplicate)
	if _, err := functionValue(graph, source); err == nil {
		t.Fatal("duplicate body identity accepted")
	}
	set(pdg.Attributes{Construct: pool.Text("other"), Extensions: attrs.Extensions})
	if _, err := functionValue(graph, source); err == nil {
		t.Fatal("body identity on non-closure accepted")
	}
	set(attrs)
	source.Result = 1
	if _, err := functionValue(graph, source); err == nil {
		t.Fatal("unresolved call result treated as a closure value")
	}
}

func testDeclarationIdentity(graph *pdg.Graph, extensions []pdg.Extension) (string, error) {
	candidate := &pdg.Graph{Tables: graph.Tables, Nodes: []pdg.Node{{Attributes: 1}}}
	if err := pdg.PackAttributes(candidate, []pdg.Attributes{{Extensions: extensions}}); err != nil {
		return "", err
	}
	return bindingflow.CallableIdentity(candidate, 1)
}
