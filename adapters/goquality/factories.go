package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Index owned results once per typed package. Literal record fields retain
// their initializer provenance; constructed containers own their backing storage.
func factoryResults(pkg *packages.Package) map[*types.Func]ast.Expr {
	result := map[*types.Func]ast.Expr{}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok {
				var value ast.Expr
				if literal := factoryReturn(fn); literal != nil {
					value = literal
				} else {
					value = localContainerReturn(pkg, fn)
				}
				if value != nil {
					result[pkg.TypesInfo.Defs[fn.Name].(*types.Func)] = value
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
