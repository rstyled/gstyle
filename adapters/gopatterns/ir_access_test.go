package gopatterns

import (
	"strings"
	"testing"

	"github.com/shanejonas/cyclo/domain/pdg"
)

func TestIRWriteTargetsAndReadModifyWrite(t *testing.T) {
	cases := []struct {
		name, parameters, statement string
		modifies, memory            bool
	}{
		{"assign", "x int", "x = 1", false, false},
		{"parenthesized assign", "x int", "(x) = 1", false, false},
		{"compound", "x int", "x += 1", true, false},
		{"increment", "x int", "x++", true, false},
		{"dereference assign", "p *int", "*p = 1", false, true},
		{"dereference compound", "p *int", "*p += 1", true, true},
		{"index assign", "xs []int", "xs[0] = 1", false, true},
		{"index increment", "xs []int", "xs[0]++", true, true},
		{"field assign", "p *struct{X int}", "p.X = 1", false, true},
		{"field compound", "p *struct{X int}", "p.X += 1", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := "package example; func f(" + tc.parameters + ") { " + tc.statement + " }"
			graph := parseIR(t, source, pdg.NewBuilder())
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			writes, targetReads, unresolvedReads := 0, 0, 0
			for _, node := range graph.Nodes {
				attrs := graph.NodeAttributes(node)
				for _, read := range attrs.Reads {
					if read.UnresolvedReason != 0 {
						unresolvedReads++
					}
				}
				span := graph.Spans[node.Span-1]
				if strings.TrimSpace(source[span.Start:span.End]) != tc.statement || len(attrs.Writes) == 0 {
					continue
				}
				writes += len(attrs.Writes)
				target := attrs.Writes[0]
				if tc.memory && target.UnresolvedReason == 0 {
					t.Fatal("memory target claims a known alias")
				}
				if !tc.memory && target.Symbol == 0 {
					t.Fatal("scalar write loses binding")
				}
				for _, read := range attrs.Reads {
					if read == target {
						targetReads++
					}
				}
			}
			wantReads := 0
			if tc.modifies {
				wantReads = 1
			}
			if writes != 1 || targetReads != wantReads {
				t.Fatalf("writes=%d target reads=%d, want 1/%d", writes, targetReads, wantReads)
			}
			if tc.memory && unresolvedReads != wantReads {
				t.Fatalf("memory target reads=%d, want %d", unresolvedReads, wantReads)
			}
		})
	}
}
