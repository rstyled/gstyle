package callflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

type callBoundary struct {
	kind   string
	inputs []pdg.Position
}

func (boundary *callBoundary) fact(name, value string) error {
	switch name {
	case "gstyle.go:call-entry":
		return boundary.entry(value)
	case "gstyle.go:resource-inputs":
		return boundary.resources(value)
	default:
		return nil
	}
}

func (boundary *callBoundary) entry(value string) error {
	if boundary.kind != "" {
		return fmt.Errorf("duplicate call entry fact")
	}
	if err := json.Unmarshal([]byte(value), &boundary.kind); err != nil {
		return err
	}
	if !validCallEntry(boundary.kind) {
		return fmt.Errorf("invalid call entry kind")
	}
	return nil
}

func (boundary *callBoundary) resources(value string) error {
	if boundary.inputs != nil {
		return fmt.Errorf("duplicate resource input inventory")
	}
	var inputs []pdg.Position
	if err := json.Unmarshal([]byte(value), &inputs); err != nil {
		return err
	}
	if inputs == nil {
		return fmt.Errorf("missing resource input inventory")
	}
	if err := distinctResourceInputs(inputs); err != nil {
		return err
	}
	boundary.inputs = inputs
	return nil
}

// EntersFunction establishes a static, non-nil free-function invocation. Method
// wrappers, builtins and unresolved values need separate entry contracts.
func (calls *CallSites) EntersFunction(ref pdg.Ref) bool {
	if calls == nil || calls.entries[ref] != "function" {
		return false
	}
	targets := calls.targets[ref]
	return targets.known && len(targets.names) > 0
}

func (calls *CallSites) ResourceInputs(ref pdg.Ref) ([]pdg.Position, bool) {
	if calls == nil {
		return nil, false
	}
	inputs := calls.resourceInputs[ref]
	return append([]pdg.Position{}, inputs...), inputs != nil
}

func readCallBoundary(graph *pdg.Graph, extensions []pdg.Extension) (callBoundary, error) {
	var boundary callBoundary
	for _, extension := range extensions {
		if err := boundary.fact(graph.Tables.Text(extension.Name), graph.Tables.Text(extension.Value)); err != nil {
			return callBoundary{}, err
		}
	}
	return boundary, nil
}

func distinctResourceInputs(inputs []pdg.Position) error {
	seen := map[pdg.Position]bool{}
	for _, position := range inputs {
		if position == 0 || seen[position] {
			return fmt.Errorf("resource inputs must be positive and distinct")
		}
		seen[position] = true
	}
	return nil
}

func validCallEntry(entry string) bool {
	switch entry {
	case "function", "method", "builtin", "unknown":
		return true
	default:
		return false
	}
}
