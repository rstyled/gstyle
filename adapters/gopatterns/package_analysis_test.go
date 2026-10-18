package gopatterns

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// Fixtures can request just summaries. Production analysis instead lends the
// already built package graphs to summarizeResources.
func packageResourceSummaries(pkg *packages.Package) (packageResourceFacts, error) {
	analysis := packageAnalysis{pkg: pkg, root: filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename), pool: pdg.NewBuilder()}
	if err := analysis.build(); err != nil {
		return packageResourceFacts{}, err
	}
	functions, err := resourceFunctions(pkg, analysis.graphs)
	if err != nil {
		return packageResourceFacts{}, err
	}
	return summarizeResources(functions)
}
func TestPackageConsumersShareCanonicalGraphs(t *testing.T) {
	pkg := loadTestPackage(t, `package p;func helper(x int)int{return x+1};func f(x int)int{return helper(x)}`)
	p := packageAnalysis{pkg: pkg, root: filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename), pool: pdg.NewBuilder()}
	if err := p.build(); err != nil {
		t.Fatal(err)
	}
	functions, err := resourceFunctions(pkg, p.graphs)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range p.graphs {
		id := source.graph.Tables.Text(source.graph.Function.ID)
		if functions[id].graph != source.graph {
			t.Fatalf("summary replaces graph %s", id)
		}
	}
	if err := p.summarize(); err != nil {
		t.Fatal(err)
	}
	if err := p.analyze(); err != nil {
		t.Fatal(err)
	}
	if len(p.result) != len(p.graphs) {
		t.Fatal("missing function report")
	}
	for i, result := range p.result {
		source := p.graphs[i].graph
		if result.Pdg.Tables != source.Tables || &result.Pdg.Nodes[0] != &source.Nodes[0] {
			t.Fatalf("consumer rebuilds graph %s", result.Name)
		}
	}
}

func TestExtractionOwnsTablesPerPackage(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":     "module example.com/tables\n\ngo 1.24\n",
		"lib/lib.go": `package lib;func Read() string{return "value"}`,
		"main.go":    `package tables;import "example.com/tables/lib";func Read()string{return lib.Read()};func Other()bool{return true}`,
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if len(extraction.Funcs) != 3 {
		t.Fatalf("functions=%d; want 3", len(extraction.Funcs))
	}
	tables := map[string]*pdg.Tables{}
	for _, function := range extraction.Funcs {
		graph := function.Pdg
		if err := pdg.Validate(&graph); err != nil {
			t.Fatal(err)
		}
		if graph.Tables.Sources != extraction.Sources {
			t.Fatal("source inventory is not shared")
		}
		pkg := filepath.Dir(function.Path)
		if previous := tables[pkg]; previous != nil && previous != graph.Tables {
			t.Fatal("package tables are not shared")
		}
		tables[pkg] = graph.Tables
	}
	if len(tables) != 2 || tables["."] == tables["lib"] {
		t.Fatal("unrelated packages share validation tables")
	}
}
