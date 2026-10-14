package pdg

// NormalFlow is validation context beside the closed PDG interchange document.
// It owns indexes only; it retains no AST, compiler object, source bytes or
// dominance matrix. Block references and PDG node references are one-based.
type NormalFlow struct {
	Blocks       []FlowBlock
	Entry        Ref
	ReadBindings uint64
}

type FlowBlock struct {
	Operations []Ref
	Successors []Ref
	Condition  Ref // Optional boolean operand; successor zero is true, one is false.
	Reachable  bool
}

// RefreshReachability publishes reachability after a producer expands flow.
// References must already be valid; this does not validate source semantics.
func (flow *NormalFlow) RefreshReachability() {
	reachable := flow.flowReachability()
	for i := range flow.Blocks {
		flow.Blocks[i].Reachable = reachable[i]
	}
}
