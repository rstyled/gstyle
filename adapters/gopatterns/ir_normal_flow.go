package gopatterns

import (
	"slices"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

const normalExecutionDescription = "Nearest dominators under deterministic source-order scalar lowering; Go leaves some operand evaluation orders unspecified. These edges are not a complete language-mandated ordering proof."

const normalDependencyDescription = "Nearest-operation dominance and post-dominator control dependence for reachable acyclic local scalar normal flow; exceptional flow is excluded."

type normalFlowAnalysis struct {
	builder                             *irBuilder
	operations                          [][]pdg.Ref
	post                                *executionDominators
	evidence, executionEvidence, origin pdg.Ref
	nextEdge                            int
}

func (a *normalFlowAnalysis) execution(dom *executionDominators) {
	for _, index := range dom.order {
		previous := dom.previousAnchor(index, a.operations)
		for _, node := range a.operations[index] {
			a.edge(previous, node, pdg.Execution)
			previous = node
		}
	}
}

func (a *normalFlowAnalysis) control(flow *cfg.CFG, dom *executionDominators) {
	for _, index := range dom.order {
		nodes := a.operations[index]
		if len(nodes) == 0 {
			continue
		}
		controller := nodes[len(nodes)-1]
		for _, next := range flow.Blocks[index].Succs {
			a.controlPath(controller, int(next.Index)+1, a.post.idom[index+1])
		}
	}
}

func (a *normalFlowAnalysis) controlPath(controller pdg.Ref, start, stop int) {
	for runner := start; runner != stop && runner != 0; runner = a.post.idom[runner] {
		for _, node := range a.operations[runner-1] {
			a.edge(controller, node, pdg.ControlEdge)
		}
	}
}

func (a *normalFlowAnalysis) edge(source, target pdg.Ref, kind pdg.EdgeKind) {
	if source == 0 || source == target {
		return
	}
	a.nextEdge++
	subkind := "chosen-order-dominator"
	evidence := a.executionEvidence
	if kind == pdg.ControlEdge {
		subkind = "post-dominator-control"
		evidence = a.evidence
	}
	b := a.builder
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("normal-e" + strconv.Itoa(a.nextEdge)), Source: source, Target: target, Kind: kind, Subkind: b.pool.Text(subkind), Evidence: evidence, Provenance: a.origin})
}

func (b *irBuilder) normalDependenciesFor(flow *cfg.CFG, dom *executionDominators) {
	if !b.scalarDefinitions {
		return
	}
	post := normalPostDominators(flow, dom)
	a := normalFlowAnalysis{builder: b, operations: make([][]pdg.Ref, len(flow.Blocks)), post: post,
		evidence: b.pool.Evidence(pdg.Supported, normalDependencyDescription), origin: b.pool.Origin(pdg.Analysis, normalDependencyDescription), executionEvidence: b.pool.Evidence(pdg.Approximated, normalExecutionDescription), nextEdge: len(b.graph.Edges)}
	for index, block := range flow.Blocks {
		a.operations[index] = b.normalOperations(block)
	}
	b.graph.Edges = slices.DeleteFunc(b.graph.Edges, func(edge pdg.Edge) bool { return edge.Kind == pdg.Execution || edge.Kind == pdg.ControlEdge })
	a.execution(dom)
	a.control(flow, dom)
	b.normalDependencies = true
	b.normalFlowContext(flow, a.operations, dom)
}

// Scalar CFG lowering has already ordered each block's semantic operations.
func (b *irBuilder) normalOperations(block *cfg.Block) []pdg.Ref {
	operations := make([]pdg.Ref, 0, len(block.Nodes))
	for _, source := range block.Nodes {
		if node := b.nodes[source]; node != 0 {
			operations = append(operations, node)
		}
	}
	return operations
}

// Reverse the reachable CFG, adding one analysis-only exit for all normal exits.
// Reusing the dominator solver keeps storage linear; no transitive closure is kept.
func normalPostDominators(flow *cfg.CFG, dom *executionDominators) *executionDominators {
	reverse := &cfg.CFG{Blocks: make([]*cfg.Block, len(flow.Blocks)+1)}
	for i := range reverse.Blocks {
		reverse.Blocks[i] = &cfg.Block{Index: int32(i)}
	}
	for _, index := range dom.order {
		reverseNormalBlock(reverse, flow.Blocks[index])
	}
	return newExecutionDominators(reverse)
}

func reverseNormalBlock(reverse *cfg.CFG, block *cfg.Block) {
	current := reverse.Blocks[block.Index+1]
	if len(block.Succs) == 0 {
		reverse.Blocks[0].Succs = append(reverse.Blocks[0].Succs, current)
		return
	}
	for _, next := range block.Succs {
		reverse.Blocks[next.Index+1].Succs = append(reverse.Blocks[next.Index+1].Succs, current)
	}
}
