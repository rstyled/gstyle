package patterns

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type parameterBinding struct {
	symbol   pdg.Ref
	position uint32
	role     pdg.Role
}

type dependencyBinding struct {
	scope    int
	role     pdg.Role
	status   pdg.Status
	typeName string
}

type dependencyEdge struct {
	kind                                         pdg.EdgeKind
	source, target, symbol, definition, location pdg.Ref
	position                                     pdg.Position
	loopCarried                                  uint8
	subkind, outcome                             string
}

// RefactorDependencyShapeMatches compares checked graphs after a complete AST
// match establishes operation correspondence. IDs and source offsets are local;
// binding partitions, definition ownership and directed constraints must agree.
// This is a conservative whole-body check, not graph isomorphism or equivalence.
func RefactorDependencyShapeMatches(a, b *pdg.Graph) bool {
	if len(a.Definitions) != len(b.Definitions) {
		return false
	}
	if !sameStorageShape(a, b) {
		return false
	}
	return sameInterfaceBindingShape(a, b) && sameOperationShape(a, b) && sameDefinitionShape(a, b) && sameEdgeShape(a, b)
}

func sameStorageShape(a, b *pdg.Graph) bool {
	if !sameDependencyProfile(a, b) {
		return false
	}
	return slices.Equal(bindingShape(a), bindingShape(b)) && sameMemoryShape(a, b)
}

func sameDependencyProfile(a, b *pdg.Graph) bool {
	ap, bp := a.Tables.Policies[a.Profile-1], b.Tables.Policies[b.Profile-1]
	return a.Tables.Text(ap.ID) == b.Tables.Text(bp.ID) && a.Tables.Text(ap.Version) == b.Tables.Text(bp.Version) &&
		a.Tables.Text(a.Producer.Language) == b.Tables.Text(b.Producer.Language)
}

func sameMemoryShape(a, b *pdg.Graph) bool {
	if len(a.Locations) != len(b.Locations) {
		return false
	}
	for i, location := range a.Locations {
		other := b.Locations[i]
		if a.Tables.Text(location.Abstraction) != b.Tables.Text(other.Abstraction) {
			return false
		}
		if !sameLocationExtensions(a.Tables, location.Extensions, b.Tables, other.Extensions) {
			return false
		}
	}
	return true
}

func sameLocationExtensions(a *pdg.Tables, ax []pdg.Extension, b *pdg.Tables, bx []pdg.Extension) bool {
	if len(ax) != len(bx) {
		return false
	}
	for i, extension := range ax {
		if a.Text(extension.Name) != b.Text(bx[i].Name) || a.Text(extension.Value) != b.Text(bx[i].Value) {
			return false
		}
	}
	return true
}

func sameInterfaceBindingShape(a, b *pdg.Graph) bool {
	return slices.Equal(parameterBindings(a.Function.Inputs), parameterBindings(b.Function.Inputs)) &&
		slices.Equal(parameterBindings(a.Function.Outputs), parameterBindings(b.Function.Outputs))
}

func parameterBindings(parameters []pdg.Parameter) []parameterBinding {
	bindings := make([]parameterBinding, len(parameters))
	for i, parameter := range parameters {
		bindings[i] = parameterBinding{parameter.Symbol, parameter.Position, parameter.Role}
	}
	return bindings
}

func sameOperationShape(a, b *pdg.Graph) bool {
	if len(a.Nodes) != len(b.Nodes) {
		return false
	}
	for i, node := range a.Nodes {
		if node.Category != b.Nodes[i].Category || !sameNodeBindings(a.NodeAttributes(node), b.NodeAttributes(b.Nodes[i])) {
			return false
		}
	}
	return true
}

func sameNodeBindings(a, b pdg.Attributes) bool {
	return a.Position == b.Position && slices.Equal(a.Symbols, b.Symbols) && slices.Equal(a.Defines, b.Defines) &&
		slices.Equal(a.Reads, b.Reads) && slices.Equal(a.Writes, b.Writes)
}

func bindingShape(graph *pdg.Graph) []dependencyBinding {
	scopes := make(map[pdg.Text]int)
	shape := make([]dependencyBinding, len(graph.Symbols))
	for i, symbol := range graph.Symbols {
		if _, exists := scopes[symbol.Scope]; !exists {
			scopes[symbol.Scope] = len(scopes)
		}
		shape[i] = dependencyBinding{scopes[symbol.Scope], symbol.Role, symbol.Type.Status, graph.Tables.Text(symbol.Type.Value)}
	}
	return shape
}

func sameDefinitionShape(a, b *pdg.Graph) bool {
	for i, definition := range a.Definitions {
		other := b.Definitions[i]
		if definition.Node != other.Node || definition.Symbol != other.Symbol || definition.Location != other.Location {
			return false
		}
	}
	return true
}

func sameEdgeShape(a, b *pdg.Graph) bool {
	if len(a.Edges) != len(b.Edges) {
		return false
	}
	counts := dependencyEdgeCounts(a)
	for _, edge := range b.Edges {
		counts[dependencyEdgeKey(b, edge)]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func dependencyEdgeCounts(graph *pdg.Graph) map[dependencyEdge]int {
	counts := make(map[dependencyEdge]int, len(graph.Edges))
	for _, edge := range graph.Edges {
		counts[dependencyEdgeKey(graph, edge)]++
	}
	return counts
}

func dependencyEdgeKey(graph *pdg.Graph, edge pdg.Edge) dependencyEdge {
	return dependencyEdge{edge.Kind, edge.Source, edge.Target, edge.Symbol, edge.Definition, edge.Location,
		edge.Position, edge.LoopCarried, graph.Tables.Text(edge.Subkind), graph.Tables.Text(edge.Outcome)}
}
