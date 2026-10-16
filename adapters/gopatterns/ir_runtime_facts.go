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
		b.runtimeZeroDeclaration(node)
		b.runtimeRangeFacts(node)
		b.runtimeIncrementFacts(node)
		if _, spawn := node.(*ast.GoStmt); spawn {
			b.graph.Extensions = append(b.graph.Extensions, b.pool.SingletonExtension("gstyle.runtime:concurrent-memory", "true")...)
		}
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
	b.runtimeInterfaceValue(expr, attrs)
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
	if value.Value != nil && value.Value.Kind() == constant.Bool {
		b.runtimeExtension(attrs, "boolean", value.Value.ExactString())
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
		b.runtimeFieldFacts(expr, attrs)
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
	if pointerType(b.info.TypeOf(expr.X)) {
		b.runtimeExtension(attrs, "pointer-index", "true")
	}
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

func (b *irBuilder) runtimeRangeFacts(node ast.Node) {
	stmt, ok := node.(*ast.RangeStmt)
	if !ok || !nonnegativeRangeKey(b.info.TypeOf(stmt.X)) {
		return
	}
	key, ok := stmt.Key.(*ast.Ident)
	if !ok {
		return
	}
	ref := b.symbols[b.info.ObjectOf(key)]
	if ref == 0 {
		return
	}
	attrs := &b.attributes[b.graph.Nodes[b.nodes[stmt]-1].Attributes-1]
	b.runtimeExtension(attrs, "range-index", strconv.Itoa(int(ref)))
}

func (b *irBuilder) runtimeFieldFacts(expr *ast.SelectorExpr, attrs *pdg.Attributes) {
	b.runtimeFieldPaths(expr, attrs)
	if !runtimeAccess(expr, b.info) {
		return
	}
	b.runtimeExtension(attrs, "dereference", "true")
	if embeddedPointerAccess(b.info.Selections[expr]) {
		b.runtimeExtension(attrs, "embedded-dereference", "true")
	}
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
	case *ast.CallExpr:
		return runtimeOutputCall(node, info)
	case *ast.StarExpr:
		return !info.Types[node].IsType()
	case *ast.SelectorExpr:
		return runtimePointerField(node, info)
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
	switch typ := typ.Underlying().(type) {
	case *types.Basic:
		return typ.Info()&types.IsString != 0
	case *types.Array, *types.Slice:
		return true
	case *types.Pointer:
		_, array := typ.Elem().Underlying().(*types.Array)
		return array
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

func nonnegativeRangeKey(typ types.Type) bool {
	if collectionType(typ) {
		return true
	}
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsInteger|types.IsString) != 0
}

// A promoted field can dereference an embedded pointer even when its receiver
// is a value. The receiver's non-nil fact does not prove that field non-nil.
func embeddedPointerAccess(selection *types.Selection) bool {
	if selection == nil || selection.Kind() != types.FieldVal {
		return false
	}
	typ := selection.Recv()
	path := selection.Index()
	for _, index := range path[:len(path)-1] {
		typ = runtimeStructType(typ).Field(index).Type()
		if pointerType(typ) {
			return true
		}
	}
	return false
}

func runtimeStructType(typ types.Type) *types.Struct {
	if pointer, ok := typ.Underlying().(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	return typ.Underlying().(*types.Struct)
}

func runtimePointerField(expr *ast.SelectorExpr, info *types.Info) bool {
	selection := info.Selections[expr]
	if selection == nil || selection.Kind() != types.FieldVal {
		return false
	}
	return pointerType(info.TypeOf(expr.X)) || embeddedPointerAccess(selection)
}

func (b *irBuilder) runtimeIncrementFacts(node ast.Node) {
	increment, ok := node.(*ast.IncDecStmt)
	if !ok {
		return
	}
	basic, ok := b.info.TypeOf(increment.X).Underlying().(*types.Basic)
	if !ok {
		return
	}
	if !b.signedStep64Type(basic) {
		return
	}
	ref := b.nodes[node]
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	b.runtimeExtension(attrs, "signed-step64", strconv.Quote(increment.Tok.String()))
}

func (b *irBuilder) signedStep64Type(basic *types.Basic) bool {
	if basic.Kind() == types.Int64 {
		return true
	}
	if basic.Kind() != types.Int || b.sizes == nil {
		return false
	}
	return b.sizes.Sizeof(basic) == 8
}

func (b *irBuilder) runtimeZeroDeclaration(node ast.Node) {
	spec, ok := node.(*ast.ValueSpec)
	if !ok || len(spec.Values) != 0 {
		return
	}
	if ref := b.nodes[node]; ref != 0 {
		b.runtimeExtension(&b.attributes[b.graph.Nodes[ref-1].Attributes-1], "zero-initialized", "true")
	}
}

func (b *irBuilder) runtimeInterfaceValue(expr ast.Expr, attrs *pdg.Attributes) {
	if kind := interfaceValueKind(expr, b.info); kind != "" {
		b.runtimeExtension(attrs, "interface-value", `"`+kind+`"`)
	}
}
