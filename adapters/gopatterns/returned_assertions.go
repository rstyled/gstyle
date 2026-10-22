package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/types/typeutil"
)

func (q *assertionGuardQuery) returnedType(ref pdg.Ref, assertion *ast.TypeAssertExpr, summaries map[types.Object]concreteReturn) bool {
	if !q.stableReturnInput(ref, assertion.X) {
		return false
	}
	origins := q.bindings.OriginsAt(ref, 1, q.facts)
	if !origins.Known || len(origins.Sources) == 0 {
		return false
	}
	return q.returnOriginsProve(ref, assertion, origins.Sources, summaries)
}

func (q *assertionGuardQuery) returnOriginsProve(ref pdg.Ref, assertion *ast.TypeAssertExpr, origins []bindingflow.ValueSource, summaries map[types.Object]concreteReturn) bool {
	nullable := false
	for _, origin := range origins {
		result := q.returnedConcreteType(origin, summaries)
		if !satisfiesTypeAssertion(result.typ, q.info.TypeOf(assertion.Type)) {
			return false
		}
		nullable = nullable || result.nullable
	}
	return !nullable || q.nonNilAssertion(ref, assertion)
}

func (q *assertionGuardQuery) stableReturnInput(ref pdg.Ref, expression ast.Expr) bool {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.CallExpr:
		return true
	case *ast.Ident:
		return q.localPath(expression, q.bindings.Operand(ref, 1)).Root != 0
	default:
		return false
	}
}

func (q *assertionGuardQuery) returnedConcreteType(origin bindingflow.ValueSource, summaries map[types.Object]concreteReturn) concreteReturn {
	if origin.Zero || origin.Node == 0 || origin.Result > 1 {
		return concreteReturn{}
	}
	call, ok := q.source[origin.Node].(*ast.CallExpr)
	if !ok {
		return concreteReturn{}
	}
	object := typeutil.Callee(q.info, call)
	if !q.resolvedReturnCall(origin.Node, object) {
		return concreteReturn{}
	}
	return summaries[object]
}

func (q *assertionGuardQuery) resolvedReturnCall(ref pdg.Ref, object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	return attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known && q.graph.Tables.Text(attrs.Callee.Value) == patterns.FuncID(function)
}

func satisfiesTypeAssertion(concrete, target types.Type) bool {
	if concrete == nil || target == nil {
		return false
	}
	if _, generic := types.Unalias(target).(*types.TypeParam); generic {
		return false
	}
	if iface, ok := target.Underlying().(*types.Interface); ok {
		return types.Implements(concrete, iface)
	}
	return types.Identical(concrete, target)
}
