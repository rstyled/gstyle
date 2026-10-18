package gopatterns

import "go/ast"

// x/tools represents branch transfers only as CFG arcs. A private syntax view
// places an operation marker before each transfer, preserving its actual source
// node and resolved CFG target. Compiler-owned syntax is never changed.
func (b *irBuilder) branchCFGBlock(block *ast.BlockStmt) *ast.BlockStmt {
	out := *block
	out.List = b.branchCFGList(block.List)
	return &out
}

func (b *irBuilder) branchCFGList(statements []ast.Stmt) []ast.Stmt {
	out := make([]ast.Stmt, len(statements))
	for index, statement := range statements {
		out[index] = b.branchCFGStatement(statement)
	}
	return out
}

func (b *irBuilder) branchCFGStatement(statement ast.Stmt) ast.Stmt {
	switch node := statement.(type) {
	case *ast.BlockStmt:
		return b.branchCFGBlock(node)
	case *ast.BranchStmt:
		marker := &ast.EmptyStmt{Semicolon: node.Pos()}
		b.nodes[marker] = b.nodes[node]
		return &ast.BlockStmt{List: []ast.Stmt{marker, node}}
	case *ast.LabeledStmt:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Stmt = b.branchCFGStatement(node.Stmt)
		return &out
	case *ast.CaseClause:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGList(node.Body)
		return &out
	case *ast.CommClause:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGList(node.Body)
		return &out
	default:
		return b.branchCFGSelection(statement)
	}
}

func (b *irBuilder) branchCFGSelection(statement ast.Stmt) ast.Stmt {
	switch node := statement.(type) {
	case *ast.IfStmt:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGBlock(node.Body)
		if node.Else != nil {
			out.Else = b.branchCFGStatement(node.Else)
		}
		return &out
	case *ast.SwitchStmt:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGBlock(node.Body)
		return &out
	case *ast.TypeSwitchStmt:
		return b.branchCFGTypeSwitch(node)
	case *ast.SelectStmt:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGBlock(node.Body)
		return &out
	default:
		return b.branchCFGLoop(statement)
	}
}

func (b *irBuilder) branchCFGLoop(statement ast.Stmt) ast.Stmt {
	switch node := statement.(type) {
	case *ast.ForStmt:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGBlock(node.Body)
		return &out
	case *ast.RangeStmt:
		out := *node
		b.nodes[&out] = b.nodes[node]
		out.Body = b.branchCFGBlock(node.Body)
		return &out
	default:
		return statement
	}
}
