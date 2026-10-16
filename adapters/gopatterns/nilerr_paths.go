package gopatterns

import (
	"go/ast"
	"go/types"
)

const nilErrPathBudget = 256

type nilErrEvidence struct {
	checked   ast.Expr
	positions []int
	info      *types.Info
	values    map[types.Object]nilErrValue
	truths    map[types.Object]bool
	results   []types.Object
	budget    *int
	guards    []nilErrCheck
}

func (c *nilErrCollector) lostPath(statements []ast.Stmt, checked ast.Expr, context nilErrEvidence) bool {
	if !sameNilErrValue(checked, checked, c.info) || nilErrUnstable(c.fn, checked, c.info) {
		return false
	}
	budget := nilErrPathBudget
	evidence := context.clone()
	evidence.checked, evidence.positions, evidence.budget = checked, c.positions, &budget
	evidence.assumeError(nilErrCheck{errVal: checked, isNotNil: true})
	return evidence.sequence(statements)
}

func nilErrStatements(statement ast.Stmt) []ast.Stmt {
	if statement == nil {
		return nil
	}
	if block, ok := statement.(*ast.BlockStmt); ok {
		return block.List
	}
	return []ast.Stmt{statement}
}

func (e nilErrEvidence) sequence(statements []ast.Stmt) bool {
	if len(statements) == 0 || *e.budget == 0 {
		return false
	}
	*e.budget--
	first, rest := statements[0], statements[1:]
	switch first := first.(type) {
	case *ast.BlockStmt:
		return e.sequence(nilErrContinue(first, rest))
	case *ast.ReturnStmt:
		return e.lostReturn(first)
	case *ast.IfStmt:
		return e.conditional(first, rest)
	default:
		return e.next(first, rest)
	}
}

func (e nilErrEvidence) next(statement ast.Stmt, rest []ast.Stmt) bool {
	if !nilErrLinearStatement(statement) || e.unknownHandling(statement) {
		return false
	}
	e.record(statement)
	return e.sequence(rest)
}

func nilErrLinearStatement(statement ast.Stmt) bool {
	switch statement.(type) {
	case *ast.ExprStmt, *ast.AssignStmt, *ast.DeclStmt, *ast.EmptyStmt:
		return true
	default:
		return false
	}
}

func (e nilErrEvidence) conditional(statement *ast.IfStmt, rest []ast.Stmt) bool {
	if e.unknownHandling(statement.Init) || e.unknownHandling(statement.Cond) {
		return false
	}
	e.record(statement.Init)
	if value, known := e.boolean(statement.Cond); known {
		return e.branch(statement, rest, value)
	}
	if !e.canSplit(statement.Cond) {
		return false
	}
	return e.branch(statement, rest, true) || e.branch(statement, rest, false)
}

func (e nilErrEvidence) branch(statement *ast.IfStmt, rest []ast.Stmt, truth bool) bool {
	path := e.clone()
	path.assume(statement.Cond, truth)
	if truth {
		return path.sequence(nilErrContinue(statement.Body, rest))
	}
	return path.sequence(nilErrContinue(statement.Else, rest))
}

func nilErrContinue(statement ast.Stmt, rest []ast.Stmt) []ast.Stmt {
	return append(append([]ast.Stmt{}, nilErrStatements(statement)...), rest...)
}

func (e nilErrEvidence) clone() nilErrEvidence {
	values, truths := e.values, e.truths
	e.values = make(map[types.Object]nilErrValue, len(values))
	e.truths = make(map[types.Object]bool, len(truths))
	for key, value := range values {
		e.values[key] = value
	}
	for key, value := range truths {
		e.truths[key] = value
	}
	return e
}

func (e nilErrEvidence) canSplit(expr ast.Expr) bool {
	return !e.mentions(expr) && e.value(expr).boolean != nil
}
