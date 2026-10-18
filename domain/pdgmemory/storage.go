package pdgmemory

import "github.com/rstyled/gstyle/domain/pdg"

func (m *Model) Places() []Place { return append([]Place(nil), m.places...) }

func (m *Model) Targets(ref pdg.Ref) []Place { return m.expand(m.targets(ref)) }

func (m *Model) AddressTargets(ref pdg.Ref) []Place { return m.expand(m.nodeValues(ref)) }

func (m *Model) AddressOnly(ref pdg.Ref) bool { return m.addressOnly[ref-1] }

func (m *Model) storage(ref pdg.Ref) []Place {
	ref = m.unwrap(ref)
	if ref == 0 {
		return []Place{{}}
	}
	if symbol := m.directSymbol(ref); symbol != 0 {
		return []Place{{Symbol: symbol}}
	}
	return m.indirectStorage(ref)
}

func (m *Model) indirectStorage(ref pdg.Ref) []Place {
	switch m.operator(ref) {
	case "dereference":
		return m.nodeValues(m.operand(ref, 1))
	case "field", "index":
		return m.selectedStorage(ref)
	case "composite":
		return []Place{{Allocation: ref}}
	default:
		return []Place{{}}
	}
}

func (m *Model) unwrap(ref pdg.Ref) pdg.Ref {
	for count := 0; count < len(m.attrs) && m.operator(ref) == "parentheses"; count++ {
		ref = m.operand(ref, 1)
	}
	return ref
}

func (m *Model) selectedStorage(ref pdg.Ref) []Place {
	base := m.operand(ref, 1)
	if base == 0 {
		return []Place{{}}
	}
	kind := m.extension(m.attrs[base-1].Extensions, "gstyle.memory:kind")
	if kind == `"immutable"` {
		return nil
	}
	if kind == `"aggregate"` {
		if symbol := m.directSymbol(base); symbol != 0 {
			return []Place{{Symbol: symbol}}
		}
		return []Place{{}}
	}
	return m.nodeValues(base)
}

func (m *Model) collectPlaces() {
	for i, attrs := range m.attrs {
		m.collectOperationPlaces(pdg.Ref(i+1), attrs)
	}
	for i, symbol := range m.graph.Symbols {
		if symbol.Role == pdg.GlobalRole || symbol.Role == pdg.CaptureRole || m.addressed[i] {
			m.places, _ = union(m.places, []Place{{Symbol: pdg.Ref(i + 1)}})
		}
	}
}

func (m *Model) collectOperationPlaces(ref pdg.Ref, attrs pdg.Attributes) {
	if m.operator(ref) == "&" {
		m.places, _ = union(m.places, m.nodeValues(ref))
	}
	if unresolvedAccess(attrs) {
		m.places, _ = union(m.places, m.targets(ref))
	}
	if OpaqueCall(m.graph, m.graph.Nodes[ref-1], attrs) {
		m.places, _ = union(m.places, []Place{{}})
	}
	for _, place := range m.nodeValues(ref) {
		if place.Allocation == ref {
			m.places, _ = union(m.places, []Place{place})
		}
	}
}

func (m *Model) targets(ref pdg.Ref) []Place {
	if len(m.addresses[ref-1]) == 0 {
		return m.storage(ref)
	}
	var result []Place
	for _, address := range m.addresses[ref-1] {
		result, _ = union(result, m.storage(address))
	}
	return result
}

func (m *Model) expand(places []Place) []Place {
	result := append([]Place(nil), places...)
	for _, place := range places {
		if place == (Place{}) {
			result, _ = union(result, m.places)
			break
		}
	}
	return result
}

func unresolvedAccess(attrs pdg.Attributes) bool {
	for _, access := range attrs.Reads {
		if access.UnresolvedReason != 0 {
			return true
		}
	}
	for _, access := range attrs.Writes {
		if access.UnresolvedReason != 0 {
			return true
		}
	}
	return false
}

// Builtin identity is recorded by the typed producer. User functions with the
// same spelling have qualified IDs and cannot inherit intrinsic effects.
func OpaqueCall(graph *pdg.Graph, node pdg.Node, attrs pdg.Attributes) bool {
	if node.Category != pdg.Call {
		return false
	}
	if attrs.Callee.Status != pdg.Known {
		return true
	}
	switch graph.Tables.Text(attrs.Callee.Value) {
	case "len", "cap", "complex", "real", "imag", "min", "max", "new", "make", "panic", "recover", "print", "println":
		return false
	default:
		return true
	}
}
