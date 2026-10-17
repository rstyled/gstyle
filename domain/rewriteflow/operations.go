package rewriteflow

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type comparison struct{ a, b *pdg.Graph }

func (pair comparison) attributes(x, y pdg.Attributes, substitute bool) bool {
	a, b := pair.a, pair.b
	if !pair.operationFacts(x, y, substitute) || !sameBindingLists(x, y) {
		return false
	}
	return sameAccesses(a, x.Reads, b, y.Reads) && sameAccesses(a, x.Writes, b, y.Writes) && sameExtensions(a, x.Extensions, b, y.Extensions)
}

func (pair comparison) operationFacts(x, y pdg.Attributes, substitute bool) bool {
	a, b := pair.a, pair.b
	if !sameOperationShape(x, y) {
		return false
	}
	if !sameFact(a, x.Operator, b, y.Operator) || !sameFact(a, x.Literal, b, y.Literal) {
		return false
	}
	return sameConstructs(a, x, b, y) && (substitute || sameFact(a, x.Callee, b, y.Callee))
}

type access struct {
	symbol, location pdg.Ref
	position         pdg.Position
	unresolved       string
}

func sameOperations(a, b *pdg.Graph, allowed map[pdg.Ref]bool) bool {
	if len(a.Nodes) != len(b.Nodes) {
		return false
	}
	for i, node := range a.Nodes {
		other := b.Nodes[i]
		if node.Category != other.Category || node.Synthetic != other.Synthetic {
			return false
		}
		if !(comparison{a, b}).attributes(a.NodeAttributes(node), b.NodeAttributes(other), allowed[pdg.Ref(i+1)]) {
			return false
		}
	}
	return true
}

func sameOperationShape(x, y pdg.Attributes) bool {
	return x.CallForm == y.CallForm && x.ExitKind == y.ExitKind && x.Position == y.Position
}

func sameConstructs(a *pdg.Graph, x pdg.Attributes, b *pdg.Graph, y pdg.Attributes) bool {
	return a.Tables.Text(x.Construct) == b.Tables.Text(y.Construct) && a.Tables.Text(x.WriteForm) == b.Tables.Text(y.WriteForm) && slices.Equal(texts(a, x.Outcomes), texts(b, y.Outcomes))
}

func sameFact(a *pdg.Graph, x pdg.Fact, b *pdg.Graph, y pdg.Fact) bool {
	return x.Status == y.Status && a.Tables.Text(x.Value) == b.Tables.Text(y.Value)
}

func sameBindingLists(x, y pdg.Attributes) bool {
	return slices.Equal(x.Symbols, y.Symbols) && slices.Equal(x.Defines, y.Defines)
}

func texts(graph *pdg.Graph, values []pdg.Text) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = graph.Tables.Text(value)
	}
	return result
}

func accesses(graph *pdg.Graph, values []pdg.Access) []access {
	result := make([]access, len(values))
	for i, value := range values {
		result[i] = access{value.Symbol, value.Location, value.Position, graph.Tables.Text(value.UnresolvedReason)}
	}
	return result
}

func sameAccesses(a *pdg.Graph, x []pdg.Access, b *pdg.Graph, y []pdg.Access) bool {
	return slices.Equal(accesses(a, x), accesses(b, y))
}
