package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"go/types"
)

func (b *irBuilder) runtimeSymbolFacts(fn *ast.FuncDecl) {
	for object, ref := range b.symbols {
		kind := runtimeTypeKind(object.Type())
		if kind != "" {
			b.graph.Symbols[ref-1].Extensions = append(b.graph.Symbols[ref-1].Extensions, b.pool.SingletonExtension("gstyle.runtime:type", `"`+kind+`"`)...)
		}
	}
	unstable := make(map[pdg.Ref]bool)
	ast.Inspect(fn.Body, func(node ast.Node) bool { return b.runtimeEscape(node, unstable) })
	for ref := range unstable {
		symbol := &b.graph.Symbols[ref-1]
		symbol.Extensions = append(symbol.Extensions, b.pool.SingletonExtension("gstyle.runtime:unstable", "true")...)
	}
}

func (b *irBuilder) runtimeUnstable(expr ast.Expr, unstable map[pdg.Ref]bool) {
	id, ok := unparen(expr).(*ast.Ident)
	if !ok {
		return
	}
	if ref := b.symbols[b.info.ObjectOf(id)]; ref != 0 {
		unstable[ref] = true
	}
}

func (b *irBuilder) runtimeCapturedWrites(body *ast.BlockStmt, unstable map[pdg.Ref]bool) {
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, target := range node.Lhs {
				b.runtimeUnstable(target, unstable)
			}
		case *ast.IncDecStmt:
			b.runtimeUnstable(node.X, unstable)
		case *ast.RangeStmt:
			b.runtimeUnstable(node.Key, unstable)
			b.runtimeUnstable(node.Value, unstable)
		}
		return true
	})
}

func (b *irBuilder) runtimeEscape(node ast.Node, unstable map[pdg.Ref]bool) bool {
	switch node := node.(type) {
	case *ast.UnaryExpr:
		if node.Op == token.AND {
			b.runtimeUnstable(node.X, unstable)
		}
	case *ast.FuncLit:
		b.runtimeCapturedWrites(node.Body, unstable)
		return false
	}
	return true
}

func runtimeTypeKind(typ types.Type) string {
	if _, generic := types.Unalias(typ).(*types.TypeParam); generic {
		return ""
	}
	switch typ := typ.Underlying().(type) {
	case *types.Pointer:
		return "pointer"
	case *types.Interface:
		return "interface"
	case *types.Basic:
		return runtimeIntegerKind(typ)
	default:
		return ""
	}
}

func runtimeIntegerKind(typ *types.Basic) string {
	if typ.Info()&types.IsInteger == 0 {
		return ""
	}
	if typ.Info()&types.IsUnsigned != 0 {
		return "unsigned"
	}
	return "signed"
}
