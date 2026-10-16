package gopatterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

const controlVertexLimit = 16384

const semanticControlDescription = "Divergence-sensitive post-dominator control dependence on operation CFG flow, including finite and infinite paths. Ordinary calls are assumed to return; built-in panic, os.Exit and runtime.Goexit terminate paths. Implicit panics, recovery, deferred execution and concurrent control are not modeled."

func (b *irBuilder) semanticControlFor(body *ast.BlockStmt, flow *cfg.CFG) error {
	if b.normalDependencies || !b.semanticControlSubset(body) {
		return nil
	}
	flow, outcomes := b.controlCFG(body, flow)
	if len(flow.Blocks) > controlVertexLimit {
		return fmt.Errorf("control dependence exceeds %d operation CFG vertices", controlVertexLimit)
	}
	dom := newExecutionDominators(flow)
	sets := maximalPostDominators(flow, dom)
	a := normalFlowAnalysis{builder: b, operations: make([][]pdg.Ref, len(flow.Blocks)), evidence: b.pool.Evidence(pdg.Approximated, semanticControlDescription), origin: b.pool.Origin(pdg.Analysis, semanticControlDescription), nextEdge: len(b.graph.Edges)}
	for _, index := range dom.order {
		a.operations[index] = b.normalOperations(flow.Blocks[index])
	}
	a.maximalControl(flow, dom, sets, outcomes)
	b.semanticControl = true
	return nil
}

func (a *normalFlowAnalysis) maximalControl(flow *cfg.CFG, dom *executionDominators, sets controlPostSets, outcomes map[int][]pdg.Position) {
	for _, from := range dom.order {
		nodes := a.operations[from]
		if len(nodes) == 0 {
			continue
		}
		for arm, next := range flow.Blocks[from].Succs {
			a.maximalControlArm(nodes[len(nodes)-1], from, int(next.Index), controlOutcome(outcomes, from, arm), dom, sets)
		}
	}
}

func controlOutcome(outcomes map[int][]pdg.Position, block, arm int) pdg.Position {
	if positions := outcomes[block]; len(positions) != 0 {
		return positions[arm]
	}
	return pdg.Position(arm + 1)
}

func (a *normalFlowAnalysis) maximalControlArm(controller pdg.Ref, from, next int, outcome pdg.Position, dom *executionDominators, sets controlPostSets) {
	for _, target := range dom.order {
		if !sets.contains(next, target) || (target != from && sets.contains(from, target)) {
			continue
		}
		for _, node := range a.operations[target] {
			a.edge(controller, node, pdg.ControlEdge, outcome)
		}
	}
}

func (b *irBuilder) semanticControlSubset(body *ast.BlockStmt) bool {
	supported := true
	ast.Inspect(body, func(node ast.Node) bool {
		supported = supported && semanticControlNode(node) && !b.recoveryCall(node)
		return supported
	})
	return supported
}

func semanticControlNode(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.FuncLit, *ast.DeferStmt, *ast.GoStmt, *ast.SelectStmt, *ast.SendStmt:
		return false
	case *ast.UnaryExpr:
		return node.Op != token.ARROW
	default:
		return true
	}
}

func (b *irBuilder) recoveryCall(node ast.Node) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	builtin, ok := b.callee(call.Fun).(*types.Builtin)
	return ok && builtin.Name() == "recover"
}
