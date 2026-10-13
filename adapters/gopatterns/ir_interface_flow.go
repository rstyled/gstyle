package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

const interfaceFlowDescription = "Exact local binding reaching definitions on acyclic normal flow; address escapes, closures, deferred writes, concurrency, loops and multi-value assignments are excluded. Call effects and memory aliases remain unknown."

type interfaceBindingScan struct {
	builder             *irBuilder
	supported, relevant bool
}

func (scan *interfaceBindingScan) visit(node ast.Node) bool {
	if !scan.supported {
		return false
	}
	scan.supported = interfaceBindingSource(node) && scan.builder.localInterfaceBinding(node)
	if expr, ok := node.(ast.Expr); ok {
		scan.relevant = scan.relevant || interfaceType(scan.builder.info.TypeOf(expr))
	}
	return scan.supported
}

// This is a binding policy, not the compat scalar extraction profile.
func (b *irBuilder) interfaceBindingSubset(fn *ast.FuncDecl) bool {
	if len(b.graph.Function.Outputs) > 1 {
		return false
	}
	scan := interfaceBindingScan{builder: b, supported: true, relevant: b.interfaceResult(fn)}
	ast.Inspect(fn.Body, scan.visit)
	return scan.supported && scan.relevant
}

// Compiler facts annotate existing semantic nodes. Dependency inference occurs
// downstream, using only the shared PDG and its explicit evidence.
func (b *irBuilder) interfaceValueFacts() {
	if !b.bindingOperations {
		return
	}
	for source, ref := range b.nodes {
		expr, ok := source.(ast.Expr)
		if !ok {
			continue
		}
		kind := interfaceValueKind(expr, b.info)
		if kind == "" {
			continue
		}
		attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
		attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle:interface-value", `"`+kind+`"`)...)
		b.interfaceComparison(expr, attrs)
	}
}

func (b *irBuilder) interfaceComparison(expr ast.Expr, attrs *pdg.Attributes) {
	binary, ok := expr.(*ast.BinaryExpr)
	if !ok || (binary.Op != token.EQL && binary.Op != token.NEQ) {
		return
	}
	position := interfaceNilPosition(binary, b.info)
	if position == "" {
		return
	}
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle:interface-nil-operand", position)...)
}

func (b *irBuilder) localInterfaceBinding(node ast.Node) bool {
	id, ok := node.(*ast.Ident)
	if !ok {
		return true
	}
	object, ok := b.info.ObjectOf(id).(*types.Var)
	if !ok || object.Pkg() == nil {
		return true
	}
	return object.Parent() != object.Pkg().Scope()
}

func (b *irBuilder) interfaceResult(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, field := range fn.Type.Results.List {
		if interfaceType(b.info.TypeOf(field.Type)) {
			return true
		}
	}
	return false
}

func interfaceBindingSource(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.FuncLit, *ast.ForStmt, *ast.RangeStmt, *ast.BranchStmt, *ast.DeferStmt, *ast.GoStmt, *ast.SelectStmt, *ast.SendStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		return false
	case *ast.UnaryExpr:
		return node.Op != token.AND && node.Op != token.ARROW
	case *ast.AssignStmt:
		return singleBindingAssignment(node)
	case *ast.ValueSpec:
		return len(node.Names) == 1
	default:
		return true
	}
}

func singleBindingAssignment(assign *ast.AssignStmt) bool {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return false
	}
	if assign.Tok != token.ASSIGN && assign.Tok != token.DEFINE {
		return false
	}
	_, local := unparen(assign.Lhs[0]).(*ast.Ident)
	return local
}

func interfaceType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	_, ok := typ.Underlying().(*types.Interface)
	return ok
}

func interfaceValueKind(expr ast.Expr, info *types.Info) string {
	typ := interfaceExpressionType(expr, info)
	if typ == nil {
		return ""
	}
	if !interfaceType(typ) {
		return "concrete"
	}
	if concreteInterfaceConversion(expr, info) {
		return "boxed"
	}
	return "interface"
}

func interfaceExpressionType(expr ast.Expr, info *types.Info) types.Type {
	value, ok := info.Types[expr]
	if !ok || value.IsType() || value.IsNil() {
		return nil
	}
	if unknownInterfaceValue(value.Type) {
		return nil
	}
	return value.Type
}

func unknownInterfaceValue(typ types.Type) bool {
	switch typ.(type) {
	case *types.TypeParam, *types.Tuple:
		return true
	default:
		return false
	}
}

func concreteInterfaceConversion(expr ast.Expr, info *types.Info) bool {
	call, ok := unparen(expr).(*ast.CallExpr)
	if !ok || !info.Types[call.Fun].IsType() || len(call.Args) != 1 {
		return false
	}
	return interfaceValueKind(call.Args[0], info) == "concrete"
}

func interfaceNilPosition(binary *ast.BinaryExpr, info *types.Info) string {
	if info.Types[binary.Y].IsNil() && interfaceType(info.TypeOf(binary.X)) {
		return "1"
	}
	if info.Types[binary.X].IsNil() && interfaceType(info.TypeOf(binary.Y)) {
		return "2"
	}
	return ""
}
