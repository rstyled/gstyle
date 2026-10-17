package gopatterns

import (
	"encoding/json"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/pdgmemory"
)

func (m *memoryBuilder) operation(ref pdg.Ref) {
	b := m.ir
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	if m.model.AddressOnly(ref) {
		for _, read := range attrs.Reads {
			if read.Symbol != 0 {
				attrs.Symbols = append(attrs.Symbols, read.Symbol)
			}
		}
		attrs.Reads = nil
	} else {
		attrs.Reads = m.resolveAccesses(ref, attrs.Reads)
	}
	attrs.Writes = m.resolveAccesses(ref, attrs.Writes)
	writes := append([]pdg.Access(nil), attrs.Writes...)
	for _, write := range writes {
		if write.Location != 0 {
			m.memoryWrite(ref, write, true)
		}
	}
	m.callEffects(ref)
	m.addressEdges(ref)
	m.captureEdges(ref)
}
func (m *memoryBuilder) resolveAccesses(ref pdg.Ref, accesses []pdg.Access) []pdg.Access {
	var result []pdg.Access
	for _, access := range accesses {
		if access.UnresolvedReason == 0 {
			result = append(result, access)
			continue
		}
		for _, place := range m.model.Targets(ref) {
			result = append(result, pdg.Access{Location: m.locations[place], Position: access.Position})
		}
	}
	return result
}
func (m *memoryBuilder) memoryWrite(ref pdg.Ref, write pdg.Access, aliasBinding bool) {
	b := m.ir
	b.weakDefinition(ref, 0, write.Location)
	if !aliasBinding {
		b.attributes[b.graph.Nodes[ref-1].Attributes-1].Writes = append(b.attributes[b.graph.Nodes[ref-1].Attributes-1].Writes, write)
		return
	}
	for place, location := range m.locations {
		if location == write.Location && place.Symbol != 0 {
			m.indirectBindingWrite(ref, place.Symbol, write.Position)
		}
	}
}
func (m *memoryBuilder) indirectBindingWrite(ref, symbol pdg.Ref, position pdg.Position) {
	b := m.ir
	if !b.weakDefinition(ref, symbol, 0) {
		return
	}
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Writes = append(attrs.Writes, pdg.Access{Symbol: symbol, Position: position})
}
func (m *memoryBuilder) callEffects(ref pdg.Ref) {
	b := m.ir
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	if !pdgmemory.OpaqueCall(&b.graph, b.graph.Nodes[ref-1], *attrs) {
		return
	}
	for _, place := range m.model.Places() {
		access := pdg.Access{Location: m.locations[place]}
		attrs.Reads = append(attrs.Reads, access)
		attrs.Writes = append(attrs.Writes, access)
		m.memoryWrite(ref, access, true)
	}
}
func (m *memoryBuilder) addressEdges(ref pdg.Ref) {
	b := m.ir
	attrs := b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	if attrs.Operator.Status != pdg.Known || b.pool.Tables.Text(attrs.Operator.Value) != "&" {
		return
	}
	for _, place := range m.model.AddressTargets(ref) {
		b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("memory-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: m.sources[place], Target: ref, Kind: pdg.Data, Subkind: b.pool.Text("address"), Location: m.locations[place], Evidence: b.pool.Evidence(pdg.Approximated, generalBindingDescription), Provenance: b.sourceOrigin})
	}
}
func (m *memoryBuilder) weakEvidence() {
	b := m.ir
	for index := range b.graph.Nodes {
		attrs := &b.attributes[b.graph.Nodes[index].Attributes-1]
		var ids []string
		for _, definition := range attrs.Defines {
			if b.weakDefinitions[definition] {
				ids = append(ids, b.pool.Tables.Text(b.graph.Definitions[definition-1].ID))
			}
		}
		if len(ids) == 0 {
			continue
		}
		data, _ := json.Marshal(ids)
		attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.memory:weak-definitions", string(data))...)
	}
}

func (m *memoryBuilder) captureEdges(ref pdg.Ref) {
	b := m.ir
	attrs := b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	if b.pool.Tables.Text(attrs.Construct) != "closure" {
		return
	}
	for _, symbol := range attrs.Symbols {
		place := pdgmemory.Place{Symbol: symbol}
		b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("capture-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: m.sources[place], Target: ref, Kind: pdg.Data, Subkind: b.pool.Text("capture-address"), Symbol: symbol, Location: m.locations[place], Evidence: b.pool.Evidence(pdg.Approximated, generalBindingDescription), Provenance: b.sourceOrigin})
	}
}
