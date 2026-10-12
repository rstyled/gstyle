package gopatterns

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Complete source matching establishes operation correspondence. Compare the
// compiler-resolved bindings on those operations, not approximate flow edges.
func matchingParameterizationBindings(a, b *pdg.Graph) bool {
	if len(a.Nodes) != len(b.Nodes) || len(a.Definitions) != len(b.Definitions) {
		return false
	}
	if !slices.Equal(parameterizationScopes(a), parameterizationScopes(b)) {
		return false
	}
	return matchingOperationBindings(a, b) && matchingDefinitionBindings(a, b)
}

type parameterizationBinding struct {
	scope      int
	role       pdg.Role
	typeStatus pdg.Status
	typeName   string
}

func parameterizationScopes(graph *pdg.Graph) []parameterizationBinding {
	scopes := make(map[pdg.Text]int)
	bindings := make([]parameterizationBinding, len(graph.Symbols))
	for i, symbol := range graph.Symbols {
		if _, exists := scopes[symbol.Scope]; !exists {
			scopes[symbol.Scope] = len(scopes)
		}
		typeName := ""
		if symbol.Type.Status == pdg.Known {
			typeName = graph.Tables.Text(symbol.Type.Value)
		}
		bindings[i] = parameterizationBinding{scopes[symbol.Scope], symbol.Role, symbol.Type.Status, typeName}
	}
	return bindings
}

func matchingOperationBindings(a, b *pdg.Graph) bool {
	for i, node := range a.Nodes {
		if node.Category != b.Nodes[i].Category {
			return false
		}
		if !matchingBindingAttributes(a.NodeAttributes(node), b.NodeAttributes(b.Nodes[i])) {
			return false
		}
	}
	return true
}

func matchingBindingAttributes(a, b pdg.Attributes) bool {
	return slices.Equal(a.Symbols, b.Symbols) && slices.Equal(a.Defines, b.Defines) &&
		slices.Equal(accessBindings(a.Reads), accessBindings(b.Reads)) && slices.Equal(accessBindings(a.Writes), accessBindings(b.Writes))
}

func accessBindings(accesses []pdg.Access) []pdg.Ref {
	bindings := make([]pdg.Ref, len(accesses))
	for i, access := range accesses {
		bindings[i] = access.Symbol
	}
	return bindings
}

func matchingDefinitionBindings(a, b *pdg.Graph) bool {
	for i, definition := range a.Definitions {
		other := b.Definitions[i]
		if definition.Node != other.Node || definition.Symbol != other.Symbol {
			return false
		}
	}
	return true
}
