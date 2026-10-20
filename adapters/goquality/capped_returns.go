package goquality

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Capacity describes the returned header, not ownership or function purity.
// Explicit full-slice bounds establish this certificate, including local copies.
func cappedReturns(pkg *packages.Package) map[*types.Func]bool {
	results := map[*types.Func]bool{}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if ok && cappedReturn(pkg, fn) {
				results[pkg.TypesInfo.Defs[fn.Name].(*types.Func)] = true
			}
		}
	}
	return results
}

func cappedReturn(pkg *packages.Package, fn *ast.FuncDecl) bool {
	signature := pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	if fn.Body == nil || signature.Results().Len() != 1 || signature.Results().At(0).Name() != "" {
		return false
	}
	if _, ok := underlying(signature.Results().At(0).Type()).(*types.Slice); !ok {
		return false
	}
	x := extractor{pkg: pkg, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}}
	x.collectBindings(fn.Body)
	return !x.fields.deferred && x.fullSliceReturns(fn.Body)
}

func (x *extractor) fullSliceReturns(body *ast.BlockStmt) bool {
	found, valid := false, true
	ast.Inspect(body, func(node ast.Node) bool {
		if !valid {
			return false
		}
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			found = true
			valid = x.fullSlice(assignmentValue(node.Results, 0, 1))
		}
		return valid
	})
	return found && valid
}
