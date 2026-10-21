package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *assertionGuardQuery) successfulAssertion(ref pdg.Ref, required assertionRequirement) bool {
	assertion, ok := q.source[ref].(*ast.TypeAssertExpr)
	if !ok || assertion.Type == nil || q.commaOK[assertion] {
		return false
	}
	if !types.Identical(q.info.TypeOf(assertion.Type), required.typ) {
		return false
	}
	path := q.localPath(assertion.X, q.bindings.Operand(ref, 1))
	return path.Root != 0 && path.Same(required.path)
}
