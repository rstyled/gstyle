package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

// Only bindings that feed an access or nil predicate need flow state. Follow
// their definitions with a worklist; unrelated locals never enter block maps.
func (v *graphView) trackBindings() {
	definitions := make([][]pdg.Ref, len(v.kinds))
	var queue []pdg.Ref
	visited := make([]bool, len(v.graph.Nodes))
	for i, node := range v.graph.Nodes {
		ref := pdg.Ref(i + 1)
		v.trackRoots(ref, visited, &queue)
		v.trackCall(ref, visited, &queue)
		v.predicateDefinitions(ref, definitions)
		for _, write := range v.graph.NodeAttributes(node).Writes {
			if write.Symbol != 0 {
				definitions[write.Symbol-1] = append(definitions[write.Symbol-1], ref)
			}
		}
	}
	for cursor := 0; cursor < len(queue); cursor++ {
		for _, ref := range definitions[queue[cursor]-1] {
			v.trackDefinitionValues(ref, visited, &queue)
		}
	}
}

func (v *graphView) trackRoots(ref pdg.Ref, visited []bool, queue *[]pdg.Ref) {
	if v.fact(ref, "index") == "true" || v.fact(ref, "dereference") == "true" {
		v.trackValue(v.operand(ref, 0), visited, queue)
		v.trackValue(v.operand(ref, 1), visited, queue)
	}
	if v.fact(v.operand(ref, 0), "nil") == "true" {
		v.track(v.symbol(v.operand(ref, 1)), queue)
	}
	if v.fact(v.operand(ref, 1), "nil") == "true" {
		v.track(v.symbol(v.operand(ref, 0)), queue)
	}
}

func (v *graphView) track(symbol pdg.Ref, queue *[]pdg.Ref) {
	if symbol == 0 || v.tracked[symbol-1] {
		return
	}
	v.tracked[symbol-1] = true
	*queue = append(*queue, symbol)
}

func (v *graphView) trackValue(ref pdg.Ref, visited []bool, queue *[]pdg.Ref) {
	if ref == 0 || visited[ref-1] {
		return
	}
	visited[ref-1] = true
	v.track(v.symbol(ref), queue)
	for _, operand := range v.operands[ref-1] {
		v.trackValue(operand, visited, queue)
	}
}

func (v *graphView) trackedBinding(symbol pdg.Ref) bool {
	return symbol != 0 && int(symbol) <= len(v.tracked) && v.tracked[symbol-1]
}

func (v *graphView) predicateDefinitions(ref pdg.Ref, definitions [][]pdg.Ref) {
	operator := operatorText(v.graph.Tables, v.attrs(ref).Operator)
	if _, comparison := inverse[operator]; !comparison {
		return
	}
	for _, operand := range v.operands[ref-1] {
		if symbol := v.symbol(operand); symbol != 0 {
			definitions[symbol-1] = append(definitions[symbol-1], ref)
		}
	}
}

func (v *graphView) trackDefinitionValues(ref pdg.Ref, visited []bool, queue *[]pdg.Ref) {
	for _, operand := range v.operands[ref-1] {
		v.trackValue(operand, visited, queue)
	}
}

func (v *graphView) trackCall(ref pdg.Ref, visited []bool, queue *[]pdg.Ref) {
	if v.graph.Nodes[ref-1].Category == pdg.Call {
		v.trackValue(v.operand(ref, 0), visited, queue)
	}
}
