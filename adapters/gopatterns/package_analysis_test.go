package gopatterns

import (
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
