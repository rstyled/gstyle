package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"golang.org/x/tools/go/cfg"
	"strconv"
)

const runtimeFlowDescription = "Typed binding operands and normal CFG outcomes; local non-escaped values, assignments and joins. Memory aliases, call results and concurrency remain unknown."

func (b *irBuilder) runtimeFlow(fn *ast.FuncDecl, flow *cfg.CFG) {
	if !b.runtimeOperations {
		return
	}
	for _, block := range flow.Blocks {
		for _, source := range block.Nodes {
			b.runtimeOperationsFor(source)
		}
	}
	b.runtimeControlEdges(fn.Body)
	b.runtimeLoopWrites(flow)
	flow = b.scalarCFG(fn.Body, flow)
	context := &pdg.NormalFlow{Entry: 1, Blocks: make([]pdg.FlowBlock, len(flow.Blocks))}
	for i, block := range flow.Blocks {
		context.Blocks[i] = b.runtimeBlock(block)
	}
	context.RefreshReachability()
	b.graph.ValueFlow = context
	b.graph.Extensions = append(b.graph.Extensions, b.pool.SingletonExtension("gstyle.runtime:flow", `"0.1.0"`)...)
}

func (b *irBuilder) runtimeBlock(block *cfg.Block) pdg.FlowBlock {
	out := pdg.FlowBlock{Reachable: block.Live}
	if block.Kind == cfg.KindRangeLoop {
		out.Operations = append(out.Operations, b.nodes[block.Stmt])
	}
	for _, source := range block.Nodes {
		out.Operations = append(out.Operations, b.runtimeOperationsFor(source)...)
	}
	for _, successor := range block.Succs {
		out.Successors = append(out.Successors, pdg.Ref(successor.Index+1))
	}
	if len(out.Successors) == 2 && len(block.Nodes) != 0 {
		source := block.Nodes[len(block.Nodes)-1]
		out.Condition = b.runtimeCondition(b.nodes[source])
	}
	return out
}

func (b *irBuilder) runtimeOperationsFor(source ast.Node) []pdg.Ref {
	var out []pdg.Ref
	var stack []ast.Node
	ast.Inspect(source, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if node != nil {
			stack = append(stack, node)
			return true
		}
		last := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		ref := b.nodes[last]
		if ref != 0 {
			out = append(out, ref)
			b.runtimeParentEdge(ref, stack, last)
		}
		return true
	})
	return out
}

func (b *irBuilder) runtimeParentEdge(source pdg.Ref, stack []ast.Node, child ast.Node) {
	for i := len(stack) - 1; i >= 0; i-- {
		if target := b.nodes[stack[i]]; target != 0 {
			position := runtimeOperandPosition(stack[i], child)
			b.runtimeValueEdge(source, target, position)
			return
		}
		child = stack[i]
	}
}

func (b *irBuilder) runtimeValueEdge(source, target pdg.Ref, position pdg.Position) {
	if position == 0 || source == 0 || target == 0 || source == target {
		return
	}
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{
		ID: b.pool.Text("runtime-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: target,
		Kind: pdg.Data, Position: position, Subkind: b.pool.Text("source-value"),
		Evidence: b.pool.Evidence(pdg.Supported, runtimeFlowDescription), Provenance: b.pool.Origin(pdg.SourceOrigin, runtimeFlowDescription),
	})
}

func (b *irBuilder) runtimeCondition(ref pdg.Ref) pdg.Ref {
	if ref == 0 {
		return 0
	}
	node := b.graph.Nodes[ref-1]
	if node.Category != pdg.Control {
		return ref
	}
	for i := len(b.graph.Edges) - 1; i >= 0; i-- {
		edge := b.graph.Edges[i]
		if runtimeConditionEdge(edge, ref) {
			return edge.Source
		}
	}
	return 0
}

func (b *irBuilder) runtimeControlEdges(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if expr := executionCondition(node); expr != nil {
			b.runtimeValueEdge(b.nodes[expr], b.nodes[node], 1)
		}
		return true
	})
}

func (b *irBuilder) runtimeLoopWrites(flow *cfg.CFG) {
	for _, block := range flow.Blocks {
		if block.Kind != cfg.KindRangeLoop {
			continue
		}
		if ref := b.nodes[block.Stmt]; ref != 0 {
			marker := &ast.EmptyStmt{}
			b.nodes[marker] = ref
			block.Nodes = append([]ast.Node{marker}, block.Nodes...)
		}
	}
}

func runtimeOperandPosition(parent, child ast.Node) pdg.Position {
	switch parent := parent.(type) {
	case *ast.StarExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	case *ast.SelectorExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	case *ast.IndexExpr:
		return operandIndex([]ast.Expr{parent.X, parent.Index}, child)
	case *ast.CallExpr:
		return operandIndex(parent.Args, child)
	default:
		return sourceOperandPosition(parent, child)
	}
}

func runtimeConditionEdge(edge pdg.Edge, target pdg.Ref) bool {
	return edge.Target == target && edge.Kind == pdg.Data && edge.Position == 1
}
