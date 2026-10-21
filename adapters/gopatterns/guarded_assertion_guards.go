package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *assertionGuardQuery) guards(edge assertionEdge, required assertionRequirement) bool {
	block := q.flow.Blocks[edge.block-1]
	condition := block.Condition
	if condition == 0 || len(block.Successors) != 2 {
		return false
	}
	if clause, ok := q.source[condition].(*ast.CaseClause); ok {
		return edge.outcome == 0 && q.caseGuard(condition, clause, required)
	}
	branch := q.conditionBranch(condition)
	if branch == nil {
		return false
	}
	return q.checkedGuard(condition, branch, edge.outcome, required)
}

func (q *assertionGuardQuery) caseGuard(ref pdg.Ref, clause *ast.CaseClause, required assertionRequirement) bool {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if attrs.Operator.Status != pdg.Known || q.graph.Tables.Text(attrs.Operator.Value) != "type-is" || len(clause.List) != 1 {
		return false
	}
	if !types.Identical(q.info.TypeOf(clause.List[0]), required.typ) {
		return false
	}
	inputs := q.controls[ref]
	return len(inputs) == 1 && q.bindings.Path(inputs[0], q.facts).Same(required.path)
}

func assertionControlOperands(graph *pdg.Graph) map[pdg.Ref][]pdg.Ref {
	inputs := map[pdg.Ref][]pdg.Ref{}
	for _, edge := range graph.Edges {
		if edge.Kind == pdg.Data && edge.Position == 1 && graph.Tables.Text(edge.Subkind) == "control-value" {
			inputs[edge.Target] = append(inputs[edge.Target], edge.Source)
		}
	}
	return inputs
}

func (q *assertionGuardQuery) conditionBranch(ref pdg.Ref) *ast.IfStmt {
	if branch, ok := q.source[ref].(*ast.IfStmt); ok {
		return branch
	}
	expression, _ := q.source[ref].(ast.Expr)
	return q.conditions[expression]
}

func (q *assertionGuardQuery) checkedGuard(ref pdg.Ref, branch *ast.IfStmt, outcome int, required assertionRequirement) bool {
	assertion, result := checkedAssertion(branch.Init)
	if assertion == nil || !types.Identical(q.info.TypeOf(assertion.Type), required.typ) {
		return false
	}
	if q.source[ref] == branch {
		ref = q.bindings.Operand(ref, 1)
	}
	if !q.checkedOutcome(ref, branch.Cond, result, outcome) {
		return false
	}
	subject := q.bindings.Operand(q.refs[assertion], 1)
	return q.bindings.Path(subject, q.facts).Same(required.path)
}

func checkedAssertion(statement ast.Stmt) (*ast.TypeAssertExpr, ast.Expr) {
	assign, ok := statement.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return nil, nil
	}
	assertion, ok := ast.Unparen(assign.Rhs[0]).(*ast.TypeAssertExpr)
	if !ok || assertion.Type == nil {
		return nil, nil
	}
	return assertion, assign.Lhs[1]
}

func (q *assertionGuardQuery) checkedOutcome(ref pdg.Ref, condition, result ast.Expr, outcome int) bool {
	id, ref, want := q.assertionOutcome(ref, condition)
	if id == nil || outcome != want {
		return false
	}
	resultID, _ := result.(*ast.Ident)
	if !q.freshCheckResult(id, resultID) {
		return false
	}
	read, ok := q.source[q.bindings.IdentitySource(ref, q.facts)].(*ast.Ident)
	return ok && q.info.ObjectOf(read) == q.info.ObjectOf(id)
}

// A reused or shared boolean can be stale. The supported if initializer
// creates its result binding immediately before the condition reads it.
func (q *assertionGuardQuery) freshCheckResult(condition, result *ast.Ident) bool {
	return condition != nil && result != nil && q.info.Defs[result] != nil && q.info.ObjectOf(condition) == q.info.Defs[result]
}

func (q *assertionGuardQuery) assertionOutcome(ref pdg.Ref, condition ast.Expr) (*ast.Ident, pdg.Ref, int) {
	want := 0
	if unary, ok := ast.Unparen(condition).(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		condition, ref, want = unary.X, q.bindings.Operand(ref, 1), 1
	}
	id, _ := ast.Unparen(condition).(*ast.Ident)
	return id, ref, want
}
