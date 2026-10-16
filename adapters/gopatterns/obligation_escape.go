package gopatterns

import (
	"go/ast"
	"go/token"
)

// Unresolved handoffs are unknown, not proof that a caller dropped the file.
func (p *releaseProof) ownershipEscapes(fn *ast.FuncDecl) bool {
	escaped := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		escaped = escaped || p.ownershipEscape(node)
		return !escaped
	})
	return escaped
}

func (p *releaseProof) ownershipEscape(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.CallExpr:
		return p.callHandoff(node)
	case *ast.AssignStmt:
		return p.aliasHandoff(node)
	case *ast.ReturnStmt:
		return p.mentionsResource(node) && !p.discharges(node)
	default:
		return p.indirectHandoff(node)
	}
}

func (p *releaseProof) indirectHandoff(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.UnaryExpr:
		return node.Op == token.AND && p.isResource(node.X)
	case *ast.FuncLit, *ast.GoStmt, *ast.SendStmt, *ast.ValueSpec:
		return p.mentionsResource(node)
	default:
		return false
	}
}

func (p *releaseProof) callHandoff(call *ast.CallExpr) bool {
	if p.releaseCall(call) || p.borrowedFileView(call) {
		return false
	}
	for _, arg := range call.Args {
		if p.mentionsResource(arg) && !p.borrowedFileExpression(arg) {
			return true
		}
	}
	return false
}

func (p *releaseProof) aliasHandoff(assign *ast.AssignStmt) bool {
	for i, rhs := range assign.Rhs {
		if !p.assignedResource(rhs) {
			continue
		}
		if i >= len(assign.Lhs) || !blankResourceTarget(assign.Lhs[i]) {
			return true
		}
	}
	return false
}

func blankResourceTarget(expr ast.Expr) bool {
	name, ok := expr.(*ast.Ident)
	return ok && name.Name == "_"
}

func (p *releaseProof) mentionsResource(node ast.Node) bool {
	return nilErrRootMention(node, p.resource, p.info)
}

// io.TeeReader returns the standard read-only teeReader wrapper, not the
// underlying file. Passing that view does not hand off the Close obligation.
func (p *releaseProof) borrowedFileView(call *ast.CallExpr) bool {
	function := fileCallObject(p.info, call)
	return function != nil && function.FullName() == "io.TeeReader"
}

func (p *releaseProof) borrowedFileExpression(expr ast.Expr) bool {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	return ok && p.borrowedFileView(call)
}

func (p *releaseProof) assignedResource(expr ast.Expr) bool {
	if _, call := ast.Unparen(expr).(*ast.CallExpr); call {
		return false
	}
	return p.mentionsResource(expr)
}
