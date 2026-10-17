package pdg

// NormalProjection retains reachable operations, actual inputs, and the value
// entities required by those operations. It trusts validated flow context.
// Nil means this producer supplies no normal-flow projection.
func NormalProjection(g *Graph) []bool {
	if g.NormalFlow == nil {
		return nil
	}
	return operationProjection(g, g.NormalFlow)
}

// ReachableProjection also accepts the producer's shared sequential operation
// flow. Legacy partial value flow does not justify removing source operations.
// This projection does not grant strict scalar conformance.
func ReachableProjection(g *Graph) []bool {
	if g.NormalFlow != nil {
		return NormalProjection(g)
	}
	flow := sharedProjectionFlow(g)
	if flow == nil {
		return nil
	}
	return operationProjection(g, flow)
}

func sharedProjectionFlow(g *Graph) *NormalFlow {
	if !sharedOperationFlow(g) {
		return nil
	}
	return g.ValueFlow
}

func sharedOperationFlow(g *Graph) bool {
	for _, extension := range g.Extensions {
		if g.Tables.Text(extension.Name) == "gstyle.pdg:operation-flow" && g.Tables.Text(extension.Value) == `"0.1.0"` {
			return true
		}
	}
	return false
}

func operationProjection(g *Graph, flow *NormalFlow) []bool {
	kept := make([]bool, len(g.Nodes))
	for i, node := range g.Nodes {
		kept[i] = node.Category == FormalInput
	}
	for _, block := range flow.Blocks {
		if !block.Reachable {
			continue
		}
		for _, node := range block.Operations {
			kept[node-1] = true
		}
	}

	retainNormalValues(g, kept)
	return kept
}

// A queue propagates value requirements once per entity, using linear temporary
// storage. Deeply nested joins never require repeated whole-graph scans.
func retainNormalValues(g *Graph, kept []bool) {
	required := normalValueRequirements(g)
	var queue []Ref
	for i, keep := range kept {
		if keep {
			queue = append(queue, Ref(i+1))
		}
	}
	for cursor := 0; cursor < len(queue); cursor++ {
		for _, node := range required[queue[cursor]-1] {
			if kept[node-1] {
				continue
			}
			kept[node-1] = true
			queue = append(queue, node)
		}
	}
}

func normalValueRequirements(g *Graph) [][]Ref {
	required := make([][]Ref, len(g.Nodes))
	for _, edge := range g.Edges {
		if edge.Kind != Data {
			continue
		}
		if projectionValue(g.Nodes[edge.Source-1].Category) {
			required[edge.Target-1] = append(required[edge.Target-1], edge.Source)
		}
		if g.Nodes[edge.Target-1].Category == FormalOutput {
			required[edge.Source-1] = append(required[edge.Source-1], edge.Target)
		}
	}
	return required
}

func projectionValue(category Category) bool {
	return nonexecutingCategory(category) || category == MemorySummary
}
