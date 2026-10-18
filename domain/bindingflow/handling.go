package bindingflow

import "github.com/rstyled/gstyle/domain/pdg"

type Dependency struct{ Used, Known bool }

type dependencySearch struct {
	boundaries map[pdg.Ref]bool
	query      *Query
	path       Path
	context    Evaluation
	pending    []dependencyItem
	seen       map[pdg.Ref]uint8
	unknown    bool
}

func (search *dependencySearch) run() Dependency {
	for len(search.pending) > 0 {
		current := search.pending[len(search.pending)-1]
		search.pending = search.pending[:len(search.pending)-1]
		used, known := search.step(current)
		search.unknown = search.unknown || !known
		if used {
			return Dependency{Used: true, Known: true}
		}
	}
	return Dependency{Known: !search.unknown}
}

func (search *dependencySearch) step(current dependencyItem) (bool, bool) {
	if current.exit {
		search.seen[current.ref] = 2
		return false, true
	}
	switch search.seen[current.ref] {
	case 1:
		return false, false
	case 2:
		return false, true
	}
	search.seen[current.ref] = 1
	if search.query.Path(current.ref, search.context.Facts).Same(search.path) {
		return true, true
	}
	if search.boundaries[current.ref] {
		search.seen[current.ref] = 2
		return false, true
	}
	inputs, known := search.query.dependencyInputs(current.ref, search.context)
	search.pushInputs(current.ref, inputs)
	return false, known
}

func (search *dependencySearch) pushInputs(ref pdg.Ref, inputs []pdg.Ref) {
	items := make([]dependencyItem, len(inputs)+1)
	items[0] = dependencyItem{ref: ref, exit: true}
	for i, input := range inputs {
		items[i+1] = dependencyItem{ref: input}
	}
	search.pending = append(search.pending, items...)
}

type dependencyItem struct {
	ref  pdg.Ref
	exit bool
}

// Depends follows value operands, not reaching-definition or lexical edges.
// Missing inventories/inputs cannot establish an absence of handling.
func (q *Query) Depends(ref pdg.Ref, path Path, context Evaluation) Dependency {
	return q.DependsUntil(ref, path, context, nil)
}

// DependsUntil stops below caller-supplied semantic boundary nodes. A matching
// path at a boundary itself still counts as used. The boundary map is borrowed.
func (q *Query) DependsUntil(ref pdg.Ref, path Path, context Evaluation, boundaries map[pdg.Ref]bool) Dependency {
	if path.Root == 0 {
		return Dependency{Known: true}
	}
	if !q.validRef(ref) {
		return Dependency{}
	}
	search := dependencySearch{boundaries: boundaries, query: q, path: path, context: context, pending: []dependencyItem{{ref: ref}}, seen: map[pdg.Ref]uint8{}}
	return search.run()
}

func (q *Query) HandlingUnknown(ref pdg.Ref, checked Path, context Evaluation) bool {
	if !q.validRef(ref) {
		return true
	}
	node := q.graph.Nodes[ref-1]
	attrs := q.graph.NodeAttributes(node)
	if node.Category == pdg.Call {
		return !plainPrint(q.graph, attrs)
	}
	if q.uncertainWrites(ref, attrs.Writes, checked) {
		return true
	}
	if !copyOperation(q.graph, node, attrs) {
		return false
	}
	return q.copyHandlingUnknown(ref, checked, context)
}

// StablePath is a conservative admission rule while indirect effects remain
// weak: address escapes and copies of a receiver prefix can obscure identity.
// This filter is not a no-alias or concurrency proof.
func (q *Query) StablePath(path Path, facts Facts) bool {
	if !q.admittedPath(path) {
		return false
	}
	for i := range q.graph.Nodes {
		if !q.reached[i] {
			continue
		}
		if q.unstableOperation(pdg.Ref(i+1), path, facts) {
			return false
		}
	}
	return true
}

func (q *Query) validRef(ref pdg.Ref) bool {
	return q != nil && ref != 0 && int(ref) <= len(q.graph.Nodes)
}

