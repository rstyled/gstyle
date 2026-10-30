package cpp

import (
	"context"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func certificateFixture(t *testing.T) (*projection, []*vertex) {
	t.Helper()
	doc, err := indexGraph(context.Background(), fixtureWire(t))
	if err != nil {
		t.Fatal(err)
	}
	groups := methodNodes(doc)
	for _, method := range doc.Methods {
		if method.text("NAME") != "nested" {
			continue
		}
		p := newProjection(doc, method)
		if _, err := p.convert(groups[method]); err != nil {
			t.Fatal(err)
		}
		return p, groups[method]
	}
	t.Fatal("missing fixture method")
	return nil, nil
}

func TestCPPImportCertificateRejectsCorruption(t *testing.T) {
	cases := map[string]func(*projection){
		"lost source relations": func(p *projection) { p.graph.Extensions = nil },
		"lost dependency":       func(p *projection) { p.graph.Edges = p.graph.Edges[1:] },
		"wrong node category": func(p *projection) {
			p.graph.Nodes[0].Category = (p.graph.Nodes[0].Category + 1) % (pdg.Other + 1)
		},
		"wrong operand direction": func(p *projection) {
			for i, edge := range p.graph.Edges {
				if strings.HasPrefix(p.pool.Tables.Text(edge.ID), "operand:") {
					p.graph.Edges[i].Source, p.graph.Edges[i].Target = edge.Target, edge.Source
					return
				}
			}
		},
		"wrong control kind": func(p *projection) {
			for i, edge := range p.graph.Edges {
				if edge.Kind == pdg.ControlEdge {
					p.graph.Edges[i].Kind = pdg.Data
					return
				}
			}
		},
		"wrong operand position": func(p *projection) {
			for i, edge := range p.graph.Edges {
				if strings.HasPrefix(p.pool.Tables.Text(edge.ID), "operand:") {
					p.graph.Edges[i].Position++
					return
				}
			}
		},
		"lost variable metadata": func(p *projection) {
			for i, edge := range p.graph.Edges {
				if edge.Annotations != 0 {
					p.graph.Edges[i].Annotations = 0
					return
				}
			}
		},
	}
	for name, corrupt := range cases {
		t.Run(name, func(t *testing.T) {
			p, nodes := certificateFixture(t)
			if err := p.verifyImport(nodes); err != nil {
				t.Fatal(err)
			}
			corrupt(p)
			if err := p.verifyImport(nodes); err == nil {
				t.Fatal("corrupt graph retains import certification")
			}
		})
	}
}
