package gopatterns

import (
	"go/ast"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Each type-switch clause has a distinct compiler binding. The source guard's
// identifier is not a shared variable definition for all clauses.
func (b *irBuilder) branchCFGTypeSwitch(statement *ast.TypeSwitchStmt) ast.Stmt {
	out := *statement
	b.nodes[&out] = b.nodes[statement]
	out.Body = b.branchCFGBlock(statement.Body)
	for index, original := range statement.Body.List {
		clause := original.(*ast.CaseClause)
		if binding := b.typeSwitchBinding(statement, clause); binding != 0 {
			marker := &ast.EmptyStmt{Semicolon: clause.Pos()}
			b.nodes[marker] = binding
			target := out.Body.List[index].(*ast.CaseClause)
			target.Body = append([]ast.Stmt{marker}, target.Body...)
		}
	}
	return &out
}

func (b *irBuilder) typeSwitchBinding(statement *ast.TypeSwitchStmt, clause *ast.CaseClause) pdg.Ref {
	if !b.generalBindings {
		return 0
	}
	symbol := b.symbols[b.info.Implicits[clause]]
	if symbol == 0 {
		return 0
	} // No uses or writes need a definition for this clause.
	attrs := pdg.Attributes{Construct: b.pool.Text("type-switch-binding"), WriteForm: b.pool.Text(":="), Writes: []pdg.Access{{Symbol: symbol, Position: 1}}, Symbols: []pdg.Ref{symbol}}
	ref := b.scalarEntity(b.scalarSourceSpan(clause), pdg.Assignment, attrs)
	b.definition(ref, symbol)
	b.typeSwitchValueEdge(b.nodes[typeSwitchValue(statement.Assign)], ref)
	return ref
}

func (b *irBuilder) typeSwitchValueEdge(source, target pdg.Ref) {
	description := "Type-switch clause binding depends on the switched interface value; dynamic representation is conservative."
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("case-value-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: target, Kind: pdg.Data, Position: 1, Subkind: b.pool.Text("type-switch-value"), Evidence: b.pool.Evidence(pdg.Approximated, description), Provenance: b.sourceOrigin})
}

func (b *irBuilder) typeSwitchGuard(statement *ast.AssignStmt) bool {
	if len(b.stack) == 0 {
		return false
	}
	parent, ok := b.stack[len(b.stack)-1].(*ast.TypeSwitchStmt)
	return ok && parent.Assign == statement
}
