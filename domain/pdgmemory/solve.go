package pdgmemory

import "github.com/rstyled/gstyle/domain/pdg"

func (m *Model) seed() {
	for i, symbol := range m.graph.Symbols {
		m.seedSymbol(pdg.Ref(i+1), symbol)
	}
	for i := range m.attrs {
		m.seedAddress(pdg.Ref(i + 1))
	}
}

func (m *Model) seedSymbol(ref pdg.Ref, symbol pdg.Symbol) {
	if !m.symbolReference(ref) {
		return
	}
	if externalBinding(symbol.Role) || m.extension(symbol.Extensions, "gstyle.runtime:unstable") == "true" {
		m.bindings[ref-1] = []Place{{}}
	}
}

func (m *Model) seedAddress(ref pdg.Ref) {
	if m.operator(ref) == "closure" {
		m.seedCaptures(ref)
		return
	}
	if m.operator(ref) != "&" {
		return
	}
	operand := m.unwrap(m.operand(ref, 1))
	if operand != 0 {
		m.addressOnly[operand-1] = true
	}
	symbol := m.directSymbol(operand)
	if symbol == 0 {
		return
	}
	m.addressed[symbol-1] = true
	if m.symbolReference(symbol) {
		m.bindings[symbol-1] = []Place{{}}
	}
}

// Constraints only add places in a finite variable/allocation/unknown domain.
// Ignoring assignment kills gives may-alias sets, never a no-alias proof.
func (m *Model) update() bool {
	changed := false
	for i := range m.attrs {
		values, grew := union(m.values[i], m.expression(pdg.Ref(i+1)))
		m.values[i] = values
		changed = changed || grew
	}
	for i, attrs := range m.attrs {
		for _, write := range attrs.Writes {
			changed = m.bindingWrite(pdg.Ref(i+1), write) || changed
		}
	}
	return changed
}

func (m *Model) bindingWrite(node pdg.Ref, write pdg.Access) bool {
	if write.Symbol == 0 || !m.symbolReference(write.Symbol) {
		return false
	}
	source := m.writeSource(node, write.Position)
	values := m.nodeValues(source)
	if source == 0 && m.graph.Nodes[node-1].Category != pdg.Declaration {
		values = []Place{{}}
	}
	joined, changed := union(m.bindings[write.Symbol-1], values)
	m.bindings[write.Symbol-1] = joined
	return changed
}

func (m *Model) expression(ref pdg.Ref) []Place {
	if !m.reference(ref) {
		return nil
	}
	if m.extension(m.attrs[ref-1].Extensions, "gstyle.runtime:nil") == "true" {
		return nil
	}
	if symbol := m.directSymbol(ref); symbol != 0 {
		return m.bindings[symbol-1]
	}
	return m.referenceExpression(ref)
}

func (m *Model) referenceExpression(ref pdg.Ref) []Place {
	switch m.operator(ref) {
	case "&":
		return m.storage(m.operand(ref, 1))
	case "parentheses", "conversion":
		return m.converted(ref)
	case "composite":
		return []Place{{Allocation: ref}}
	default:
		return m.referenceResult(ref)
	}
}

func (m *Model) writeSource(node pdg.Ref, position pdg.Position) pdg.Ref {
	if position == 0 {
		position = 1
	}
	source := m.operand(node, position)
	if source == 0 && len(m.operands[node-1]) == 1 {
		return m.operand(node, 1)
	}
	return source
}

func (m *Model) converted(ref pdg.Ref) []Place {
	operand := m.operand(ref, 1)
	if operand == 0 {
		return []Place{{}}
	}
	if m.reference(operand) || m.extension(m.attrs[operand-1].Extensions, "gstyle.runtime:nil") == "true" {
		return m.nodeValues(operand)
	}
	return []Place{{}}
}

func (m *Model) referenceResult(ref pdg.Ref) []Place {
	attrs := m.attrs[ref-1]
	if m.graph.Nodes[ref-1].Category == pdg.Call && attrs.Callee.Status == pdg.Known {
		callee := m.graph.Tables.Text(attrs.Callee.Value)
		if callee == "new" || callee == "make" {
			return []Place{{Allocation: ref}}
		}
	}
	return []Place{{}}
}

func (m *Model) directSymbol(ref pdg.Ref) pdg.Ref {
	for count := 0; count < len(m.attrs) && ref != 0; count++ {
		if symbol := m.readSymbol(ref); symbol != 0 {
			return symbol
		}
		if m.operator(ref) != "parentheses" {
			return 0
		}
		ref = m.operand(ref, 1)
	}
	return 0
}

func (m *Model) readSymbol(ref pdg.Ref) pdg.Ref {
	attrs := m.attrs[ref-1]
	if m.graph.Nodes[ref-1].Category != pdg.Read || len(attrs.Reads) != 1 {
		return 0
	}
	return attrs.Reads[0].Symbol
}

func (m *Model) seedCaptures(ref pdg.Ref) {
	for _, symbol := range m.attrs[ref-1].Symbols {
		m.addressed[symbol-1] = true
		if m.symbolReference(symbol) {
			m.bindings[symbol-1] = []Place{{}}
		}
	}
}

func externalBinding(role pdg.Role) bool {
	return role == pdg.ParameterRole || role == pdg.ReceiverRole || role == pdg.GlobalRole
}
