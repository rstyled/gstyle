package bindingflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

// CallShape describes receiving arity. Canonical operand edges still identify
// the evaluated expressions; Tuple selects the results of their sole producer.
type CallShape struct {
	Fixed    uint32 `json:"fixed"`
	Variadic bool   `json:"variadic"`
	Spread   bool   `json:"spread"`
	Tuple    uint32 `json:"tuple"`
}

// CallShapeAt returns the compiler receiving contract, without borrowing maps.
func (q *Query) CallShapeAt(ref pdg.Ref) (CallShape, bool) {
	if q == nil {
		return CallShape{}, false
	}
	shape, found := q.callShapes[ref]
	return shape, found
}

func (q *Query) callShapeFact(ref pdg.Ref, extension pdg.Extension) error {
	if q.graph.Nodes[ref-1].Category != pdg.Call {
		return fmt.Errorf("call shape requires a call")
	}
	if _, found := q.callShapes[ref]; found {
		return fmt.Errorf("duplicate call shape")
	}
	shape, err := decodeCallShape(q.graph.Tables.Text(extension.Value))
	if err != nil {
		return err
	}
	q.callShapes[ref] = shape
	return nil
}

func (q *Query) tupleArityFact(ref pdg.Ref, extension pdg.Extension) error {
	if _, found := q.tupleArities[ref]; found {
		return fmt.Errorf("duplicate tuple arity")
	}
	arity, err := decodeTupleArity(q.graph.Tables.Text(extension.Value))
	if err != nil {
		return err
	}
	if arity < 2 || !q.tupleValue(ref) {
		return fmt.Errorf("tuple arity requires a tuple value")
	}
	q.tupleArities[ref] = arity
	return nil
}

func decodeCallShape(raw string) (CallShape, error) {
	if !json.Valid([]byte(raw)) {
		return CallShape{}, fmt.Errorf("invalid call shape JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	token, _ := decoder.Token()
	if token != json.Delim('{') {
		return CallShape{}, fmt.Errorf("call shape requires an object")
	}
	fields := map[string]any{}
	for decoder.More() {
		if err := readShapeField(decoder, fields); err != nil {
			return CallShape{}, err
		}
	}
	return shapeFields(fields)
}

func readShapeField(decoder *json.Decoder, fields map[string]any) error {
	key, _ := decoder.Token()
	name := key.(string)
	if _, found := fields[name]; found {
		return fmt.Errorf("duplicate call shape field")
	}
	value, _ := decoder.Token()
	switch name {
	case "fixed", "tuple":
		number, err := shapeUint(value)
		if err != nil {
			return err
		}
		fields[name] = number
	case "variadic", "spread":
		flag, ok := value.(bool)
		if !ok {
			return fmt.Errorf("call shape flag requires Boolean")
		}
		fields[name] = flag
	default:
		return fmt.Errorf("unknown call shape field")
	}
	return nil
}

func shapeUint(value any) (uint32, error) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("call shape arity requires integer")
	}
	value64, err := strconv.ParseUint(string(number), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid call shape arity: %w", err)
	}
	return uint32(value64), nil
}

func shapeFields(fields map[string]any) (CallShape, error) {
	if len(fields) != 4 {
		return CallShape{}, fmt.Errorf("incomplete call shape")
	}
	shape := CallShape{Fixed: fields["fixed"].(uint32), Tuple: fields["tuple"].(uint32), Variadic: fields["variadic"].(bool), Spread: fields["spread"].(bool)}
	if shape.Tuple == 1 {
		return CallShape{}, fmt.Errorf("tuple requires multiple results")
	}
	if shape.Spread && (!shape.Variadic || shape.Tuple != 0) {
		return CallShape{}, fmt.Errorf("invalid spread call shape")
	}
	return shape, nil
}

func decodeTupleArity(raw string) (uint32, error) {
	if !json.Valid([]byte(raw)) {
		return 0, fmt.Errorf("invalid tuple arity JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	token, _ := decoder.Token()
	return shapeUint(token)
}
