package bindingflow

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type Path struct {
	Root   pdg.Ref
	Fields []uint32
}

func (path Path) Key() PathKey { return PathKey{Root: path.Root, Fields: fmt.Sprint(path.Fields)} }

func (path Path) Same(other Path) bool {
	return path.Root != 0 && path.Root == other.Root && slices.Equal(path.Fields, other.Fields)
}

func (path Path) PrefixOf(other Path) bool {
	if path.Root == 0 || path.Root != other.Root || len(path.Fields) > len(other.Fields) {
		return false
	}
	return slices.Equal(path.Fields, other.Fields[:len(path.Fields)])
}

type PathKey struct {
	Root   pdg.Ref
	Fields string
}

// Path composes compiler field indexes through actual value operands. It does
// not equate different bindings merely because their pointers may alias.
func (q *Query) Path(ref pdg.Ref, facts Facts) Path {
	if q == nil {
		return Path{}
	}
	var groups [][]uint32
	for range q.graph.Nodes {
		ref = q.IdentitySource(ref, facts)
		if ref == 0 {
			return Path{}
		}
		if fields, field := q.fields[ref]; field {
			groups = append(groups, fields)
			ref = q.Operand(ref, 1)
			continue
		}
		symbol := q.pathSymbol(ref)
		if symbol == 0 {
			return Path{}
		}
		return Path{Root: symbol, Fields: joinFieldPaths(groups)}
	}
	return Path{}
}

func (q *Query) pathSymbol(ref pdg.Ref) pdg.Ref {
	node := q.graph.Nodes[ref-1]
	attrs := q.graph.NodeAttributes(node)
	if len(attrs.Reads) == 1 && directBinding(attrs.Reads[0]) {
		return attrs.Reads[0].Symbol
	}
	if !pathBindingNode(node.Category) {
		return 0
	}
	// Address-only identifiers retain a symbol, not a fabricated value read.
	if len(attrs.Reads) == 0 && len(attrs.Symbols) == 1 {
		return attrs.Symbols[0]
	}
	return 0
}

func (q *Query) indexDependencyFacts() error {
	for i, node := range q.graph.Nodes {
		attrs := q.graph.NodeAttributes(node)
		for _, extension := range attrs.Extensions {
			if err := q.dependencyFact(pdg.Ref(i+1), extension); err != nil {
				return err
			}
		}
	}
	return nil
}

func (q *Query) dependencyFact(ref pdg.Ref, extension pdg.Extension) error {
	name := q.graph.Tables.Text(extension.Name)
	switch name {
	case "gstyle.go:call-shape":
		return q.callShapeFact(ref, extension)
	case "gstyle.go:tuple-arity":
		return q.tupleArityFact(ref, extension)
	case "gstyle.go:input-representation":
		return q.representationFact(ref, extension)
	case "gstyle.go:field-path":
		return q.fieldFact(ref, extension)
	case "gstyle.go:value-operands":
		return q.operandFact(ref, extension)
	}
	return nil
}

func (q *Query) fieldFact(ref pdg.Ref, extension pdg.Extension) error {
	fields, err := numericFact(q.graph.Tables.Text(extension.Value))
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return fmt.Errorf("field path must be nonempty")
	}
	q.fields[ref] = fields
	return nil
}

func (q *Query) operandFact(ref pdg.Ref, extension pdg.Extension) error {
	values, err := numericFact(q.graph.Tables.Text(extension.Value))
	if err != nil {
		return err
	}
	positions, err := operandPositions(values)
	if err != nil {
		return err
	}
	q.required[ref] = positions
	return nil
}

func joinFieldPaths(groups [][]uint32) []uint32 {
	var result []uint32
	for i := len(groups) - 1; i >= 0; i-- {
		result = append(result, groups[i]...)
	}
	return result
}

func numericFact(text string) ([]uint32, error) {
	var values []uint32
	if err := json.Unmarshal([]byte(text), &values); err != nil {
		return nil, fmt.Errorf("invalid dependency fact: %w", err)
	}
	if values == nil {
		return nil, fmt.Errorf("dependency fact must be an array")
	}
	return values, nil
}

func operandPositions(values []uint32) ([]pdg.Position, error) {
	var positions []pdg.Position
	seen := map[uint32]bool{}
	for _, value := range values {
		if value == 0 || seen[value] {
			return nil, fmt.Errorf("operand position must be distinct and nonzero")
		}
		seen[value] = true
		positions = append(positions, pdg.Position(value))
	}
	return positions, nil
}

func pathBindingNode(category pdg.Category) bool {
	return category == pdg.Read || category == pdg.FormalInput
}
