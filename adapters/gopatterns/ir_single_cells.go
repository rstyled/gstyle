package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

type cellSourceFacts struct {
	stack []ast.Node
	local map[types.Object]bool
	loops int
	gotos bool
	info  *types.Info
}

func (b *irBuilder) singleCellFacts(fn *ast.FuncDecl) {
	facts := cellSourceFacts{local: map[types.Object]bool{}, info: b.info}
	ast.Inspect(fn.Body, facts.visit)
	for object, ref := range b.symbols {
		symbol := &b.graph.Symbols[ref-1]
		single := facts.single(object, symbol.Role)
		symbol.Extensions = append(symbol.Extensions, b.pool.SingletonExtension("gstyle.memory:single-cell", strconv.FormatBool(single))...)
	}
}

func (facts *cellSourceFacts) visit(node ast.Node) bool {
	if _, closure := node.(*ast.FuncLit); closure {
		return false
	}
	if node == nil {
		last := facts.stack[len(facts.stack)-1]
		facts.stack = facts.stack[:len(facts.stack)-1]
		facts.loops -= cellLoop(last)
		return true
	}
	facts.stack = append(facts.stack, node)
	facts.loops += cellLoop(node)
	facts.record(node)
	return true
}

func (facts *cellSourceFacts) record(node ast.Node) {
	if branch, ok := node.(*ast.BranchStmt); ok && branch.Tok == token.GOTO {
		facts.gotos = true
	}
	id, ok := node.(*ast.Ident)
	if !ok {
		return
	}
	if object, variable := facts.info.Defs[id].(*types.Var); variable {
		facts.local[object] = facts.loops == 0
	}
}

func (facts *cellSourceFacts) single(object types.Object, role pdg.Role) bool {
	switch role {
	case pdg.ParameterRole, pdg.ReceiverRole, pdg.OutputRole, pdg.ReturnRole, pdg.CaptureRole:
		return true
	case pdg.LocalRole:
		return facts.local[object] && !facts.gotos
	default:
		return false
	}
}

func cellLoop(node ast.Node) int {
	switch node.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return 1
	default:
		return 0
	}
}