func (q *Query) dependencyInputs(ref pdg.Ref, context Evaluation) ([]pdg.Ref, bool) {
	if ref == 0 || int(ref) > len(q.graph.Nodes) {
		return nil, false
	}
	// Whole-value non-nilness does not erase the value's payload inputs.
	if fact := context.Facts.Nodes[ref]; fact.Known && !fact.Value.NonNil {
		return nil, true
	}
	positions, known := q.required[ref]
	if !known {
		return nil, q.Path(ref, context.Facts).Root != 0
	}
	return q.requiredInputs(ref, positions, context)
}

func (q *Query) requiredInputs(ref pdg.Ref, positions []pdg.Position, context Evaluation) ([]pdg.Ref, bool) {
	var inputs []pdg.Ref
	for _, position := range positions {
		if q.skippedLogicalInput(ref, position, context) {
			continue
		}
		input := q.Operand(ref, position)
		if input == 0 {
			return inputs, false
		}
		inputs = append(inputs, input)
	}
	return inputs, true
}

func (q *Query) skippedLogicalInput(ref pdg.Ref, position pdg.Position, context Evaluation) bool {
	if position != 2 {
		return false
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	operator := q.operator(attrs)
	if operator != "&&" && operator != "||" {
		return false
	}
	truth, known := q.Evaluate(q.Operand(ref, 1), context).Bool()
	_, skip := logicalShortCircuit(operator, truth, known)
	return skip
}

func (q *Query) copyHandlingUnknown(ref pdg.Ref, checked Path, context Evaluation) bool {
	for _, write := range q.Writes(ref) {
		if write.Zero {
			continue
		}
		dependency := q.Depends(write.Source, checked, context)
		if write.Source == 0 || dependency.Used || !dependency.Known {
			return true
		}
	}
	return false
}

func (q *Query) uncertainWrites(ref pdg.Ref, writes []pdg.Access, checked Path) bool {
	for _, write := range writes {
		if q.freshStorageWrite(ref, write) {
			continue
		}
		if !directBinding(write) || write.UnresolvedReason != 0 {
			return true
		}
		if write.Symbol == checked.Root {
			return true
		}
	}
	return false
}

func (q *Query) addressesRoot(ref pdg.Ref, attrs pdg.Attributes, path Path, facts Facts) bool {
	if q.operator(attrs) != "&" {
		return false
	}
	source := q.Operand(ref, 1)
	if source == 0 {
		return true
	}
	return q.Path(source, facts).Root == path.Root
}

func (q *Query) aliasesReceiver(ref pdg.Ref, attrs pdg.Attributes, path Path, facts Facts) bool {
	if len(path.Fields) == 0 {
		return false
	}
	for _, write := range attrs.Writes {
		source := q.Path(q.Operand(ref, write.Position), facts)
		if len(source.Fields) < len(path.Fields) && source.PrefixOf(path) {
			return true
		}
	}
	return false
}

func (q *Query) admittedPath(path Path) bool { return q != nil && path.Root != 0 && q.reached != nil }

func (q *Query) unstableOperation(ref pdg.Ref, path Path, facts Facts) bool {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	return q.addressesRoot(ref, attrs, path, facts) || q.aliasesReceiver(ref, attrs, path, facts)
}

func (q *Query) freshStorageWrite(ref pdg.Ref, access pdg.Access) bool {
	node := q.graph.Nodes[ref-1]
	return q.ownsAllocation(ref, access) && node.Category == pdg.Computation && q.operator(q.graph.NodeAttributes(node)) == "composite"
}

func (q *Query) ownsAllocation(ref pdg.Ref, access pdg.Access) bool {
	return access.Symbol == 0 && access.UnresolvedReason == 0 && access.Location != 0 && q.allocations[access.Location] == ref
}

func plainPrint(graph *pdg.Graph, attrs pdg.Attributes) bool {
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return false
	}
	name := graph.Tables.Text(attrs.Callee.Value)
	return name == "print" || name == "println"
}
