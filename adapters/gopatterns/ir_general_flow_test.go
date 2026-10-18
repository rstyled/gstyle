package gopatterns

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/adapters/pdgjson"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestGeneralBindingDefinitionsReplaceFirstBindingGuesses(t *testing.T) {
	cases := []struct {
		name, source string
		want, absent []string
	}{
		{"string overwrite", `func f(x string)string{x="old";x="new";return x}`, []string{`x="new"`}, []string{`x string`, `x="old"`}},
		{"pointer overwrite", `func f(p,q *int)*int{p=q;return p}`, []string{`p=q`}, []string{`p,q *int`}},
		{"loop join", `func f(x string,c bool)string{for c{x="next"};return x}`, []string{`x string`, `x="next"`}, nil},
		{"range join", `func f(xs []string,x string)string{for _,v:=range xs{x=v};return x}`, []string{`x string`, `x=v`}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package p;" + tc.source
			graph := parseIR(t, source, pdg.NewBuilder())
			if graph.ValueFlow == nil {
				t.Fatal("general binding analysis lacks canonical operation flow")
			}
			got := generalReturnProducers(t, &graph, source)
			for _, want := range tc.want {
				if !slices.Contains(got, want) {
					t.Fatalf("missing %q in return producers %v", want, got)
				}
			}
			for _, absent := range tc.absent {
				if slices.Contains(got, absent) {
					t.Fatalf("overwritten definition %q reaches return: %v", absent, got)
				}
			}
			for _, edge := range graph.Edges {
				if graph.Tables.Text(edge.Subkind) == "mining-value" {
					t.Fatal("first-binding guess remains in general PDG")
				}
			}
		})
	}
}

func generalReturnProducers(t *testing.T, graph *pdg.Graph, source string) []string {
	t.Helper()
	if err := pdg.Validate(graph); err != nil {
		t.Fatal(err)
	}
	var result []string
	for _, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) != "reaching-definition" || graph.Nodes[edge.Target-1].Category != pdg.Read {
			continue
		}
		target := generalReadConsumer(graph, edge.Target)
		if graph.Nodes[target-1].Category != pdg.Return {
			continue
		}
		for _, producer := range reachingProducers(*graph, edge.Source) {
			span := graph.Spans[graph.Nodes[producer-1].Span-1]
			result = append(result, strings.TrimSpace(source[span.Start:span.End]))
		}
	}
	return result
}

func TestGeneralMemoryWriteReachesAliasedRead(t *testing.T) {
	source := `package p;func f()int{x:=1;p:=&x;*p=2;return x}`
	graph := parseIR(t, source, pdg.NewBuilder())
	got := generalReturnProducers(t, &graph, source)
	if !slices.Contains(got, "*p=2") {
		t.Fatalf("indirect write does not reach local binding: %v", got)
	}
	if len(graph.Locations) == 0 {
		t.Fatal("memory aliases have no explicit locations")
	}
}

func TestUnknownCallCanWriteAnEscapedBinding(t *testing.T) {
	source := `package p;func f(mutate func(*int))int{x:=1;mutate(&x);return x}`
	graph := parseIR(t, source, pdg.NewBuilder())
	got := generalReturnProducers(t, &graph, source)
	if !slices.Contains(got, "mutate(&x)") {
		t.Fatalf("escaped local ignores call writes: %v", got)
	}
	if !slices.Contains(got, "x:=1") {
		t.Fatalf("may-write call incorrectly kills unchanged value: %v", got)
	}
}

func generalReadConsumer(graph *pdg.Graph, node pdg.Ref) pdg.Ref {
	for _, edge := range graph.Edges {
		if edge.Source == node && graph.Tables.Text(edge.Subkind) == "source-value" {
			return edge.Target
		}
	}
	return node
}

func TestGeneralAliasAndStorageRegressions(t *testing.T) {
	cases := []struct {
		name, body   string
		want, absent []string
	}{
		{"load through address", `x:=1;p:=&x;return *p`, []string{"x:=1"}, nil},
		{"disjoint addresses", `x:=1;y:=2;p:=&x;q:=&y;_=q;*p=3;return y`, []string{"y:=2"}, []string{"*p=3"}},
		{"branch alias union", `x:=1;y:=2;p:=&x;if c{p=&y};*p=3;return x`, []string{"x:=1", "*p=3"}, nil},
		{"parallel declaration", `x:=1;y:=2;var p,q = &x,&y;_=p;*q=3;return y`, []string{"*q=3"}, nil},
		{"nested pointers", `x:=1;y:=2;p:=&x;pp:=&p;*pp=&y;*p=3;return y`, []string{"*p=3"}, nil},
		{"fresh allocation", `p:=new(int);*p=2;return *p`, []string{"*p=2"}, nil},
		{"pure intrinsic", `x:=1;p:=&x;_=p;_=len([]int{1,2});return x`, []string{"x:=1"}, []string{"len([]int{1,2})"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package p;func f(c bool)int{" + tc.body + "}"
			graph := parseIR(t, source, pdg.NewBuilder())
			got := generalReturnProducers(t, &graph, source)
			for _, want := range tc.want {
				if !slices.Contains(got, want) {
					t.Fatalf("missing %q among %v", want, got)
				}
			}
			for _, absent := range tc.absent {
				if slices.Contains(got, absent) {
					t.Fatalf("unrelated store %q reaches return: %v", absent, got)
				}
			}
		})
	}
}

func TestGeneralRangeElementsRetainValueDependencies(t *testing.T) {
	graph := parseIR(t, `package p;func f(xs []string)string{var x string;for _,v:=range xs{x=v};return x}`, pdg.NewBuilder())
	found := false
	for _, edge := range graph.Edges {
		if graph.Tables.Text(edge.Subkind) == "iteration-value" {
			found = true
		}
	}
	if !found {
		t.Fatal("iteration binding loses its collection value dependency")
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
}

func TestGeneralMemoryExportIsDeterministic(t *testing.T) {
	source := `package p;func f(change func(*int),c bool)int{x,y:=1,2;p:=&x;if c{p=&y};change(p);*p=3;return x+y}`
	var expected bytes.Buffer
	first := parseIR(t, source, pdg.NewBuilder())
	if err := pdgjson.Write(&expected, &first); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		graph := parseIR(t, source, pdg.NewBuilder())
		var actual bytes.Buffer
		if err := pdgjson.Write(&actual, &graph); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual.Bytes(), expected.Bytes()) {
			t.Fatal("general memory export depends on map iteration order")
		}
	}
}
