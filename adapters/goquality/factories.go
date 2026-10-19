package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Index literal results once per typed package. A fresh outer result does not
// prove its fields: their original initializer expressions retain provenance.
func literalFactories(pkg *packages.Package) map[*types.Func]*ast.CompositeLit {
	result := map[*types.Func]*ast.CompositeLit{}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok {
				if literal := factoryReturn(fn); literal != nil {
					result[pkg.TypesInfo.Defs[fn.Name].(*types.Func)] = literal
				}
			}
		}
	}
	return result
}

func factoryReturn(fn *ast.FuncDecl) *ast.CompositeLit {
	if fn.Recv != nil || fn.Type.TypeParams != nil || fn.Body == nil {
		return nil
	}
	if len(fn.Body.List) != 1 {
		return nil
	}
	statement, _ := fn.Body.List[0].(*ast.ReturnStmt)
	return returnedLiteral(statement, fn.Type.Results)
}

func returnedLiteral(statement *ast.ReturnStmt, results *ast.FieldList) *ast.CompositeLit {
	if statement == nil || len(statement.Results) != 1 {
		return nil
	}
	if len(results.List[0].Names) != 0 {
		return nil
	}
	value := ast.Unparen(statement.Results[0])
	if address, ok := value.(*ast.UnaryExpr); ok && address.Op == token.AND {
		value = ast.Unparen(address.X)
	}
	literal, _ := value.(*ast.CompositeLit)
	return literal
}
