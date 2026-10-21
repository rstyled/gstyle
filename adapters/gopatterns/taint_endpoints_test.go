package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/taintflow"
)

func TestTaintCatalogEndpoints(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         int
	}{
		{"neither endpoint", `func copy(v string,_ *http.Request)string{return v}`, 0},
		{"source only", `func input(r *http.Request)string{return r.FormValue("q")}`, 0},
		{"sink only", `func output(v string,_ *http.Request){db.Exec(v)}`, 0},
		{"separate functions", `func input(r *http.Request)string{return r.FormValue("q")};func output(v string){db.Exec(v)};func handler(r *http.Request){output(input(r))}`, 1},
		{"separate closures", `func handler(r *http.Request){input:=func()string{return r.FormValue("q")};output:=func(v string){db.Exec(v)};output(input())}`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if flows := sourceTaintFlows(t, test.source); len(flows) != test.want {
				t.Fatalf("got %+v, want %d flows", flows, test.want)
			}
		})
	}
}

// Repeated choices create many symbolic field paths even though no modeled
// source or sink exists. The query still validates the graph before skipping.
func BenchmarkTaintWithoutEndpoints(b *testing.B) {
	graph := parseIR(b, `package p
type node struct{left,right *node;value string}
func walk(n *node,keep,choose bool)string{
 for keep{if choose{n=n.left}else{n=n.right}}
 return n.value
}`, pdg.NewBuilder())
	graphs := map[string]*pdg.Graph{graph.Tables.Text(graph.Function.ID): &graph}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		flows, err := taintflow.Analyze(graphs, taintflow.Models{})
		if err != nil || len(flows) != 0 {
			b.Fatalf("unexpected result: %+v %v", flows, err)
		}
	}
}
