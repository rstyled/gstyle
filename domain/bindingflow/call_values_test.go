package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func receivingModel(t *testing.T, fixed, count, mode, spreadValue int, variadic bool) (*Query, pdg.Ref, []pdg.Parameter, Evaluation) {
	t.Helper()
	expressions := count
	if mode == 1 {
		expressions = 1
	}
	operators := make([]string, expressions+1)
	for i := range operators {
		operators[i] = "leaf"
	}
	operators[expressions] = "call"
	q := valueQuery(t, operators...)
	owner := pdg.Ref(len(operators))
	q.callShapes = map[pdg.Ref]CallShape{owner: {Fixed: uint32(fixed), Variadic: variadic, Spread: mode == 2}}
	q.tupleArities = map[pdg.Ref]uint32{}
	q.required[owner] = []pdg.Position{}
	q.representations[owner] = map[pdg.Position]Representation{}
	facts := map[pdg.Ref]Value{}
	for i := range expressions {
		position := pdg.Position(i + 1)
		q.required[owner] = append(q.required[owner], position)
		valueEdge(q, pdg.Ref(i+1), owner, position)
		facts[pdg.Ref(i+1)] = boolValue(1 + i%2)
	}
	for i := range count {
		q.representations[owner][pdg.Position(i+1)] = UnchangedRepresentation
	}
	if mode == 1 {
		shape := q.callShapes[owner]
		shape.Tuple = uint32(count)
		q.callShapes[owner] = shape
		q.tupleArities[1] = uint32(count)
		attrs := q.graph.NodeAttributes(q.graph.Nodes[0])
		attrs.Extensions = []pdg.Extension{{Name: testText(q, "gstyle.memory:kind"), Value: testText(q, `"tuple"`)}}
		replaceValueAttrs(t, q, 1, pdg.Computation, attrs)
		facts = map[pdg.Ref]Value{}
		for i := range count {
			if i%2 == 1 {
				q.representations[owner][pdg.Position(i+1)] = BoxedRepresentation
			}
		}
	}
	if mode == 2 && expressions > 0 {
		switch spreadValue {
		case 0:
			facts[pdg.Ref(expressions)] = Value{Nil: true}
		case 1:
			facts[pdg.Ref(expressions)] = Value{NonNil: true}
		default:
			delete(facts, pdg.Ref(expressions))
		}
	}
	formalCount := fixed
	if variadic {
		formalCount++
	}
	inputs := make([]pdg.Parameter, formalCount)
	for i := range inputs {
		inputs[i] = pdg.Parameter{Position: uint32(i), Role: pdg.ParameterRole, Variadic: pdg.BooleanFact{Status: pdg.Known, Value: variadic && i == fixed}}
	}
	return q, owner, inputs, Evaluation{Facts: knownValues(facts)}
}
func TestCallReceivingValuesAgainstArityModel(t *testing.T) {
	observations := 0
	for fixed := 0; fixed <= 2; fixed++ {
		for _, variadic := range []bool{false, true} {
			for count := 0; count <= 4; count++ {
				for mode := 0; mode <= 2; mode++ {
					if mode == 1 && count < 2 {
						continue
					}
					if mode == 2 && !variadic {
						continue
					}
					variants := 1
					if mode == 2 {
						variants = 3
					}
					for variant := 0; variant < variants; variant++ {
						q, owner, inputs, context := receivingModel(t, fixed, count, mode, variant, variadic)
						values, known := q.CallValues(owner, inputs, context)
						want := count == fixed
						if variadic {
							want = count >= fixed
						}
						if mode == 2 {
							want = count == fixed+1
						}
						if known != want {
							t.Fatalf("fixed=%d count=%d mode=%d variadic=%v known=%v", fixed, count, mode, variadic, known)
						}
						observations++
						if !known {
							continue
						}
						for i := range fixed {
							if mode == 1 {
								if values[i].NonNil != (i%2 == 1) || values[i].Nil {
									t.Fatalf("tuple slot %d: %+v", i, values[i])
								}
							} else if flag, exact := values[i].Bool(); !exact || flag != (i%2 == 1) {
								t.Fatalf("fixed slot %d: %+v", i, values[i])
							}
						}
						if !variadic {
							continue
						}
						nilValue, nonNilValue := count == fixed, count > fixed
						if mode == 2 {
							nilValue, nonNilValue = variant == 0, variant == 1
						}
						if values[fixed].Nil != nilValue || values[fixed].NonNil != nonNilValue {
							t.Fatalf("pack fixed=%d count=%d mode=%d variant=%d value=%+v", fixed, count, mode, variant, values[fixed])
						}
					}
				}
			}
		}
	}
	if observations != 93 {
		t.Fatalf("observations=%d", observations)
	}
}
func TestLegacyTupleProducerCannotBecomeOneScalarArgument(t *testing.T) {
	q, owner, _, context := receivingModel(t, 2, 2, 1, 0, false)
	delete(q.callShapes, owner)
	inputs := []pdg.Parameter{{Position: 0, Role: pdg.ParameterRole, Variadic: pdg.BooleanFact{Status: pdg.Known}}}
	if _, known := q.CallValues(owner, inputs, context); known {
		t.Fatal("tuple without receiving shape admitted as scalar")
	}
}
