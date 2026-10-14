package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// Capture receiver facts while compiler objects are available. The rewrite
// consumes only this compact projection and binding-use ranges.
func sourceReceiver(pkg *packages.Package, function *types.Func, call *ast.CallExpr) pdg.SourceReceiver {
	if len(call.Args) == 0 {
		return pdg.SourceReceiver{}
	}
	signature := function.Type().(*types.Signature)
	if signature.Params().Len() == 0 {
		return pdg.SourceReceiver{}
	}
	arg := call.Args[0]
	exact := types.Identical(pkg.TypesInfo.TypeOf(arg), signature.Params().At(0).Type())
	result := pdg.SourceReceiver{ExactType: exact, Parenthesize: receiverNeedsParens(arg)}
	if ident := sourceLiteralCopy(pkg, arg); ident != nil {
		result.Copy = nodeByteRange(pkg.Fset, ident)
	}
	return result
}

func receiverNeedsParens(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.CompositeLit, *ast.UnaryExpr, *ast.BinaryExpr, *ast.StarExpr:
		return true
	default:
		return false
	}
}

func sourceLiteralCopy(pkg *packages.Package, expr ast.Expr) *ast.Ident {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	structure, ok := pkg.TypesInfo.TypeOf(literal).Underlying().(*types.Struct)
	if !ok || len(literal.Elts) != structure.NumFields() {
		return nil
	}
	return completeLiteralCopy(pkg, literal)
}

func completeLiteralCopy(pkg *packages.Package, literal *ast.CompositeLit) *ast.Ident {
	if len(literal.Elts) == 0 {
		return nil
	}
	first := copiedLiteralField(pkg, literal.Elts[0])
	if first == nil {
		return nil
	}
	if !types.Identical(pkg.TypesInfo.TypeOf(first), pkg.TypesInfo.TypeOf(literal)) {
		return nil
	}
	for _, element := range literal.Elts[1:] {
		if !fieldCopiesBinding(pkg, element, first) {
			return nil
		}
	}
	return first
}

func fieldCopiesBinding(pkg *packages.Package, element ast.Expr, first *ast.Ident) bool {
	ident := copiedLiteralField(pkg, element)
	return ident != nil && pkg.TypesInfo.Uses[ident] == pkg.TypesInfo.Uses[first]
}

func copiedLiteralField(pkg *packages.Package, expr ast.Expr) *ast.Ident {
	field, ok := expr.(*ast.KeyValueExpr)
	if !ok {
		return nil
	}
	selector, ok := field.Value.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	selection := pkg.TypesInfo.Selections[selector]
	if selection == nil || len(selection.Index()) != 1 {
		return nil
	}
	if pkg.TypesInfo.Uses[field.Key.(*ast.Ident)] != selection.Obj() {
		return nil
	}
	ident, _ := selector.X.(*ast.Ident)
	return ident
}
