package rewriteflow

import (
	"maps"

	"github.com/rstyled/gstyle/domain/pdg"
)

type extension struct{ name, value string }

func sameStorage(a, b *pdg.Graph) bool {
	if len(a.Symbols) != len(b.Symbols) || len(a.Definitions) != len(b.Definitions) || len(a.Locations) != len(b.Locations) {
		return false
	}
	return sameSymbols(a, b) && sameDefinitions(a, b) && sameLocations(a, b)
}

func sameSymbols(a, b *pdg.Graph) bool {
	for i, symbol := range a.Symbols {
		other := b.Symbols[i]
		if symbol.Role != other.Role || !sameFact(a, symbol.Type, b, other.Type) {
			return false
		}
		if !sameExtensions(a, symbol.Extensions, b, other.Extensions) {
			return false
		}
	}
	return true
}

func sameDefinitions(a, b *pdg.Graph) bool {
	for i, definition := range a.Definitions {
		other := b.Definitions[i]
		if definition.Node != other.Node || definition.Symbol != other.Symbol || definition.Location != other.Location {
			return false
		}
	}
	return true
}

func sameLocations(a, b *pdg.Graph) bool {
	for i, location := range a.Locations {
		other := b.Locations[i]
		if a.Tables.Text(location.Abstraction) != b.Tables.Text(other.Abstraction) || !sameExtensions(a, location.Extensions, b, other.Extensions) {
			return false
		}
	}
	return true
}

func sameExtensions(a *pdg.Graph, x []pdg.Extension, b *pdg.Graph, y []pdg.Extension) bool {
	return maps.Equal(semanticExtensions(a, x), semanticExtensions(b, y))
}

// Only callable source spelling is excluded: canonical callable type is retained,
// and import normalization may change the source qualifier. Unknown extensions
// must still match so future dependency metadata cannot be silently ignored.
func semanticExtensions(graph *pdg.Graph, values []pdg.Extension) map[extension]bool {
	result := map[extension]bool{}
	for _, value := range values {
		name := graph.Tables.Text(value.Name)
		if name != "gstyle:callable_source_type" {
			result[extension{name, graph.Tables.Text(value.Value)}] = true
		}
	}
	return result
}
