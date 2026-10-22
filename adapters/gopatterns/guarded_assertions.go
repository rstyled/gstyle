package gopatterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

type assertionGuardQuery struct {
	graph      *pdg.Graph
	flow       *pdg.NormalFlow
	bindings   *bindingflow.Query
	facts      bindingflow.Facts
	info       *types.Info
	source     map[pdg.Ref]ast.Node
	refs       map[ast.Node]pdg.Ref
	conditions map[ast.Expr]*ast.IfStmt
	previous   [][]assertionEdge
	controls   map[pdg.Ref][]pdg.Ref
	commaOK    map[*ast.TypeAssertExpr]bool
}

// Syntax identifies typed guard and return boundaries. Only canonical operands,
// writes and paths can prove an assertion safe. Missing flow keeps warnings.
func provenTypeAssertions(pkg *packages.Package, source functionGraph, summaries map[types.Object]types.Type, prepared ...*bindingflow.Query) map[*ast.TypeAssertExpr]bool {
	fn, fset, info, graph := source.declaration, pkg.Fset, pkg.TypesInfo, source.graph
	if !assertionGuardAvailable(fn, info, graph) {
		return nil
	}
	if !assertionProofNeeded(fn, fset, summaries) {
		return nil
	}
	bindings, err := nilErrBindings(graph, prepared)
	if err != nil {
		return nil
	}
	facts, err := bindings.ValueFacts()
	if err != nil {
		return nil
	}
	flow := pdg.BodyFlow(graph)
	sourceNodes, refs := assertionSources(fn, fset, graph)
	query := assertionGuardQuery{graph: graph, flow: flow, bindings: bindings, facts: facts, info: info, source: sourceNodes, refs: refs, conditions: assertionConditions(fn), previous: assertionPredecessors(flow), controls: assertionControlOperands(graph), commaOK: collectCommaOk(fn.Body)}
	return query.proven(summaries)
}

func assertionProofNeeded(fn *ast.FuncDecl, fset *token.FileSet, summaries map[types.Object]types.Type) bool {
	hits := len(findForceTypeAssertHits(fn, fset))
	return hits > 0 && (hits > 1 || len(summaries) > 0 || hasAssertionGuard(fn))
}

func assertionGuardAvailable(fn *ast.FuncDecl, info *types.Info, graph *pdg.Graph) bool {
	return fn.Body != nil && info != nil && graph != nil && pdg.ReachableProjection(graph) != nil
}

func hasAssertionGuard(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.TypeSwitchStmt:
			found = true
		case *ast.IfStmt:
			found = found || node.Init != nil
		}
		return !found
	})
	return found
}

func assertionSources(fn *ast.FuncDecl, fset *token.FileSet, graph *pdg.Graph) (map[pdg.Ref]ast.Node, map[ast.Node]pdg.Ref) {
	nodes := nilErrSourceNodes(fn, fset)
	sourceNodes, refs := map[pdg.Ref]ast.Node{}, map[ast.Node]pdg.Ref{}
	for i, node := range graph.Nodes {
		if node.Span == 0 {
			continue
		}
		span := graph.Spans[node.Span-1]
		source := nodes[fmt.Sprintf("%s:%d:%d", graph.Tables.Text(node.OriginalKind.Value), span.Start, span.End)]
		if graph.Tables.Text(graph.NodeAttributes(node).Operator.Value) == "type-is" {
			source = nodes[fmt.Sprintf("*ast.CaseClause:%d:%d", span.Start, span.End)]
		}
		sourceNodes[pdg.Ref(i+1)], refs[source] = source, pdg.Ref(i+1)
	}
	return sourceNodes, refs
}

func assertionConditions(fn *ast.FuncDecl) map[ast.Expr]*ast.IfStmt {
	conditions := map[ast.Expr]*ast.IfStmt{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if branch, ok := node.(*ast.IfStmt); ok {
			conditions[branch.Cond] = branch
		}
		return true
	})
	return conditions
}

func (q *assertionGuardQuery) proven(summaries map[types.Object]types.Type) map[*ast.TypeAssertExpr]bool {
	proven := map[*ast.TypeAssertExpr]bool{}
	for source, ref := range q.refs {
		assertion, ok := source.(*ast.TypeAssertExpr)
		if !ok || assertion.Type == nil {
			continue
		}
		if q.protectedAssertion(ref, assertion) || q.returnedType(ref, assertion, summaries) {
			proven[assertion] = true
		}
	}
	return proven
}

func (q *assertionGuardQuery) protectedAssertion(ref pdg.Ref, assertion *ast.TypeAssertExpr) bool {
	path := q.localPath(assertion.X, q.bindings.Operand(ref, 1))
	return path.Root != 0 && q.protected(ref, assertionRequirement{path, q.info.TypeOf(assertion.Type)})
}

func (q *assertionGuardQuery) localPath(expression ast.Expr, ref pdg.Ref) bindingflow.Path {
	id, ok := ast.Unparen(expression).(*ast.Ident)
	if !ok {
		return bindingflow.Path{}
	}
	object := localAssertionObject(q.info, id)
	if object == nil {
		return bindingflow.Path{}
	}
	read, ok := q.source[q.bindings.IdentitySource(ref, q.facts)].(*ast.Ident)
	if !ok || q.info.ObjectOf(read) != object {
		return bindingflow.Path{}
	}
	return q.stablePath(ref)
}

func localAssertionObject(info *types.Info, id *ast.Ident) types.Object {
	object := info.ObjectOf(id)
	if object == nil || object.Pkg() == nil || object.Parent() == object.Pkg().Scope() {
		return nil
	}
	return object
}

func (q *assertionGuardQuery) stablePath(ref pdg.Ref) bindingflow.Path {
	path := q.bindings.Path(ref, q.facts)
	if len(path.Fields) != 0 || q.facts.CapturedSymbols[path.Root] || !q.bindings.StablePath(path, q.facts) {
		return bindingflow.Path{}
	}
	return path
}
