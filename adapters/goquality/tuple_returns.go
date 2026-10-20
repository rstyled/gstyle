package goquality

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/packages"
)

// Start with candidate results, then remove any that fail the existing storage
// proof. A recursive cycle is valid only if every terminating return owns its
// outer storage. Borrowed branches and escapes invalidate callers as well.
func tupleFactories(pkg *packages.Package, capacity map[*types.Func]bool, results map[*types.Func]factoryResult) {
	candidates := tupleFunctions(pkg, results)
	extraction := packageExtraction{pkg: pkg, factories: results, cappedReturns: capacity}
	for changed := true; changed; {
		changed = false
		for fn, function := range candidates {
			changed = extraction.refineTuple(fn, results[function].tuple) || changed
		}
	}
}

func tupleFunctions(pkg *packages.Package, results map[*types.Func]factoryResult) map[*ast.FuncDecl]*types.Func {
	candidates := map[*ast.FuncDecl]*types.Func{}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			function := pkg.TypesInfo.Defs[fn.Name].(*types.Func)
			owned := tupleCandidates(function.Type().(*types.Signature).Results())
			if slices.Contains(owned, true) {
				results[function] = factoryResult{tuple: owned}
				candidates[fn] = function
			}
		}
	}
	return candidates
}

func tupleCandidates(results *types.Tuple) []bool {
	if results.Len() < 2 {
		return nil
	}
	owned := make([]bool, results.Len())
	for index := range results.Len() {
		result := results.At(index)
		if result.Name() != "" {
			continue
		}
		switch underlying(result.Type()).(type) {
		case *types.Map, *types.Slice:
			owned[index] = true
		}
	}
	return owned
}

func (extraction packageExtraction) refineTuple(fn *ast.FuncDecl, owned []bool) bool {
	x := extractor{pkg: extraction.pkg, factories: extraction.factories, cappedReturns: extraction.cappedReturns, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}}
	x.addParameters(fn.Type.Params)
	x.addParameters(fn.Recv)
	x.collectBindings(fn.Body)
	changed := false
	for index, candidate := range owned {
		if candidate && x.ownedContainerReturn(fn.Body, index, len(owned)) == nil {
			owned[index] = false
			changed = true
		}
	}
	return changed
}

func (x *extractor) boundValue(values []ast.Expr, index, count int) ast.Expr {
	value := assignmentValue(values, index, count)
	if len(values) != count && x.ownedTuple(value, index) {
		return nil // The existing local-binding marker describes outer storage only.
	}
	return value
}

func (x *extractor) ownedTuple(value ast.Expr, index int) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok {
		return false
	}
	function, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
	if !ok {
		return false
	}
	owned := x.factories[function.Origin()].tuple
	return index < len(owned) && owned[index]
}
