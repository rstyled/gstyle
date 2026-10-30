package cpp

import (
	"context"
	"strconv"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCPPRecordedFlowPreservesCFGAndCycles(t *testing.T) {
	wire := fixtureWire(t)
	doc, err := indexGraph(context.Background(), wire)
	if err != nil {
		t.Fatal(err)
	}
	extraction, err := convert(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range extraction.Functions {
		graph := f.Graph
		flow := graph.ValueFlow
		if flow == nil {
			t.Fatalf("missing CFG for %s", f.Score.Name)
		}
		if err := pdg.ValidateFlow(graph, flow); err != nil {
			t.Fatal(err)
		}
		got := map[[2]string]bool{}
		for i, block := range flow.Blocks {
			for _, next := range block.Successors {
				got[[2]string{graph.Tables.Text(graph.Nodes[i].ID), graph.Tables.Text(graph.Nodes[next-1].ID)}] = true
			}
		}
		for _, edge := range doc.Edges {
			if edge.Label != "CFG" || edge.Source.Owner != edge.Target.Owner {
				continue
			}
			if edge.Source.Owner.text("FULL_NAME") != f.Score.Name {
				continue
			}
			key := [2]string{strconv.FormatUint(edge.Source.ID, 10), strconv.FormatUint(edge.Target.ID, 10)}
			if !got[key] {
				t.Fatalf("missing CFG relation %v in %s", key, f.Score.Name)
			}
			delete(got, key)
		}
		if len(got) != 0 {
			t.Fatalf("invented CFG relations: %v", got)
		}
		if graph.NormalFlow != nil || pdg.BodyFlow(graph) != nil {
			t.Fatal("recorded CFG grants unverified native execution semantics")
		}
	}
}

func TestCPPReachingDefinitionsKeepBindingIdentity(t *testing.T) {
	extraction, err := Load(context.Background(), fixture)
	if err != nil {
		t.Fatal(err)
	}
	linked := 0
	for _, f := range extraction.Functions {
		g := f.Graph
		for _, edge := range g.Edges {
			if edge.Definition == 0 {
				continue
			}
			definition := g.Definitions[edge.Definition-1]
			if definition.Node != edge.Source || definition.Symbol != edge.Symbol {
				t.Fatal("reaching edge loses source definition identity")
			}
			if g.Tables.Text(edge.Subkind) != "REACHING_DEF" {
				t.Fatal("operand or control relation claims a reaching definition")
			}
			linked++
		}
	}
	if linked == 0 {
		t.Fatal("C++ definitions never reach canonical edges")
	}
}
