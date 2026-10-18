package gopatterns

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/pdgmemory"
)

const generalBindingDescription = "Conservative fixed-point binding and may-alias storage dependencies on the shared normal operation CFG. Direct local writes kill bindings; indirect and opaque call writes are weak. Memory is field-insensitive; unknown targets retain all possible locations. Closure construction retains captured cells; lexical body links and external captured storage are explicit. Runtime environments, exceptional, deferred and concurrent flow remain incomplete."

type memoryBuilder struct {
	ir        *irBuilder
	model     *pdgmemory.Model
	locations map[pdgmemory.Place]pdg.Ref
	sources   map[pdgmemory.Place]pdg.Ref
	nodeCount int
}

func (m *memoryBuilder) location(place pdgmemory.Place) {
	b := m.ir
	ref := pdg.Ref(len(b.graph.Locations) + 1)
	id := "memory-external"
	if place.Symbol != 0 {
		id = "memory-s" + strconv.Itoa(int(place.Symbol))
	}
	if place.Allocation != 0 {
		id = "memory-n" + strconv.Itoa(int(place.Allocation))
	}
	b.graph.Locations = append(b.graph.Locations, pdg.Location{ID: b.pool.Text(id), Abstraction: b.pool.Text("gstyle.go.may-alias-storage/0.1.0"), Description: b.pool.Text(fmt.Sprintf("May-alias storage %s; allocation sites can represent multiple objects.", id)), Evidence: b.pool.Evidence(pdg.Approximated, generalBindingDescription), Extensions: m.locationExtensions(place)})
	m.locations[place] = ref
	m.memorySource(place, ref)
}

func (m *memoryBuilder) memorySource(place pdgmemory.Place, location pdg.Ref) {
	b := m.ir
	if place.Allocation != 0 {
		m.sources[place] = place.Allocation
		m.memoryWrite(place.Allocation, pdg.Access{Location: location}, false)
		return
	}
	attrs := pdg.Attributes{Construct: b.pool.Text("storage-identity"), Description: b.pool.Text("Storage identity; not an executable value computation.")}
	source := b.scalarEntity(0, pdg.MemorySummary, attrs)
	m.sources[place] = source
	if place.Symbol == 0 {
		m.memoryInput(source, 0, location)
		return
	}
	role := b.graph.Symbols[place.Symbol-1].Role
	if role == pdg.GlobalRole || role == pdg.CaptureRole {
		m.memoryInput(source, place.Symbol, location)
	}
}

func (m *memoryBuilder) memoryInput(node, symbol, location pdg.Ref) {
	b := m.ir
	b.attributes[b.graph.Nodes[node-1].Attributes-1].Construct = b.pool.Text("memory-input")
	if symbol != 0 {
		b.storageDefinition(node, symbol, 0)
	}
	b.storageDefinition(node, 0, location)
}

func (m *memoryBuilder) bindingStorage(definition pdg.Definition) {
	location := m.locations[pdgmemory.Place{Symbol: definition.Symbol}]
	if definition.Symbol == 0 || location == 0 {
		return
	}
	m.ir.weakDefinition(definition.Node, 0, location)
}

func (m *memoryBuilder) locationExtensions(place pdgmemory.Place) []pdg.Extension {
	if place.Allocation == 0 {
		return nil
	}
	b := m.ir
	return b.pool.SingletonExtension("gstyle.memory:allocation", strconv.Quote(b.pool.Tables.Text(b.graph.Nodes[place.Allocation-1].ID)))
}

func (b *irBuilder) currentAttributes(node pdg.Node) pdg.Attributes {
	return b.attributes[node.Attributes-1]
}

func (b *irBuilder) generalMemory() error {
	model, err := pdgmemory.Analyze(&b.graph, b.currentAttributes)
	if err != nil {
		return err
	}
	m := memoryBuilder{ir: b, model: model, locations: map[pdgmemory.Place]pdg.Ref{}, sources: map[pdgmemory.Place]pdg.Ref{}, nodeCount: len(b.graph.Nodes)}
	b.weakDefinitions = map[pdg.Ref]bool{}
	definitions := slices.Clone(b.graph.Definitions)
	for _, place := range model.Places() {
		m.location(place)
	}
	for _, definition := range definitions {
		m.bindingStorage(definition)
	}
	for index := 0; index < m.nodeCount; index++ {
		m.operation(pdg.Ref(index + 1))
	}
	m.weakEvidence()
	return nil
}

func (b *irBuilder) weakDefinition(node, symbol, location pdg.Ref) bool {
	attrs := b.attributes[b.graph.Nodes[node-1].Attributes-1]
	for _, ref := range attrs.Defines {
		definition := b.graph.Definitions[ref-1]
		if definition.Symbol == symbol && definition.Location == location {
			return false
		}
	}
	ref := b.storageDefinition(node, symbol, location)
	b.weakDefinitions[ref] = true
	return true
}
