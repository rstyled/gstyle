package gopatterns

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// Private functions are closed only when every compiler use is a modeled direct
// call. Function values, foreign files and directives invalidate that boundary.
func privateAssertionProofs(pkg *packages.Package, functions map[types.Object]resourceFunction) map[*ast.TypeAssertExpr]bool {
	if !privatePackageBoundary(pkg) {
		return nil
	}
	index := newPrivateAssertionIndex(pkg, functions)
	proven := map[*ast.TypeAssertExpr]bool{}
	for _, function := range functions {
		if !privateAssertionFunction(function) {
			continue
		}
		index.prove(function, proven)
	}
	return proven
}

func privatePackageBoundary(pkg *packages.Package) bool {
	if len(pkg.OtherFiles) != 0 {
		return false
	}
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if privateCallDirective(comment.Text) {
					return false
				}
			}
		}
	}
	return true
}

func privateCallDirective(text string) bool {
	return strings.HasPrefix(text, "//go:") || strings.HasPrefix(text, "//export ")
}

func privateAssertionFunction(function resourceFunction) bool {
	if function.object.Exported() || function.declaration.Recv != nil {
		return false
	}
	signature, ok := function.object.Type().(*types.Signature)
	return ok && privateAssertionSignature(signature)
}

func privateAssertionSignature(signature *types.Signature) bool {
	return signature.TypeParams().Len() == 0 && !signature.Variadic()
}

func (index *privateAssertionIndex) prove(function resourceFunction, proven map[*ast.TypeAssertExpr]bool) {
	if len(findForceTypeAssertHits(function.declaration, index.pkg.Fset)) == 0 {
		return
	}
	query := index.query(function)
	if query == nil {
		return
	}
	for node, ref := range query.refs {
		if assertion := index.provenAssertion(query, node, ref, function.object); assertion != nil {
			proven[assertion] = true
		}
	}
}

func (index *privateAssertionIndex) provenAssertion(query *assertionGuardQuery, node ast.Node, ref pdg.Ref, object types.Object) *ast.TypeAssertExpr {
	assertion, ok := node.(*ast.TypeAssertExpr)
	if !ok || assertion.Type == nil {
		return nil
	}
	position := query.privateParameter(ref, assertion)
	if position == 0 || !index.allTyped(object, position, index.pkg.TypesInfo.TypeOf(assertion.Type)) {
		return nil
	}
	return assertion
}

func (q *assertionGuardQuery) privateParameter(ref pdg.Ref, assertion *ast.TypeAssertExpr) pdg.Position {
	if q.localPath(assertion.X, q.bindings.Operand(ref, 1)).Root == 0 {
		return 0
	}
	return privateOriginPosition(q.graph, q.bindings.OriginsOf(q.bindings.Operand(ref, 1), q.facts))
}

func privateOriginPosition(graph *pdg.Graph, origins bindingflow.Origins) pdg.Position {
	if !origins.Known || len(origins.Sources) != 1 {
		return 0
	}
	origin := origins.Sources[0]
	if !privateFormalOrigin(graph, origin) {
		return 0
	}
	for position, input := range graph.Function.Inputs {
		if input.Symbol == origin.Symbol {
			return pdg.Position(position + 1)
		}
	}
	return 0
}

func privateFormalOrigin(graph *pdg.Graph, origin bindingflow.ValueSource) bool {
	return origin.Node != 0 && int(origin.Node) <= len(graph.Nodes) && !origin.Zero && origin.Representation == bindingflow.UnchangedRepresentation && graph.Nodes[origin.Node-1].Category == pdg.FormalInput
}
