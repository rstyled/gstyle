package pdg

import "fmt"

// ValidateFlow checks structural references, unique operation sites and claimed
// reachability. It accepts cycles and does not grant scalar profile conformance.
func ValidateFlow(g *Graph, flow *NormalFlow) error {
	if g == nil || flow == nil {
		return fmt.Errorf("operation-flow context is required")
	}
	if err := flowRefs([]Ref{flow.Entry}, len(flow.Blocks), "entry"); err != nil {
		return err
	}
	for _, block := range flow.Blocks {
		if err := operationBlockReferences(g, flow, block); err != nil {
			return err
		}
	}
	return validateFlowSites(flow)
}
func validateFlowSites(flow *NormalFlow) error {
	reachable := flow.flowReachability()
	seen := make(map[Ref]bool)
	for i, block := range flow.Blocks {
		if block.Reachable != reachable[i] {
			return fmt.Errorf("operation-flow reachability disagrees at block %d", i+1)
		}
		for _, ref := range block.Operations {
			if seen[ref] {
				return fmt.Errorf("operation appears at multiple operation-flow sites")
			}
			seen[ref] = true
		}
	}
	return nil
}

// The shared executable-flow profile also excludes value-only entities.
func validateOperationFlow(g *Graph) error {
	for _, flow := range []*NormalFlow{BodyFlow(g), ExitBodyFlow(g)} {
		if err := validateExecutableFlow(g, flow); err != nil {
			return err
		}
	}
	return nil
}

func validateExecutableFlow(g *Graph, flow *NormalFlow) error {
	if flow == nil {
		return nil
	}
	if err := ValidateFlow(g, flow); err != nil {
		return err
	}
	for _, block := range flow.Blocks {
		for _, ref := range block.Operations {
			if projectionValue(g.Nodes[ref-1].Category) {
				return fmt.Errorf("value entity appears as executable operation-flow operation")
			}
		}
	}
	return nil
}
func operationBlockReferences(g *Graph, flow *NormalFlow, block FlowBlock) error {
	if int(block.Condition) > len(g.Nodes) {
		return fmt.Errorf("operation-flow condition is out of bounds")
	}
	if err := flowRefs(block.Operations, len(g.Nodes), "operation"); err != nil {
		return err
	}
	return flowRefs(block.Successors, len(flow.Blocks), "successor")
}
