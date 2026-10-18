package bindingflow

import (
	"encoding/json"
	"fmt"

	"github.com/rstyled/gstyle/domain/pdg"
)

type Representation uint8

const (
	UnchangedRepresentation Representation = 0
	BoxedRepresentation     Representation = 1
	UnknownRepresentation   Representation = 2
)

type representationInput struct {
	Position pdg.Position `json:"position"`
	Kind     string       `json:"kind"`
}

func (q *Query) representationFact(ref pdg.Ref, extension pdg.Extension) error {
	var inputs []representationInput
	if err := json.Unmarshal([]byte(q.graph.Tables.Text(extension.Value)), &inputs); err != nil {
		return fmt.Errorf("decode input representation: %w", err)
	}
	if inputs == nil || q.representations[ref] != nil {
		return fmt.Errorf("missing or duplicate input representation inventory")
	}
	representations := map[pdg.Position]Representation{}
	for _, input := range inputs {
		if err := addRepresentation(representations, input); err != nil {
			return err
		}
	}
	q.representations[ref] = representations
	return nil
}

func (q *Query) inputRepresentation(owner pdg.Ref, position pdg.Position) Representation {
	if kind, found := q.representations[owner][position]; found {
		return kind
	}
	return UnknownRepresentation
}

func addRepresentation(representations map[pdg.Position]Representation, input representationInput) error {
	if input.Position == 0 {
		return fmt.Errorf("input representation position is missing")
	}
	if _, found := representations[input.Position]; found {
		return fmt.Errorf("duplicate input representation position")
	}
	kind, known := representationCode(input.Kind)
	if !known {
		return fmt.Errorf("unknown input representation kind")
	}
	representations[input.Position] = kind
	return nil
}

func representationCode(kind string) (Representation, bool) {
	switch kind {
	case "identity":
		return UnchangedRepresentation, true
	case "boxed":
		return BoxedRepresentation, true
	case "unknown":
		return UnknownRepresentation, true
	default:
		return UnknownRepresentation, false
	}
}

func originRepresentation(step originStep, vertex originVertex) originStep {
	source := step.source
	source.Representation |= vertex.representation
	return originStep{inputs: representOrigins(step.inputs, vertex.representation), source: source, known: step.known}
}

func representOrigins(inputs []originVertex, representation Representation) []originVertex {
	represented := make([]originVertex, len(inputs))
	for i, input := range inputs {
		represented[i] = originVertex{ref: input.ref, result: input.result, definition: input.definition, representation: input.representation | representation}
	}
	return represented
}
