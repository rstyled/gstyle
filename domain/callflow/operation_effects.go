package callflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// OperationEffects reads per-operation effects. Operand evaluation, call bodies
// and deferred invocation are separate canonical operations and queries.
type OperationEffects struct {
	guards            map[pdg.Ref][]flowGuard
	orders            map[pdg.Ref]uint32
	graph             *pdg.Graph
	paths             *bindingflow.Query
	regions           map[pdg.Ref]uint32
	members           map[uint32][]pdg.Ref
	incompleteRegions bool
	effects           map[pdg.Ref]operationEffect
}

func NewOperationEffects(graph *pdg.Graph) (*OperationEffects, error) {
	if graph == nil {
		return nil, nil
	}
	paths, err := bindingflow.New(graph)
	if err != nil {
		return nil, err
	}
	return newOperationEffects(graph, paths)
}

// The validated binding query is immutable after construction. Exit evaluation
// shares its indexes with call and operation queries instead of rebuilding them.
func newOperationEffects(graph *pdg.Graph, paths *bindingflow.Query) (*OperationEffects, error) {
	operations := &OperationEffects{guards: map[pdg.Ref][]flowGuard{}, orders: map[pdg.Ref]uint32{}, graph: graph, paths: paths, regions: map[pdg.Ref]uint32{}, members: map[uint32][]pdg.Ref{}, effects: map[pdg.Ref]operationEffect{}}
	for i, node := range graph.Nodes {
		if err := operations.index(pdg.Ref(i+1), graph.NodeAttributes(node).Extensions); err != nil {
			return nil, err
		}
	}
	operations.indexRegionMembers()
	operations.indexFlowGuards()
	return operations, nil
}

// Outcomes includes possible normal completion. Unknown never establishes that
// later cleanup runs; blocking retains a possible nontermination outcome.
func (operations *OperationEffects) Outcomes(ref pdg.Ref) Outcomes {
	if operations == nil || ref == 0 || int(ref) > len(operations.graph.Nodes) {
		return UnknownExit
	}
	if operations.graph.Nodes[ref-1].Category == pdg.Call {
		return UnknownExit
	}
	return effectOutcomes(operations.effect(ref))
}

// OutcomesWithNonNilInput requires proof that canonical operand one is non-nil.
// It discharges only the nil-input fault, never any other possible panic.
func (operations *OperationEffects) OutcomesWithNonNilInput(ref pdg.Ref) Outcomes {
	if operations == nil || ref == 0 || int(ref) > len(operations.graph.Nodes) {
		return UnknownExit
	}
	if operations.graph.Nodes[ref-1].Category == pdg.Call {
		return UnknownExit
	}
	effect := operations.effect(ref)
	effect.NilInput = false
	return effectOutcomes(effect)
}

func (operations *OperationEffects) index(ref pdg.Ref, extensions []pdg.Extension) error {
	for _, extension := range extensions {
		if err := operations.extension(ref, extension); err != nil {
			return err
		}
	}
	return nil
}

func (operations *OperationEffects) extension(ref pdg.Ref, extension pdg.Extension) error {
	value := operations.graph.Tables.Text(extension.Value)
	switch operations.graph.Tables.Text(extension.Name) {
	case "gstyle.go:operation-effects":
		return operations.indexEffect(ref, value)
	case "gstyle.go:invocation-order":
		return operations.indexInvocationOrder(ref, value)
	case "gstyle.go:evaluation-region":
		return operations.indexRegion(ref, value)
	default:
		return nil
	}
}

func (operations *OperationEffects) indexEffect(ref pdg.Ref, value string) error {
	if _, duplicate := operations.effects[ref]; duplicate {
		return fmt.Errorf("duplicate operation effects")
	}
	effect, err := decodeOperationEffect(value)
	if err != nil {
		return err
	}
	if err := operations.validEffect(ref, effect); err != nil {
		return err
	}
	operations.effects[ref] = effect
	return nil
}

func (operations *OperationEffects) effect(ref pdg.Ref) operationEffect {
	if effect, known := operations.effects[ref]; known {
		return effect
	}
	node := operations.graph.Nodes[ref-1]
	if node.Category == pdg.Return || node.Category == pdg.Merge {
		return operationEffect{}
	}
	attrs := operations.graph.NodeAttributes(node)
	if directReadOperation(node.Category, attrs) {
		return operationEffect{}
	}
	if knownControlOperation(operations.graph.Tables.Text(attrs.Construct)) {
		return operationEffect{}
	}
	return operationEffect{Unknown: true}
}

func (operations *OperationEffects) validEffect(ref pdg.Ref, effect operationEffect) error {
	if effect.NilInput && operations.paths.Operand(ref, 1) == 0 {
		return fmt.Errorf("nil-input fault has no canonical operand")
	}
	return nil
}

func effectOutcomes(effect operationEffect) Outcomes {
	if effect.Unknown {
		return UnknownExit
	}
	result := Returns
	if effect.Panic || effect.NilInput {
		result |= Panics
	}
	if effect.Block {
		result |= Diverges
	}
	return result
}

func directReadOperation(category pdg.Category, attrs pdg.Attributes) bool {
	if category != pdg.Read {
		return false
	}
	if len(attrs.Reads) == 0 {
		return false
	}
	for _, read := range attrs.Reads {
		if !directOperationAccess(read) {
			return false
		}
	}
	return true
}

func knownControlOperation(construct string) bool {
	switch construct {
	case "if", "for", "switch", "type-switch", "short-circuit", "defer-register", "implicit-return", "goto", "break", "continue", "fallthrough":
		return true
	default:
		return false
	}
}
