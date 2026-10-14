package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (b *irBuilder) runtimeValueFacts(fn *ast.FuncDecl) {
	if !b.runtimeOperations {
		return
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		b.runtimeExpression(node)
		return true
	})
	b.runtimeAddressCancellation(fn.Body)
	b.runtimeSymbolFacts(fn)
}

func (b *irBuilder) runtimeExpression(node ast.Node) {
	expr, ok := node.(ast.Expr)
	if !ok {
		return
	}
	if b.info.Types[expr].IsType() {
		return
	}
	ref := b.nodes[node]
	if runtimeMissingNode(ref, expr, b.info) {
		ref = b.addNode(node, pdg.Computation, pdg.Attributes{})
	}
	if ref == 0 {
		return
	}
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	b.runtimeConstant(expr, attrs)
	b.runtimeConversionType(expr, attrs)
	b.runtimeAccessFacts(expr, attrs)
	if runtimeNonNil(expr, b.info) {
		b.runtimeExtension(attrs, "non-nil", "true")
	}
}

func (b *irBuilder) runtimeExtension(attrs *pdg.Attributes, name, value string) {
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.runtime:"+name, value)...)
}

func (b *irBuilder) runtimeConstant(expr ast.Expr, attrs *pdg.Attributes) {
	value := b.info.Types[expr]
	if value.IsNil() {
		b.runtimeExtension(attrs, "nil", "true")
		return
	}
	if value.Value != nil && value.Value.Kind() == constant.Int {
		b.runtimeExtension(attrs, "integer", value.Value.ExactString())
	}
}

func (b *irBuilder) runtimeAccessFacts(expr ast.Expr, attrs *pdg.Attributes) {
	switch expr := expr.(type) {
	case *ast.StarExpr:
		b.runtimeExtension(attrs, "dereference", "true")
	case *ast.SelectorExpr:
		if runtimeAccess(expr, b.info) {
			b.runtimeExtension(attrs, "dereference", "true")
		}
	case *ast.IndexExpr:
		b.runtimeIndexFacts(expr, attrs)
	case *ast.CompositeLit:
		b.runtimeExtentFacts(expr, attrs)
	}
}

func (b *irBuilder) runtimeIndexFacts(expr *ast.IndexExpr, attrs *pdg.Attributes) {
	if !collectionType(b.info.TypeOf(expr.X)) {
		return
	}
	b.runtimeExtension(attrs, "index", "true")
	if _, literal := unparen(expr.X).(*ast.CompositeLit); literal {
		b.runtimeExtension(attrs, "literal-index", "true")
	}
}

func (b *irBuilder) runtimeExtentFacts(expr *ast.CompositeLit, attrs *pdg.Attributes) {
	if size, known := runtimeLiteralExtent(expr, b.info); known {
		b.runtimeExtension(attrs, "literal-extent", strconv.FormatInt(size, 10))
	}
}

func (b *irBuilder) runtimeConversionType(expr ast.Expr, attrs *pdg.Attributes) {
	if b.pool.Tables.Text(attrs.Operator.Value) != "conversion" {
		return
	}
	kind := runtimeTypeKind(b.info.TypeOf(expr))
	if kind != "" {
		b.runtimeExtension(attrs, "type", strconv.Quote(kind))
	}
}

// Go cancels &*p without dereferencing p, including a nil pointer.
func (b *irBuilder) runtimeAddressCancellation(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		unary, ok := node.(*ast.UnaryExpr)
		if ok && unary.Op == token.AND {
			b.cancelRuntimeDereference(unary.X)
		}
		return true
	})
}

func (b *irBuilder) cancelRuntimeDereference(expr ast.Expr) {
	star, ok := unparen(expr).(*ast.StarExpr)
	if !ok {
		return
	}
	ref := b.nodes[star]
	if ref == 0 {
		return
	}
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = slices.DeleteFunc(attrs.Extensions, func(value pdg.Extension) bool { return b.pool.Tables.Text(value.Name) == "gstyle.runtime:dereference" })
}

// Compiler facts annotate the canonical PDG once. They describe values and
// accesses, not findings or safety decisions.
func runtimeRelevant(fn *ast.FuncDecl, info *types.Info) bool {
	relevant := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		relevant = relevant || runtimeAccess(node, info)
		return !relevant
	})
	return relevant
}

func runtimeAccess(node ast.Node, info *types.Info) bool {
	switch node := node.(type) {
	case *ast.StarExpr:
		return !info.Types[node].IsType()
	case *ast.SelectorExpr:
		selection := info.Selections[node]
		return selection != nil && selection.Kind() == types.FieldVal && pointerType(info.TypeOf(node.X))
	case *ast.IndexExpr:
		return collectionType(info.TypeOf(node.X))
	default:
		return false
	}
}

func pointerType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	_, ok := typ.Underlying().(*types.Pointer)
	return ok
}

func collectionType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch typ.Underlying().(type) {
	case *types.Array, *types.Slice:
		return true
	default:
		return false
	}
}

func runtimeFactExpression(expr ast.Expr, info *types.Info) bool {
	return info.Types[expr].IsNil() || info.Types[expr].Value != nil || runtimeAccess(expr, info)
}

func runtimeNonNil(expr ast.Expr, info *types.Info) bool {
	switch expr := expr.(type) {
	case *ast.UnaryExpr:
		_, dereference := unparen(expr.X).(*ast.StarExpr)
		return expr.Op == token.AND && !dereference
	case *ast.CallExpr:
		id, ok := unparen(expr.Fun).(*ast.Ident)
		if !ok {
			return false
		}
		builtin, ok := info.ObjectOf(id).(*types.Builtin)
		return ok && builtin.Name() == "new"
	default:
		return false
	}
}

func runtimeLiteralExtent(lit *ast.CompositeLit, info *types.Info) (int64, bool) {
	switch typ := info.TypeOf(lit).Underlying().(type) {
	case *types.Array:
		return typ.Len(), true
	case *types.Slice:
		return runtimeSliceExtent(lit, info)
	default:
		return 0, false
	}
}

func runtimeSliceExtent(lit *ast.CompositeLit, info *types.Info) (int64, bool) {
	var next, size int64
	for _, element := range lit.Elts {
		if keyed, ok := element.(*ast.KeyValueExpr); ok {
			value := info.Types[keyed.Key].Value
			var known bool
			next, known = constant.Int64Val(value)
			if !known {
				return 0, false
			}
		}
		next++
		size = max(size, next)
	}
	return size, true
}

func runtimeMissingNode(ref pdg.Ref, expr ast.Expr, info *types.Info) bool {
	return ref == 0 && runtimeFactExpression(expr, info)
}
