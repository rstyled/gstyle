package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Caller proofs never depend on the private assertions being established here.
// This is a single-hop proof, not a recursive assumption about helper contracts.
func (index *privateAssertionIndex) allTyped(object types.Object, position pdg.Position, target types.Type) bool {
	uses := index.references[object]
	if target == nil || len(uses) == 0 {
		return false
	}
	if !index.referenceBoundary(object, len(uses)) {
		return false
	}
	for _, use := range uses {
		if !index.callTyped(use, position, target) {
			return false
		}
	}
	return true
}

func (index *privateAssertionIndex) callTyped(use *ast.Ident, position pdg.Position, target types.Type) bool {
	site, found := index.calls[use]
	if !found || site.function.object == index.pkg.TypesInfo.ObjectOf(use) {
		return false
	}
	query := index.query(site.function)
	if query == nil || !query.resolvedReturnCall(query.refs[site.call], index.pkg.TypesInfo.ObjectOf(use)) {
		return false
	}
	return query.safeCallArgument(site.call, position, target)
}

func (q *assertionGuardQuery) safeCallArgument(call *ast.CallExpr, position pdg.Position, target types.Type) bool {
	argument := q.privateCallArgument(call, position)
	if argument == nil {
		return false
	}
	if satisfiesTypeAssertion(concreteInterfaceValueType(q.info.TypeOf(argument)), target) {
		return true
	}
	path := q.localPath(argument, q.refs[argument])
	return path.Root != 0 && q.protected(q.refs[call], assertionRequirement{path, target})
}

func (q *assertionGuardQuery) privateCallArgument(call *ast.CallExpr, position pdg.Position) ast.Expr {
	if position == 0 || int(position) > len(call.Args) {
		return nil
	}
	argument := call.Args[position-1]
	ref := q.refs[argument]
	if ref == 0 || q.bindings.Operand(q.refs[call], position) != ref {
		return nil
	}
	return argument
}
