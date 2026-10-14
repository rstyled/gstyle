package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Only a single top-level local definition proves non-nil. Writes, address
// escapes and jumps invalidate this bounded proof, including writes in closures.
func nonNilLocalDefinitions(fn *ast.FuncDecl, info *types.Info) map[types.Object]token.Pos {
	allocated := map[types.Object]token.Pos{}
	for _, stmt := range fn.Body.List {
		recordNonNilDefinition(stmt, info, allocated)
	}
	uses := allocationUses{info: info, writes: map[types.Object]int{}, escaped: map[types.Object]bool{}}
	ast.Inspect(fn.Body, uses.visit)
	for obj := range allocated {
		if uses.jumps || uses.writes[obj] != 1 || uses.escaped[obj] {
			delete(allocated, obj)
		}
	}
	return allocated
}

func recordNonNilDefinition(stmt ast.Stmt, info *types.Info, allocated map[types.Object]token.Pos) {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || !singleLocalDefinition(assign) {
		return
	}
	id, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || !definitelyNonNil(assign.Rhs[0], info) {
		return
	}
	obj := info.Defs[id]
	if isPointerVar(obj) {
		allocated[obj] = stmt.End()
	}
}

func definitelyNonNil(expr ast.Expr, info *types.Info) bool {
	switch expr := unparen(expr).(type) {
	case *ast.UnaryExpr:
		return expr.Op == token.AND
	case *ast.CallExpr:
		id, ok := unparen(expr.Fun).(*ast.Ident)
		if !ok {
			return false
		}
		builtin, ok := info.ObjectOf(id).(*types.Builtin)
		return ok && builtin.Name() == "new"
	default:
		return false
	}
}

type allocationUses struct {
	info    *types.Info
	writes  map[types.Object]int
	escaped map[types.Object]bool
	jumps   bool
}

func (u *allocationUses) visit(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.AssignStmt:
		u.writeExpressions(node.Lhs)
	case *ast.ValueSpec:
		u.writeNames(node.Names)
	case *ast.UnaryExpr:
		u.address(node)
	case *ast.BranchStmt:
		u.jumps = u.jumps || node.Tok == token.GOTO
	}
	return true
}

func (u *allocationUses) write(expr ast.Expr) {
	id, ok := unparen(expr).(*ast.Ident)
	if ok {
		u.writes[u.info.ObjectOf(id)]++
	}
}

func (u *allocationUses) escape(expr ast.Expr) {
	id, ok := unparen(expr).(*ast.Ident)
	if ok {
		u.escaped[u.info.ObjectOf(id)] = true
	}
}

func allocationPrecedes(allocated map[types.Object]token.Pos, deref nilDeref) bool {
	pos := allocated[deref.obj]
	return pos != token.NoPos && pos < deref.pos
}

func singleLocalDefinition(assign *ast.AssignStmt) bool {
	return assign.Tok == token.DEFINE && len(assign.Lhs) == 1 && len(assign.Rhs) == 1
}

func (u *allocationUses) writeExpressions(expressions []ast.Expr) {
	for _, expr := range expressions {
		u.write(expr)
	}
}

func (u *allocationUses) writeNames(names []*ast.Ident) {
	for _, name := range names {
		u.write(name)
	}
}

func (u *allocationUses) address(expr *ast.UnaryExpr) {
	if expr.Op == token.AND {
		u.escape(expr.X)
	}
}
