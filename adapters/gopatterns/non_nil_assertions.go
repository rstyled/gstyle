package gopatterns

import (
	"go/ast"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *assertionGuardQuery) nonNilAssertion(ref pdg.Ref, assertion *ast.TypeAssertExpr) bool {
	path := q.localPath(assertion.X, q.bindings.Operand(ref, 1))
	return path.Root != 0 && q.protected(ref, assertionRequirement{path: path})
}

func (q *assertionGuardQuery) nonNilGuard(condition pdg.Ref, outcome int, required bindingflow.Path) bool {
	subject, notNil := q.bindings.NilPredicate(condition, q.facts)
	expression, ok := q.source[subject].(ast.Expr)
	if !ok || (outcome == 0) != notNil {
		return false
	}
	path := q.localPath(expression, subject)
	return path.Root != 0 && path.Same(required)
}
