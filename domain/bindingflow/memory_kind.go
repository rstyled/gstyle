package bindingflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
)

// ImmutableValue and ImmutableSymbol require one compiler-owned memory-kind
// fact. Missing or conflicting facts do not establish an immutable value.
func (q *Query) ImmutableValue(ref pdg.Ref) bool {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Nodes) {
		return false
	}
	return immutableMemoryKind(q.graph.Tables, q.graph.NodeAttributes(q.graph.Nodes[ref-1]).Extensions)
}
func (q *Query) ImmutableSymbol(ref pdg.Ref) bool {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Symbols) {
		return false
	}
	return immutableMemoryKind(q.graph.Tables, q.graph.Symbols[ref-1].Extensions)
}
func immutableMemoryKind(tables *pdg.Tables, extensions []pdg.Extension) bool {
	count := 0
	kind := ""
	for _, extension := range extensions {
		if tables.Text(extension.Name) != "gstyle.memory:kind" {
			continue
		}
		count++
		if json.Unmarshal([]byte(tables.Text(extension.Value)), &kind) != nil {
			return false
		}
	}
	return count == 1 && kind == "immutable"
}
