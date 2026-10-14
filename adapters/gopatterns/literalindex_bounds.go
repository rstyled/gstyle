package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

type indexBounds uint8

const (
	indexLower indexBounds = 1 << iota
	indexUpper
	indexComplete = indexLower | indexUpper
)

// This bounded proof uses compiler constants and structured source paths. It
// never treats a comparison elsewhere in the function as a dominating guard.
type literalBoundProof struct {
	info   *types.Info
	object types.Object
	target *ast.IndexExpr
	size   int64
}

func (p literalBoundProof) block(block *ast.BlockStmt, bounds indexBounds) indexBounds {
	for _, stmt := range block.List {
		if containsIndex(stmt, p.target) {
			return p.inside(stmt, bounds)
		}
		if p.bindingChanges(stmt) {
			bounds = p.inherentBounds()
		}
		bounds |= p.after(stmt)
	}
	return 0
}

func (p literalBoundProof) inside(stmt ast.Stmt, bounds indexBounds) indexBounds {
	switch stmt := stmt.(type) {
	case *ast.BlockStmt:
		return p.block(stmt, bounds)
	case *ast.IfStmt:
		return p.branch(stmt, bounds)
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.SelectStmt, *ast.LabeledStmt:
		return 0 // No loop, jump or multi-way-flow proof in this subset.
	default:
		if p.bindingChanges(stmt) {
			return 0
		}
		return bounds
	}
}

func (p literalBoundProof) branch(stmt *ast.IfStmt, bounds indexBounds) indexBounds {
	if p.bindingChanges(stmt.Init) {
		bounds = p.inherentBounds()
	}
	if containsIndex(stmt.Body, p.target) {
		return p.block(stmt.Body, bounds|p.condition(stmt.Cond, true))
	}
	if stmt.Else != nil && containsIndex(stmt.Else, p.target) {
		return p.inside(stmt.Else, bounds|p.condition(stmt.Cond, false))
	}
	return 0
}

func (p literalBoundProof) after(stmt ast.Stmt) indexBounds {
	branch, ok := stmt.(*ast.IfStmt)
	if !ok || p.bindingChanges(branch) {
		return 0
	}
	if branchReturns(branch.Body) {
		return p.condition(branch.Cond, false)
	}
	if branch.Else != nil && statementReturns(branch.Else) {
		return p.condition(branch.Cond, true)
	}
	return 0
}

func (p literalBoundProof) condition(expr ast.Expr, truth bool) indexBounds {
	expr = unparen(expr)
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		return p.condition(unary.X, !truth)
	}
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return 0
	}
	if bin.Op == token.LAND || bin.Op == token.LOR {
		return p.boolean(bin, truth)
	}
	return p.comparison(bin, truth)
}

func (p literalBoundProof) boolean(bin *ast.BinaryExpr, truth bool) indexBounds {
	left, right := p.condition(bin.X, truth), p.condition(bin.Y, truth)
	if (bin.Op == token.LAND) == truth {
		return left | right
	}
	return left & right
}

func (p literalBoundProof) comparison(bin *ast.BinaryExpr, truth bool) indexBounds {
	op := bin.Op
	if !truth {
		op = inverseComparison(op)
	}
	if p.isIndex(bin.X) {
		return p.constantBounds(op, bin.Y)
	}
	if p.isIndex(bin.Y) {
		return p.constantBounds(reverseComparison(op), bin.X)
	}
	return 0
}

func (p literalBoundProof) isIndex(expr ast.Expr) bool {
	id, ok := unparen(expr).(*ast.Ident)
	return ok && p.info.ObjectOf(id) == p.object
}

func (p literalBoundProof) constantBounds(op token.Token, expr ast.Expr) indexBounds {
	value, known := integerConstant(expr, p.info)
	if !known {
		return 0
	}
	return p.bounds(op, value)
}

func (p literalBoundProof) bounds(op token.Token, value int64) indexBounds {
	switch op {
	case token.GEQ:
		return boundWhen(value >= 0, indexLower)
	case token.GTR:
		return boundWhen(value >= -1, indexLower)
	case token.LSS:
		return boundWhen(value <= p.size, indexUpper)
	case token.LEQ:
		return boundWhen(value < p.size, indexUpper)
	case token.EQL:
		return boundWhen(p.withinTable(value), indexComplete)
	default:
		return 0
	}
}

func (p literalBoundProof) bindingChanges(node ast.Node) bool {
	if node == nil {
		return false
	}
	uses := allocationUses{info: p.info, writes: map[types.Object]int{}, escaped: map[types.Object]bool{}}
	ast.Inspect(node, uses.visit)
	return uses.jumps || uses.writes[p.object] != 0 || uses.escaped[p.object]
}

func (p literalBoundProof) withinTable(value int64) bool {
	return value >= 0 && value < p.size
}

func (p literalBoundProof) inherentBounds() indexBounds {
	basic, ok := p.object.Type().Underlying().(*types.Basic)
	if ok && basic.Info()&types.IsUnsigned != 0 {
		return indexLower
	}
	return 0
}

func literalIndexBounded(fn *ast.FuncDecl, idx *ast.IndexExpr, info *types.Info) bool {
	id, ok := unparen(idx.Index).(*ast.Ident)
	if !ok {
		return false
	}
	size, known := literalTableSize(idx.X, info)
	if !known {
		return false
	}
	p := literalBoundProof{info: info, object: info.ObjectOf(id), target: idx, size: size}
	bounds := p.inherentBounds()
	return p.block(fn.Body, bounds) == indexComplete
}

func literalTableSize(expr ast.Expr, info *types.Info) (int64, bool) {
	lit, ok := unparen(expr).(*ast.CompositeLit)
	if !ok {
		return 0, false
	}
	if array, ok := info.TypeOf(lit).Underlying().(*types.Array); ok {
		return array.Len(), true
	}
	var next, size int64
	for _, element := range lit.Elts {
		if keyed, ok := element.(*ast.KeyValueExpr); ok {
			var known bool
			next, known = integerConstant(keyed.Key, info)
			if !known {
				return 0, false
			}
		}
		next++
		size = max(size, next)
	}
	return size, true
}

func boundWhen(known bool, bounds indexBounds) indexBounds {
	if known {
		return bounds
	}
	return 0
}

func integerConstant(expr ast.Expr, info *types.Info) (int64, bool) {
	value := info.Types[expr].Value
	if value == nil || value.Kind() != constant.Int {
		return 0, false
	}
	return constant.Int64Val(value)
}

func containsIndex(node ast.Node, idx *ast.IndexExpr) bool {
	return node.Pos() <= idx.Pos() && idx.End() <= node.End()
}

func inverseComparison(op token.Token) token.Token {
	return map[token.Token]token.Token{
		token.LSS: token.GEQ, token.LEQ: token.GTR, token.GTR: token.LEQ,
		token.GEQ: token.LSS, token.EQL: token.NEQ, token.NEQ: token.EQL,
	}[op]
}

func reverseComparison(op token.Token) token.Token {
	return map[token.Token]token.Token{
		token.LSS: token.GTR, token.LEQ: token.GEQ, token.GTR: token.LSS,
		token.GEQ: token.LEQ, token.EQL: token.EQL, token.NEQ: token.NEQ,
	}[op]
}
