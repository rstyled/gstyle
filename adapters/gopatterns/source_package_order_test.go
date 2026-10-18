package gopatterns

import (
	"golang.org/x/tools/go/packages"
	"reflect"
	"testing"
)

func TestSourceBatchDependenciesPrecedeImporters(t *testing.T) {
	families := []sourceFamily{
		{name: "root", query: "root", imports: []string{"outside", "middle"}, bytes: sourceBatchBytes},
		{name: "middle", query: "middle", imports: []string{"leaf"}, bytes: sourceBatchBytes},
		{name: "leaf", query: "leaf", bytes: sourceBatchBytes},
	}
	batches := planSourceBatches(families, nil)
	if !reflect.DeepEqual(families[0].imports, []string{"outside", "middle"}) {
		t.Fatal("batch planning mutates borrowed import metadata")
	}
	want := [][]string{{"leaf"}, {"middle"}, {"root"}}
	if !reflect.DeepEqual(batches, want) {
		t.Fatalf("dependency batches=%v want %v", batches, want)
	}
}

func TestPackageOrderUsesResolvedImportIdentity(t *testing.T) {
	dependency := &packages.Package{PkgPath: "resolved/dependency"}
	caller := &packages.Package{PkgPath: "root", Imports: map[string]*packages.Package{"source/import": dependency}}
	ordered := orderedSourcePackages([]*packages.Package{caller, dependency})
	if len(ordered) != 2 || ordered[0] != dependency || ordered[1] != caller {
		t.Fatal("resolved dependency follows its importer")
	}
}
