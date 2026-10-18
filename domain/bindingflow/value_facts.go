package bindingflow

import (
	"encoding/json"
	"fmt"
	"go/constant"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

// ValueFacts reads compiler-owned values without source syntax or dependencies.
func (q *Query) ValueFacts() (Facts, error) {
	facts, err := q.symbolValueFacts()
	if err != nil {
		return Facts{}, err
	}
	for i, node := range q.graph.Nodes {
		fact, found, err := q.nodeValueFact(q.graph.NodeAttributes(node))
		if err != nil {
			return Facts{}, fmt.Errorf("node %d value fact: %w", i+1, err)
		}
		if found {
			facts.Nodes[pdg.Ref(i+1)] = fact
		}
	}
	return facts, nil
}

func (q *Query) symbolValueFacts() (Facts, error) {
	if err := q.validateParameterTypes(); err != nil {
		return Facts{}, err
	}
	booleans, err := q.booleanSymbols()
	if err != nil {
		return Facts{}, err
	}
	captures, err := q.capturedSymbols()
	if err != nil {
		return Facts{}, err
	}
	single, err := q.singleCellSymbols()
	if err != nil {
		return Facts{}, err
	}
	return Facts{Nodes: map[pdg.Ref]Fact{}, BooleanSymbols: booleans, CapturedSymbols: captures, SingleCellSymbols: single}, nil
}

func (q *Query) nodeValueFact(attrs pdg.Attributes) (Fact, bool, error) {
	scalar, found, err := q.scalarNodeValueFact(attrs)
	if err != nil {
		return Fact{}, false, err
	}
	callable, known, err := q.callableValueFact(attrs)
	if err != nil {
		return Fact{}, false, err
	}
	if !known {
		return scalar, found, nil
	}
	if found && !scalar.Value.NonNil {
		return Fact{}, false, fmt.Errorf("conflicting callable value fact")
	}
	return callable, true, nil
}

func (q *Query) scalarNodeValueFact(attrs pdg.Attributes) (Fact, bool, error) {
	var fact Fact
	found := false
	for _, extension := range attrs.Extensions {
		if q.graph.Tables.Text(extension.Name) != "gstyle.go:value-fact" {
			continue
		}
		if found {
			return Fact{}, false, fmt.Errorf("duplicate record")
		}
		parsed, err := decodeValueFact(q.graph.Tables.Text(extension.Value))
		if err != nil {
			return Fact{}, false, err
		}
		fact, found = parsed, true
	}
	if found {
		return fact, true, nil
	}
	return q.legacyValueFact(attrs)
}

func (q *Query) legacyValueFact(attrs pdg.Attributes) (Fact, bool, error) {
	var fact Fact
	found := false
	for _, extension := range attrs.Extensions {
		record := q.legacyValueRecord(extension)
		if record == "" {
			continue
		}
		next, err := decodeValueFact(record)
		if err != nil {
			return Fact{}, false, err
		}
		if found && !sameLeafValue(fact.Value, next.Value) {
			return Fact{}, false, fmt.Errorf("conflicting legacy facts")
		}
		fact, found = next, true
	}
	return fact, found, nil
}

func (q *Query) legacyValueRecord(extension pdg.Extension) string {
	name, value := q.graph.Tables.Text(extension.Name), q.graph.Tables.Text(extension.Value)
	switch name {
	case "gstyle.runtime:nil":
		return `{"nil":` + value + `}`
	case "gstyle.runtime:boolean":
		return `{"boolean":` + value + `}`
	default:
		return ""
	}
}

func sameLeafValue(left, right Value) bool {
	if left.Nil || right.Nil {
		return left.Nil == right.Nil
	}
	l, lk := left.Bool()
	r, rk := right.Bool()
	return lk && rk && l == r
}

func decodeValueFact(raw string) (Fact, error) {
	if !json.Valid([]byte(raw)) {
		return Fact{}, fmt.Errorf("invalid JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	token, _ := decoder.Token()
	if token != json.Delim('{') || !decoder.More() {
		return Fact{}, fmt.Errorf("require one field")
	}
	key, _ := decoder.Token()
	value, _ := decoder.Token()
	if decoder.More() {
		return Fact{}, fmt.Errorf("require one field")
	}
	return valueFactField(key, value)
}

func valueFactField(key, value any) (Fact, error) {
	switch key {
	case "nil", "non_nil":
		return nilnessFact(key, value)
	case "boolean":
		if flag, ok := value.(bool); ok {
			return Fact{Known: true, Value: Value{Scalar: constant.MakeBool(flag)}}, nil
		}
	case "conversion":
		return conversionFact(value)
	}
	return Fact{}, fmt.Errorf("invalid value fact field %v", key)
}

func conversionFact(value any) (Fact, error) {
	switch value {
	case "identity":
		return Fact{Conversion: IdentityConversion}, nil
	case "nil":
		return Fact{Conversion: NilConversion}, nil
	case "boxed":
		return Fact{Conversion: BoxedConversion}, nil
	case "nilness":
		return Fact{Conversion: NilnessConversion}, nil
	case "unknown":
		return Fact{}, nil
	default:
		return Fact{}, fmt.Errorf("invalid conversion")
	}
}

func nilnessFact(key, value any) (Fact, error) {
	if value != true {
		return Fact{}, fmt.Errorf("nilness fact requires true")
	}
	return Fact{Known: true, Value: Value{Nil: key == "nil", NonNil: key == "non_nil"}}, nil
}
