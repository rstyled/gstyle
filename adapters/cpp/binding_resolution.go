package cpp

import (
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) readSymbol(node *vertex) pdg.Ref {
	if len(node.Out["REF"]) != 0 {
		return p.referredSymbol(node)
	}
	target := p.reachingDeclaration(node)
	if target == nil || p.symbols[target] == 0 {
		return 0
	}
	attrs := &p.attrs[p.refs[node]-1]
	attrs.Extensions = append(attrs.Extensions, p.jsonExtension("gstyle.cpp:binding-resolution", map[string]any{"declaration": strconv.FormatUint(target.ID, 10), "policy": "matching recorded REACHING_DEF variable and one declaration identity across all recorded definitions"}))
	return p.symbols[target]
}

func (p *projection) reachingDeclaration(node *vertex) *vertex {
	var result *vertex
	for _, edge := range p.doc.MethodEdges[p.method] {
		if !bindingDefinition(edge, node) {
			continue
		}
		target := definitionDeclaration(edge.Source, edge.Variable)
		if target == nil {
			return nil
		}
		if result != nil && result != target {
			return nil
		}
		result = target
	}
	return result
}

func bindingDefinition(edge edge, node *vertex) bool {
	return edge.Label == "REACHING_DEF" && edge.Target == node && edge.Variable != "" && edge.Variable == node.text("NAME")
}

func definitionDeclaration(node *vertex, variable string) *vertex {
	target := sourceDeclaration(node)
	if target == nil || target.text("NAME") != variable {
		return nil
	}
	return target
}

func sourceDeclaration(node *vertex) *vertex {
	if node.Label == "METHOD_PARAMETER_IN" {
		return node
	}
	if category(node) == pdg.Assignment {
		node = argument(node, 1)
	}
	if node == nil || node.Label != "IDENTIFIER" || len(node.Out["REF"]) != 1 {
		return nil
	}
	return node.Out["REF"][0]
}
