package goquality

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
	"golang.org/x/tools/go/packages"
)

// Restrict constructed returns to scalar containers and builtin operations.
// Existing effect facts reject publication through calls or outside writes.
func localContainerReturn(pkg *packages.Package, fn *ast.FuncDecl) ast.Expr {
	signature := pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	if fn.Body == nil || signature.Results().Len() != 1 {
		return nil
	}
	if !stableTableType(signature.Results().At(0).Type()) {
		return nil
	}
	x := extractor{pkg: pkg, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}}
	x.addParameters(fn.Type.Params)
	x.addParameters(fn.Recv)
	x.collectBindings(fn.Body)
	ast.Inspect(fn.Body, x.visit)
	if !localConstruction(x.fact) {
		return nil
	}
	return x.ownedContainerReturn(fn.Body)
}

func localConstruction(fact quality.Function) bool {
	if len(fact.Effects) != 0 || len(fact.Calls) != 0 {
		return false
	}
	for _, mutation := range fact.Mutations {
		if mutation.Provenance != quality.Local {
			return false
		}
	}
	return true
}

func (x *extractor) ownedContainerReturn(body *ast.BlockStmt) ast.Expr {
	var result ast.Expr
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit, *ast.DeferStmt:
			valid = false
		case *ast.ReturnStmt:
			value := assignmentValue(node.Results, 0, 1)
			if x.origin(value, map[types.Object]bool{}) != quality.Local {
				valid = false
			}
			result = value
		}
		return valid
	})
	if !valid {
		return nil
	}
	return result
}
