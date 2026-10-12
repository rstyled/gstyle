package gopatterns

import (
	"context"
	"go/ast"
	"os"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func memoryGraphs(pkgs []*packages.Package) ([]pdg.Graph, error) {
	pool := pdg.NewBuilder()
	var graphs []pdg.Graph
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			path := pkg.Fset.PositionFor(file.Pos(), false).Filename
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				graph, err := buildGraph(pkg, fn, path, pool)
				if err != nil {
					return nil, err
				}
				graphs = append(graphs, graph)
			}
		}
	}
	return graphs, nil
}

func memoryCounts(graphs []pdg.Graph) (functions, nodes, edges int) {
	functions = len(graphs)
	for _, g := range graphs {
		nodes += len(g.Nodes)
		edges += len(g.Edges)
	}
	return
}

// TestValidatePDGCorpus validates every graph extracted from an opt-in corpus.
func TestValidatePDGCorpus(t *testing.T) {
	root := os.Getenv("GSTYLE_PDG_VALIDATE_ROOT")
	if root == "" {
		t.Skip("opt-in corpus validation")
	}
	pkgs, err := loadPatternPackages(context.Background(), root, []string{"./..."})
	if err != nil {
		t.Fatal(err)
	}
	graphs, err := memoryGraphs(pkgs)
	if err != nil {
		t.Fatal(err)
	}
	for i := range graphs {
		if err := pdg.Validate(&graphs[i]); err != nil {
			t.Fatalf("%s: %v", graphs[i].Tables.Text(graphs[i].Function.ID), err)
		}
	}
	t.Logf("validated %d canonical graphs", len(graphs))
}
