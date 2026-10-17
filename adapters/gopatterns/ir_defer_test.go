package gopatterns

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestDeferredRegistrationSeparatesInvocation(t *testing.T) {
	graph := parseIR(t, `package p;func f(fn func(int),x int){defer fn(argument());x++;println(x)};func argument()int{return 1}`, pdg.NewBuilder())
	before, _ := json.Marshal(&graph)
	sites, err := deferflow.Sites(&graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 1 {
		t.Fatalf("sites=%+v", sites)
	}
	site := sites[0]
	if len(site.Captures) != 2 || site.Captures[0].Kind != "target" || site.Captures[1].Kind != "argument" {
		t.Fatalf("captures=%+v", site.Captures)
	}
	if graph.ValueFlow == nil {
		t.Fatal("registration has no evaluation flow")
	}
	var operations []pdg.Ref
	for _, block := range graph.ValueFlow.Blocks {
		operations = append(operations, block.Operations...)
	}
	if !slices.Contains(operations, site.Registration) || slices.Contains(operations, site.Invocation) {
		t.Fatalf("registration/invocation mixed: %+v", operations)
	}
	for _, capture := range site.Captures {
		if !slices.Contains(operations, capture.Source) {
			t.Fatalf("capture input does not evaluate: %+v", capture)
		}
	}
	for _, edge := range graph.Edges {
		if edge.Source == site.Invocation && edge.Target == site.Registration {
			t.Fatal("registration consumes an invocation result")
		}
	}
	after, _ := json.Marshal(&graph)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("defer query changes borrowed graph")
	}
	site.Captures[0].Source = 0
	fresh, err := deferflow.Sites(&graph)
	if err != nil || fresh[0].Captures[0].Source == 0 {
		t.Fatal("capture results share mutable state")
	}
}

func TestDeferredCaptureForms(t *testing.T) {
	cases := []struct {
		name, source  string
		modes         []string
		spread, tuple bool
	}{
		{"static", `func f(x int){defer work(x)};func work(int){}`, []string{"value"}, false, false},
		{"nil function", `func f(){var fn func();defer fn();println("later")}`, nil, false, false},
		{"pointer receiver", `func f(x *T){defer x.Close()};type T struct{};func(*T)Close(){}`, []string{"value"}, false, false},
		{"implicit address", `func f(x T){defer x.Close()};type T struct{};func(*T)Close(){}`, []string{"address"}, false, false},
		{"implicit dereference", `func f(x *T){defer x.Close()};type T struct{};func(T)Close(){}`, []string{"dereference"}, false, false},
		{"promoted receiver", `func f(x T){defer x.Close()};type T struct{*U};type U struct{};func(*U)Close(){}`, []string{"unknown"}, false, false},
		{"tuple", `func f(){defer work(pair())};func pair()(int,int){return 1,2};func work(int,int){}`, []string{"value"}, false, true},
		{"spread", `func f(x []int){defer work(x...)};func work(...int){}`, []string{"value"}, true, false},
		{"closure construction", `func f(){defer func(){println("body")}()}`, nil, false, false},
		{"deferred panic", `func f(){defer panic("exit");println("later")}`, []string{"value"}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, "package p;"+tc.source, pdg.NewBuilder())
			sites, err := deferflow.Sites(&graph)
			if err != nil || len(sites) != 1 {
				t.Fatalf("sites=%+v err=%v", sites, err)
			}
			site := sites[0]
			var modes []string
			for _, capture := range site.Captures {
				if capture.Kind != "target" {
					modes = append(modes, capture.Mode)
				}
			}
			if !reflect.DeepEqual(modes, tc.modes) || site.Spread != tc.spread || site.Tuple != tc.tuple {
				t.Fatalf("site=%+v want modes %v spread %v tuple %v", site, tc.modes, tc.spread, tc.tuple)
			}
			for _, block := range graph.ValueFlow.Blocks {
				if slices.Contains(block.Operations, site.Invocation) {
					t.Fatal("invocation executes at registration")
				}
			}
		})
	}
}

func TestDeferredCaptureFollowsGraph(t *testing.T) {
	graph := parseIR(t, `package p;func f(a,b int){defer work(a);println(b)};func work(int){}`, pdg.NewBuilder())
	sites, err := deferflow.Sites(&graph)
	if err != nil {
		t.Fatal(err)
	}
	var replacement pdg.Ref
	for i, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category == pdg.Read && len(attrs.Reads) == 1 && graph.Tables.Text(graph.Symbols[attrs.Reads[0].Symbol-1].Name.Value) == "b" {
			replacement = pdg.Ref(i + 1)
		}
	}
	if replacement == 0 {
		t.Fatal("missing replacement read")
	}
	for i, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) == "defer-argument" {
			graph.Edges[i].Source = replacement
		}
	}
	redirected, err := deferflow.Sites(&graph)
	if err != nil || redirected[0].Captures[0].Source != replacement || redirected[0].Captures[0].Source == sites[0].Captures[0].Source {
		t.Fatalf("redirected=%+v err=%v", redirected, err)
	}
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool { return graph.Tables.Text(edge.Subkind) == "defer-argument" })
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	if _, err := deferflow.Sites(&graph); err == nil {
		t.Fatal("source syntax replaces missing capture edge")
	}
}

