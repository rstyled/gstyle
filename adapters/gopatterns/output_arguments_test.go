package gopatterns

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/outputflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestOutputArgumentsUseCanonicalEvaluation(t *testing.T) {
	for _, tc := range []struct {
		name, argument string
		want           bool
	}{
		{"arithmetic", "n+1", true},
		{"parentheses", "((n))", true},
		{"unary", "-n", true},
		{"comparison", "n==0", true},
		{"nil comparison", "a==nil", true},
		{"package value", "io.ErrClosedPipe", true},
		{"call", "other()", false},
		{"division", "1/n", false},
		{"shift", "1<<n", false},
		{"interface equality", "a==b", false},
		{"dereference", "*p", false},
		{"index", "s[n]", false},
		{"receive", "<-ch", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := `package p;import("fmt";"io");func other()int{return 0};func f(w io.Writer,n int,a,b any,p *int,s []int,ch chan int)error{fmt.Fprintln(w,` + tc.argument + `);return nil}`
			file, fset, info, pkg := typeCheck(t, src)
			_, got := graphErrorCheckFacts(t, findFunc(file, "f"), fset, info, pkg)
			if (len(got) == 1) != tc.want {
				t.Fatalf("success=%v want %v", got, tc.want)
			}
		})
	}
}
func outputRef(t *testing.T, graph *pdg.Graph) pdg.Ref {
	t.Helper()
	for i, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category == pdg.Call && graph.Tables.Text(attrs.Callee.Value) == "fmt.Fprintln" {
			return pdg.Ref(i + 1)
		}
	}
	t.Fatal("output call missing")
	return 0
}
func TestOutputArgumentEvidenceCannotUseSourceFallback(t *testing.T) {
	for _, change := range []string{"missing input", "missing inventory", "empty inventory", "callee", "operation", "cyclic operand", "nil writer", "missing nil type"} {
		t.Run(change, func(t *testing.T) {
			graph := parseIR(t, `package p;import("fmt";"io");func f(w io.Writer,n int)error{fmt.Fprintln(w,n+1);return nil}`, pdg.NewBuilder())
			call := outputRef(t, &graph)
			proof, err := outputflow.NewSuccessFlow(&graph, 1)
			if err != nil {
				t.Fatal(err)
			}
			if !proof.Output(call, false) {
				t.Fatal("baseline lacks output evidence")
			}
			var argument pdg.Ref
			for _, edge := range graph.Edges {
				if canonicalResourceValueEdge(&graph, edge) && edge.Target == call && edge.Position == 2 {
					argument = edge.Source
				}
			}
			if argument == 0 {
				t.Fatal("argument missing")
			}
			switch change {
			case "missing input":
				graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
					return canonicalResourceValueEdge(&graph, edge) && edge.Target == call && edge.Position == 2
				})
			case "missing inventory", "empty inventory":
				attrs := graph.Attributes[graph.Nodes[call-1].Attributes-1]
				extensions := graph.AttributeTables.Extensions[attrs.Extensions-1]
				if change == "missing inventory" {
					graph.AttributeTables.Extensions[attrs.Extensions-1] = slices.DeleteFunc(extensions, func(ext pdg.Extension) bool { return graph.Tables.Text(ext.Name) == "gstyle.go:value-operands" })
					break
				}
				for i, ext := range extensions {
					if graph.Tables.Text(ext.Name) == "gstyle.go:value-operands" {
						graph.Tables.Strings = append(graph.Tables.Strings, "[]")
						extensions[i].Value = pdg.Text(len(graph.Tables.Strings))
					}
				}
			case "callee":
				graph.Attributes[graph.Nodes[call-1].Attributes-1].Callee = pdg.Fact{Status: pdg.Unknown}
			case "operation":
				attrs := graph.Attributes[graph.Nodes[argument-1].Attributes-1]
				graph.AttributeTables.Extensions[attrs.Extensions-1] = slices.DeleteFunc(graph.AttributeTables.Extensions[attrs.Extensions-1], func(ext pdg.Extension) bool { return graph.Tables.Text(ext.Name) == "gstyle.runtime:normal-completion" })
			case "cyclic operand":
				for i, edge := range graph.Edges {
					if canonicalResourceValueEdge(&graph, edge) && edge.Target == argument && edge.Position == 1 {
						graph.Edges[i].Source = argument
					}
				}
			case "nil writer", "missing nil type":
				for _, edge := range graph.Edges {
					if !canonicalResourceValueEdge(&graph, edge) || edge.Target != call || edge.Position != 1 {
						continue
					}
					attrs := graph.Attributes[graph.Nodes[edge.Source-1].Attributes-1]
					extensions := graph.AttributeTables.Extensions[attrs.Extensions-1]
					if change == "missing nil type" {
						graph.AttributeTables.Extensions[attrs.Extensions-1] = slices.DeleteFunc(extensions, func(ext pdg.Extension) bool { return graph.Tables.Text(ext.Name) == "gstyle.memory:kind" })
					}
				}
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			proof, err = outputflow.NewSuccessFlow(&graph, 1)
			if err != nil {
				t.Fatal(err)
			}
			knownNil := change == "nil writer" || change == "missing nil type"
			if proof.Output(call, knownNil) {
				t.Fatal("changed graph uses source completion evidence")
			}
		})
	}
}
