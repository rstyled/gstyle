package pdg

import "fmt"

// ValidateNormalFlow checks retained CFG context, recomputing reachability and
// read counts. It does not establish source correspondence from producer claims.
func validateNormalFlow(g *Graph) error {
	flow := g.NormalFlow
	if flow == nil {
		return fmt.Errorf("normal-flow validation context is required")
	}
	if flow.Entry == 0 || int(flow.Entry) > len(flow.Blocks) {
		return fmt.Errorf("normal-flow entry is out of bounds")
	}
	if err := flowReferences(g); err != nil {
		return err
	}
	if err := validateFlowBlocks(g); err != nil {
		return err
	}
	return validateAcyclicFlow(flow)
}

func validateFlowBlocks(g *Graph) error {
	reachable := flowReachability(g.NormalFlow)
	seen, reads := map[Ref]bool{}, map[Ref]bool{}
	for i, block := range g.NormalFlow.Blocks {
		if block.Reachable != reachable[i] {
			return fmt.Errorf("normal-flow reachability disagrees at block %d", i+1)
		}
		if err := flowOperations(g, block, seen, reads); err != nil {
			return err
		}
	}
	return flowCounts(g, uint64(len(reads)))
}

func flowReferences(g *Graph) error {
	for _, block := range g.NormalFlow.Blocks {
		if err := flowRefs(block.Successors, len(g.NormalFlow.Blocks), "successor"); err != nil {
			return err
		}
		if err := flowRefs(block.Operations, len(g.Nodes), "operation"); err != nil {
			return err
		}
	}
	return nil
}

func flowRefs(refs []Ref, length int, kind string) error {
	for _, ref := range refs {
		if ref == 0 || int(ref) > length {
			return fmt.Errorf("normal-flow %s is out of bounds", kind)
		}
	}
	return nil
}

func flowReachability(flow *NormalFlow) []bool {
	reachable := make([]bool, len(flow.Blocks))
	queue := []Ref{flow.Entry}
	reachable[flow.Entry-1] = true
	for i := 0; i < len(queue); i++ {
		for _, next := range flow.Blocks[queue[i]-1].Successors {
			if reachable[next-1] {
				continue
			}
			reachable[next-1] = true
			queue = append(queue, next)
		}
	}
	return reachable
}

func flowOperations(g *Graph, block FlowBlock, seen, reads map[Ref]bool) error {
	for _, ref := range block.Operations {
		if seen[ref] {
			return fmt.Errorf("operation appears at multiple CFG sites")
		}
		seen[ref] = true
		node := g.Nodes[ref-1]
		if nonexecutingCategory(node.Category) {
			return fmt.Errorf("value entity appears as executable CFG operation")
		}
		if err := reachableFlowRead(g, node, block.Reachable, reads); err != nil {
			return err
		}
	}
	return nil
}

func nonexecutingCategory(category Category) bool {
	return category == FormalInput || category == FormalOutput || category == Merge
}

func flowReadBindings(g *Graph, node Node, reads map[Ref]bool) error {
	accesses := g.NodeAttributes(node).Reads
	if len(accesses) == 0 {
		return fmt.Errorf("normal-flow read has no binding")
	}
	for _, access := range accesses {
		if !resolvedScalarAccess(access, len(g.Symbols)) {
			return fmt.Errorf("normal-flow read binding is unresolved")
		}
		role := g.Symbols[access.Symbol-1].Role
		if role == LocalRole || role == ParameterRole {
			reads[access.Symbol] = true
		}
	}
	return nil
}

func flowCounts(g *Graph, count uint64) error {
	if g.NormalFlow.ReadBindings != count {
		return fmt.Errorf("normal-flow reference count disagrees with reachable reads")
	}
	if g.Function.ReferenceCount.Status != Known || g.Function.ReferenceCount.Value != count {
		return fmt.Errorf("PDG reference count disagrees with normal flow")
	}
	return nil
}

func resolvedScalarAccess(access Access, symbols int) bool {
	return access.Symbol != 0 && int(access.Symbol) <= symbols && access.Location == 0 && access.UnresolvedReason == 0
}

func reachableFlowRead(g *Graph, node Node, reachable bool, reads map[Ref]bool) error {
	if !reachable || node.Category != Read {
		return nil
	}
	return flowReadBindings(g, node, reads)
}
