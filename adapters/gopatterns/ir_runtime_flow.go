package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"golang.org/x/tools/go/cfg"
	"strconv"
)

const runtimeFlowDescription = "Typed binding operands and normal CFG outcomes; local non-escaped values, assignments and joins. Memory aliases, call results and concurrency remain unknown. Local summaries remove proven nonreturning continuations. Deferred inputs evaluate at registration; invocation and unwind remain outside this flow."

func (b *irBuilder) runtimeFlow(body *ast.BlockStmt, flow *cfg.CFG) error {
	if !b.runtimeOperations {
		return b.bindingValueFlow(body, flow)
	}
	for _, block := range flow.Blocks {
		for _, source := range block.Nodes {
			b.runtimeOperationsFor(source)
		}
	}
	b.runtimeControlEdges(body)
	flow, err := b.runtimeFlowCFG(body, flow)
	if err != nil {
		return err
	}
	context := &pdg.NormalFlow{Entry: 1, Blocks: make([]pdg.FlowBlock, len(flow.Blocks))}
	for i, block := range flow.Blocks {
		context.Blocks[i] = b.runtimeBlock(block)
	}
	b.publishRuntimeFlow(body, context)
	return nil
}

func (b *irBuilder) publishRuntimeFlow(body *ast.BlockStmt, context *pdg.NormalFlow) {
	context.RefreshReachability()
	b.graph.ValueFlow = context
	b.graph.Extensions = append(b.graph.Extensions, b.runtimeFlowExtensions(body)...)
}
func (b *irBuilder) runtimeFlowExtensions(body *ast.BlockStmt) []pdg.Extension {
	exitFamily := ""
	if b.exitOperationSubset(body) {
		exitFamily = "gstyle.pdg:exit-body-flow"
	}
	families := [3]string{b.bodyFlowFamily(body), exitFamily, "gstyle.runtime:flow"}
	var extensions []pdg.Extension
	for _, family := range families {
		if family == "" {
			continue
		}
		version := `"0.1.0"`
		if family == "gstyle.runtime:flow" {
			version = `"0.2.0"`
		}
		extensions = append(extensions, b.pool.SingletonExtension(family, version)...)
	}
	return extensions
}
func (b *irBuilder) bodyFlowFamily(body *ast.BlockStmt) string {
	if b.semanticControlSubset(body) {
		return "gstyle.pdg:operation-flow"
	}
	if b.registrationFlowSubset(body) {
		return "gstyle.pdg:registration-flow"
	}
	return ""
}

func (b *irBuilder) runtimeFlowCFG(body *ast.BlockStmt, flow *cfg.CFG) (*cfg.CFG, error) {
	if b.exitOperationSubset(body) {
		return b.semanticControlCFG(body, flow)
	}
	b.runtimeLoopWrites(flow)
	return b.scalarCFG(body, flow), nil
}

func (b *irBuilder) runtimeBlock(block *cfg.Block) pdg.FlowBlock {
	out := pdg.FlowBlock{Reachable: block.Live, Condition: b.runtimeCondition(block)}
	if block.Kind == cfg.KindRangeLoop {
		out.Operations = append(out.Operations, b.nodes[block.Stmt])
	}
	for _, source := range block.Nodes {
		out.Operations = append(out.Operations, b.runtimeOperationsFor(source)...)
	}
	for _, successor := range block.Succs {
		out.Successors = append(out.Successors, pdg.Ref(successor.Index+1))
	}
	return out
}

// Synthetic operation-exit alternatives are not Boolean source conditions.
func (b *irBuilder) runtimeCondition(block *cfg.Block) pdg.Ref {
	if len(block.Succs) != 2 || len(block.Nodes) == 0 {
		return 0
	}
	ref := b.nodes[block.Nodes[len(block.Nodes)-1]]
	if b.operationExits[ref] != 0 {
		return 0
	}
	return ref
}

func (b *irBuilder) runtimeOperationsFor(source ast.Node) []pdg.Ref {
	var out []pdg.Ref
	var stack []ast.Node
	ast.Inspect(source, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			ref := b.nodes[node]
			out = append(out, ref)
			b.runtimeParentEdge(ref, stack, node)
			return false
		}
		if node != nil {
			for _, read := range b.implicitReads[node] {
				attrs := b.attributes[b.graph.Nodes[read-1].Attributes-1]
				b.runtimeValueEdge(read, b.nodes[node], attrs.Reads[0].Position)
				out = append(out, read)
			}
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
			position := b.runtimeOperandPosition(stack[i], child)
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

func (b *irBuilder) runtimeOperandPosition(parent, child ast.Node) pdg.Position {
	switch parent := parent.(type) {
	case *ast.StarExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	case *ast.SelectorExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	case *ast.IndexExpr:
		return operandIndex([]ast.Expr{parent.X, parent.Index}, child)
	case *ast.CallExpr:
		return b.callOperandPosition(parent, child)
	default:
		return sourceOperandPosition(parent, child)
	}
}

func (b *irBuilder) callOperandPosition(call *ast.CallExpr, child ast.Node) pdg.Position {
	position := operandIndex(call.Args, child)
	if !b.implicitReceiver(call.Fun) {
		return position
	}
	if child == call.Fun {
		return 1
	}
	if position != 0 {
		return position + 1
	}
	return 0
}

// Binding-only functions retain canonical sites without adding a second set of
// value operands. Their binding-value edges remain the typed interface facts.
func (b *irBuilder) bindingValueFlow(body *ast.BlockStmt, flow *cfg.CFG) error {
	if !b.bindingOperations && !b.exitOperationSubset(body) {
		return nil
	}
	shared, err := b.semanticControlCFG(body, flow)
	if err != nil {
		return err
	}
	context := &pdg.NormalFlow{Entry: 1, Blocks: make([]pdg.FlowBlock, len(shared.Blocks))}
	for index, block := range shared.Blocks {
		context.Blocks[index] = b.runtimeBlock(block)
	}
	b.publishRuntimeFlow(body, context)
	return nil
}
