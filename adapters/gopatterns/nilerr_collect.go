package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"go/ast"
	"go/token"
	"go/types"
)

type nilErrCollector struct {
	fn        *ast.FuncDecl
	fset      *token.FileSet
	info      *types.Info
	positions []int
	hits      []patterns.NilErrHit
	scope     nilErrEvidence
	blocked   bool
	remaining int
	seen      map[token.Pos]bool
}

func (c *nilErrCollector) sequence(statements []ast.Stmt) {
	if len(statements) == 0 || c.remaining == 0 {
		return
	}
	c.remaining--
	first, rest := statements[0], statements[1:]
	switch first := first.(type) {
	case *ast.ReturnStmt:
		return
	case *ast.BlockStmt:
		c.sequence(nilErrContinue(first, rest))
	case *ast.IfStmt:
		c.conditional(first, rest)
	default:
		c.linear(first, rest)
	}
}

func (c *nilErrCollector) linear(statement ast.Stmt, rest []ast.Stmt) {
	if !nilErrLinearStatement(statement) {
		return
	}
	if c.scope.unknownHandling(statement) {
		c.blocked = c.blocked || c.hasContext()
	}
	if nilErrTerminal(statement, c.info) {
		return
	}
	c.scope.record(statement)
	c.sequence(rest)
}

func (c *nilErrCollector) conditional(statement *ast.IfStmt, rest []ast.Stmt) {
	context := c.scope.clone()
	context.initialize(c.fn, statement.Pos())
	context.record(statement.Init)
	if context.unknownHandling(statement.Cond) {
		return
	}
	if truth, known := context.boolean(statement.Cond); known {
		c.branch(statement, rest, truth)
		return
	}
	c.check(statement, rest, context)
	if !nilErrContextCondition(statement.Cond, c.info) {
		return
	}
	c.branch(statement, rest, true)
	c.branch(statement, rest, false)
}

func (c *nilErrCollector) check(statement *ast.IfStmt, rest []ast.Stmt, context nilErrEvidence) {
	if c.blocked || c.seen[statement.Pos()] {
		return
	}
	if hit := c.finding(statement, rest, context); hit != nil {
		c.hits = append(c.hits, *hit)
		c.seen[statement.Pos()] = true
	}
}

func nilErrContextCondition(expr ast.Expr, info *types.Info) bool {
	return nilErrBooleanName(expr, info) || parseNilErrCond(expr, info) != nil
}

func (c *nilErrCollector) branch(statement *ast.IfStmt, rest []ast.Stmt, truth bool) {
	scope, blocked := c.scope, c.blocked
	c.scope = c.scope.clone()
	c.scope.record(statement.Init)
	c.scope.assume(statement.Cond, truth)
	branch := statement.Else
	if truth {
		branch = statement.Body
	}
	c.sequence(nilErrContinue(branch, rest))
	c.scope, c.blocked = scope, blocked
}

func (c *nilErrCollector) hasContext() bool {
	return len(c.scope.truths) > 0 || len(c.scope.guards) > 0
}

func nilErrTerminal(statement ast.Stmt, info *types.Info) bool {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := ast.Unparen(expression.X).(*ast.CallExpr)
	if !ok {
		return false
	}
	return nilErrTerminalCall(call, info)
}

func nilErrTerminalCall(call *ast.CallExpr, info *types.Info) bool {
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		builtin, ok := info.ObjectOf(fun).(*types.Builtin)
		return ok && builtin.Name() == "panic"
	case *ast.SelectorExpr:
		return nilErrTerminalFunction(info.ObjectOf(fun.Sel))
	default:
		return false
	}
}

func nilErrTerminalFunction(object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return false
	}
	name := function.Name()
	switch function.Pkg().Path() {
	case "os":
		return name == "Exit"
	case "runtime":
		return name == "Goexit"
	case "log":
		return nilErrTerminalLogName(name)
	default:
		return false
	}
}

func nilErrTerminalLogName(name string) bool {
	return name == "Fatal" || name == "Fatalf" || name == "Fatalln" || name == "Panic" || name == "Panicf" || name == "Panicln"
}
