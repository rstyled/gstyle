package gopatterns

import (
	"go/ast"
	"slices"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

// reachingState owns sorted definition sets by binding. Writes kill the old
// set; joins union predecessor sets. It is temporary, never retained in the IR.
type reachingState map[pdg.Ref][]pdg.Ref

func (state reachingState) merge(other reachingState) {
	for symbol, definitions := range other {
		values := append(state[symbol], definitions...)
		slices.Sort(values)
		state[symbol] = slices.Compact(values)
	}
}

type reachingAnalysis struct {
	builder          *irBuilder
	controls         map[ast.Node]pdg.Ref
	outputs          []reachingState
	predecessors     [][]int
	remaining        []int
	nextEdgeID       int
	evidence, origin pdg.Ref
}

func (a *reachingAnalysis) edgeID() pdg.Text {
	a.nextEdgeID++
	return a.builder.pool.Text("e" + strconv.Itoa(a.nextEdgeID))
}

func (a *reachingAnalysis) input(index int) reachingState {
	state := reachingState{}
	if index == 0 {
		a.seed(state)
		return state
	}
	for _, previous := range a.predecessors[index] {
		if a.outputs[previous] == nil {
			continue
		}
		state.merge(a.outputs[previous])
		a.remaining[previous]--
		if a.remaining[previous] == 0 {
			a.outputs[previous] = nil
		}
	}
	return state
}

func (a *reachingAnalysis) seed(state reachingState) {
	for i, definition := range a.builder.graph.Definitions {
		category := a.builder.graph.Nodes[definition.Node-1].Category
		if a.initialDefinition(definition, category) {
			state[a.key(definition.Symbol, definition.Location)] = []pdg.Ref{pdg.Ref(i + 1)}
		}
	}
}

func (a *reachingAnalysis) block(block *cfg.Block, state reachingState) {
	for _, source := range block.Nodes {
		if ref := a.builder.nodes[source]; ref != 0 {
			a.operation(ref, state)
		}
	}
}

// Postorder processes RHS reads before the enclosing assignment publishes its
// writes, including parallel and read-modify-write assignments.
func (a *reachingAnalysis) operations(source ast.Node) []pdg.Ref {
	var operations []pdg.Ref
	var stack []ast.Node
	ast.Inspect(source, func(node ast.Node) bool {
		if node != nil {
			stack = append(stack, node)
			for _, read := range a.builder.implicitReads[node] {
				a.valueEdge(read, a.builder.nodes[node], a.builder.attributes[a.builder.graph.Nodes[read-1].Attributes-1].Reads[0].Position)
				operations = append(operations, read)
			}
			return true
		}
		last := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if ref := a.builder.nodes[last]; ref != 0 {
			parent, position := a.parent(stack, last)
			a.valueEdge(ref, parent, position)
			operations = append(operations, ref)
		}
		return true
	})
	if ref := a.controls[source]; ref != 0 && !slices.Contains(operations, ref) {
		a.valueEdge(a.builder.nodes[source], ref, 1)
		operations = append(operations, ref)
	}
	return operations
}

func (a *reachingAnalysis) parent(stack []ast.Node, child ast.Node) (pdg.Ref, pdg.Position) {
	for i := len(stack) - 1; i >= 0; i-- {
		if ref := a.builder.nodes[stack[i]]; ref != 0 {
			return ref, a.builder.runtimeOperandPosition(stack[i], child)
		}
		child = stack[i]
	}
	return 0, 0
}

func (a *reachingAnalysis) valueEdge(source, target pdg.Ref, position pdg.Position) {
	if source == 0 || target == 0 || source == target {
		return
	}
	b := a.builder
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{
		ID:     a.edgeID(),
		Source: source, Target: target, Kind: pdg.Data,
		Position: position,
		Subkind:  b.pool.Text(b.valueSubkind()), Evidence: a.evidence, Provenance: a.origin,
	})
}

func (a *reachingAnalysis) operation(node pdg.Ref, state reachingState) {
	attributes := a.builder.attributes[a.builder.graph.Nodes[node-1].Attributes-1]
	for _, read := range attributes.Reads {
		key := a.key(read.Symbol, read.Location)
		definitions := state[key]
		if len(definitions) == 0 {
			continue
		}
		definition := a.joinedDefinition(key, node, definitions)
		state[key] = []pdg.Ref{definition}
		a.edge(definition, node, read.Position)
	}
	for _, ref := range attributes.Defines {
		a.writeDefinition(ref, state)
	}
}

func (a *reachingAnalysis) edge(ref, target pdg.Ref, position pdg.Position) {
	b := a.builder
	definition := b.graph.Definitions[ref-1]
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{
		ID:     a.edgeID(),
		Source: definition.Node, Target: target, Kind: pdg.Data,
		Position: position,
		Subkind:  b.pool.Text("reaching-definition"), Symbol: definition.Symbol,
		Definition: ref, Location: definition.Location, Evidence: a.evidence, Provenance: a.origin,
	})
}

func (a *reachingAnalysis) sourceValues(flow *cfg.CFG, dom *executionDominators) {
	for _, index := range dom.order {
		for _, source := range flow.Blocks[index].Nodes {
			a.operations(source)
		}
	}
}

