package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestImplicitReturnIsAnExplicitCanonicalBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		reachable    int
	}{
		{"empty", `package p;func f(){}`, 1},
		{"deferred", `package p;func f(){defer println("x")}`, 1},
		{"deferred panic", `package p;func f(){defer panic(7)}`, 1},
		{"conditional return", `package p;func f(b bool){if b{return};println("x")}`, 1},
		{"panic", `package p;func f(){panic(7)}`, 0},
		{"Goexit", `package p;import "runtime";func f(){runtime.Goexit()}`, 0},
		{"process exit", `package p;import "os";func f(){os.Exit(3)}`, 0},
		{"infinite", `package p;func f(p *int){_=p;for{}}`, 0},
		{"constant infinite", `package p;func f(p *int){_=p;for true{}}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, tc.source, pdg.NewBuilder())
			flow := pdg.BodyFlow(&graph)
			if flow == nil {
				t.Fatal("canonical body flow missing")
			}
			active := map[pdg.Ref]bool{}
			for _, block := range flow.Blocks {
				if block.Reachable {
					for _, ref := range block.Operations {
						active[ref] = true
					}
				}
			}
			found := 0
			for i, node := range graph.Nodes {
				attrs := graph.NodeAttributes(node)
				if node.Category != pdg.Return || graph.Tables.Text(attrs.Construct) != "implicit-return" {
					continue
				}
				if !node.Synthetic || node.OriginalKind.Status != pdg.NotApplicable {
					t.Fatal("implicit return claims source syntax")
				}
				span := graph.Spans[node.Span-1]
				if span.Start != span.End || int(span.End) > len(tc.source) {
					t.Fatalf("implicit source boundary=%+v", span)
				}
				if attrs.ExitKind != pdg.NormalExit {
					t.Fatal("fallthrough is not a normal return request")
				}
				if active[pdg.Ref(i+1)] {
					found++
				}
			}
			if found != tc.reachable {
				t.Fatalf("reachable implicit returns=%d want %d", found, tc.reachable)
			}
		})
	}
}
