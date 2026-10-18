package gopatterns

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

func TestResourceReleaseFollowsExplicitExitKinds(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"normal", "defer resource.Close();return", true},
		{"panic", "defer resource.Close();panic(7)", true},
		{"nil panic", "defer resource.Close();panic(nil)", true},
		{"Goexit", "defer resource.Close();runtime.Goexit()", true},
		{"process exit", "defer resource.Close();os.Exit(3)", false},
		{"deferred panic", "defer resource.Close();defer panic(7);return", true},
		{"deferred Goexit", "defer resource.Close();defer runtime.Goexit();return", true},
		{"deferred process exit first", "defer resource.Close();defer os.Exit(3);return", false},
		{"close before process exit", "defer os.Exit(3);defer resource.Close();return", true},
		{"Goexit then process exit", "defer os.Exit(3);defer resource.Close();runtime.Goexit()", true},
		{"panic after process exit registered", "defer resource.Close();defer os.Exit(3);panic(7)", false},
		{"unknown before close", "defer resource.Close();defer unknown();panic(7)", false},
		{"close before unknown", "defer unknown();defer resource.Close();panic(7)", true},
		{"no release on panic", "panic(7)", false},
		{"no release on Goexit", "runtime.Goexit()", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package p;import("os";"runtime");var _=os.Args;var _=runtime.GOOS;func f(resource *T){` + tc.body + `};type T struct{};func(*T)Close(){};func unknown(){}`
			graph := parseIR(t, source, pdg.NewBuilder())
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			if got := query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}); got != tc.want {
				t.Fatalf("release=%v want %v", got, tc.want)
			}
		})
	}
}
func TestExceptionalUnwindCannotTransferUnreturnedResource(t *testing.T) {
	for _, tail := range []string{"defer panic(7)", "defer runtime.Goexit()", "defer os.Exit(3)"} {
		graph := parseIR(t, `package p;import("runtime";"os");var _=os.Args;var _=runtime.GOOS;func f(resource *int)*int{`+tail+`;return resource}`, pdg.NewBuilder())
		query, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		if query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, TransferReturn: true}) {
			t.Fatalf("aborted return transfers ownership: %s", tail)
		}
	}
}
func TestExceptionalResourceReleaseUsesCanonicalExitTarget(t *testing.T) {
	for _, change := range []string{"abort target", "unknown target", "captured receiver", "order"} {
		t.Run(change, func(t *testing.T) {
			body := "defer resource.Close();defer panic(7)"
			if change == "order" {
				body = "defer panic(7);defer resource.Close()"
			}
			graph := parseIR(t, `package p;func f(resource,other *T){_=other;`+body+`;return};type T struct{};func(*T)Close(){}`, pdg.NewBuilder())
			policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}
			summary := func() bool {
				q, err := resourceflow.New(&graph)
				if err != nil {
					t.Fatal(err)
				}
				return q.FromEntry(policy)
			}
			if !summary() {
				t.Fatal("baseline lacks exceptional release")
			}
			switch change {
			case "abort target", "unknown target":
				for _, node := range graph.Nodes {
					if node.Category != pdg.Call {
						continue
					}
					attrs := &graph.Attributes[node.Attributes-1]
					if graph.Tables.Text(attrs.Callee.Value) != "panic" {
						continue
					}
					if change == "unknown target" {
						attrs.Callee.Status = pdg.Unknown
						continue
					}
					graph.Tables.Strings = append(graph.Tables.Strings, "os.Exit")
					attrs.Callee.Value = pdg.Text(len(graph.Tables.Strings))
				}
			case "captured receiver":
				var other pdg.Ref
				for i, node := range graph.Nodes {
					attrs := graph.NodeAttributes(node)
					if node.Category == pdg.Read && len(attrs.Reads) == 1 && attrs.Reads[0].Symbol == graph.Function.Inputs[1].Symbol {
						other = pdg.Ref(i + 1)
					}
				}
				if other == 0 {
					t.Fatal("other receiver missing")
				}
				for i, edge := range graph.Edges {
					if graph.Tables.Text(edge.Subkind) == "defer-receiver" {
						graph.Edges[i].Source = other
					}
				}
			case "order":
				sites, err := deferflow.Sites(&graph)
				if err != nil {
					t.Fatal(err)
				}
				var closeSite, panicSite pdg.Ref
				for _, site := range sites {
					attrs := graph.NodeAttributes(graph.Nodes[site.Invocation-1])
					if graph.Tables.Text(attrs.Callee.Value) == "panic" {
						panicSite = site.Registration
					} else {
						closeSite = site.Registration
					}
				}
				for _, node := range graph.Nodes {
					if node.Category != pdg.Call {
						continue
					}
					attrs := &graph.Attributes[node.Attributes-1]
					if graph.Tables.Text(attrs.Callee.Value) != "panic" {
						continue
					}
					graph.Tables.Strings = append(graph.Tables.Strings, "os.Exit")
					attrs.Callee.Value = pdg.Text(len(graph.Tables.Strings))
				}
				if !summary() {
					t.Fatal("close must still precede the pending process exit")
				}
				// Swap unique registration sites; dynamic LIFO order must follow the graph.
				for i, block := range graph.ValueFlow.Blocks {
					graph.ValueFlow.Blocks[i].Operations = slices.Clone(block.Operations)
					for j, ref := range block.Operations {
						if ref == closeSite {
							graph.ValueFlow.Blocks[i].Operations[j] = panicSite
						}
						if ref == panicSite {
							graph.ValueFlow.Blocks[i].Operations[j] = closeSite
						}
					}
				}
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			if summary() {
				t.Fatal("source syntax replaces changed exceptional evidence")
			}
		})
	}
}
