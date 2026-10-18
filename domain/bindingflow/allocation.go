package bindingflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *Query) indexMetadata() error {
	if err := q.indexWeak(); err != nil {
		return err
	}
	if err := q.indexDependencyFacts(); err != nil {
		return err
	}
	return q.indexAllocations()
}
func (q *Query) indexAllocations() error {
	nodes := map[string]pdg.Ref{}
	for i, node := range q.graph.Nodes {
		nodes[q.graph.Tables.Text(node.ID)] = pdg.Ref(i + 1)
	}
	for i, location := range q.graph.Locations {
		for _, extension := range location.Extensions {
			if err := q.allocationFact(pdg.Ref(i+1), extension, nodes); err != nil {
				return err
			}
		}
	}
	return nil
}
func (q *Query) allocationFact(location pdg.Ref, extension pdg.Extension, nodes map[string]pdg.Ref) error {
	if q.graph.Tables.Text(extension.Name) != "gstyle.memory:allocation" {
		return nil
	}
	var id string
	if err := json.Unmarshal([]byte(q.graph.Tables.Text(extension.Value)), &id); err != nil {
		return fmt.Errorf("invalid allocation identity: %w", err)
	}
	owner := nodes[id]
	if owner == 0 {
		return fmt.Errorf("allocation owner %q is not a canonical node", id)
	}
	q.allocations[location] = owner
	return nil
}

// IsAllocation reports a compiler-recorded allocation owner. It does not prove
// that a site executes only once or that aliases cannot escape.
func (q *Query) IsAllocation(ref pdg.Ref) bool {
	if q == nil || ref == 0 {
		return false
	}
	for _, owner := range q.allocations {
		if owner == ref {
			return true
		}
	}
	return false
}
