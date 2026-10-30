package cpp

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) operationFlow() error {
	flow, used := p.cfgBlocks()
	if !used[flow.Entry-1] {
		return nil
	}
	p.flowOperations(flow, used)
	flow.RefreshReachability()
	if err := pdg.ValidateFlow(p.graph, flow); err != nil {
		return err
	}
	p.graph.ValueFlow = flow
	p.extension("gstyle.cpp:operation-flow", "1")
	return nil
}

func (p *projection) cfgBlocks() (*pdg.NormalFlow, []bool) {
	flow := &pdg.NormalFlow{Entry: p.refs[p.method], Blocks: make([]pdg.FlowBlock, len(p.graph.Nodes))}
	used := make([]bool, len(p.graph.Nodes))
	for _, link := range p.doc.MethodEdges[p.method] {
		if link.Label != "CFG" || externalDependency(link, p.method) {
			continue
		}
		source, target := p.refs[link.Source], p.refs[link.Target]
		used[source-1], used[target-1] = true, true
		block := &flow.Blocks[source-1]
		if !slices.Contains(block.Successors, target) {
			block.Successors = append(block.Successors, target)
		}
	}
	return flow, used
}

func (p *projection) flowOperations(flow *pdg.NormalFlow, used []bool) {
	for i, node := range p.graph.Nodes {
		if !used[i] || nonexecuting(node.Category) {
			continue
		}
		flow.Blocks[i].Operations = []pdg.Ref{pdg.Ref(i + 1)}
	}
}

func nonexecuting(category pdg.Category) bool {
	return category == pdg.FormalInput || category == pdg.FormalOutput || category == pdg.Merge || category == pdg.MemorySummary
}
