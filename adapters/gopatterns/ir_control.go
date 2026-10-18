package gopatterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

const controlVertexLimit = 16384

const semanticControlDescription = "Operation-CFG dominance and divergence-sensitive post-dominator control dependence, including finite and infinite paths. Package-local summaries and ordinary nil-input facts remove proven nonreturning continuations; unresolved calls may return; built-in panic, os.Exit, runtime.Goexit and resolved log Fatal/Panic variants terminate paths. Other implicit panics, recovery, deferred execution and concurrent control are not modeled."

func (b *irBuilder) semanticControlFor(body *ast.BlockStmt, flow *cfg.CFG) error {
	if b.normalDependencies || !b.semanticControlSubset(body) {
		return nil
	}
	flow, err := b.semanticControlCFG(body, flow)
	if err != nil {
		return err
	}
	dom := newExecutionDominators(flow)
	sets := maximalPostDominators(flow, dom)
	a := normalFlowAnalysis{builder: b, operations: make([][]pdg.Ref, len(flow.Blocks)), evidence: b.pool.Evidence(pdg.Approximated, semanticControlDescription), origin: b.pool.Origin(pdg.Analysis, semanticControlDescription), nextEdge: len(b.graph.Edges)}
	for _, index := range dom.order {
		a.operations[index] = b.normalOperations(flow.Blocks[index])
	}
	// Execution and control must use the same operation sites and transfers.
	// Statement anchors can place range targets before their iteration decision.
	b.graph.Edges = slices.DeleteFunc(b.graph.Edges, func(edge pdg.Edge) bool { return edge.Kind == pdg.Execution })
	a.executionEvidence = b.pool.Evidence(pdg.Approximated, generalExecutionDescription)
	a.execution(dom)
	a.maximalControl(flow, dom, sets, b.controlOutcomes)
	b.semanticControl = true
	return nil
}

func (b *irBuilder) semanticControlSubset(body *ast.BlockStmt) bool {
	return b.bodyOperationSubset(body, false)
}

func (b *irBuilder) registrationFlowSubset(body *ast.BlockStmt) bool {
	return b.bodyOperationSubset(body, true)
}

func (b *irBuilder) bodyOperationSubset(body *ast.BlockStmt, allowDefer bool) bool {
	supported := true
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		supported = supported && bodyOperationNode(node, allowDefer) && !b.recoveryCall(node)
		return supported
	})
	return supported
}

func (b *irBuilder) recoveryCall(node ast.Node) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	builtin, ok := b.callee(call.Fun).(*types.Builtin)
	return ok && builtin.Name() == "recover"
}

// Value and control analysis share the same lowered CFG when both are enabled.
// This preserves operation identity for synthetic switch and short-circuit tests.
func (b *irBuilder) semanticControlCFG(body *ast.BlockStmt, original *cfg.CFG) (*cfg.CFG, error) {
	if b.controlFlow != nil {
		return b.controlFlow, nil
	}
	flow, outcomes := b.controlCFG(body, original)
	b.applyOperationContinuations(flow)
	if len(flow.Blocks) > controlVertexLimit {
		return nil, fmt.Errorf("control dependence exceeds %d operation CFG vertices", controlVertexLimit)
	}
	b.controlFlow, b.controlOutcomes = flow, outcomes
	return flow, nil
}

// exitOperationSubset admits recovery only for the separate exit-body context.
// Ordinary control consumers must continue to reject recovery semantics.
func (b *irBuilder) exitOperationSubset(body *ast.BlockStmt) bool {
	supported := true
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		supported = supported && bodyOperationNode(node, true)
		return supported
	})
	return supported
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

func controlOutcome(outcomes map[int][]pdg.Position, block, arm int) pdg.Position {
	if positions := outcomes[block]; len(positions) != 0 {
		return positions[arm]
	}
	return pdg.Position(arm + 1)
}

func bodyOperationNode(node ast.Node, allowDefer bool) bool {
	switch node := node.(type) {
	case *ast.DeferStmt:
		return allowDefer
	case *ast.GoStmt, *ast.SelectStmt, *ast.SendStmt:
		return false
	case *ast.UnaryExpr:
		return node.Op != token.ARROW
	default:
		return true
	}
}
