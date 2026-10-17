package outputflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type operationSite struct {
	block, offset int
	present       bool
}

// SuccessFlow borrows the canonical graph. It proves only normal completion:
// unknown effects and potentially panicking operations cannot establish success.
// It does not assert that the initial output call itself completes.
type SuccessFlow struct {
	graph  *pdg.Graph
	inputs *bindingflow.Query
	slot   pdg.Position
	sites  []operationSite
	good   []bool
}

func NewSuccessFlow(graph *pdg.Graph, slot pdg.Position) (*SuccessFlow, error) {
	if graph == nil {
		return nil, nil
	}
	inputs, err := bindingflow.New(graph)
	if err != nil {
		return nil, err
	}
	if !hasCompletionFlow(graph) {
		return nil, nil
	}
	if err := completionSlot(graph, slot); err != nil {
		return nil, err
	}
	proof := &SuccessFlow{graph: graph, inputs: inputs, slot: slot, sites: make([]operationSite, len(graph.Nodes)), good: make([]bool, len(graph.ValueFlow.Blocks))}
	if err := proof.index(); err != nil {
		return nil, err
	}
	proof.solve()
	return proof, nil
}

// After accepts a call only when its complete normal continuation is proved.
// A least fixed point excludes infinite paths even when they also have exits.
func (p *SuccessFlow) After(call pdg.Ref) bool {
	if p == nil {
		return false
	}
	site, found := p.callSite(call)
	return found && p.suffix(site.block, site.offset+1)
}

func (p *SuccessFlow) index() error {
	if err := pdg.ValidateFlow(p.graph, p.graph.ValueFlow); err != nil {
		return err
	}
	p.indexSites()
	return nil
}

func (p *SuccessFlow) indexSites() {
	for block, flow := range p.graph.ValueFlow.Blocks {
		for offset, ref := range flow.Operations {
			p.sites[ref-1] = operationSite{block: block, offset: offset, present: true}
		}
	}
}

func (p *SuccessFlow) solve() {
	predecessors := completionPredecessors(p.graph.ValueFlow)
	queue := make([]int, len(p.good))
	for i := range queue {
		queue[i] = i
	}
	for cursor := 0; cursor < len(queue); cursor++ {
		block := queue[cursor]
		if p.good[block] || !p.suffix(block, 0) {
			continue
		}
		p.good[block] = true
		queue = append(queue, predecessors[block]...)
	}
}

func (p *SuccessFlow) callSite(call pdg.Ref) (operationSite, bool) {
	if call == 0 || int(call) > len(p.sites) {
		return operationSite{}, false
	}
	site := p.sites[call-1]
	if !site.present || p.graph.Nodes[call-1].Category != pdg.Call {
		return operationSite{}, false
	}
	return site, p.graph.ValueFlow.Blocks[site.block].Reachable
}

func (p *SuccessFlow) suffix(index, offset int) bool {
	block := p.graph.ValueFlow.Blocks[index]
	good := p.successors(block.Successors)
	for i := len(block.Operations) - 1; i >= offset; i-- {
		ref := block.Operations[i]
		if p.graph.Nodes[ref-1].Category == pdg.Return {
			good = p.nilResult(ref)
			continue
		}
		if !p.inert(ref) {
			good = false
		}
	}
	return good
}

func (p *SuccessFlow) successors(next []pdg.Ref) bool {
	if len(next) == 0 {
		return false
	}
	for _, ref := range next {
		if !p.good[ref-1] {
			return false
		}
	}
	return true
}

func (p *SuccessFlow) nilResult(ref pdg.Ref) bool {
	operand := p.inputs.Operand(ref, p.slot)
	for range len(p.graph.Nodes) {
		if operand == 0 {
			return false
		}
		attrs := p.graph.NodeAttributes(p.graph.Nodes[operand-1])
		if runtimeFact(p.graph, attrs, "nil") == "true" {
			return true
		}
		if attrs.Operator.Status != pdg.Known || p.graph.Tables.Text(attrs.Operator.Value) != "parentheses" {
			return false
		}
		operand = p.inputs.Operand(operand, 1)
	}
	return false
}

func hasCompletionFlow(graph *pdg.Graph) bool {
	return graph.ValueFlow != nil && pdg.ReachableProjection(graph) != nil
}

func completionSlot(graph *pdg.Graph, slot pdg.Position) error {
	if slot == 0 || int(slot) > len(graph.Function.Outputs) {
		return fmt.Errorf("error output slot is out of bounds")
	}
	return nil
}

// completionPredecessors indexes the canonical successor relation for the
// backward fixed point; it does not infer additional control paths.
func completionPredecessors(flow *pdg.NormalFlow) [][]int {
	predecessors := make([][]int, len(flow.Blocks))
	for i, block := range flow.Blocks {
		for _, next := range block.Successors {
			predecessors[next-1] = append(predecessors[next-1], i)
		}
	}
	return predecessors
}

func runtimeFact(graph *pdg.Graph, attrs pdg.Attributes, name string) string {
	for _, extension := range attrs.Extensions {
		if graph.Tables.Text(extension.Name) == "gstyle.runtime:"+name {
			return graph.Tables.Text(extension.Value)
		}
	}
	return ""
}
