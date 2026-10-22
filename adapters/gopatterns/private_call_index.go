package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type privateCallSite struct {
	function resourceFunction
	call     *ast.CallExpr
}

type privateAssertionIndex struct {
	sources    *pdg.SourceContext
	pkg        *packages.Package
	calls      map[*ast.Ident]privateCallSite
	references map[types.Object][]*ast.Ident
	queries    map[types.Object]*assertionGuardQuery
}

func newPrivateAssertionIndex(pkg *packages.Package, functions map[types.Object]resourceFunction) *privateAssertionIndex {
	index := &privateAssertionIndex{sources: privateReferenceInventory(functions), pkg: pkg, calls: map[*ast.Ident]privateCallSite{}, references: map[types.Object][]*ast.Ident{}, queries: map[types.Object]*assertionGuardQuery{}}
	for ident, object := range pkg.TypesInfo.Uses {
		index.references[object] = append(index.references[object], ident)
	}
	for _, function := range functions {
		index.recordCalls(function)
	}
	return index
}

func (index *privateAssertionIndex) recordCalls(function resourceFunction) {
	ast.Inspect(function.declaration.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		index.recordCall(node, function)
		return true
	})
}

func (index *privateAssertionIndex) recordCall(node ast.Node, function resourceFunction) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return
	}
	ident, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if ok {
		index.calls[ident] = privateCallSite{function, call}
	}
}

func (index *privateAssertionIndex) query(function resourceFunction) *assertionGuardQuery {
	if query, found := index.queries[function.object]; found {
		return query
	}
	source := functionGraph{declaration: function.declaration, graph: function.graph}
	query := newAssertionGuardQuery(index.pkg, source, nil)
	index.queries[function.object] = query
	return query
}

func privateReferenceInventory(functions map[types.Object]resourceFunction) *pdg.SourceContext {
	for _, function := range functions {
		if function.graph.Tables.Sources != nil {
			return function.graph.Tables.Sources
		}
	}
	return nil
}

func (index *privateAssertionIndex) referenceBoundary(object types.Object, count int) bool {
	if index.sources == nil {
		return true
	}
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	identity := patterns.FuncID(function)
	total := 0
	for _, reference := range index.sources.References {
		if reference.Target == identity {
			total++
		}
	}
	return total == count
}
