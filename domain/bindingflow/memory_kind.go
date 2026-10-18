package bindingflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
)

// ImmutableValue and ImmutableSymbol require one compiler-owned memory-kind
// fact. Missing or conflicting facts do not establish an immutable value.
func (q *Query) ImmutableValue(ref pdg.Ref) bool { return q.MemoryKind(ref) == "immutable" }

// MemoryKind requires one exact compiler-owned kind record. Unknown, malformed
// and conflicting records return an empty kind.
func (q *Query) MemoryKind(ref pdg.Ref) string {
	if q == nil || q.graph == nil || ref == 0 || int(ref) > len(q.graph.Nodes) {
		return ""
	}
	return sourceMemoryKind(q.graph.Tables, q.graph.NodeAttributes(q.graph.Nodes[ref-1]).Extensions)
}
func (q *Query) ImmutableSymbol(ref pdg.Ref) bool {
	if q == nil || q.graph == nil || ref == 0 || int(ref) > len(q.graph.Symbols) {
		return false
	}
	return immutableMemoryKind(q.graph.Tables, q.graph.Symbols[ref-1].Extensions)
}
func immutableMemoryKind(tables *pdg.Tables, extensions []pdg.Extension) bool {
	return sourceMemoryKind(tables, extensions) == "immutable"
}
func sourceMemoryKind(tables *pdg.Tables, extensions []pdg.Extension) string {
	count := 0
	kind := ""
	for _, extension := range extensions {
		if tables.Text(extension.Name) != "gstyle.memory:kind" {
			continue
		}
		count++
		if json.Unmarshal([]byte(tables.Text(extension.Value)), &kind) != nil {
			return ""
		}
	}
	if count != 1 {
		return ""
	}
	return kind
}
