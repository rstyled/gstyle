package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

// The compiler skips x when a range has at most one iteration variable and
// x or len(x) is constant. A trailing blank variable does not count (Go spec,
// For statements with range clause; Length and capacity).
func (b *irBuilder) constantRangeBound(statement *ast.RangeStmt) string {
	if !blankRangeValue(statement.Value) {
		return ""
	}
	if value := b.info.Types[statement.X].Value; value != nil {
		return value.ExactString()
	}
	length, array := rangeArrayLength(b.info.TypeOf(statement.X))
	if !array || rangeExpressionEffects(statement.X, b.info) {
		return ""
	}
	return strconv.FormatInt(length, 10)
}

func (b *irBuilder) controlRangeBounds(body *ast.BlockStmt) map[ast.Node]pdg.Ref {
	bounds := map[ast.Node]pdg.Ref{}
	b.rangeBounds = map[pdg.Ref]pdg.Ref{}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		statement, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		if value := b.constantRangeBound(statement); value != "" {
			ref := b.rangeBound(statement, value)
			bounds[statement.X], b.rangeBounds[b.nodes[statement]] = ref, ref
		}
		return true
	})
	return bounds
}

func (b *irBuilder) rangeBound(statement *ast.RangeStmt, value string) pdg.Ref {
	attrs := pdg.Attributes{Construct: b.pool.Text("range-bound"), Operator: b.pool.Known("literal"), Literal: b.pool.Known(value)}
	ref := b.scalarEntity(b.scalarSourceSpan(statement.X), pdg.Computation, attrs)
	b.graph.Nodes[ref-1].Evidence = b.pool.Evidence(pdg.Supported, "Compiler-constant range bound; Go omits evaluation of the original expression.")
	return ref
}

func (b *irBuilder) rangeSource(statement *ast.RangeStmt) pdg.Ref {
	if ref := b.rangeBounds[b.nodes[statement]]; ref != 0 {
		return ref
	}
	return b.nodes[statement.X]
}

// Range preheader lowering either emits the compiler constant or evaluates the
// expression. Assignment targets belong to the selected iteration edge.
func (a *scalarCFGBuilder) controlSource(source ast.Node) {
	if bound := a.rangeBounds[source]; bound != 0 {
		a.operation(bound)
		return
	}
	if !a.rangeTargets[source] {
		a.source(source)
	}
}

func blankRangeValue(value ast.Expr) bool {
	if value == nil {
		return true
	}
	identifier, ok := unparen(value).(*ast.Ident)
	return ok && identifier.Name == "_"
}

func rangeArrayLength(typ types.Type) (int64, bool) {
	if typ == nil {
		return 0, false
	}
	typ = typ.Underlying()
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem().Underlying()
	}
	array, ok := typ.(*types.Array)
	if !ok {
		return 0, false
	}
	return array.Len(), true
}

func rangeExpressionEffects(expression ast.Expr, info *types.Info) bool {
	effects := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if effects {
			return false
		}
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if expr, ok := node.(ast.Expr); ok && info.Types[expr].Value != nil {
			return false
		}
		effects = rangeEffect(node, info)
		return !effects
	})
	return effects
}

func rangeEffect(node ast.Node, info *types.Info) bool {
	switch node := node.(type) {
	case *ast.UnaryExpr:
		return node.Op == token.ARROW
	case *ast.CallExpr:
		return !info.Types[node.Fun].IsType()
	default:
		return false
	}
}