func TestDeferredFactsRejectBrokenContext(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*pdg.Graph, deferflow.Site)
	}{
		{"missing registration link", func(g *pdg.Graph, s deferflow.Site) {
			g.Edges = slices.DeleteFunc(g.Edges, func(e pdg.Edge) bool { return g.Tables.Text(e.Subkind) == "defer-registration" })
		}},
		{"invocation in evaluation flow", func(g *pdg.Graph, s deferflow.Site) {
			g.ValueFlow.Blocks[0].Operations = append(g.ValueFlow.Blocks[0].Operations, s.Invocation)
		}},
		{"capture own invocation", func(g *pdg.Graph, s deferflow.Site) {
			for i, e := range g.Edges {
				if g.Tables.Text(e.Subkind) == "defer-argument" {
					g.Edges[i].Source = s.Invocation
				}
			}
		}},
		{"missing inventory", func(g *pdg.Graph, s deferflow.Site) {
			rewriteDeferredFact(g, s, func(f *deferflow.Fact) { f.Slots = nil })
		}},
		{"missing invocation", func(g *pdg.Graph, s deferflow.Site) {
			rewriteDeferredFact(g, s, func(f *deferflow.Fact) { f.Invocation = "missing" })
		}},
		{"duplicate slot", func(g *pdg.Graph, s deferflow.Site) {
			rewriteDeferredFact(g, s, func(f *deferflow.Fact) { f.Slots = append(f.Slots, f.Slots[0]) })
		}},
		{"wrong capture mode", func(g *pdg.Graph, s deferflow.Site) {
			rewriteDeferredFact(g, s, func(f *deferflow.Fact) { f.Slots[0].Mode = "invented" })
		}},
		{"wrong operation", func(g *pdg.Graph, s deferflow.Site) { g.Nodes[s.Registration-1].Category = pdg.Computation }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, `package p;func f(x int){defer work(x)};func work(int){}`, pdg.NewBuilder())
			sites, err := deferflow.Sites(&graph)
			if err != nil {
				t.Fatal(err)
			}
			tc.change(&graph, sites[0])
			if err := pdg.Validate(&graph); err != nil {
				t.Fatalf("test breaks base graph: %v", err)
			}
			if _, err := deferflow.Sites(&graph); err == nil {
				t.Fatal("broken deferred context accepted")
			}
		})
	}
}
func rewriteDeferredFact(graph *pdg.Graph, site deferflow.Site, change func(*deferflow.Fact)) {
	attrs := graph.Attributes[graph.Nodes[site.Registration-1].Attributes-1]
	extensions := graph.AttributeTables.Extensions[attrs.Extensions-1]
	for i, extension := range extensions {
		if graph.Tables.Text(extension.Name) != deferflow.Extension {
			continue
		}
		var fact deferflow.Fact
		if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &fact); err != nil {
			panic(err)
		}
		change(&fact)
		data, err := json.Marshal(fact)
		if err != nil {
			panic(err)
		}
		graph.Tables.Strings = append(graph.Tables.Strings, string(data))
		extensions[i].Value = pdg.Text(len(graph.Tables.Strings))
	}
}

func TestDeferredRegistrationsRetainDynamicSites(t *testing.T) {
	graph := parseIR(t, `package p;func f(n int){for i:=0;i<n;i++{defer work(i)};defer panic("exit");println("registered")};func work(int){}`, pdg.NewBuilder())
	sites, err := deferflow.Sites(&graph)
	if err != nil || len(sites) != 2 {
		t.Fatalf("sites=%+v err=%v", sites, err)
	}
	var later bool
	for _, block := range graph.ValueFlow.Blocks {
		for _, ref := range block.Operations {
			for _, site := range sites {
				if ref == site.Invocation {
					t.Fatal("deferred invocation executes in loop/body flow")
				}
			}
			attrs := graph.NodeAttributes(graph.Nodes[ref-1])
			if attrs.Callee.Status == pdg.Known && graph.Tables.Text(attrs.Callee.Value) == "println" {
				later = true
			}
		}
	}
	if !later {
		t.Fatal("deferred panic removes subsequent evaluation")
	}
	// A loop site stays one registration instruction. Its dynamic stack depth is
	// not bounded by the number of sites; exit-time lowering must preserve that.
}
