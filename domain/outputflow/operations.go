package outputflow

import "github.com/rstyled/gstyle/domain/pdg"

func (p *SuccessFlow) inert(ref pdg.Ref) bool {
	node := p.graph.Nodes[ref-1]
	attrs := p.graph.NodeAttributes(node)
	return !memoryAccess(attrs.Reads) && !memoryAccess(attrs.Writes) && p.inertOperation(node.Category, attrs)
}

func (p *SuccessFlow) inertOperation(category pdg.Category, attrs pdg.Attributes) bool {
	switch category {
	case pdg.Read:
		return normalRead(p.graph, attrs)
	case pdg.Declaration, pdg.Assignment:
		return inertWrite(p.graph.Tables.Text(attrs.WriteForm))
	case pdg.Control:
		return normalControl(p.graph, attrs)
	case pdg.Computation:
		return runtimeFact(p.graph, attrs, "normal-completion") == "true"
	case pdg.Other:
		return normalOther(p.graph, attrs)
	default:
		return false
	}
}

func memoryAccess(accesses []pdg.Access) bool {
	for _, access := range accesses {
		if !access.DirectSymbol() {
			return true
		}
	}
	return false
}

func inertWrite(form string) bool {
	switch form {
	case "", ":=", "=", "++", "--", "+=", "-=", "*=", "&=", "|=", "^=", "&^=":
		return true
	default:
		return false
	}
}

func normalControl(graph *pdg.Graph, attrs pdg.Attributes) bool {
	if runtimeFact(graph, attrs, "normal-completion") == "true" {
		return true
	}
	switch graph.Tables.Text(attrs.Construct) {
	case "if", "for", "switch", "type-switch", "short-circuit":
		return true
	case "switch-case":
		return normalPredicate(graph.Tables.Text(attrs.Operator.Value))
	default:
		return false
	}
}

func normalPredicate(operator string) bool { return operator == "type-is" || operator == "truth-test" }

func inertTransfer(construct string) bool {
	switch construct {
	case "goto", "break", "continue", "fallthrough":
		return true
	default:
		return false
	}
}

func normalRead(graph *pdg.Graph, attrs pdg.Attributes) bool {
	if len(attrs.Writes) != 0 {
		return false
	}
	return len(attrs.Reads) != 0 || runtimeFact(graph, attrs, "normal-completion") == "true"
}

func normalOther(graph *pdg.Graph, attrs pdg.Attributes) bool {
	return inertTransfer(graph.Tables.Text(attrs.Construct)) || runtimeFact(graph, attrs, "normal-completion") == "true"
}
