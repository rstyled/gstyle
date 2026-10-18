package bindingflow

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *Query) indexWeak() error {
	definitions := map[string]pdg.Ref{}
	for i, definition := range q.graph.Definitions {
		definitions[q.graph.Tables.Text(definition.ID)] = pdg.Ref(i + 1)
	}
	for i, node := range q.graph.Nodes {
		attrs := q.graph.NodeAttributes(node)
		for _, extension := range attrs.Extensions {
			if q.graph.Tables.Text(extension.Name) != "gstyle.memory:weak-definitions" {
				continue
			}
			if err := q.weakFact(pdg.Ref(i+1), attrs, extension, definitions); err != nil {
				return err
			}
		}
	}
	return nil
}
func (q *Query) weakFact(owner pdg.Ref, attrs pdg.Attributes, extension pdg.Extension, definitions map[string]pdg.Ref) error {
	ids, err := decodeWeakFact(q.graph.Tables.Text(extension.Value))
	if err != nil {
		return err
	}
	for _, id := range ids {
		ref := definitions[id]
		if ref == 0 || !slices.Contains(attrs.Defines, ref) {
			return fmt.Errorf("weak definition %q is not owned by operation", id)
		}
		if q.graph.Definitions[ref-1].Node != owner {
			return fmt.Errorf("weak definition %q has another owner", id)
		}
		q.weak[ref] = true
	}
	return nil
}
func (q *Query) strongWrite(attrs pdg.Attributes, symbol pdg.Ref) bool {
	found := false
	for _, ref := range attrs.Defines {
		definition := q.graph.Definitions[ref-1]
		if !bindingDefinition(definition, symbol) {
			continue
		}
		if q.weak[ref] {
			return false
		}
		found = true
	}
	return found
}
func bindingDefinition(definition pdg.Definition, symbol pdg.Ref) bool {
	return definition.Symbol == symbol && definition.Location == 0
}

func decodeWeakFact(text string) ([]string, error) {
	var ids []string
	if err := json.Unmarshal([]byte(text), &ids); err != nil {
		return nil, fmt.Errorf("invalid weak definition fact: %w", err)
	}
	if ids == nil {
		return nil, fmt.Errorf("weak definition fact must be an array")
	}
	return ids, nil
}
