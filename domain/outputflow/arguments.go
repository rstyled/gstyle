package outputflow

import "github.com/rstyled/gstyle/domain/pdg"

// Output proves the existing formatted-output contract from canonical call
// identity, argument evaluation, writer nilness and normal continuations.
// The writer implementation itself remains the output protocol boundary.
func (p *SuccessFlow) Output(call pdg.Ref, writerNil bool) bool {
	if !p.After(call) || p.inputs == nil || !p.formatted(call) {
		return false
	}
	return !p.nilWriter(call, writerNil) && p.arguments(call, map[pdg.Ref]uint8{})
}

func (p *SuccessFlow) formatted(call pdg.Ref) bool {
	attrs := p.graph.NodeAttributes(p.graph.Nodes[call-1])
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return false
	}
	switch p.graph.Tables.Text(attrs.Callee.Value) {
	case "fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln":
		return true
	default:
		return false
	}
}

func (p *SuccessFlow) nilWriter(call pdg.Ref, knownNil bool) bool {
	writer := p.inputs.Operand(call, 1)
	if writer == 0 {
		return true
	}
	attrs := p.graph.NodeAttributes(p.graph.Nodes[writer-1])
	if runtimeFact(p.graph, attrs, "nil") == "true" {
		return true
	}
	if !knownNil {
		return false
	}
	return !concreteNilWriter(memoryKind(p.graph, attrs))
}

func (p *SuccessFlow) arguments(owner pdg.Ref, seen map[pdg.Ref]uint8) bool {
	values, known := p.inputs.InputValues(owner)
	if !known {
		return false
	}
	for _, value := range values {
		if !p.argument(value, seen) {
			return false
		}
	}
	return true
}

func (p *SuccessFlow) argument(ref pdg.Ref, seen map[pdg.Ref]uint8) bool {
	if ref == 0 {
		return false
	}
	if state := seen[ref]; state != 0 {
		return state == 2
	}
	if !p.inert(ref) {
		return false
	}
	seen[ref] = 1
	if !p.arguments(ref, seen) {
		return false
	}
	seen[ref] = 2
	return true
}

func concreteNilWriter(kind string) bool {
	switch kind {
	case `"pointer"`, `"map"`, `"slice"`, `"channel"`, `"function"`:
		return true
	default:
		return false
	}
}

func memoryKind(graph *pdg.Graph, attrs pdg.Attributes) string {
	for _, extension := range attrs.Extensions {
		if graph.Tables.Text(extension.Name) == "gstyle.memory:kind" {
			return graph.Tables.Text(extension.Value)
		}
	}
	return ""
}
