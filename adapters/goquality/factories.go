package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Index owned results once per typed package. Literal record fields retain
// their initializer provenance; constructed containers own their backing storage.
type factoryResult struct {
	value     ast.Expr
	parameter *int
	tuple     []bool
}

// Grow only proven certificates. The bound keeps long dependency chains unknown.
const factoryRounds = 8

func factoryResults(pkg *packages.Package, capacity map[*types.Func]bool) map[*types.Func]factoryResult {
	results := map[*types.Func]factoryResult{}
	functions := factoryFunctions(pkg, results)
	extraction := packageExtraction{pkg: pkg, factories: results, cappedReturns: capacity}
	for range factoryRounds {
		if !extraction.composeFactories(functions) {
			break
		}
	}
	tupleFactories(pkg, capacity, results)
	return results
}

func factoryFunctions(pkg *packages.Package, results map[*types.Func]factoryResult) []*ast.FuncDecl {
	var functions []*ast.FuncDecl
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			functions = append(functions, fn)
			if literal := factoryReturn(fn); literal != nil {
				results[pkg.TypesInfo.Defs[fn.Name].(*types.Func)] = factoryResult{value: literal}
			}
		}
	}
	return functions
}

func (extraction packageExtraction) composeFactories(functions []*ast.FuncDecl) bool {
	changed := false
	for _, fn := range functions {
		function := extraction.pkg.TypesInfo.Defs[fn.Name].(*types.Func)
		if extraction.factories[function].value != nil {
			continue
		}
		if result := extraction.containerFactory(fn); result.value != nil {
			extraction.factories[function] = result
			changed = true
		}
	}
	return changed
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
