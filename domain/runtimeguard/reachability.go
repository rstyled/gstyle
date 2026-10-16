package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

// ReachableNodes shares Analyze's branch constraints and normal-flow model.
// Membership is model reachability, not an interprocedural feasibility proof.
func ReachableNodes(graph *pdg.Graph) ([]pdg.Ref, error) {
	a, err := solve(graph)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, nil
	}
	var out []pdg.Ref
	for i, block := range graph.ValueFlow.Blocks {
		if a.inputs[i] != nil {
			out = append(out, block.Operations...)
		}
	}
	return out, nil
}
