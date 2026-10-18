package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

// Control lowering owns its CFG. It does not change compiler syntax or the
// scalar/value-flow profiles. Each operation gets its own vertex so a back edge
// cannot turn an entire basic block into false self-control dependencies.
func (b *irBuilder) controlCFG(body *ast.BlockStmt, original *cfg.CFG) (*cfg.CFG, map[int][]pdg.Position) {
	a := scalarCFGBuilder{builder: b, graph: &cfg.CFG{}, controls: b.executionControls(body), maximal: true, outcomes: map[int][]pdg.Position{}, conditions: controlConditions(body), selections: controlSelections(body), selectors: map[*ast.CaseClause]int{}, rangeTargets: controlRangeTargets(body), rangeBounds: b.controlRangeBounds(body)}
	for range original.Blocks {
		a.block()
	}
	for index, block := range original.Blocks {
		a.current = a.graph.Blocks[index]
		for _, source := range block.Nodes {
			a.controlSource(source)
		}
		b.controlDecision(&a, block)
		next := make([]*cfg.Block, len(block.Succs))
		for arm, successor := range block.Succs {
			next[arm] = a.graph.Blocks[successor.Index]
		}
		a.transfer(controlCondition(block, a.conditions), a.rangeSuccessors(block, next))
	}
	return splitControlOperations(a.graph, a.outcomes)
}

// x/tools emits real switch transfers but no operation for type tests or range
// decisions. Give those decisions explicit source-backed predicate operations.
func (b *irBuilder) controlDecision(a *scalarCFGBuilder, block *cfg.Block) {
	if len(block.Succs) != 2 {
		return
	}
	if block.Kind == cfg.KindRangeLoop {
		ref := b.nodes[block.Stmt]
		b.attributes[b.graph.Nodes[ref-1].Attributes-1].Outcomes = []pdg.Text{b.pool.Text("true"), b.pool.Text("false")}
		a.operation(ref)
		b.controlValueEdge(b.rangeSource(block.Stmt.(*ast.RangeStmt)), ref, 1)
		return
	}
	clause, ok := block.Succs[0].Stmt.(*ast.CaseClause)
	if !ok {
		return
	}
	construct := "switch-case"
	ref := a.selectionPredicate(clause)
	b.graph.Nodes[ref-1].MatchLabel = b.pool.MatchLabel(pdg.MatchLabel{Kind: b.pool.Text(string(patterns.Branch)), Detail: b.pool.Text(construct)})
	a.operation(ref)
}

// A compiler constant or a short-circuit constant result can remove an
// impossible outcome without evaluating an unexecuted operand.
func (b *irBuilder) controlBoolean(expression ast.Expr) (bool, bool) {
	if value := b.info.Types[expression].Value; value != nil && value.Kind() == constant.Bool {
		return constant.BoolVal(value), true
	}
	binary, ok := logicalExpression(unparen(expression))
	if !ok {
		return false, false
	}
	left, known := b.controlBoolean(binary.X)
	if !known {
		return false, false
	}
	if shortCircuitSkips(binary.Op, left) {
		return left, true
	}
	return b.controlBoolean(binary.Y)
}

func (a *scalarCFGBuilder) transfer(condition ast.Node, successors []*cfg.Block) {
	a.current.Succs = successors
	if !a.maximal || len(successors) != 2 {
		return
	}
	expression, ok := condition.(ast.Expr)
	if !ok {
		return
	}
	value, known := a.builder.controlBoolean(expression)
	if !known {
		return
	}
	arm := 1
	if value {
		arm = 0
	}
	a.current.Succs = successors[arm : arm+1]
	a.outcomes[int(a.current.Index)] = []pdg.Position{pdg.Position(arm + 1)}
}

func controlConditions(body *ast.BlockStmt) map[ast.Node]bool {
	conditions := map[ast.Node]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IfStmt:
			conditions[node.Cond] = true
		case *ast.ForStmt:
			conditions[node.Cond] = true
		case *ast.SwitchStmt:
			if node.Tag == nil {
				markBooleanCases(conditions, node.Body)
			}
		}
		return true
	})
	return conditions
}

func controlCondition(block *cfg.Block, conditions map[ast.Node]bool) ast.Node {
	if len(block.Nodes) == 0 {
		return nil
	}
	last := block.Nodes[len(block.Nodes)-1]
	if !conditions[last] {
		return nil
	}
	return last
}

func splitControlOperations(flow *cfg.CFG, outcomes map[int][]pdg.Position) (*cfg.CFG, map[int][]pdg.Position) {
	out := &cfg.CFG{Blocks: make([]*cfg.Block, len(flow.Blocks))}
	for index := range out.Blocks {
		out.Blocks[index] = &cfg.Block{Index: int32(index)}
	}
	positions := map[int][]pdg.Position{}
	for index, block := range flow.Blocks {
		last := splitControlBlock(out, out.Blocks[index], block.Nodes)
		for _, next := range block.Succs {
			last.Succs = append(last.Succs, out.Blocks[next.Index])
		}
		positions[int(last.Index)] = outcomes[index]
	}
	return out, positions
}

func splitControlBlock(out *cfg.CFG, first *cfg.Block, nodes []ast.Node) *cfg.Block {
	current := first
	for index, node := range nodes {
		if index != 0 {
			next := &cfg.Block{Index: int32(len(out.Blocks))}
			out.Blocks = append(out.Blocks, next)
			current.Succs = []*cfg.Block{next}
			current = next
		}
		current.Nodes = []ast.Node{node}
	}
	return current
}

func shortCircuitSkips(operator token.Token, left bool) bool {
	return (operator == token.LAND && !left) || (operator == token.LOR && left)
}

func markBooleanCases(conditions map[ast.Node]bool, body *ast.BlockStmt) {
	for _, statement := range body.List {
		for _, expression := range statement.(*ast.CaseClause).List {
			conditions[expression] = true
		}
	}
}
