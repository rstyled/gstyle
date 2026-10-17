package resourceflow

import "github.com/rstyled/gstyle/domain/pdg"

func (q *Query) invalidates(ref pdg.Ref, policy Policy) bool {
	node := q.graph.Nodes[ref-1]
	attrs := q.graph.NodeAttributes(node)
	if q.addressesTracked(ref, attrs, policy) {
		return true
	}
	for _, write := range attrs.Writes {
		if write.Symbol != 0 && write.Symbol == policy.AcquisitionError {
			return true
		}
		if indirectStore(node.Category, write) {
			return true
		}
	}
	return q.capturesTracked(attrs, policy)
}

func (q *Query) addressesTracked(ref pdg.Ref, attrs pdg.Attributes, policy Policy) bool {
	if attrs.Operator.Status != pdg.Known || q.graph.Tables.Text(attrs.Operator.Value) != "&" {
		return false
	}
	source := q.paths.Operand(ref, 1)
	return source == 0 || ownedSymbol(q.symbol(source), policy)
}

func (q *Query) capturesTracked(attrs pdg.Attributes, policy Policy) bool {
	if q.graph.Tables.Text(attrs.Construct) != "closure" {
		return false
	}
	for _, symbol := range attrs.Symbols {
		if ownedSymbol(symbol, policy) {
			return true
		}
	}
	return false
}

func indirectStore(category pdg.Category, write pdg.Access) bool {
	if category != pdg.Assignment && category != pdg.Declaration {
		return false
	}
	return write.Location != 0 || write.Symbol == 0 || write.UnresolvedReason != 0
}
