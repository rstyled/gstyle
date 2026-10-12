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
	Reachable  bool
}
