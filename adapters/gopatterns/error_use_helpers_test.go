package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
)

// Legacy call-site tests now exercise the same canonical graph as extraction.
func findErrorCheckSites(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) []patterns.ErrorCheckSite {
	pkg := info.Defs[fn.Name].Pkg()
	loaded := &packages.Package{Fset: fset, TypesInfo: info, Types: pkg, PkgPath: pkg.Path()}
	graph, err := buildGraph(loaded, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		panic(err)
	}
	sites, _, err := reachableErrorCheckFacts(fn, fset, info, &graph)
	if err != nil {
		panic(err)
	}
	return sites
}