func (a *reachingAnalysis) acyclic(flow *cfg.CFG, dom *executionDominators) {
	a.remaining = reachingConsumers(flow)

	for _, index := range dom.order {
		state := a.input(index)
		a.block(flow.Blocks[index], state)
		if a.remaining[index] > 0 {
			a.outputs[index] = state
		}
	}
}

func (a *reachingAnalysis) sourceReachingValues(flow *cfg.CFG, dom *executionDominators) {
	if a.builder.generalBindings || a.builder.runtimeOperations {
		return
	}
	a.sourceValues(flow, dom)
}

func (a *reachingAnalysis) initialDefinition(definition pdg.Definition, category pdg.Category) bool {
	if category == pdg.FormalInput || category == pdg.FormalOutput {
		return true
	}
	attrs := a.builder.attributes[a.builder.graph.Nodes[definition.Node-1].Attributes-1]
	return category == pdg.MemorySummary && a.builder.pool.Tables.Text(attrs.Construct) == "memory-input"
}

func (a *reachingAnalysis) key(symbol, location pdg.Ref) pdg.Ref {
	if symbol != 0 {
		return symbol
	}
	if location == 0 {
		return 0
	}
	return pdg.Ref(len(a.builder.graph.Symbols)) + location
}

func (a *reachingAnalysis) writeDefinition(ref pdg.Ref, state reachingState) {
	definition := a.builder.graph.Definitions[ref-1]
	key := a.key(definition.Symbol, definition.Location)
	if a.builder.weakDefinitions[ref] {
		state.merge(reachingState{key: []pdg.Ref{ref}})
		return
	}
	state[key] = []pdg.Ref{ref}
}

const scalarDefinitionDescription = "Exact scalar reaching definitions for acyclic normal flow: local integer/boolean bindings, assignments, if branches, short-circuit expressions, explicit value joins and returns; calls, memory and loops are excluded."

func (b *irBuilder) reachingDefinitions(body *ast.BlockStmt, flow *cfg.CFG, dom *executionDominators) (*cfg.CFG, *executionDominators, error) {
	if !b.reachingEnabled() {
		return flow, dom, nil
	}
	nextEdgeID := len(b.graph.Edges)
	b.removeFirstBindingEdges()

	a := reachingAnalysis{builder: b, controls: b.executionControls(body), nextEdgeID: nextEdgeID, evidence: b.pool.Evidence(b.bindingStatus(), b.bindingDescription()), origin: b.pool.Origin(pdg.Analysis, b.bindingDescription())}
	a.sourceReachingValues(flow, dom)
	var err error
	flow, err = b.reachingCFG(body, flow)
	if err != nil {
		return nil, nil, err
	}
	dom = newExecutionDominators(flow)
	if b.generalBindings {
		if err := b.generalMemory(); err != nil {
			return nil, nil, err
		}
	}
	a.outputs = make([]reachingState, len(flow.Blocks))
	a.predecessors = dom.predecessors
	if b.fixedPointBindings() {
		a.fixedPoint(flow, dom)
		b.normalReturnTransfers(flow, dom)
		return flow, dom, nil
	}
	a.acyclic(flow, dom)
	b.scalarDefinitions = b.scalarOperations
	return flow, dom, nil
}

// Source scalar flow replaces the old abstraction's first-binding data edges.
// Existing IDs stay stable. New occurrences use the original high-water mark.
func (b *irBuilder) removeFirstBindingEdges() {
	b.graph.Edges = slices.DeleteFunc(b.graph.Edges, func(edge pdg.Edge) bool {
		return edge.Kind == pdg.Data && b.pool.Tables.Text(edge.Subkind) == "mining-value"
	})
}

func (b *irBuilder) bindingDescription() string {
	if b.generalBindings {
		return generalBindingDescription
	}
	if b.extendedScalar {
		return cyclicDefinitionDescription
	}
	if b.bindingOperations {
		return interfaceFlowDescription
	}
	return scalarDefinitionDescription
}

func (b *irBuilder) valueSubkind() string {
	if b.bindingOperations {
		return "binding-value"
	}
	return "scalar-value"
}

func (b *irBuilder) reachingCFG(body *ast.BlockStmt, flow *cfg.CFG) (*cfg.CFG, error) {
	if !b.fixedPointBindings() && !b.bindingOperations {
		return b.scalarCFG(body, flow), nil
	}
	return b.semanticControlCFG(body, flow)
}

func (b *irBuilder) reachingEnabled() bool {
	return b.scalarOperations || b.bindingOperations || b.generalBindings
}

func (b *irBuilder) fixedPointBindings() bool { return b.extendedScalar || b.generalBindings }

func (b *irBuilder) bindingStatus() pdg.Status {
	if b.generalBindings {
		return pdg.Approximated
	}
	return pdg.Supported
}

func reachingConsumers(flow *cfg.CFG) []int {
	consumers := make([]int, len(flow.Blocks))
	for i, block := range flow.Blocks {
		consumers[i] = len(block.Succs)
	}
	return consumers
}
